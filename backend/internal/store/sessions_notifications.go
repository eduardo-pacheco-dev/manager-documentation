package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
)

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type Session struct {
	ID        string    `db:"id"`
	UserID    int64     `db:"user_id"`
	CSRFToken string    `db:"csrf_token"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func (s *Store) CreateSession(userID int64, ttl time.Duration) (rawToken string, csrf string, err error) {
	rawToken, err = randomToken(32)
	if err != nil {
		return "", "", err
	}
	csrf, err = randomToken(24)
	if err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	_, err = s.DB.Exec(`
		INSERT INTO sessions (id, user_id, csrf_token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		hashToken(rawToken), userID, csrf, now.Add(ttl), now)
	return rawToken, csrf, err
}

func (s *Store) FindSession(rawToken string) (Session, error) {
	var session Session
	err := s.DB.Get(&session, `SELECT * FROM sessions WHERE id = ?`, hashToken(rawToken))
	if errors.Is(err, sql.ErrNoRows) {
		return session, ErrNotFound
	}
	if err != nil {
		return session, err
	}
	if session.ExpiresAt.Before(time.Now().UTC()) {
		_, _ = s.DB.Exec(`DELETE FROM sessions WHERE id = ?`, session.ID)
		return session, ErrNotFound
	}
	return session, nil
}

func (s *Store) DeleteSession(rawToken string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE id = ?`, hashToken(rawToken))
	return err
}

func (s *Store) DeleteUserSessions(userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

type APIToken struct {
	ID         int64      `db:"id"`
	UserID     int64      `db:"user_id"`
	Name       string     `db:"name"`
	TokenHash  string     `db:"token_hash"`
	LastUsedAt *time.Time `db:"last_used_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

func (s *Store) CreateAPIToken(userID int64, name string) (string, error) {
	raw, err := randomToken(32)
	if err != nil {
		return "", err
	}
	if name == "" {
		name = "mobile"
	}
	_, err = s.DB.Exec(`
		INSERT INTO api_tokens (user_id, name, token_hash, created_at)
		VALUES (?, ?, ?, ?)`, userID, name, hashToken(raw), time.Now().UTC())
	return raw, err
}

func (s *Store) FindAPIToken(raw string) (APIToken, error) {
	var token APIToken
	err := s.DB.Get(&token, `SELECT * FROM api_tokens WHERE token_hash = ?`, hashToken(raw))
	if errors.Is(err, sql.ErrNoRows) {
		return token, ErrNotFound
	}
	if err != nil {
		return token, err
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now().UTC()) {
		return token, ErrNotFound
	}
	_, _ = s.DB.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, time.Now().UTC(), token.ID)
	return token, nil
}

func (s *Store) DeleteAPIToken(raw string) error {
	_, err := s.DB.Exec(`DELETE FROM api_tokens WHERE token_hash = ?`, hashToken(raw))
	return err
}

func (s *Store) DeleteAPITokenByID(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM api_tokens WHERE id = ?`, id)
	return err
}

/* --------------------------------------------------------------------------
 * Notifications
 * ----------------------------------------------------------------------- */

func (s *Store) CreateNotification(notification *models.Notification) error {
	_, err := s.DB.Exec(`
		INSERT INTO notifications (id, user_id, title, body, url, level, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		notification.ID, notification.UserID, notification.Title, notification.Body,
		notification.URL, notification.Level, notification.ReadAt, notification.CreatedAt)
	return err
}

func (s *Store) ListNotifications(userID int64, page, perPage int) ([]models.Notification, int, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	var total, unread int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM notifications WHERE user_id = ?`, userID); err != nil {
		return nil, 0, 0, err
	}
	if err := s.DB.Get(&unread, `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`, userID); err != nil {
		return nil, 0, 0, err
	}
	items := []models.Notification{}
	err := s.DB.Select(&items, `
		SELECT * FROM notifications WHERE user_id = ?
		ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`,
		userID, perPage, (page-1)*perPage)
	return items, total, unread, err
}

func (s *Store) LatestNotifications(userID int64, limit int) ([]models.Notification, int, error) {
	items := []models.Notification{}
	if err := s.DB.Select(&items, `
		SELECT * FROM notifications WHERE user_id = ?
		ORDER BY created_at DESC, id DESC LIMIT ?`, userID, limit); err != nil {
		return nil, 0, err
	}
	var unread int
	err := s.DB.Get(&unread, `SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`, userID)
	return items, unread, err
}

func (s *Store) MarkNotificationRead(userID int64, id string) (bool, error) {
	result, err := s.DB.Exec(`
		UPDATE notifications SET read_at = ? WHERE id = ? AND user_id = ? AND read_at IS NULL`,
		time.Now().UTC(), id, userID)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *Store) MarkAllNotificationsRead(userID int64) error {
	_, err := s.DB.Exec(`UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`,
		time.Now().UTC(), userID)
	return err
}

func (s *Store) DeleteNotification(userID int64, id string) error {
	_, err := s.DB.Exec(`DELETE FROM notifications WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) ClearNotifications(userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM notifications WHERE user_id = ?`, userID)
	return err
}

func (s *Store) NotificationOwnedBy(userID int64, id string) (bool, error) {
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM notifications WHERE id = ? AND user_id = ?`, id, userID)
	return count > 0, err
}

/* --------------------------------------------------------------------------
 * Settings
 * ----------------------------------------------------------------------- */

func (s *Store) GetSetting(key string) (string, error) {
	var value sql.NullString
	err := s.DB.Get(&value, `SELECT value FROM settings WHERE key = ?`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return value.String, nil
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().UTC())
	return err
}

func (s *Store) GetStorageSettings() (models.StorageSettings, error) {
	var settings models.StorageSettings
	err := s.DB.Get(&settings, `SELECT * FROM storage_settings WHERE id = 1`)
	return settings, err
}

func (s *Store) SaveStorageSettings(settings *models.StorageSettings) error {
	settings.UpdatedAt = time.Now().UTC()
	_, err := s.DB.Exec(`
		UPDATE storage_settings SET
			provider = ?, max_upload_mb = ?, aws_access_key = ?, aws_secret_key = ?, aws_region = ?,
			aws_bucket = ?, aws_endpoint = ?, aws_public_url = ?, aws_use_path_style = ?,
			cloudflare_account_id = ?, cloudflare_access_key_id = ?, cloudflare_secret_access_key = ?,
			cloudflare_bucket = ?, cloudflare_public_url = ?, updated_at = ?
		WHERE id = 1`,
		settings.Provider, settings.MaxUploadMB, settings.AwsAccessKey, settings.AwsSecretKey, settings.AwsRegion,
		settings.AwsBucket, settings.AwsEndpoint, settings.AwsPublicURL, settings.AwsUsePathStyle,
		settings.CloudflareAccountID, settings.CloudflareAccessKeyID, settings.CloudflareSecretAccessKey,
		settings.CloudflareBucket, settings.CloudflarePublicURL, settings.UpdatedAt)
	return err
}
