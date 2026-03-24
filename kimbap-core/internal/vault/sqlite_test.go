package vault

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	corecrypto "github.com/dunialabs/kimbap-core/internal/crypto"
)

func TestSQLiteStoreCreateAndGetRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "GITHUB_TOKEN", SecretTypeBearerToken, []byte("ghp_abc123"), map[string]string{"env": "dev"}, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	value, err := store.GetValue(ctx, "tenant-a", "GITHUB_TOKEN")
	if err != nil {
		t.Fatalf("get value: %v", err)
	}
	if !bytes.Equal(value, []byte("ghp_abc123")) {
		t.Fatalf("unexpected value")
	}
}

func TestSQLiteStoreDuplicateNameFails(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "DUPLICATE", SecretTypeAPIKey, []byte("one"), nil, "tester")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = store.Create(ctx, "tenant-a", "DUPLICATE", SecretTypeAPIKey, []byte("two"), nil, "tester")
	if !errors.Is(err, ErrSecretAlreadyExists) {
		t.Fatalf("expected ErrSecretAlreadyExists, got %v", err)
	}
}

func TestSQLiteStoreMetadataRetrieval(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "STRIPE_KEY", SecretTypeAPIKey, []byte("sk_live_123"), map[string]string{"service": "billing"}, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	meta, err := store.GetMeta(ctx, "tenant-a", "STRIPE_KEY")
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	if meta.Name != "STRIPE_KEY" {
		t.Fatalf("unexpected name: %s", meta.Name)
	}
	if meta.Type != SecretTypeAPIKey {
		t.Fatalf("unexpected type: %s", meta.Type)
	}
	if meta.Labels["service"] != "billing" {
		t.Fatalf("unexpected labels")
	}
}

func TestSQLiteStoreRotationCreatesNewVersion(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "ROTATE_ME", SecretTypePassword, []byte("v1"), nil, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	record, err := store.Rotate(ctx, "tenant-a", "ROTATE_ME", []byte("v2"), "rotator")
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	if record.CurrentVersion != 2 || record.VersionCount != 2 {
		t.Fatalf("unexpected version metadata: current=%d count=%d", record.CurrentVersion, record.VersionCount)
	}

	v1, err := store.GetVersion(ctx, "tenant-a", "ROTATE_ME", 1)
	if err != nil {
		t.Fatalf("get v1: %v", err)
	}
	v2, err := store.GetVersion(ctx, "tenant-a", "ROTATE_ME", 2)
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}

	if !bytes.Equal(v1, []byte("v1")) || !bytes.Equal(v2, []byte("v2")) {
		t.Fatalf("version payload mismatch")
	}
}

func TestSQLiteStoreDeleteRemovesAllVersions(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "DELETE_ME", SecretTypePassword, []byte("v1"), nil, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = store.Rotate(ctx, "tenant-a", "DELETE_ME", []byte("v2"), "tester")
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	if err := store.Delete(ctx, "tenant-a", "DELETE_ME"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = store.GetValue(ctx, "tenant-a", "DELETE_ME")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got %v", err)
	}
}

func TestSQLiteStoreTenantIsolation(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "SHARED_NAME", SecretTypeBearerToken, []byte("tenant-a-value"), nil, "tester")
	if err != nil {
		t.Fatalf("create tenant-a: %v", err)
	}

	_, err = store.GetValue(ctx, "tenant-b", "SHARED_NAME")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("expected tenant isolation error, got %v", err)
	}
}

