package vault

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	corecrypto "github.com/dunialabs/kimbap-core/internal/crypto"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var (
	ErrSecretNotFound      = errors.New("secret not found")
	ErrSecretAlreadyExists = errors.New("secret already exists")
)

type SQLiteStore struct {
	db       *sql.DB
	envelope *corecrypto.EnvelopeService
}

func NewSQLiteStore(db *sql.DB, envelope *corecrypto.EnvelopeService) (*SQLiteStore, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	if envelope == nil {
		return nil, errors.New("envelope service is required")
	}

	s := &SQLiteStore{db: db, envelope: envelope}
	if err := s.initSchema(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func OpenSQLiteStore(dsn string, envelope *corecrypto.EnvelopeService) (*SQLiteStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("dsn is required")
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	store, err := NewSQLiteStore(db, envelope)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Create(ctx context.Context, tenantID string, name string, secretType SecretType, plaintext []byte, labels map[string]string, createdBy string) (*SecretRecord, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenant ID is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("name is required")
	}
	if strings.TrimSpace(createdBy) == "" {
		return nil, errors.New("createdBy is required")
	}

	envelope, err := s.encryptForTenant(tenantID, plaintext)
	if err != nil {
		return nil, err
	}

	labelsJSON, err := marshalLabels(labels)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	secretID := uuid.NewString()
	versionID := uuid.NewString()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO secrets (
			id, tenant_id, name, type, labels, created_at, updated_at, version_count, current_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, 1, 1)
	`, secretID, tenantID, name, string(secretType), labelsJSON, now, now)
	if err != nil {
		if isSQLiteUniqueViolation(err) {
			return nil, ErrSecretAlreadyExists
		}
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO secret_versions (
			id, secret_id, version, ciphertext, nonce, salt, key_id, algorithm, created_at, created_by, active, wrapped_dek, dek_nonce
		) VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, versionID, secretID, envelope.Ciphertext, envelope.Nonce, envelope.Salt, envelope.KeyID, envelope.Algorithm, now, createdBy, envelope.WrappedDEK, envelope.DEKNonce)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetMeta(ctx, tenantID, name)
}

func (s *SQLiteStore) Upsert(ctx context.Context, tenantID string, name string, secretType SecretType, plaintext []byte, labels map[string]string, createdBy string) (*SecretRecord, error) {
	rec, err := s.Create(ctx, tenantID, name, secretType, plaintext, labels, createdBy)
	if err == ErrSecretAlreadyExists {
		return s.Rotate(ctx, tenantID, name, plaintext, createdBy)
	}
	return rec, err
}

func (s *SQLiteStore) GetMeta(ctx context.Context, tenantID string, name string) (*SecretRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, tenant_id, name, type, labels, created_at, updated_at,
			last_used_at, rotated_at, version_count, current_version
		FROM secrets
		WHERE tenant_id = ? AND name = ?
	`, tenantID, name)

	rec, err := scanSecretRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}

	return rec, nil
}

func (s *SQLiteStore) GetValue(ctx context.Context, tenantID string, name string) ([]byte, error) {
	envelope, err := s.currentEnvelope(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	return s.envelope.Decrypt(envelope)
}

func (s *SQLiteStore) List(ctx context.Context, tenantID string, opts ListOptions) ([]SecretRecord, error) {
	args := []any{tenantID}
	query := `
		SELECT
			id, tenant_id, name, type, labels, created_at, updated_at,
			last_used_at, rotated_at, version_count, current_version
		FROM secrets
		WHERE tenant_id = ?
	`

	if opts.Type != nil {
		query += " AND type = ?"
		args = append(args, string(*opts.Type))
	}

	query += " ORDER BY name ASC"
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	} else if opts.Offset > 0 {
		query += " LIMIT -1"
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]SecretRecord, 0)
	for rows.Next() {
		rec, err := scanSecretRecord(rows)
		if err != nil {
			return nil, err
		}
		if matchesLabels(rec.Labels, opts.Labels) {
			records = append(records, *rec)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (s *SQLiteStore) Delete(ctx context.Context, tenantID string, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var secretID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM secrets WHERE tenant_id = ? AND name = ?`, tenantID, name).Scan(&secretID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSecretNotFound
		}
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM secret_versions WHERE secret_id = ?`, secretID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM secrets WHERE id = ?`, secretID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStore) Rotate(ctx context.Context, tenantID string, name string, newPlaintext []byte, rotatedBy string) (*SecretRecord, error) {
	if strings.TrimSpace(rotatedBy) == "" {
		return nil, errors.New("rotatedBy is required")
	}

	envelope, err := s.encryptForTenant(tenantID, newPlaintext)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var secretID string
	var currentVersion int
	if err := tx.QueryRowContext(ctx, `SELECT id, current_version FROM secrets WHERE tenant_id = ? AND name = ?`, tenantID, name).Scan(&secretID, &currentVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}

	newVersion := currentVersion + 1
	versionID := uuid.NewString()

	if _, err := tx.ExecContext(ctx, `UPDATE secret_versions SET active = 0 WHERE secret_id = ?`, secretID); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_versions (
			id, secret_id, version, ciphertext, nonce, salt, key_id, algorithm, created_at, created_by, active, wrapped_dek, dek_nonce
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, versionID, secretID, newVersion, envelope.Ciphertext, envelope.Nonce, envelope.Salt, envelope.KeyID, envelope.Algorithm, now, rotatedBy, envelope.WrappedDEK, envelope.DEKNonce); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE secrets
		SET updated_at = ?, rotated_at = ?, version_count = version_count + 1, current_version = ?
		WHERE id = ?
	`, now, now, newVersion, secretID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetMeta(ctx, tenantID, name)
}

func (s *SQLiteStore) GetVersion(ctx context.Context, tenantID string, name string, version int) ([]byte, error) {
	if version <= 0 {
		return nil, errors.New("version must be greater than zero")
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT sv.ciphertext, sv.nonce, sv.salt, sv.key_id, sv.algorithm, sv.wrapped_dek, sv.dek_nonce
		FROM secrets s
		JOIN secret_versions sv ON sv.secret_id = s.id
		WHERE s.tenant_id = ? AND s.name = ? AND sv.version = ?
	`, tenantID, name, version)

	envelope, err := scanEnvelope(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}

	return s.envelope.Decrypt(envelope)
}

func (s *SQLiteStore) MarkUsed(ctx context.Context, tenantID string, name string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE secrets SET last_used_at = ?, updated_at = ? WHERE tenant_id = ? AND name = ?`, now, now, tenantID, name)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSecretNotFound
	}
	return nil
}

func (s *SQLiteStore) Exists(ctx context.Context, tenantID string, name string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM secrets WHERE tenant_id = ? AND name = ?
		)
	`, tenantID, name).Scan(&exists)
	return exists, err
}

func (s *SQLiteStore) currentEnvelope(ctx context.Context, tenantID string, name string) (*corecrypto.EncryptedEnvelope, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT sv.ciphertext, sv.nonce, sv.salt, sv.key_id, sv.algorithm, sv.wrapped_dek, sv.dek_nonce
		FROM secrets s
		JOIN secret_versions sv
			ON sv.secret_id = s.id
			AND sv.version = s.current_version
		WHERE s.tenant_id = ? AND s.name = ?
	`, tenantID, name)

	envelope, err := scanEnvelope(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSecretNotFound
		}
		return nil, err
	}

	return envelope, nil
}

