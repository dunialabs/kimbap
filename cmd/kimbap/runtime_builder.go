package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/dunialabs/kimbap/internal/app"
	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	corecrypto "github.com/dunialabs/kimbap/internal/crypto"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
)

var (
	initVaultStoreForBuild     = initVaultStore
	openRuntimeStoreForBuild   = openRuntimeStore
	openConnectorStoreForBuild = openConnectorStore
	closeVaultStoreForBuild    = closeVaultStoreIfPossible
	closeRuntimeStoreForBuild  = func(st *store.SQLStore) {
		if st != nil {
			_ = st.Close()
		}
	}
	closeConnectorStoreForBuild = closeConnectorStoreIfPossible
	buildRuntimeForConfig       = app.BuildRuntime
)

func closeVaultStoreIfPossible(st vault.Store) {
	if closer, ok := st.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func initVaultStore(cfg *config.KimbapConfig) (vault.Store, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Vault.Path), 0o700); err != nil {
		return nil, fmt.Errorf("create vault db dir: %w", err)
	}

	masterKey, err := resolveVaultMasterKey(cfg)
	if err != nil {
		return nil, err
	}
	envelope, err := corecrypto.NewEnvelopeService(masterKey)
	if err != nil {
		return nil, err
	}

	store, err := vault.OpenSQLiteStore(cfg.Vault.Path, envelope)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func resolveVaultMasterKey(cfg *config.KimbapConfig) ([]byte, error) {
	if decoded, err, present := decodeMasterKeyHexEnv(); present {
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}

	devEnabled := strings.EqualFold(strings.TrimSpace(cfg.Mode), "dev")
	if !devEnabled {
		if rawDev, ok := os.LookupEnv("KIMBAP_DEV"); ok {
			parsed, err := strconv.ParseBool(strings.TrimSpace(rawDev))
			if err != nil {
				return nil, fmt.Errorf("parse KIMBAP_DEV: %w", err)
			}
			devEnabled = parsed
		}
	}

	if !devEnabled {
		return nil, fmt.Errorf("vault master key is required: set KIMBAP_MASTER_KEY_HEX or enable dev mode (--mode dev or KIMBAP_DEV=true)")
	}
	devKeyPath := filepath.Join(cfg.DataDir, ".dev-master-key")
	existing, readErr := readPersistedDevMasterKey(devKeyPath)
	if readErr == nil {
		return existing, nil
	}
	if !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("read dev master key %s: %w", devKeyPath, readErr)
	}
	key, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("generate dev master key: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	f, err := os.OpenFile(devKeyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			existing, readErr := readPersistedDevMasterKey(devKeyPath)
			if readErr == nil {
				return existing, nil
			}
			return nil, fmt.Errorf("read dev master key %s after concurrent create: %w", devKeyPath, readErr)
		}
		return nil, fmt.Errorf("persist dev master key: %w", err)
	}
	_, writeErr := f.Write(key)
	_ = f.Close()
	if writeErr != nil {
		_ = os.Remove(devKeyPath)
		return nil, fmt.Errorf("write dev master key: %w", writeErr)
	}
	return key, nil
}

