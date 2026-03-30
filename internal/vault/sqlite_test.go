package vault

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	corecrypto "github.com/dunialabs/kimbap/internal/crypto"
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

func TestSQLiteStoreGetValueTrimsInputs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "TRIM_ME", SecretTypeBearerToken, []byte("value"), nil, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	value, err := store.GetValue(ctx, " tenant-a ", " TRIM_ME ")
	if err != nil {
		t.Fatalf("get value with trimmed inputs: %v", err)
	}
	if !bytes.Equal(value, []byte("value")) {
		t.Fatalf("unexpected value")
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

func TestSQLiteStoreListOffsetWithoutLimit(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, _ = store.Create(ctx, "tenant-a", "A", SecretTypeAPIKey, []byte("a"), nil, "tester")
	_, _ = store.Create(ctx, "tenant-a", "B", SecretTypeAPIKey, []byte("b"), nil, "tester")
	_, _ = store.Create(ctx, "tenant-a", "C", SecretTypeAPIKey, []byte("c"), nil, "tester")

	list, err := store.List(ctx, "tenant-a", ListOptions{Offset: 1})
	if err != nil {
		t.Fatalf("list with offset only: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 secrets after offset, got %d", len(list))
	}
	if list[0].Name != "B" || list[1].Name != "C" {
		t.Fatalf("unexpected offset list: %+v", list)
	}
}

func TestSQLiteStoreListLabelsPaginationUsesFilteredOrder(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, _ = store.Create(ctx, "tenant-a", "A", SecretTypeAPIKey, []byte("a"), map[string]string{"env": "dev"}, "tester")
	_, _ = store.Create(ctx, "tenant-a", "B", SecretTypeAPIKey, []byte("b"), map[string]string{"env": "prod"}, "tester")
	_, _ = store.Create(ctx, "tenant-a", "C", SecretTypeAPIKey, []byte("c"), map[string]string{"env": "dev"}, "tester")
	_, _ = store.Create(ctx, "tenant-a", "D", SecretTypeAPIKey, []byte("d"), map[string]string{"env": "prod"}, "tester")

	list, err := store.List(ctx, "tenant-a", ListOptions{
		Labels: map[string]string{"env": "prod"},
		Offset: 1,
		Limit:  1,
	})
	if err != nil {
		t.Fatalf("list with label pagination: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one filtered record, got %d", len(list))
	}
	if list[0].Name != "D" {
		t.Fatalf("expected filtered offset record D, got %s", list[0].Name)
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

func TestSQLiteStoreUpsertRequiresCreatedByWhenUpdatingExistingSecret(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Create(ctx, "tenant-a", "UPSERT_CREATED_BY", SecretTypeAPIKey, []byte("v1"), nil, "tester"); err != nil {
		t.Fatalf("create seed: %v", err)
	}

	if _, err := store.Upsert(ctx, "tenant-a", "UPSERT_CREATED_BY", SecretTypeAPIKey, []byte("v2"), nil, "   "); err == nil || err.Error() != "createdBy is required" {
		t.Fatalf("expected createdBy validation error, got %v", err)
	}

	meta, err := store.GetMeta(ctx, "tenant-a", "UPSERT_CREATED_BY")
	if err != nil {
		t.Fatalf("get meta after failed upsert: %v", err)
	}
	if meta.CurrentVersion != 1 || meta.VersionCount != 1 {
		t.Fatalf("expected failed upsert to preserve version metadata, current=%d count=%d", meta.CurrentVersion, meta.VersionCount)
	}

	value, err := store.GetValue(ctx, "tenant-a", "UPSERT_CREATED_BY")
	if err != nil {
		t.Fatalf("get value after failed upsert: %v", err)
	}
	if !bytes.Equal(value, []byte("v1")) {
		t.Fatalf("expected failed upsert to preserve original value, got %q", string(value))
	}
}

func TestSQLiteStoreUpsertUpdatesMetadataWhenExists(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "UPSERT_META", SecretTypeAPIKey, []byte("v1"), map[string]string{"env": "dev"}, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rec, err := store.Upsert(ctx, "tenant-a", "UPSERT_META", SecretTypeBearerToken, []byte("v2"), map[string]string{"env": "prod", "team": "platform"}, "tester")
	if err != nil {
		t.Fatalf("upsert metadata: %v", err)
	}
	if rec.Type != SecretTypeBearerToken {
		t.Fatalf("expected type %s, got %s", SecretTypeBearerToken, rec.Type)
	}
	if rec.Labels["env"] != "prod" || rec.Labels["team"] != "platform" {
		t.Fatalf("expected updated labels, got %+v", rec.Labels)
	}

	value, err := store.GetValue(ctx, "tenant-a", "UPSERT_META")
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

func TestSQLiteStoreCreateReturnsDetachedLabelsMap(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	labels := map[string]string{"env": "dev"}

	rec, err := store.Create(ctx, "tenant-a", "DETACHED_LABELS", SecretTypeAPIKey, []byte("v1"), labels, "tester")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	labels["env"] = "prod"
	if rec.Labels["env"] != "dev" {
		t.Fatalf("expected returned labels to be detached from caller map, got %+v", rec.Labels)
	}
}

func TestSQLiteStoreUpsertReturnsDetachedLabelsMap(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, "tenant-a", "UPSERT_DETACHED", SecretTypeAPIKey, []byte("v1"), map[string]string{"env": "dev"}, "tester")
	if err != nil {
		t.Fatalf("create seed: %v", err)
	}

	labels := map[string]string{"env": "prod"}
	rec, err := store.Upsert(ctx, "tenant-a", "UPSERT_DETACHED", SecretTypeAPIKey, []byte("v2"), labels, "tester")
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	labels["env"] = "staging"
	if rec.Labels["env"] != "prod" {
		t.Fatalf("expected upsert result labels detached from caller map, got %+v", rec.Labels)
	}
}

func TestSQLiteStoreRotateFailsOnCorruptedLabelsJSON(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Create(ctx, "tenant-a", "ROTATE_BAD_LABELS", SecretTypeAPIKey, []byte("v1"), map[string]string{"env": "dev"}, "tester"); err != nil {
		t.Fatalf("create seed: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE secrets SET labels = ? WHERE tenant_id = ? AND name = ?`, "{", "tenant-a", "ROTATE_BAD_LABELS"); err != nil {
		t.Fatalf("corrupt labels JSON: %v", err)
	}

	if _, err := store.Rotate(ctx, "tenant-a", "ROTATE_BAD_LABELS", []byte("v2"), "tester"); err == nil {
		t.Fatal("expected rotate to fail when labels JSON is corrupted")
	}

	var currentVersion, versionCount int
	if err := store.db.QueryRowContext(ctx, `SELECT current_version, version_count FROM secrets WHERE tenant_id = ? AND name = ?`, "tenant-a", "ROTATE_BAD_LABELS").Scan(&currentVersion, &versionCount); err != nil {
		t.Fatalf("query version metadata after failed rotate: %v", err)
	}
	if currentVersion != 1 || versionCount != 1 {
		t.Fatalf("expected failed rotate to keep version metadata unchanged, current=%d count=%d", currentVersion, versionCount)
	}

	value, err := store.GetValue(ctx, "tenant-a", "ROTATE_BAD_LABELS")
	if err != nil {
		t.Fatalf("get value after failed rotate: %v", err)
	}
	if !bytes.Equal(value, []byte("v1")) {
		t.Fatalf("expected failed rotate to preserve previous value, got %q", string(value))
	}
}

func TestSQLiteStoreUpsertFailsOnCorruptedExistingLabelsJSON(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Create(ctx, "tenant-a", "UPSERT_BAD_LABELS", SecretTypeAPIKey, []byte("v1"), map[string]string{"env": "dev"}, "tester"); err != nil {
		t.Fatalf("create seed: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE secrets SET labels = ? WHERE tenant_id = ? AND name = ?`, "{", "tenant-a", "UPSERT_BAD_LABELS"); err != nil {
		t.Fatalf("corrupt labels JSON: %v", err)
	}

	if _, err := store.Upsert(ctx, "tenant-a", "UPSERT_BAD_LABELS", SecretTypeAPIKey, []byte("v2"), nil, "tester"); err == nil {
		t.Fatal("expected upsert to fail when existing labels JSON is corrupted")
	}

	var currentVersion, versionCount int
	if err := store.db.QueryRowContext(ctx, `SELECT current_version, version_count FROM secrets WHERE tenant_id = ? AND name = ?`, "tenant-a", "UPSERT_BAD_LABELS").Scan(&currentVersion, &versionCount); err != nil {
		t.Fatalf("query version metadata after failed upsert: %v", err)
	}
	if currentVersion != 1 || versionCount != 1 {
		t.Fatalf("expected failed upsert to keep version metadata unchanged, current=%d count=%d", currentVersion, versionCount)
	}

	value, err := store.GetValue(ctx, "tenant-a", "UPSERT_BAD_LABELS")
	if err != nil {
		t.Fatalf("get value after failed upsert: %v", err)
	}
	if !bytes.Equal(value, []byte("v1")) {
		t.Fatalf("expected failed upsert to preserve previous value, got %q", string(value))
	}
}

func TestSQLiteStoreRotateDeactivatesOldVersions(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Create(ctx, "tenant-a", "ROTATE_ACTIVE", SecretTypeAPIKey, []byte("v1"), map[string]string{"env": "dev"}, "tester"); err != nil {
		t.Fatalf("create seed: %v", err)
	}

	rec, err := store.Rotate(ctx, "tenant-a", "ROTATE_ACTIVE", []byte("v2"), "tester")
	if err != nil {
		t.Fatalf("rotate to v2: %v", err)
	}
	if rec.CurrentVersion != 2 {
		t.Fatalf("expected current_version=2, got %d", rec.CurrentVersion)
	}

	// After rotate, exactly one version should be active and it should match current_version.
	var activeCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secret_versions WHERE secret_id = ? AND active = 1`, rec.ID).Scan(&activeCount); err != nil {
		t.Fatalf("count active versions: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected exactly 1 active version after rotate, got %d", activeCount)
	}

	var activeVersion int
	if err := store.db.QueryRowContext(ctx, `SELECT version FROM secret_versions WHERE secret_id = ? AND active = 1`, rec.ID).Scan(&activeVersion); err != nil {
		t.Fatalf("query active version: %v", err)
	}
	if activeVersion != rec.CurrentVersion {
		t.Fatalf("active version (%d) does not match current_version (%d)", activeVersion, rec.CurrentVersion)
	}

	// Rotate again to v3 and re-verify.
	rec2, err := store.Rotate(ctx, "tenant-a", "ROTATE_ACTIVE", []byte("v3"), "tester")
	if err != nil {
		t.Fatalf("rotate to v3: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM secret_versions WHERE secret_id = ? AND active = 1`, rec2.ID).Scan(&activeCount); err != nil {
		t.Fatalf("count active versions after v3: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected exactly 1 active version after second rotate, got %d", activeCount)
	}

	value, err := store.GetValue(ctx, "tenant-a", "ROTATE_ACTIVE")
	if err != nil {
		t.Fatalf("get value after rotations: %v", err)
	}
	if !bytes.Equal(value, []byte("v3")) {
		t.Fatalf("expected v3, got %s", string(value))
	}
}

func TestOpenSQLiteStoreAppliesConnectionPragmas(t *testing.T) {
	masterKey, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	envelope, err := corecrypto.NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}
	tenantKey, err := corecrypto.GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate tenant key: %v", err)
	}
	if err := envelope.RotateKey("default", "tenant-a", tenantKey); err != nil {
		t.Fatalf("configure tenant key: %v", err)
	}

	store, err := OpenSQLiteStore(filepath.Join(t.TempDir(), "vault-pragmas.sqlite"), envelope)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	stats := store.db.Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("expected max open connections to be 1, got %d", stats.MaxOpenConnections)
	}

	var busyTimeout int
	if err := store.db.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout pragma: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("expected busy_timeout=5000, got %d", busyTimeout)
	}

	var foreignKeys int
	if err := store.db.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign_keys pragma enabled, got %d", foreignKeys)
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
