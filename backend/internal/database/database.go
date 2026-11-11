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
	{
		`CREATE TABLE agenda_calendars (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			description TEXT,
			color TEXT NOT NULL DEFAULT 'blueberry',
			timezone TEXT NOT NULL DEFAULT 'UTC',
			is_default INTEGER NOT NULL DEFAULT 0,
			is_visible INTEGER NOT NULL DEFAULT 1,
			position INTEGER NOT NULL DEFAULT 0,
			default_reminders TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_agenda_calendars_user ON agenda_calendars(user_id, position)`,
		`CREATE TABLE agenda_calendar_shares (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agenda_calendar_id INTEGER NOT NULL REFERENCES agenda_calendars(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role TEXT NOT NULL DEFAULT 'reader',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(agenda_calendar_id, user_id)
		)`,
		`CREATE TABLE agenda_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agenda_calendar_id INTEGER NOT NULL REFERENCES agenda_calendars(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT,
			location TEXT,
			meeting_url TEXT,
			start_at DATETIME NOT NULL,
			end_at DATETIME NOT NULL,
			is_all_day INTEGER NOT NULL DEFAULT 0,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			status TEXT NOT NULL DEFAULT 'confirmed',
			visibility TEXT NOT NULL DEFAULT 'default',
			transparency TEXT NOT NULL DEFAULT 'opaque',
			color TEXT,
			recurrence TEXT,
			recurrence_until TEXT,
			attendees_can_invite_others INTEGER NOT NULL DEFAULT 1,
			attendees_can_modify INTEGER NOT NULL DEFAULT 0,
			attendees_can_see_other_attendees INTEGER NOT NULL DEFAULT 1,
			attachments TEXT,
			sequence INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_agenda_events_calendar_start ON agenda_events(agenda_calendar_id, start_at)`,
		`CREATE INDEX idx_agenda_events_user_start ON agenda_events(user_id, start_at)`,
		`CREATE TABLE agenda_event_attendees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agenda_event_id INTEGER NOT NULL REFERENCES agenda_events(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			email TEXT NOT NULL,
			name TEXT,
			status TEXT NOT NULL DEFAULT 'needs_action',
			is_optional INTEGER NOT NULL DEFAULT 0,
			is_organizer INTEGER NOT NULL DEFAULT 0,
			is_resource INTEGER NOT NULL DEFAULT 0,
			comment TEXT,
			responded_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(agenda_event_id, email)
		)`,
		`CREATE TABLE agenda_event_reminders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agenda_event_id INTEGER NOT NULL REFERENCES agenda_events(id) ON DELETE CASCADE,
			method TEXT NOT NULL DEFAULT 'popup',
			minutes_before INTEGER,
			trigger_at DATETIME,
			sent_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE agenda_event_exceptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agenda_event_id INTEGER NOT NULL REFERENCES agenda_events(id) ON DELETE CASCADE,
			recurrence_id DATETIME NOT NULL,
			is_cancelled INTEGER NOT NULL DEFAULT 0,
			overrides TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(agenda_event_id, recurrence_id)
		)`,
		`CREATE TABLE todo_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT 'indigo',
			icon TEXT NOT NULL DEFAULT 'check-circle',
			position INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME
		)`,
		`CREATE INDEX idx_todo_lists_user ON todo_lists(user_id, position)`,
		`CREATE TABLE todo_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			todo_list_id INTEGER NOT NULL REFERENCES todo_lists(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			notes TEXT,
			due_at TEXT,
			remind_at DATETIME,
			repeat_type TEXT NOT NULL DEFAULT 'none',
			is_important INTEGER NOT NULL DEFAULT 0,
			my_day_on TEXT,
			completed_at DATETIME,
			position INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_todo_tasks_user_completed ON todo_tasks(user_id, completed_at)`,
		`CREATE INDEX idx_todo_tasks_list_position ON todo_tasks(todo_list_id, position)`,
		`CREATE TABLE todo_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			todo_task_id INTEGER NOT NULL REFERENCES todo_tasks(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			completed_at DATETIME,
			position INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_todo_steps_task ON todo_steps(todo_task_id, position)`,
		`CREATE TABLE follow_up_clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE follow_up_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL REFERENCES follow_up_clients(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(client_id, name)
		)`,
		`CREATE TABLE follow_up_assignees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE follow_ups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			code TEXT,
			description TEXT,
			client_id INTEGER REFERENCES follow_up_clients(id) ON DELETE SET NULL,
			project_id INTEGER REFERENCES follow_up_projects(id) ON DELETE SET NULL,
			status TEXT NOT NULL DEFAULT 'pendente',
			priority TEXT NOT NULL DEFAULT 'media',
			due_date TEXT,
			assignee_id INTEGER REFERENCES follow_up_assignees(id) ON DELETE SET NULL,
			created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
			completed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_follow_ups_status_due ON follow_ups(status, due_date)`,
		`CREATE TABLE follow_up_updates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			follow_up_id INTEGER NOT NULL REFERENCES follow_ups(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			body TEXT NOT NULL,
			status_from TEXT,
			status_to TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_follow_up_updates_follow ON follow_up_updates(follow_up_id, created_at)`,
		`CREATE TABLE follow_up_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			per_page INTEGER NOT NULL DEFAULT 15,
			due_soon_days INTEGER NOT NULL DEFAULT 7,
			widget_limit INTEGER NOT NULL DEFAULT 5,
			email_summary_enabled INTEGER NOT NULL DEFAULT 1,
			email_subject_prefix TEXT NOT NULL DEFAULT 'Follow-up',
			summary_period TEXT NOT NULL DEFAULT 'all',
			summary_times TEXT NOT NULL DEFAULT '08:00,17:00',
			summary_days TEXT NOT NULL DEFAULT '0,1,2,3,4,5,6',
			summary_recipients TEXT,
			summary_cc_recipients TEXT,
			updated_at DATETIME NOT NULL
		)`,
		`INSERT INTO follow_up_settings (id, updated_at) VALUES (1, CURRENT_TIMESTAMP)`,
		`CREATE TABLE stations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id TEXT NOT NULL,
			endereco_id TEXT,
			tipo_de_elemento TEXT,
			tecnologia TEXT,
			tipo_de_conexao TEXT,
			classificacao TEXT,
			detentor_area TEXT,
			tipo_de_infra TEXT,
			tipo_de_ev TEXT,
			tipo_de_logradouro TEXT,
			logradouro TEXT,
			numero TEXT,
			complemento TEXT,
			bairro TEXT,
			municipio TEXT,
			estado TEXT,
			cep TEXT,
			regional TEXT,
			latitude TEXT,
			longitude TEXT,
			status TEXT,
			tipo_da_torre TEXT,
			station_id TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_stations_site ON stations(site_id)`,
		`CREATE INDEX idx_stations_station ON stations(station_id)`,
		`CREATE INDEX idx_stations_status ON stations(status)`,
		`CREATE TABLE station_attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			station_id INTEGER NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
			uploaded_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
			category TEXT NOT NULL DEFAULT 'attachment',
			original_name TEXT NOT NULL,
			path TEXT NOT NULL,
			disk TEXT NOT NULL DEFAULT 'local',
			mime_type TEXT,
			size INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE station_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			station_id INTEGER NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE nokia_roms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			phone TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE nokia_sites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id TEXT NOT NULL,
			end_id TEXT,
			oc TEXT UNIQUE,
			os_fam_entrega TEXT,
			os_fam_instalacao TEXT,
			regional TEXT,
			subcon TEXT,
			interruption_reason TEXT,
			interruption_return_days INTEGER,
			interruption_started_at DATETIME,
			checklist_fam_entrega INTEGER NOT NULL DEFAULT 0,
			checklist_fam_entrega_status TEXT NOT NULL DEFAULT 'pending',
			checklist_fam_instalacao INTEGER NOT NULL DEFAULT 0,
			checklist_fam_instalacao_status TEXT NOT NULL DEFAULT 'pending',
			checklist_fam_panoramica INTEGER NOT NULL DEFAULT 0,
			checklist_fam_panoramica_status TEXT NOT NULL DEFAULT 'pending',
			mos_plan_at TEXT,
			mos_real_at TEXT,
			instalacao_plan_at TEXT,
			instalacao_real_at TEXT,
			integracao_plan_at TEXT,
			integracao_real_at TEXT,
			rfa_plan_at TEXT,
			rfa_real_at TEXT,
			fam_entrega_approved_at TEXT,
			fam_instalacao_approved_at TEXT,
			fam_panoramica_approved_at TEXT,
			nokia_rom_id INTEGER REFERENCES nokia_roms(id) ON DELETE SET NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_nokia_sites_site ON nokia_sites(site_id)`,
		`CREATE INDEX idx_nokia_sites_regional ON nokia_sites(regional)`,
		`CREATE TABLE nokia_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE nokia_site_task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nokia_site_id INTEGER NOT NULL REFERENCES nokia_sites(id) ON DELETE CASCADE,
			nokia_task_id INTEGER NOT NULL REFERENCES nokia_tasks(id) ON DELETE CASCADE,
			completed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			UNIQUE(nokia_site_id, nokia_task_id)
		)`,
		`CREATE TABLE nokia_site_attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nokia_site_id INTEGER NOT NULL REFERENCES nokia_sites(id) ON DELETE CASCADE,
			category TEXT NOT NULL,
			original_name TEXT NOT NULL,
			path TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			mime_type TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_nokia_attachments_site ON nokia_site_attachments(nokia_site_id, category)`,
		`CREATE TABLE nokia_site_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nokia_site_id INTEGER NOT NULL REFERENCES nokia_sites(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE nokia_site_rejections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			nokia_site_id INTEGER NOT NULL REFERENCES nokia_sites(id) ON DELETE CASCADE,
			checklist_field TEXT NOT NULL,
			reason TEXT NOT NULL,
			return_days INTEGER NOT NULL,
			requires_field_return INTEGER NOT NULL DEFAULT 0,
			started_at DATETIME NOT NULL,
			resolved_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
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
