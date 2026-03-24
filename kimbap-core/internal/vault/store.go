package vault

import "context"

type Store interface {
	Create(ctx context.Context, tenantID string, name string, secretType SecretType, plaintext []byte, labels map[string]string, createdBy string) (*SecretRecord, error)
	Upsert(ctx context.Context, tenantID string, name string, secretType SecretType, plaintext []byte, labels map[string]string, createdBy string) (*SecretRecord, error)
	GetMeta(ctx context.Context, tenantID string, name string) (*SecretRecord, error)
	GetValue(ctx context.Context, tenantID string, name string) ([]byte, error)
	List(ctx context.Context, tenantID string, opts ListOptions) ([]SecretRecord, error)
	Delete(ctx context.Context, tenantID string, name string) error

	Rotate(ctx context.Context, tenantID string, name string, newPlaintext []byte, rotatedBy string) (*SecretRecord, error)
	GetVersion(ctx context.Context, tenantID string, name string, version int) ([]byte, error)

	MarkUsed(ctx context.Context, tenantID string, name string) error

	Exists(ctx context.Context, tenantID string, name string) (bool, error)
}

type ListOptions struct {
	Type   *SecretType
	Labels map[string]string
	Limit  int
	Offset int
}
