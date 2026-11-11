package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Open opens (and creates when needed) the SQLite database and applies all
// pending migrations.
func Open(path string) (*sqlx.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}

	dsn := "file:" + filepath.ToSlash(path) +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sqlx.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for index, statements := range migrations {
		version := index + 1
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version); err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		tx, err := db.Beginx()
		if err != nil {
			return err
		}
		for _, statement := range statements {
			if _, err := tx.Exec(statement); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("migration %d: %w", version, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("database: applied migration %d", version)
	}
	return nil
}

// migrations is intentionally append-only. Add new slices for future schema
// changes instead of editing existing entries.
var migrations = [][]string{
	{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			surname TEXT,
			email TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash TEXT NOT NULL,
			is_admin INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			theme TEXT NOT NULL DEFAULT 'system',
			enabled_modules TEXT,
			module_order TEXT,
			module_sections TEXT,
			dashboard_hidden_widgets TEXT,
			dashboard_widget_order TEXT,
			dashboard_widget_spans TEXT,
			email_notifications INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_users_created_at ON users(created_at)`,
		`CREATE TABLE dev_docs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			category TEXT NOT NULL,
			summary TEXT,
			content TEXT NOT NULL,
			is_published INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_dev_docs_category_published ON dev_docs(category, is_published)`,
		`CREATE INDEX idx_dev_docs_updated_at ON dev_docs(updated_at)`,
		`CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE storage_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			provider TEXT NOT NULL DEFAULT 'local',
			max_upload_mb INTEGER NOT NULL DEFAULT 20,
			aws_access_key TEXT,
			aws_secret_key TEXT,
			aws_region TEXT NOT NULL DEFAULT 'us-east-1',
			aws_bucket TEXT,
			aws_endpoint TEXT,
			aws_public_url TEXT,
			aws_use_path_style INTEGER NOT NULL DEFAULT 0,
			cloudflare_account_id TEXT,
			cloudflare_access_key_id TEXT,
			cloudflare_secret_access_key TEXT,
			cloudflare_bucket TEXT,
			cloudflare_public_url TEXT,
			updated_at DATETIME NOT NULL
		)`,
		`INSERT INTO storage_settings (id, provider, max_upload_mb, aws_region, aws_use_path_style, updated_at)
		 VALUES (1, 'local', 20, 'us-east-1', 0, CURRENT_TIMESTAMP)`,
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			csrf_token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX idx_sessions_expires_at ON sessions(expires_at)`,
		`CREATE TABLE api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			last_used_at DATETIME,
			expires_at DATETIME,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_api_tokens_user_id ON api_tokens(user_id)`,
		`CREATE TABLE notifications (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			url TEXT,
			level TEXT NOT NULL DEFAULT 'info',
			read_at DATETIME,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC)`,
	},
}

// CleanupSessions removes expired sessions and tokens. It is safe to call at
// startup and periodically.
func CleanupSessions(db *sqlx.DB) {
	now := time.Now().UTC()
	_, _ = db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	_, _ = db.Exec(`DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at < ?`, now)
}

// ParseTime is a small helper used by all repositories.
func ParseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, sql.ErrNoRows
	}
	return time.Parse(time.RFC3339, value)
}