func TestSQLiteStoreListWithFilters(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, _ = store.Create(ctx, "tenant-a", "A", SecretTypeAPIKey, []byte("a"), map[string]string{"env": "prod"}, "tester")
	_, _ = store.Create(ctx, "tenant-a", "B", SecretTypeBearerToken, []byte("b"), map[string]string{"env": "prod"}, "tester")
	_, _ = store.Create(ctx, "tenant-a", "C", SecretTypeAPIKey, []byte("c"), map[string]string{"env": "dev"}, "tester")

	secretType := SecretTypeAPIKey
	list, err := store.List(ctx, "tenant-a", ListOptions{Type: &secretType, Labels: map[string]string{"env": "prod"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(list) != 1 || list[0].Name != "A" {
		t.Fatalf("unexpected filtered list: %+v", list)
	}
}

func TestSQLiteStoreMarkUsedUpdatesTimestamp(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "USED_SECRET", SecretTypeRefreshToken, []byte("refresh"), nil, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	metaBefore, err := store.GetMeta(ctx, "tenant-a", "USED_SECRET")
	if err != nil {
		t.Fatalf("meta before: %v", err)
	}
	if metaBefore.LastUsedAt != nil {
		t.Fatalf("expected nil last_used_at before mark")
	}

	if err := store.MarkUsed(ctx, "tenant-a", "USED_SECRET"); err != nil {
		t.Fatalf("mark used: %v", err)
	}

	metaAfter, err := store.GetMeta(ctx, "tenant-a", "USED_SECRET")
	if err != nil {
		t.Fatalf("meta after: %v", err)
	}
	if metaAfter.LastUsedAt == nil {
		t.Fatalf("expected last_used_at to be updated")
	}
}

func TestSQLiteStoreUnicodeAndBinaryHandling(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	value := append([]byte("토큰🍙"), []byte{0x00, 0x01, 0xFE, 0xFF}...)
	_, err := store.Create(ctx, "tenant-a", "UNICODE_BINARY", SecretTypeBearerToken, value, nil, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	readValue, err := store.GetValue(ctx, "tenant-a", "UNICODE_BINARY")
	if err != nil {
		t.Fatalf("get value: %v", err)
	}

	if !bytes.Equal(readValue, value) {
		t.Fatalf("unicode/binary mismatch")
	}
}

func TestSQLiteStoreLargeSecretHandling(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	value := bytes.Repeat([]byte("abcd1234"), 256*1024)
	_, err := store.Create(ctx, "tenant-a", "LARGE_SECRET", SecretTypeCertificate, value, nil, "tester")
	if err != nil {
		t.Fatalf("create large: %v", err)
	}

	readValue, err := store.GetValue(ctx, "tenant-a", "LARGE_SECRET")
	if err != nil {
		t.Fatalf("get large: %v", err)
	}

	if !bytes.Equal(readValue, value) {
		t.Fatalf("large payload mismatch")
	}
}

func TestSQLiteStoreUpsertCreatesWhenNew(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	rec, err := store.Upsert(ctx, "tenant-a", "UPSERT_NEW", SecretTypeAPIKey, []byte("first"), nil, "tester")
	if err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	if rec.CurrentVersion != 1 {
		t.Fatalf("expected version 1, got %d", rec.CurrentVersion)
	}

	value, err := store.GetValue(ctx, "tenant-a", "UPSERT_NEW")
	if err != nil {
		t.Fatalf("get value: %v", err)
	}
	if !bytes.Equal(value, []byte("first")) {
		t.Fatalf("unexpected value")
	}
}

func TestSQLiteStoreUpsertRotatesWhenExists(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "UPSERT_EXIST", SecretTypeAPIKey, []byte("v1"), nil, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rec, err := store.Upsert(ctx, "tenant-a", "UPSERT_EXIST", SecretTypeAPIKey, []byte("v2"), nil, "tester")
	if err != nil {
		t.Fatalf("upsert overwrite: %v", err)
	}
	if rec.CurrentVersion != 2 {
		t.Fatalf("expected version 2, got %d", rec.CurrentVersion)
	}

	value, err := store.GetValue(ctx, "tenant-a", "UPSERT_EXIST")
	if err != nil {
		t.Fatalf("get value: %v", err)
	}
	if !bytes.Equal(value, []byte("v2")) {
		t.Fatalf("expected v2, got %s", string(value))
	}
}

func TestSQLiteStoreCreateSucceedsForAnyTenant(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	entry, err := store.Create(ctx, "tenant-other", "SOME_KEY", SecretTypeAPIKey, []byte("x"), nil, "tester")
	if err != nil {
		t.Fatalf("expected non-default tenant to use default KEK, got error: %v", err)
	}
	if entry.TenantID != "tenant-other" {
		t.Fatalf("expected tenant-other, got %s", entry.TenantID)
	}
}

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()

	masterKey, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	envelope, err := corecrypto.NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}
	tenantAKey, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate tenant-a key: %v", err)
	}
	if err := envelope.RotateKey("default", "tenant-a", tenantAKey); err != nil {
		t.Fatalf("configure tenant-a key: %v", err)
	}
	tenantBKey, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate tenant-b key: %v", err)
	}
	if err := envelope.RotateKey("default", "tenant-b", tenantBKey); err != nil {
		t.Fatalf("configure tenant-b key: %v", err)
	}

	dsn := filepath.Join(t.TempDir(), "vault.sqlite")
	store, err := OpenSQLiteStore(dsn, envelope)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	t.Cleanup(func() {
		_ = store.db.Close()
	})

	return store
}
