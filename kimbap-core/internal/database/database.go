package database

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB       *gorm.DB
	initOnce sync.Once
	initErr  error
)

func Initialize(databaseURL string) error {
	initOnce.Do(func() {
		initErr = actualInitialize(databaseURL)
	})
	return initErr
}

func actualInitialize(databaseURL string) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("database URL is required")
	}

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return errors.New("database URL must use postgres/postgresql scheme")
	}

	logLevel := logger.Warn
	if strings.EqualFold(os.Getenv("NODE_ENV"), "production") || strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		logLevel = logger.Error
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if shouldAutoMigrate() {
		if strings.EqualFold(os.Getenv("NODE_ENV"), "production") || strings.EqualFold(os.Getenv("APP_ENV"), "production") {
			log.Warn().Msg("AUTO_MIGRATE is enabled in production environment — use versioned migrations for production deployments")
		}
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
			log.Warn().Err(err).Msg("could not create pgcrypto extension (gen_random_uuid is built-in on PG 13+)")
		}
		// Disable FK constraint creation during migration to avoid ordering
		// issues (e.g. OAuthClient has-many OAuthToken/OAuthAuthorizationCode).
		// GORM may attempt to create reverse FK constraints before the
		// referenced tables exist, causing errors on fresh databases.
		db.Config.DisableForeignKeyConstraintWhenMigrating = true
		if err := db.AutoMigrate(
			&Proxy{}, &User{}, &Server{}, &Log{}, &Event{},
			&License{},
			&OAuthClient{}, &OAuthAuthorizationCode{}, &OAuthToken{},
			&ApprovalRequest{}, &ToolPolicySet{},
		); err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return fmt.Errorf("auto-migrate failed: %w", err)
		}
		// Create partial unique index for approval request dedup (GORM can't express partial indexes)
		if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "approval_request_request_hash_active_uq" ON "approval_request" ("request_hash") WHERE "status" IN ('PENDING', 'APPROVED', 'EXECUTING')`).Error; err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return fmt.Errorf("create approval request partial unique index failed: %w", err)
		}

		// Deduplicate existing rows before creating unique indexes
		if err := db.Exec(`WITH numbered AS (SELECT id, ROW_NUMBER() OVER (PARTITION BY "server_id" ORDER BY "created_at", id) AS rn FROM "tool_policy_set" WHERE "server_id" IS NOT NULL) UPDATE "tool_policy_set" SET "version" = numbered.rn FROM numbered WHERE "tool_policy_set".id = numbered.id`).Error; err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return fmt.Errorf("dedup tool policy set server versions failed: %w", err)
		}
		if err := db.Exec(`WITH numbered AS (SELECT id, ROW_NUMBER() OVER (ORDER BY "created_at", id) AS rn FROM "tool_policy_set" WHERE "server_id" IS NULL) UPDATE "tool_policy_set" SET "version" = numbered.rn FROM numbered WHERE "tool_policy_set".id = numbered.id`).Error; err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return fmt.Errorf("dedup tool policy set global versions failed: %w", err)
		}
		if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "tool_policy_set_server_version_uq" ON "tool_policy_set" ("server_id", "version") WHERE "server_id" IS NOT NULL`).Error; err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return fmt.Errorf("create tool policy set server/version unique index failed: %w", err)
		}
		if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS "tool_policy_set_global_version_uq" ON "tool_policy_set" ("version") WHERE "server_id" IS NULL`).Error; err != nil {
			if sqlDB, sqlErr := db.DB(); sqlErr == nil {
				_ = sqlDB.Close()
			}
			return fmt.Errorf("create tool policy set global version unique index failed: %w", err)
		}
		db.Config.DisableForeignKeyConstraintWhenMigrating = false
	}

	if err := ensureApprovalExecutionResultColumn(db); err != nil {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
		return err
	}
	if err := ensureApprovalAuditColumns(db); err != nil {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
		return err
	}

	DB = db

	return nil
}

func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	DB = nil
	return sqlDB.Close()
}

func shouldAutoMigrate() bool {
	raw := strings.TrimSpace(os.Getenv("AUTO_MIGRATE"))
	if raw == "" {
		return false
	}
	return raw == "1" || strings.EqualFold(raw, "true")
}

func ensureApprovalExecutionResultColumn(db *gorm.DB) error {
	var tableExists bool
	tableCheckQuery := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'approval_request'
		)
	`
	if err := db.Raw(tableCheckQuery).Scan(&tableExists).Error; err != nil {
		return fmt.Errorf("check approval_request table failed: %w", err)
	}
	if !tableExists {
		return nil
	}

	var exists bool
	checkQuery := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'approval_request'
			  AND column_name = 'execution_result'
		)
	`
	if err := db.Raw(checkQuery).Scan(&exists).Error; err != nil {
		return fmt.Errorf("check approval_request.execution_result column failed: %w", err)
	}
	if exists {
		return nil
	}

	if err := db.Exec(`ALTER TABLE IF EXISTS "approval_request" ADD COLUMN IF NOT EXISTS "execution_result" JSONB`).Error; err != nil {
		return fmt.Errorf("ensure approval_request.execution_result column failed: %w", err)
	}

	if err := db.Raw(checkQuery).Scan(&exists).Error; err != nil {
		return fmt.Errorf("recheck approval_request.execution_result column failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("required column approval_request.execution_result is missing")
	}
	return nil
}

func ensureApprovalAuditColumns(db *gorm.DB) error {
	var tableExists bool
	tableCheckQuery := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'approval_request'
		)
	`
	if err := db.Raw(tableCheckQuery).Scan(&tableExists).Error; err != nil {
		return fmt.Errorf("check approval_request table failed: %w", err)
	}
	if !tableExists {
		return nil
	}

	columns := []struct {
		name string
		ddl  string
	}{
		{"decided_by_user_id", `ALTER TABLE "approval_request" ADD COLUMN IF NOT EXISTS "decided_by_user_id" VARCHAR(64)`},
		{"decided_by_role", `ALTER TABLE "approval_request" ADD COLUMN IF NOT EXISTS "decided_by_role" INTEGER`},
		{"decision_channel", `ALTER TABLE "approval_request" ADD COLUMN IF NOT EXISTS "decision_channel" VARCHAR(32)`},
	}

	for _, col := range columns {
		if err := db.Exec(col.ddl).Error; err != nil {
			return fmt.Errorf("ensure approval_request.%s column failed: %w", col.name, err)
		}
	}
	return nil
}
