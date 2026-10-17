package services

import (
	"embed"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate applies immutable, ordered SQL migrations.  Schema changes belong in a
// new file; AutoMigrate is deliberately not used in the running service.
func Migrate(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`).Error; err != nil {
		return err
	}
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var applied int64
		if err := db.Raw(`SELECT count(*) FROM schema_migrations WHERE version = ?`, name).Scan(&applied).Error; err != nil {
			return err
		}
		if applied != 0 {
			continue
		}
		contents, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(contents)).Error; err != nil {
				return fmt.Errorf("apply %s: %w", name, err)
			}
			return tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, name).Error
		}); err != nil {
			return err
		}
	}
	return nil
}
