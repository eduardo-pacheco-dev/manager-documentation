package database

import (
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
)

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// Open connects to MySQL, creating the configured database when it does not
// exist yet, and applies all pending migrations.
func Open(cfg config.Config) (*sqlx.DB, error) {
	if strings.TrimSpace(cfg.DatabaseDSN) != "" {
		db, err := sqlx.Connect("mysql", cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("open database: %w", err)
		}
		configurePool(db)
		if err := migrate(db); err != nil {
			_ = db.Close()
			return nil, err
		}
		return db, nil
	}

	if !databaseNamePattern.MatchString(cfg.DatabaseName) {
		return nil, fmt.Errorf("invalid database name %q", cfg.DatabaseName)
	}

	dsn := buildDSN(cfg, cfg.DatabaseName)
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if !errors.As(err, &mysqlErr) || mysqlErr.Number != 1049 {
			return nil, fmt.Errorf("open database: %w", err)
		}
		if err := createDatabase(cfg); err != nil {
			return nil, err
		}
		db, err = sqlx.Connect("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("open database: %w", err)
		}
	}
	configurePool(db)

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func configurePool(db *sqlx.DB) {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
}

func buildDSN(cfg config.Config, databaseName string) string {
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = cfg.DatabaseUser
	mysqlConfig.Passwd = cfg.DatabasePassword
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(cfg.DatabaseHost, strconv.Itoa(cfg.DatabasePort))
	mysqlConfig.DBName = databaseName
	mysqlConfig.ParseTime = true
	mysqlConfig.Loc = time.UTC
	mysqlConfig.Collation = "utf8mb4_unicode_ci"
	mysqlConfig.Params = map[string]string{"charset": "utf8mb4"}
	mysqlConfig.Timeout = 10 * time.Second
	mysqlConfig.ReadTimeout = 30 * time.Second
	mysqlConfig.WriteTimeout = 30 * time.Second
	return mysqlConfig.FormatDSN()
}

func createDatabase(cfg config.Config) error {
	admin, err := sqlx.Connect("mysql", buildDSN(cfg, ""))
	if err != nil {
		return fmt.Errorf("connect to mysql server: %w", err)
	}
	defer admin.Close()
	statement := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.DatabaseName,
	)
	if _, err := admin.Exec(statement); err != nil {
		return fmt.Errorf("create database %q: %w", cfg.DatabaseName, err)
	}
	return nil
}

// DropDatabase removes a database. It is used by the test suite to clean up
// isolated databases.
func DropDatabase(cfg config.Config, databaseName string) error {
	if !databaseNamePattern.MatchString(databaseName) {
		return fmt.Errorf("invalid database name %q", databaseName)
	}
	admin, err := sqlx.Connect("mysql", buildDSN(cfg, ""))
	if err != nil {
		return fmt.Errorf("connect to mysql server: %w", err)
	}
	defer admin.Close()
	_, err = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", databaseName))
	return err
}