func (s *SQLiteStore) encryptForTenant(tenantID string, plaintext []byte) (*corecrypto.EncryptedEnvelope, error) {
	keyID := tenantKeyID(tenantID)
	if err := s.envelope.EnsureKey(keyID); err != nil {
		return nil, err
	}
	return s.envelope.Encrypt(plaintext, keyID)
}

func tenantKeyID(tenantID string) string {
	tid := strings.TrimSpace(tenantID)
	if tid == "" || tid == "default" {
		return "default"
	}
	return "tenant:" + tid
}

func (s *SQLiteStore) initSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			labels TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_used_at DATETIME,
			rotated_at DATETIME,
			version_count INTEGER DEFAULT 1,
			current_version INTEGER DEFAULT 1,
			UNIQUE(tenant_id, name)
		);

		CREATE TABLE IF NOT EXISTS secret_versions (
			id TEXT PRIMARY KEY,
			secret_id TEXT NOT NULL REFERENCES secrets(id),
			version INTEGER NOT NULL,
			ciphertext BLOB NOT NULL,
			nonce BLOB NOT NULL,
			salt BLOB,
			key_id TEXT NOT NULL,
			algorithm TEXT NOT NULL DEFAULT 'AES-256-GCM',
			created_at DATETIME NOT NULL,
			created_by TEXT NOT NULL,
			active BOOLEAN DEFAULT TRUE,
			wrapped_dek BLOB NOT NULL,
			dek_nonce BLOB NOT NULL,
			UNIQUE(secret_id, version)
		);
	`)
	return err
}

func scanSecretRecord(scanner interface{ Scan(dest ...any) error }) (*SecretRecord, error) {
	var rec SecretRecord
	var recordType string
	var labelsRaw sql.NullString
	var lastUsed sql.NullTime
	var rotated sql.NullTime

	err := scanner.Scan(
		&rec.ID,
		&rec.TenantID,
		&rec.Name,
		&recordType,
		&labelsRaw,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&lastUsed,
		&rotated,
		&rec.VersionCount,
		&rec.CurrentVersion,
	)
	if err != nil {
		return nil, err
	}

	rec.Type = SecretType(recordType)
	if labelsRaw.Valid {
		if err := json.Unmarshal([]byte(labelsRaw.String), &rec.Labels); err != nil {
			return nil, err
		}
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if lastUsed.Valid {
		ts := lastUsed.Time
		rec.LastUsedAt = &ts
	}
	if rotated.Valid {
		ts := rotated.Time
		rec.RotatedAt = &ts
	}

	return &rec, nil
}

func scanEnvelope(scanner interface{ Scan(dest ...any) error }) (*corecrypto.EncryptedEnvelope, error) {
	envelope := &corecrypto.EncryptedEnvelope{Version: 1}
	if err := scanner.Scan(
		&envelope.Ciphertext,
		&envelope.Nonce,
		&envelope.Salt,
		&envelope.KeyID,
		&envelope.Algorithm,
		&envelope.WrappedDEK,
		&envelope.DEKNonce,
	); err != nil {
		return nil, err
	}
	return envelope, nil
}

func marshalLabels(labels map[string]string) (string, error) {
	if labels == nil {
		labels = map[string]string{}
	}
	b, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func matchesLabels(candidate map[string]string, wanted map[string]string) bool {
	if len(wanted) == 0 {
		return true
	}
	for key, value := range wanted {
		if candidate[key] != value {
			return false
		}
	}
	return true
}

func isSQLiteUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed")
}
