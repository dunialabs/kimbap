package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap/internal/api"
	"github.com/dunialabs/kimbap/internal/app"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	"github.com/dunialabs/kimbap/internal/jobs"
	"github.com/dunialabs/kimbap/internal/observability"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/spf13/cobra"
)

const webhookEventPersistTimeout = 2 * time.Second
const webhookEventPersistQueueSize = 256
const webhookEventPersistEnqueueWait = 250 * time.Millisecond

func serverDisplayURL(addr string) string {
	if addr == "" {
		return "http://localhost:8080"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://localhost:8080"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func consoleDisplayURL(addr string) string {
	return serverDisplayURL(addr) + "/console"
}

func newServeCommand() *cobra.Command {
	var (
		addr        string
		port        int
		withConsole bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start connected-mode REST API server",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			st, err := openRuntimeStore(cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			listenAddr := strings.TrimSpace(addr)
			if listenAddr == "" {
				listenAddr = strings.TrimSpace(cfg.ListenAddr)
			}
			if port > 0 {
				listenAddr = withPort(listenAddr, port)
			}

			// vaultStore is intentionally not deferred-closed: it is passed into the
			// runtime and API server, remaining open for the process lifetime.
			vaultStore, err := initVaultStore(cfg)
			if err != nil {
				return err
			}
			defer closeVaultStoreIfPossible(vaultStore)

			var rt *runtime.Runtime
			rt, runtimeCleanup, buildErr := buildServeRuntime(cfg, st, vaultStore)
			if buildErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, "warning: runtime unavailable, action execution disabled:", buildErr)
			} else {
				defer runtimeCleanup()
			}

			enableConsole := withConsole || cfg.Console.Enabled
			dispatcher := webhooks.NewDispatcher()
			cleanupWebhookSink, err := configureWebhookDispatcherFromStore(runCtx, dispatcher, st)
			if err != nil {
				return err
			}
			defer cleanupWebhookSink()
			srv := api.NewServer(listenAddr, st, buildServeServerOptions(rt, vaultStore, dispatcher, enableConsole)...)

			logger := observability.NewLogger(cfg.LogLevel, cfg.LogFormat)
			bgWorker := jobs.NewWorker(time.Minute, &storeApprovalExpirer{st: st, dispatcher: dispatcher}, logger)

			bgWorker.Start(runCtx)
			defer bgWorker.Stop()

			_, _ = fmt.Fprintf(os.Stdout, "Starting on %s\n", serverDisplayURL(listenAddr))
			if enableConsole {
				_, _ = fmt.Fprintf(os.Stdout, "Console: %s\n", consoleDisplayURL(listenAddr))
			}

			if err := srv.Start(runCtx); err != nil {
				return fmt.Errorf("start api server: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "", "API listen address (default from config)")
	cmd.Flags().IntVar(&port, "port", 0, "API listen port override")
	cmd.Flags().BoolVar(&withConsole, "console", false, "serve embedded console UI at /console")

	return cmd
}

func buildServeServerOptions(rt *runtime.Runtime, vaultStore vault.Store, dispatcher *webhooks.Dispatcher, enableConsole bool) []api.ServerOption {
	if dispatcher == nil {
		dispatcher = webhooks.NewDispatcher()
	}
	opts := []api.ServerOption{api.WithWebhookDispatcher(dispatcher)}
	if enableConsole {
		opts = append(opts, api.WithConsole())
	}
	if vaultStore != nil {
		opts = append(opts, api.WithVaultStore(vaultStore))
	}
	if rt != nil {
		opts = append(opts, api.WithRuntime(rt))
	}
	return opts
}

func configureWebhookDispatcherFromStore(ctx context.Context, dispatcher *webhooks.Dispatcher, st *store.SQLStore) (func(), error) {
	if dispatcher == nil || st == nil {
		return func() {}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	subs, err := st.ListWebhookSubscriptions(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("load webhook subscriptions: %w", err)
	}
	for _, sub := range subs {
		dispatcher.Subscribe(webhookRecordToSubscription(sub))
	}

	events, err := st.ListWebhookEvents(ctx, "", 1000)
	if err != nil {
		return nil, fmt.Errorf("load webhook events: %w", err)
	}
	if len(events) > 0 {
		hydrated := make([]webhooks.Event, 0, len(events))
		for _, rec := range events {
			hydrated = append(hydrated, webhookEventRecordToEvent(rec))
		}
		dispatcher.ReplaceRecentEvents(hydrated)
	}

	sinkCtx, cancelSink := context.WithCancel(ctx)
	var sinkGateMu sync.Mutex
	var sinkWG sync.WaitGroup
	sinkClosing := false

	persistEvent := func(event webhooks.Event) {
		persistCtx, cancel := context.WithTimeout(context.Background(), webhookEventPersistTimeout)
		err := st.WriteWebhookEvent(persistCtx, webhookEventToRecord(event))
		cancel()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: persist webhook event failed: %v\n", err)
		}
	}

	eventSinkQueue := make(chan webhooks.Event, webhookEventPersistQueueSize)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		for {
			select {
			case event := <-eventSinkQueue:
				persistEvent(event)
			case <-sinkCtx.Done():
				for {
					select {
					case event := <-eventSinkQueue:
						persistEvent(event)
					default:
						return
					}
				}
			}
		}
	}()

	dispatcher.SetEventSink(func(event webhooks.Event) {
		sinkGateMu.Lock()
		if sinkClosing {
			sinkGateMu.Unlock()
			return
		}
		sinkWG.Add(1)
		sinkGateMu.Unlock()
		defer sinkWG.Done()

		select {
		case <-sinkCtx.Done():
			return
		default:
		}

		select {
		case eventSinkQueue <- event:
			return
		default:
		}

		t := time.NewTimer(webhookEventPersistEnqueueWait)
		defer t.Stop()
		select {
		case eventSinkQueue <- event:
		case <-t.C:
			_, _ = fmt.Fprintf(os.Stderr, "warning: dropping webhook event persistence due to saturated queue: %s\n", event.ID)
		case <-sinkCtx.Done():
			return
		}
	})

	return func() {
		sinkGateMu.Lock()
		sinkClosing = true
		sinkGateMu.Unlock()
		sinkWG.Wait()
		cancelSink()
		<-workerDone
	}, nil
}

func webhookRecordToSubscription(rec store.WebhookSubscriptionRecord) webhooks.Subscription {
	return webhooks.Subscription{
		ID:       rec.ID,
		URL:      rec.URL,
		Secret:   rec.Secret,
		Events:   parseWebhookEventTypes(rec.EventsJSON),
		TenantID: rec.TenantID,
		Active:   rec.Active,
	}
}

func parseWebhookEventTypes(raw string) []webhooks.EventType {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var events []webhooks.EventType
	if err := json.Unmarshal([]byte(raw), &events); err != nil {
		return nil
	}
	return events
}

func webhookEventRecordToEvent(rec store.WebhookEventRecord) webhooks.Event {
	var data map[string]any
	if strings.TrimSpace(rec.DataJSON) != "" {
		_ = json.Unmarshal([]byte(rec.DataJSON), &data)
	}
	return webhooks.Event{
		ID:        rec.ID,
		Type:      webhooks.EventType(rec.Type),
		TenantID:  rec.TenantID,
		Timestamp: rec.Timestamp,
		Data:      data,
	}
}

func webhookEventToRecord(event webhooks.Event) *store.WebhookEventRecord {
	dataJSON := "{}"
	if event.Data != nil {
		if b, err := json.Marshal(event.Data); err == nil {
			dataJSON = string(b)
		}
	}
	return &store.WebhookEventRecord{
		ID:        event.ID,
		TenantID:  event.TenantID,
		Type:      string(event.Type),
		Timestamp: event.Timestamp,
		DataJSON:  dataJSON,
	}
}

type storeApprovalExpirer struct {
	st         *store.SQLStore
	dispatcher *webhooks.Dispatcher
}

func (e *storeApprovalExpirer) ExpireStale(ctx context.Context) (int, error) {
	if e == nil || e.st == nil {
		return 0, nil
	}
	return expirePendingApprovalsWithSideEffects(ctx, e.st, "", func(approval store.ApprovalRecord) {
		if e.dispatcher == nil {
			return
		}
		e.dispatcher.EmitForTenant(approval.TenantID, webhooks.EventApprovalExpired, map[string]any{
			"approval_id": approval.ID,
			"tenant_id":   approval.TenantID,
			"request_id":  approval.RequestID,
			"agent_name":  approval.AgentName,
			"service":     approval.Service,
			"action":      approval.Action,
			"status":      "expired",
		})
	})
}

func buildServeRuntime(cfg *config.KimbapConfig, st *store.SQLStore, vaultStore vault.Store) (*runtime.Runtime, func(), error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is required")
	}

	var writers []audit.Writer
	auditPath := strings.TrimSpace(cfg.Audit.Path)
	if auditPath != "" {
		jw, jwErr := audit.NewJSONLWriter(auditPath)
		if jwErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "warning: JSONL audit writer init failed:", jwErr)
		} else {
			writers = append(writers, audit.NewRedactingWriter(jw))
		}
	}
	if st != nil {
		writers = append(writers, audit.NewRedactingWriter(&storeAuditWriter{st: st}))
	}
	var auditWriter runtime.AuditWriter
	var auditCloser interface{ Close() error }
	if len(writers) > 0 {
		multiWriter := audit.NewMultiWriter(writers...)
		auditWriter = app.NewAuditWriterAdapter(multiWriter)
		auditCloser = multiWriter
	}

	approvalMgr := approvals.NewApprovalManager(
		&storeApprovalStoreAdapter{st: st},
		buildNotifierFromConfig(cfg),
		defaultApprovalTTL,
	)

	var connStore connectors.ConnectorStore
	var connConfigs []connectors.ConnectorConfig
	if cs, csErr := openConnectorStore(cfg); csErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: connector store unavailable, OAuth credential resolution disabled: %v\n", csErr)
	} else {
		connStore = cs
		for _, prov := range providers.ListProviders() {
			creds := resolveOAuthCreds(cfg, prov.ID)
			connConfigs = append(connConfigs, connectors.ConnectorConfig{
				Name:         prov.ID,
				Provider:     prov.ID,
				ClientID:     creds.ClientID,
				ClientSecret: creds.ClientSecret,
				AuthMethod:   creds.AuthMethod,
				TokenURL:     prov.TokenEndpoint,
				DeviceURL:    prov.DeviceEndpoint,
				Scopes:       prov.DefaultScopes,
			})
		}
	}

	rt, err := app.BuildRuntime(app.RuntimeDeps{
		Config:           cfg,
		VaultStore:       vaultStore,
		ConnectorStore:   connStore,
		ConnectorConfigs: connConfigs,
		PolicyStore:      st,
		PolicyPath:       cfg.Policy.Path,
		ServicesDir:      cfg.Services.Dir,
		AuditWriter:      auditWriter,
		ApprovalManager:  app.NewApprovalManagerAdapter(approvalMgr),
		HeldStore:        st,
	})
	if err != nil {
		if auditCloser != nil {
			_ = auditCloser.Close()
		}
		closeConnectorStoreIfPossible(connStore)
		return nil, nil, err
	}
	cleanup := func() {
		if auditCloser != nil {
			_ = auditCloser.Close()
		}
		closeConnectorStoreIfPossible(connStore)
	}
	return rt, cleanup, nil
}

