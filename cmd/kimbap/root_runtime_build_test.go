package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/app"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
)

type buildFailConnectorStore struct {
	closed int
}

type buildFailVaultStore struct {
	closed int
}

func (s *buildFailVaultStore) Create(context.Context, string, string, vault.SecretType, []byte, map[string]string, string) (*vault.SecretRecord, error) {
	return nil, nil
}

func (s *buildFailVaultStore) Upsert(context.Context, string, string, vault.SecretType, []byte, map[string]string, string) (*vault.SecretRecord, error) {
	return nil, nil
}

func (s *buildFailVaultStore) GetMeta(context.Context, string, string) (*vault.SecretRecord, error) {
	return nil, nil
}

func (s *buildFailVaultStore) GetValue(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (s *buildFailVaultStore) List(context.Context, string, vault.ListOptions) ([]vault.SecretRecord, error) {
	return nil, nil
}

func (s *buildFailVaultStore) Delete(context.Context, string, string) error {
	return nil
}

func (s *buildFailVaultStore) Rotate(context.Context, string, string, []byte, string) (*vault.SecretRecord, error) {
	return nil, nil
}

func (s *buildFailVaultStore) GetVersion(context.Context, string, string, int) ([]byte, error) {
	return nil, nil
}

func (s *buildFailVaultStore) MarkUsed(context.Context, string, string) error {
	return nil
}

func (s *buildFailVaultStore) Exists(context.Context, string, string) (bool, error) {
	return false, nil
}

func (s *buildFailVaultStore) Close() error {
	s.closed++
	return nil
}

func (s *buildFailConnectorStore) Save(context.Context, *connectors.ConnectorState) error {
	return nil
}

func (s *buildFailConnectorStore) Get(context.Context, string, string) (*connectors.ConnectorState, error) {
	return nil, nil
}

func (s *buildFailConnectorStore) List(context.Context, string) ([]connectors.ConnectorState, error) {
	return nil, nil
}

func (s *buildFailConnectorStore) Delete(context.Context, string, string) error {
	return nil
}

func (s *buildFailConnectorStore) ResolveCredential(context.Context, string, actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	return nil, nil
}

func (s *buildFailConnectorStore) Close() error {
	s.closed++
	return nil
}

func TestBuildRuntimeFromConfigClosesStoresOnBuildFailure(t *testing.T) {
	dataDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Mode = "dev"
	config.ApplyDataDirOverride(cfg, dataDir)
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = filepath.Join(dataDir, "runtime.db")

	runtimeStore, err := store.OpenSQLiteStore(filepath.Join(dataDir, "approval.db"))
	if err != nil {
		t.Fatalf("open runtime store: %v", err)
	}
	t.Cleanup(func() {
		_ = runtimeStore.Close()
	})

	connectorStore := &buildFailConnectorStore{}
	vaultStore := &buildFailVaultStore{}

	prevInitVault := initVaultStoreForBuild
	prevOpenRuntime := openRuntimeStoreForBuild
	prevOpenConnector := openConnectorStoreForBuild
	prevCloseVault := closeVaultStoreForBuild
	prevCloseRuntime := closeRuntimeStoreForBuild
	prevCloseConnector := closeConnectorStoreForBuild
	prevBuildRuntime := buildRuntimeForConfig
	defer func() {
		initVaultStoreForBuild = prevInitVault
		openRuntimeStoreForBuild = prevOpenRuntime
		openConnectorStoreForBuild = prevOpenConnector
		closeVaultStoreForBuild = prevCloseVault
		closeRuntimeStoreForBuild = prevCloseRuntime
		closeConnectorStoreForBuild = prevCloseConnector
		buildRuntimeForConfig = prevBuildRuntime
	}()

	vaultCloseCalls := 0
	runtimeCloseCalls := 0
	connectorCloseCalls := 0

	initVaultStoreForBuild = func(*config.KimbapConfig) (vault.Store, error) {
		return vaultStore, nil
	}
	openRuntimeStoreForBuild = func(*config.KimbapConfig) (*store.SQLStore, error) {
		return runtimeStore, nil
	}
	openConnectorStoreForBuild = func(*config.KimbapConfig) (connectors.ConnectorStore, error) {
		return connectorStore, nil
	}
	closeVaultStoreForBuild = func(st vault.Store) {
		vaultCloseCalls++
		if closer, ok := st.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	closeRuntimeStoreForBuild = func(st *store.SQLStore) {
		runtimeCloseCalls++
		if st != nil {
			_ = st.Close()
		}
	}
	closeConnectorStoreForBuild = func(st connectors.ConnectorStore) {
		connectorCloseCalls++
		closeConnectorStoreIfPossible(st)
	}
	buildRuntimeForConfig = func(app.RuntimeDeps) (*runtimepkg.Runtime, error) {
		return nil, errors.New("forced runtime build failure")
	}

	_, err = buildRuntimeFromConfig(cfg)
	if err == nil {
		t.Fatal("expected build failure")
	}
	if vaultCloseCalls != 1 {
		t.Fatalf("expected vault store close once, got %d", vaultCloseCalls)
	}
	if vaultStore.closed != 1 {
		t.Fatalf("expected vault store Close once, got %d", vaultStore.closed)
	}
	if runtimeCloseCalls != 1 {
		t.Fatalf("expected runtime store close once, got %d", runtimeCloseCalls)
	}
	if connectorCloseCalls != 1 {
		t.Fatalf("expected connector store close once, got %d", connectorCloseCalls)
	}
	if connectorStore.closed != 1 {
		t.Fatalf("expected connector store Close once, got %d", connectorStore.closed)
	}
}