func migrate(db *sqlx.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INT NOT NULL,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (version)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
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

		for _, statement := range statements {
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("migration %d: %w", version, err)
			}
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
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
			id BIGINT NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			surname VARCHAR(255),
			email VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			is_admin TINYINT(1) NOT NULL DEFAULT 0,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			theme VARCHAR(32) NOT NULL DEFAULT 'system',
			enabled_modules TEXT,
			module_order TEXT,
			module_sections TEXT,
			dashboard_hidden_widgets TEXT,
			dashboard_widget_order TEXT,
			dashboard_widget_spans TEXT,
			email_notifications TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_users_email (email),
			KEY idx_users_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE dev_docs (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			slug VARCHAR(255) NOT NULL,
			category VARCHAR(100) NOT NULL,
			summary TEXT,
			content MEDIUMTEXT NOT NULL,
			is_published TINYINT(1) NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_dev_docs_slug (slug),
			KEY idx_dev_docs_category_published (category, is_published),
			KEY idx_dev_docs_updated_at (updated_at),
			CONSTRAINT fk_dev_docs_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE settings (
			` + "`key`" + ` VARCHAR(191) NOT NULL,
			value MEDIUMTEXT,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (` + "`key`" + `)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE storage_settings (
			id TINYINT NOT NULL,
			provider VARCHAR(32) NOT NULL DEFAULT 'local',
			max_upload_mb INT NOT NULL DEFAULT 20,
			aws_access_key TEXT,
			aws_secret_key TEXT,
			aws_region VARCHAR(64) NOT NULL DEFAULT 'us-east-1',
			aws_bucket VARCHAR(255),
			aws_endpoint VARCHAR(1024),
			aws_public_url VARCHAR(1024),
			aws_use_path_style TINYINT(1) NOT NULL DEFAULT 0,
			cloudflare_account_id VARCHAR(255),
			cloudflare_access_key_id VARCHAR(255),
			cloudflare_secret_access_key TEXT,
			cloudflare_bucket VARCHAR(255),
			cloudflare_public_url VARCHAR(1024),
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`INSERT INTO storage_settings (id, provider, max_upload_mb, aws_region, aws_use_path_style, updated_at)
		 VALUES (1, 'local', 20, 'us-east-1', 0, UTC_TIMESTAMP())`,
		`CREATE TABLE sessions (
			id VARCHAR(64) NOT NULL,
			user_id BIGINT NOT NULL,
			csrf_token VARCHAR(255) NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_sessions_user_id (user_id),
			KEY idx_sessions_expires_at (expires_at),
			CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE api_tokens (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			token_hash VARCHAR(64) NOT NULL,
			last_used_at DATETIME,
			expires_at DATETIME,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_api_tokens_hash (token_hash),
			KEY idx_api_tokens_user_id (user_id),
			CONSTRAINT fk_api_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE notifications (
			id VARCHAR(64) NOT NULL,
			user_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			body TEXT NOT NULL,
			url VARCHAR(1024),
			level VARCHAR(20) NOT NULL DEFAULT 'info',
			read_at DATETIME,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_notifications_user_created (user_id, created_at),
			CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		`CREATE TABLE agenda_calendars (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			color VARCHAR(32) NOT NULL DEFAULT 'blueberry',
			timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
			is_default TINYINT(1) NOT NULL DEFAULT 0,
			is_visible TINYINT(1) NOT NULL DEFAULT 1,
			position INT NOT NULL DEFAULT 0,
			default_reminders TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_agenda_calendars_user (user_id, position),
			CONSTRAINT fk_agenda_calendars_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE agenda_calendar_shares (
			id BIGINT NOT NULL AUTO_INCREMENT,
			agenda_calendar_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			role VARCHAR(32) NOT NULL DEFAULT 'reader',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_agenda_calendar_shares (agenda_calendar_id, user_id),
			CONSTRAINT fk_agenda_shares_calendar FOREIGN KEY (agenda_calendar_id) REFERENCES agenda_calendars(id) ON DELETE CASCADE,
			CONSTRAINT fk_agenda_shares_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE agenda_events (
			id BIGINT NOT NULL AUTO_INCREMENT,
			agenda_calendar_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			location VARCHAR(512),
			meeting_url VARCHAR(1024),
			start_at DATETIME NOT NULL,
			end_at DATETIME NOT NULL,
			is_all_day TINYINT(1) NOT NULL DEFAULT 0,
			timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
			status VARCHAR(32) NOT NULL DEFAULT 'confirmed',
			visibility VARCHAR(32) NOT NULL DEFAULT 'default',
			transparency VARCHAR(32) NOT NULL DEFAULT 'opaque',
			color VARCHAR(32),
			recurrence TEXT,
			recurrence_until VARCHAR(32),
			attendees_can_invite_others TINYINT(1) NOT NULL DEFAULT 1,
			attendees_can_modify TINYINT(1) NOT NULL DEFAULT 0,
			attendees_can_see_other_attendees TINYINT(1) NOT NULL DEFAULT 1,
			attachments TEXT,
			sequence INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_agenda_events_calendar_start (agenda_calendar_id, start_at),
			KEY idx_agenda_events_user_start (user_id, start_at),
			CONSTRAINT fk_agenda_events_calendar FOREIGN KEY (agenda_calendar_id) REFERENCES agenda_calendars(id) ON DELETE CASCADE,
			CONSTRAINT fk_agenda_events_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE agenda_event_attendees (
			id BIGINT NOT NULL AUTO_INCREMENT,
			agenda_event_id BIGINT NOT NULL,
			user_id BIGINT,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(255),
			status VARCHAR(32) NOT NULL DEFAULT 'needs_action',
			is_optional TINYINT(1) NOT NULL DEFAULT 0,
			is_organizer TINYINT(1) NOT NULL DEFAULT 0,
			is_resource TINYINT(1) NOT NULL DEFAULT 0,
			comment TEXT,
			responded_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_agenda_attendee_email (agenda_event_id, email),
			CONSTRAINT fk_agenda_attendees_event FOREIGN KEY (agenda_event_id) REFERENCES agenda_events(id) ON DELETE CASCADE,
			CONSTRAINT fk_agenda_attendees_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE agenda_event_reminders (
			id BIGINT NOT NULL AUTO_INCREMENT,
			agenda_event_id BIGINT NOT NULL,
			method VARCHAR(32) NOT NULL DEFAULT 'popup',
			minutes_before INT,
			trigger_at DATETIME,
			sent_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_agenda_reminders_event FOREIGN KEY (agenda_event_id) REFERENCES agenda_events(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE agenda_event_exceptions (
			id BIGINT NOT NULL AUTO_INCREMENT,
			agenda_event_id BIGINT NOT NULL,
			recurrence_id DATETIME NOT NULL,
			is_cancelled TINYINT(1) NOT NULL DEFAULT 0,
			overrides TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_agenda_exception (agenda_event_id, recurrence_id),
			CONSTRAINT fk_agenda_exceptions_event FOREIGN KEY (agenda_event_id) REFERENCES agenda_events(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE todo_lists (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			color VARCHAR(32) NOT NULL DEFAULT 'indigo',
			icon VARCHAR(64) NOT NULL DEFAULT 'check-circle',
			position INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME,
			PRIMARY KEY (id),
			KEY idx_todo_lists_user (user_id, position),
			CONSTRAINT fk_todo_lists_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE todo_tasks (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			todo_list_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			notes TEXT,
			due_at VARCHAR(10),
			remind_at DATETIME,
			repeat_type VARCHAR(32) NOT NULL DEFAULT 'none',
			is_important TINYINT(1) NOT NULL DEFAULT 0,
			my_day_on VARCHAR(10),
			completed_at DATETIME,
			position INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_todo_tasks_user_completed (user_id, completed_at),
			KEY idx_todo_tasks_list_position (todo_list_id, position),
			CONSTRAINT fk_todo_tasks_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_todo_tasks_list FOREIGN KEY (todo_list_id) REFERENCES todo_lists(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE todo_steps (
			id BIGINT NOT NULL AUTO_INCREMENT,
			todo_task_id BIGINT NOT NULL,
			title VARCHAR(255) NOT NULL,
			completed_at DATETIME,
			position INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_todo_steps_task (todo_task_id, position),
			CONSTRAINT fk_todo_steps_task FOREIGN KEY (todo_task_id) REFERENCES todo_tasks(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE follow_up_clients (
			id BIGINT NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_follow_up_clients_name (name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE follow_up_projects (
			id BIGINT NOT NULL AUTO_INCREMENT,
			client_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_follow_up_projects (client_id, name),
			CONSTRAINT fk_follow_up_projects_client FOREIGN KEY (client_id) REFERENCES follow_up_clients(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE follow_up_assignees (
			id BIGINT NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE follow_ups (
			id BIGINT NOT NULL AUTO_INCREMENT,
			title VARCHAR(255) NOT NULL,
			code VARCHAR(64),
			description TEXT,
			client_id BIGINT,
			project_id BIGINT,
			status VARCHAR(32) NOT NULL DEFAULT 'pendente',
			priority VARCHAR(32) NOT NULL DEFAULT 'media',
			due_date VARCHAR(10),
			assignee_id BIGINT,
			created_by BIGINT,
			completed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_follow_ups_status_due (status, due_date),
			CONSTRAINT fk_follow_ups_client FOREIGN KEY (client_id) REFERENCES follow_up_clients(id) ON DELETE SET NULL,
			CONSTRAINT fk_follow_ups_project FOREIGN KEY (project_id) REFERENCES follow_up_projects(id) ON DELETE SET NULL,
			CONSTRAINT fk_follow_ups_assignee FOREIGN KEY (assignee_id) REFERENCES follow_up_assignees(id) ON DELETE SET NULL,
			CONSTRAINT fk_follow_ups_creator FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE follow_up_updates (
			id BIGINT NOT NULL AUTO_INCREMENT,
			follow_up_id BIGINT NOT NULL,
			user_id BIGINT,
			body TEXT NOT NULL,
			status_from VARCHAR(32),
			status_to VARCHAR(32),
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_follow_up_updates_follow (follow_up_id, created_at),
			CONSTRAINT fk_follow_up_updates_follow FOREIGN KEY (follow_up_id) REFERENCES follow_ups(id) ON DELETE CASCADE,
			CONSTRAINT fk_follow_up_updates_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE follow_up_settings (
			id TINYINT NOT NULL,
			per_page INT NOT NULL DEFAULT 15,
			due_soon_days INT NOT NULL DEFAULT 7,
			widget_limit INT NOT NULL DEFAULT 5,
			email_summary_enabled TINYINT(1) NOT NULL DEFAULT 1,
			email_subject_prefix VARCHAR(255) NOT NULL DEFAULT 'Follow-up',
			summary_period VARCHAR(32) NOT NULL DEFAULT 'all',
			summary_times VARCHAR(255) NOT NULL DEFAULT '08:00,17:00',
			summary_days VARCHAR(64) NOT NULL DEFAULT '0,1,2,3,4,5,6',
			summary_recipients TEXT,
			summary_cc_recipients TEXT,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`INSERT INTO follow_up_settings (id, updated_at) VALUES (1, UTC_TIMESTAMP())`,
		`CREATE TABLE stations (
			id BIGINT NOT NULL AUTO_INCREMENT,
			site_id VARCHAR(255) NOT NULL,
			endereco_id VARCHAR(255),
			tipo_de_elemento VARCHAR(255),
			tecnologia VARCHAR(255),
			tipo_de_conexao VARCHAR(255),
			classificacao VARCHAR(255),
			detentor_area VARCHAR(255),
			tipo_de_infra VARCHAR(255),
			tipo_de_ev VARCHAR(255),
			tipo_de_logradouro VARCHAR(255),
			logradouro VARCHAR(255),
			numero VARCHAR(255),
			complemento VARCHAR(255),
			bairro VARCHAR(255),
			municipio VARCHAR(255),
			estado VARCHAR(255),
			cep VARCHAR(255),
			regional VARCHAR(255),
			latitude VARCHAR(255),
			longitude VARCHAR(255),
			status VARCHAR(255),
			tipo_da_torre VARCHAR(255),
			station_id VARCHAR(255),
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_stations_site (site_id),
			KEY idx_stations_station (station_id),
			KEY idx_stations_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE station_attachments (
			id BIGINT NOT NULL AUTO_INCREMENT,
			station_id BIGINT NOT NULL,
			uploaded_by BIGINT,
			category VARCHAR(32) NOT NULL DEFAULT 'attachment',
			original_name VARCHAR(512) NOT NULL,
			path VARCHAR(1024) NOT NULL,
			disk VARCHAR(32) NOT NULL DEFAULT 'local',
			mime_type VARCHAR(255),
			size BIGINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_station_attachments_station FOREIGN KEY (station_id) REFERENCES stations(id) ON DELETE CASCADE,
			CONSTRAINT fk_station_attachments_user FOREIGN KEY (uploaded_by) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE station_comments (
			id BIGINT NOT NULL AUTO_INCREMENT,
			station_id BIGINT NOT NULL,
			user_id BIGINT,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_station_comments_station FOREIGN KEY (station_id) REFERENCES stations(id) ON DELETE CASCADE,
			CONSTRAINT fk_station_comments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_roms (
			id BIGINT NOT NULL AUTO_INCREMENT,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			phone VARCHAR(64) NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_nokia_roms_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_sites (
			id BIGINT NOT NULL AUTO_INCREMENT,
			site_id VARCHAR(255) NOT NULL,
			end_id VARCHAR(255),
			oc VARCHAR(255),
			os_fam_entrega VARCHAR(255),
			os_fam_instalacao VARCHAR(255),
			regional VARCHAR(64),
			subcon VARCHAR(255),
			interruption_reason TEXT,
			interruption_return_days INT,
			interruption_started_at DATETIME,
			checklist_fam_entrega TINYINT(1) NOT NULL DEFAULT 0,
			checklist_fam_entrega_status VARCHAR(32) NOT NULL DEFAULT 'pending',
			checklist_fam_instalacao TINYINT(1) NOT NULL DEFAULT 0,
			checklist_fam_instalacao_status VARCHAR(32) NOT NULL DEFAULT 'pending',
			checklist_fam_panoramica TINYINT(1) NOT NULL DEFAULT 0,
			checklist_fam_panoramica_status VARCHAR(32) NOT NULL DEFAULT 'pending',
			mos_plan_at VARCHAR(10),
			mos_real_at VARCHAR(10),
			instalacao_plan_at VARCHAR(10),
			instalacao_real_at VARCHAR(10),
			integracao_plan_at VARCHAR(10),
			integracao_real_at VARCHAR(10),
			rfa_plan_at VARCHAR(10),
			rfa_real_at VARCHAR(10),
			fam_entrega_approved_at VARCHAR(10),
			fam_instalacao_approved_at VARCHAR(10),
			fam_panoramica_approved_at VARCHAR(10),
			nokia_rom_id BIGINT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_nokia_sites_oc (oc),
			KEY idx_nokia_sites_site (site_id),
			KEY idx_nokia_sites_regional (regional),
			CONSTRAINT fk_nokia_sites_rom FOREIGN KEY (nokia_rom_id) REFERENCES nokia_roms(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_tasks (
			id BIGINT NOT NULL AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			position INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_site_task (
			id BIGINT NOT NULL AUTO_INCREMENT,
			nokia_site_id BIGINT NOT NULL,
			nokia_task_id BIGINT NOT NULL,
			completed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_nokia_site_task (nokia_site_id, nokia_task_id),
			CONSTRAINT fk_nokia_site_task_site FOREIGN KEY (nokia_site_id) REFERENCES nokia_sites(id) ON DELETE CASCADE,
			CONSTRAINT fk_nokia_site_task_task FOREIGN KEY (nokia_task_id) REFERENCES nokia_tasks(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_site_attachments (
			id BIGINT NOT NULL AUTO_INCREMENT,
			nokia_site_id BIGINT NOT NULL,
			category VARCHAR(32) NOT NULL,
			original_name VARCHAR(512) NOT NULL,
			path VARCHAR(1024) NOT NULL,
			size BIGINT NOT NULL DEFAULT 0,
			mime_type VARCHAR(255),
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_nokia_attachments_site (nokia_site_id, category),
			CONSTRAINT fk_nokia_attachments_site FOREIGN KEY (nokia_site_id) REFERENCES nokia_sites(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_site_comments (
			id BIGINT NOT NULL AUTO_INCREMENT,
			nokia_site_id BIGINT NOT NULL,
			user_id BIGINT,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_nokia_comments_site FOREIGN KEY (nokia_site_id) REFERENCES nokia_sites(id) ON DELETE CASCADE,
			CONSTRAINT fk_nokia_comments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE nokia_site_rejections (
			id BIGINT NOT NULL AUTO_INCREMENT,
			nokia_site_id BIGINT NOT NULL,
			checklist_field VARCHAR(64) NOT NULL,
			reason TEXT NOT NULL,
			return_days INT NOT NULL,
			requires_field_return TINYINT(1) NOT NULL DEFAULT 0,
			started_at DATETIME NOT NULL,
			resolved_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			CONSTRAINT fk_nokia_rejections_site FOREIGN KEY (nokia_site_id) REFERENCES nokia_sites(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		`ALTER TABLE stations ADD COLUMN nokia_site_id BIGINT NULL`,
		`CREATE UNIQUE INDEX idx_stations_nokia_site ON stations(nokia_site_id)`,
	},
	{
		`CREATE TABLE entity_links (
			id BIGINT NOT NULL AUTO_INCREMENT,
			module VARCHAR(32) NOT NULL,
			entity_id BIGINT NOT NULL,
			resource_type VARCHAR(32) NOT NULL,
			resource_id BIGINT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_entity_links (module, entity_id, resource_type, resource_id),
			KEY idx_entity_links_resource (resource_type, resource_id),
			KEY idx_entity_links_entity (module, entity_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`ALTER TABLE notifications ADD COLUMN dedupe_key VARCHAR(191) NULL`,
		`CREATE UNIQUE INDEX idx_notifications_dedupe ON notifications(user_id, dedupe_key)`,
	},
	{
		`CREATE TABLE nokia_site_agenda_events (
			id BIGINT NOT NULL AUTO_INCREMENT,
			nokia_site_id BIGINT NOT NULL,
			date_field VARCHAR(64) NOT NULL,
			agenda_event_id BIGINT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_nokia_agenda_field (nokia_site_id, date_field),
			KEY idx_nokia_agenda_event (agenda_event_id),
			CONSTRAINT fk_nokia_agenda_site FOREIGN KEY (nokia_site_id) REFERENCES nokia_sites(id) ON DELETE CASCADE,
			CONSTRAINT fk_nokia_agenda_event FOREIGN KEY (agenda_event_id) REFERENCES agenda_events(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		`CREATE TABLE short_links (
			id BIGINT NOT NULL AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			code VARCHAR(32) NOT NULL,
			target_url VARCHAR(2048) NOT NULL,
			title VARCHAR(255),
			description TEXT,
			is_active TINYINT(1) NOT NULL DEFAULT 1,
			expires_at DATETIME,
			click_count BIGINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uq_short_links_code (code),
			KEY idx_short_links_user (user_id),
			KEY idx_short_links_created (created_at),
			CONSTRAINT fk_short_links_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE short_link_clicks (
			id BIGINT NOT NULL AUTO_INCREMENT,
			short_link_id BIGINT NOT NULL,
			visitor_hash VARCHAR(64),
			referrer VARCHAR(1024),
			user_agent VARCHAR(512),
			clicked_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_short_link_clicks_link (short_link_id, clicked_at),
			CONSTRAINT fk_short_link_clicks_link FOREIGN KEY (short_link_id) REFERENCES short_links(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
	{
		`CREATE TABLE short_link_files (
			id BIGINT NOT NULL AUTO_INCREMENT,
			short_link_id BIGINT NOT NULL,
			uploaded_by BIGINT,
			original_name VARCHAR(512) NOT NULL,
			path VARCHAR(1024) NOT NULL,
			url VARCHAR(2048) NOT NULL,
			disk VARCHAR(32) NOT NULL DEFAULT 'local',
			mime_type VARCHAR(255),
			size BIGINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (id),
			KEY idx_short_link_files_link (short_link_id),
			CONSTRAINT fk_short_link_files_link FOREIGN KEY (short_link_id) REFERENCES short_links(id) ON DELETE CASCADE,
			CONSTRAINT fk_short_link_files_user FOREIGN KEY (uploaded_by) REFERENCES users(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	},
}

// CleanupSessions removes expired sessions and tokens. It is safe to call at
// startup and periodically.
func CleanupSessions(db *sqlx.DB) {
	now := time.Now().UTC()
	_, _ = db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	_, _ = db.Exec(`DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at < ?`, now)
}