func buildRuntimeFromConfig(cfg *config.KimbapConfig) (*runtime.Runtime, error) {
	rt, _, err := buildRuntimeFromConfigWithCleanup(cfg)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func buildRuntimeFromConfigWithCleanup(cfg *config.KimbapConfig) (*runtime.Runtime, func(), error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("config is required to build runtime")
	}
	vaultStore, err := initVaultStoreForBuild(cfg)
	if err != nil {
		return nil, nil, err
	}

	var auditWriter runtime.AuditWriter
	var auditCloser interface{ Close() error }
	auditPath := strings.TrimSpace(cfg.Audit.Path)
	if auditPath != "" {
		jw, jwErr := audit.NewJSONLWriter(auditPath)
		if jwErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: audit writer init failed (%v), audit events will not be recorded\n", jwErr)
		} else {
			auditWriter = app.NewAuditWriterAdapter(audit.NewRedactingWriter(jw))
			auditCloser = jw
		}
	}

	var approvalManager runtime.ApprovalManager
	var runtimeStoreForCleanup *store.SQLStore
	if runtimeStore, rsErr := openRuntimeStoreForBuild(cfg); rsErr == nil {
		runtimeStoreForCleanup = runtimeStore
		approvalMgr := approvals.NewApprovalManager(
			&storeApprovalStoreAdapter{st: runtimeStore},
			buildNotifierFromConfig(cfg),
			defaultApprovalTTL,
		)
		approvalManager = app.NewApprovalManagerAdapter(approvalMgr)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s; approval-requiring actions will fail\n", unavailableMessage(componentApprovalStore, rsErr))
	}

	var connStore connectors.ConnectorStore
	var connConfigs []connectors.ConnectorConfig
	if cs, csErr := openConnectorStoreForBuild(cfg); csErr == nil {
		connStore = cs
		connConfigs = buildConnectorConfigs(cfg)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "warning: %s; oauth-backed credential resolution may fail\n", unavailableMessage(componentConnectorStore, csErr))
	}

	rt, buildErr := buildRuntimeForConfig(app.RuntimeDeps{
		Config:           cfg,
		VaultStore:       vaultStore,
		ConnectorStore:   connStore,
		ConnectorConfigs: connConfigs,
		PolicyPath:       cfg.Policy.Path,
		ServicesDir:      cfg.Services.Dir,
		AuditWriter:      auditWriter,
		ApprovalManager:  approvalManager,
		HeldStore:        runtimeStoreForCleanup,
	})
	if buildErr != nil {
		if auditCloser != nil {
			_ = auditCloser.Close()
		}
		closeVaultStoreForBuild(vaultStore)
		if runtimeStoreForCleanup != nil {
			closeRuntimeStoreForBuild(runtimeStoreForCleanup)
		}
		closeConnectorStoreForBuild(connStore)
		return nil, nil, buildErr
	}
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if auditCloser != nil {
				_ = auditCloser.Close()
			}
			closeVaultStoreForBuild(vaultStore)
			if runtimeStoreForCleanup != nil {
				closeRuntimeStoreForBuild(runtimeStoreForCleanup)
			}
			closeConnectorStoreForBuild(connStore)
		})
	}
	return rt, cleanup, nil
}

// buildNotifierFromConfig constructs an approval notifier from the kimbap configuration.
// If no notification adapters are configured, it returns a LogNotifier as fallback.
// Uses best-effort delivery: individual adapter failures are logged but do not block approval creation.
func buildNotifierFromConfig(cfg *config.KimbapConfig) approvals.Notifier {
	if cfg == nil {
		return &approvals.LogNotifier{}
	}
	n := cfg.Notifications
	var notifiers []approvals.Notifier

	if strings.TrimSpace(n.Slack.WebhookURL) != "" {
		notifiers = append(notifiers, approvals.NewSlackNotifier(n.Slack.WebhookURL))
	}
	if strings.TrimSpace(n.Telegram.BotToken) != "" && strings.TrimSpace(n.Telegram.ChatID) != "" {
		notifiers = append(notifiers, approvals.NewTelegramNotifier(n.Telegram.BotToken, n.Telegram.ChatID))
	}
	if strings.TrimSpace(n.Email.SMTPHost) != "" && strings.TrimSpace(n.Email.From) != "" && len(n.Email.To) > 0 {
		notifiers = append(notifiers, approvals.NewEmailNotifier(
			n.Email.SMTPHost, n.Email.SMTPPort, n.Email.From, n.Email.To,
			n.Email.Username, n.Email.Password,
		))
	}
	if strings.TrimSpace(n.Webhook.URL) != "" {
		notifiers = append(notifiers, approvals.NewWebhookNotifier(n.Webhook.URL, []byte(n.Webhook.SignKey)))
	}

	if len(notifiers) == 0 {
		return &approvals.LogNotifier{}
	}
	return approvals.NewMultiNotifier(notifiers...)
}