type storeAuditWriter struct {
	st *store.SQLStore
}

func (w *storeAuditWriter) Write(ctx context.Context, event audit.AuditEvent) error {
	if w == nil || w.st == nil {
		return nil
	}
	errCode, errMsg := "", ""
	if event.Error != nil {
		errCode = event.Error.Code
		errMsg = event.Error.Message
	}
	metaJSON := "{}"
	if event.Meta != nil {
		if b, err := json.Marshal(event.Meta); err == nil {
			metaJSON = string(b)
		}
	}
	inputJSON := "{}"
	if event.Input != nil {
		if b, err := json.Marshal(event.Input); err == nil {
			inputJSON = string(b)
		}
	}
	return w.st.WriteAuditEvent(ctx, &store.AuditRecord{
		Timestamp:      event.Timestamp,
		RequestID:      event.RequestID,
		TraceID:        event.TraceID,
		TenantID:       event.TenantID,
		PrincipalID:    event.PrincipalID,
		AgentName:      event.AgentName,
		Service:        event.Service,
		Action:         event.Action,
		Mode:           event.Mode,
		Status:         string(event.Status),
		PolicyDecision: event.PolicyDecision,
		DurationMS:     event.DurationMS,
		ErrorCode:      errCode,
		ErrorMessage:   errMsg,
		InputJSON:      inputJSON,
		MetaJSON:       metaJSON,
	})
}

func (w *storeAuditWriter) Close() error { return nil }
