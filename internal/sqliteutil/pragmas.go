package sqliteutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func ApplyConnectionPragmas(ctx context.Context, db *sql.DB, pragmas []string) error {
	if db == nil {
		return errors.New("database is required")
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("apply sqlite pragma %q: %w", pragma, err)
		}
	}
	return nil
}
