package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/api"
	"github.com/dunialabs/kimbap-core/internal/app"
	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/audit"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	"github.com/dunialabs/kimbap-core/internal/jobs"
	"github.com/dunialabs/kimbap-core/internal/observability"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/store"
	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	var (
		addr string
		port int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start connected-mode REST API server",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

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

			var serverOpts []api.ServerOption
			rt, buildErr := buildServeRuntime(cfg, st)
			if buildErr != nil {
				_, _ = fmt.Fprintln(os.Stderr, "warning: runtime unavailable, action execution disabled:", buildErr)
			} else {
				serverOpts = append(serverOpts, api.WithRuntime(rt))
			}

			srv := api.NewServer(listenAddr, st, serverOpts...)

			logger := observability.NewLogger(cfg.LogLevel, cfg.LogFormat)
			bgWorker := jobs.NewWorker(time.Minute, &storeApprovalExpirer{st: st}, logger)

			runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			bgWorker.Start(runCtx)
			defer bgWorker.Stop()

			if err := srv.Start(runCtx); err != nil {
				return fmt.Errorf("start api server: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "", "API listen address (default from config)")
	cmd.Flags().IntVar(&port, "port", 0, "API listen port override")

	return cmd
}

type storeApprovalExpirer struct {
	st *store.SQLStore
}

func (e *storeApprovalExpirer) ExpireStale(ctx context.Context) (int, error) {
	if e == nil || e.st == nil {
		return 0, nil
	}
	return e.st.ExpirePendingApprovals(ctx)
}

func buildServeRuntime(cfg *config.KimbapConfig, st *store.SQLStore) (*runtime.Runtime, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	vaultStore, err := initVaultStore(cfg)
	if err != nil {
		return nil, err
	}

	var writers []audit.Writer
	auditPath := strings.TrimSpace(cfg.Audit.Path)
	if auditPath != "" {
		jw, jwErr := audit.NewJSONLWriter(auditPath)
		if jwErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "warning: JSONL audit writer init failed:", jwErr)
		} else {
			writers = append(writers, jw)
		}
	}
	if st != nil {
		writers = append(writers, &storeAuditWriter{st: st})
	}
	var auditWriter runtime.AuditWriter
	if len(writers) > 0 {
		auditWriter = app.NewAuditWriterAdapter(audit.NewMultiWriter(writers...))
	}

	approvalMgr := approvals.NewApprovalManager(
		&storeApprovalStoreAdapter{st: st},
		buildNotifierFromConfig(cfg),
		defaultApprovalTTL,
	)

	var connStore connectors.ConnectorStore
	var connConfigs []connectors.ConnectorConfig
	if cs, csErr := openConnectorStore(cfg); csErr == nil {
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

	return app.BuildRuntime(app.RuntimeDeps{
		Config:           cfg,
		VaultStore:       vaultStore,
		ConnectorStore:   connStore,
		ConnectorConfigs: connConfigs,
		PolicyPath:       cfg.Policy.Path,
		SkillsDir:        cfg.Services.Dir,
		AuditWriter:      auditWriter,
		ApprovalManager:  app.NewApprovalManagerAdapter(approvalMgr),
	})
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

type storeApprovalStoreAdapter struct {
	st *store.SQLStore
}

func (a *storeApprovalStoreAdapter) Create(ctx context.Context, req *approvals.ApprovalRequest) error {
	if a.st == nil {
		return fmt.Errorf("store unavailable")
	}
	inputJSON := "{}"
	if req.Input != nil {
		if b, err := json.Marshal(req.Input); err == nil {
			inputJSON = string(b)
		}
	}
	return a.st.CreateApproval(ctx, &store.ApprovalRecord{
		ID:        req.ID,
		TenantID:  req.TenantID,
		RequestID: req.RequestID,
		AgentName: req.AgentName,
		Service:   req.Service,
		Action:    req.Action,
		Status:    string(req.Status),
		InputJSON: inputJSON,
		CreatedAt: req.CreatedAt,
		ExpiresAt: req.ExpiresAt,
	})
}

func (a *storeApprovalStoreAdapter) Get(ctx context.Context, id string) (*approvals.ApprovalRequest, error) {
	rec, err := a.st.GetApproval(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &approvals.ApprovalRequest{
		ID:         rec.ID,
		TenantID:   rec.TenantID,
		RequestID:  rec.RequestID,
		AgentName:  rec.AgentName,
		Service:    rec.Service,
		Action:     rec.Action,
		Status:     approvals.ApprovalStatus(rec.Status),
		CreatedAt:  rec.CreatedAt,
		ExpiresAt:  rec.ExpiresAt,
		ResolvedBy: rec.ResolvedBy,
		DenyReason: rec.Reason,
	}
	if rec.ResolvedAt != nil {
		result.ResolvedAt = rec.ResolvedAt
	}
	return result, nil
}

func (a *storeApprovalStoreAdapter) Update(ctx context.Context, req *approvals.ApprovalRequest) error {
	return a.st.UpdateApprovalStatus(ctx, req.ID, string(req.Status), req.ResolvedBy, req.DenyReason)
}

func (a *storeApprovalStoreAdapter) ListPending(ctx context.Context, tenantID string) ([]approvals.ApprovalRequest, error) {
	recs, err := a.st.ListApprovals(ctx, tenantID, "pending")
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, len(recs))
	for i, r := range recs {
		out[i] = approvals.ApprovalRequest{ID: r.ID, TenantID: r.TenantID, RequestID: r.RequestID, AgentName: r.AgentName, Service: r.Service, Action: r.Action, Status: approvals.ApprovalStatus(r.Status)}
	}
	return out, nil
}

func (a *storeApprovalStoreAdapter) ListAll(ctx context.Context, tenantID string, filter approvals.ApprovalFilter) ([]approvals.ApprovalRequest, error) {
	status := ""
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	recs, err := a.st.ListApprovals(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}
	out := make([]approvals.ApprovalRequest, len(recs))
	for i, r := range recs {
		out[i] = approvals.ApprovalRequest{ID: r.ID, TenantID: r.TenantID, RequestID: r.RequestID, AgentName: r.AgentName, Service: r.Service, Action: r.Action, Status: approvals.ApprovalStatus(r.Status)}
	}
	return out, nil
}

func (a *storeApprovalStoreAdapter) ExpireOld(ctx context.Context) (int, error) {
	return a.st.ExpirePendingApprovals(ctx)
}
