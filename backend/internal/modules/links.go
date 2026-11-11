package modules

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Link shortener module
 * ----------------------------------------------------------------------- */

type ShortLink struct {
	ID           int64      `db:"id" json:"id"`
	UserID       int64      `db:"user_id" json:"user_id"`
	Code         string     `db:"code" json:"code"`
	TargetURL    string     `db:"target_url" json:"target_url"`
	Title        *string    `db:"title" json:"title"`
	Description  *string    `db:"description" json:"description"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	ExpiresAt    *time.Time `db:"expires_at" json:"expires_at"`
	ClickCount   int64      `db:"click_count" json:"click_count"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	CreatorName  *string    `db:"creator_name" json:"creator_name"`
	UniqueClicks int64      `db:"unique_clicks" json:"unique_clicks"`
	LastClickAt  *time.Time `db:"last_click_at" json:"last_click_at"`
	FileCount    int        `db:"file_count" json:"file_count"`
	Expired      bool       `db:"-" json:"expired"`
	ShortURL     string     `db:"-" json:"short_url"`
}

type ShortLinkFile struct {
	ID           int64     `db:"id" json:"id"`
	ShortLinkID  int64     `db:"short_link_id" json:"short_link_id"`
	UploadedBy   *int64    `db:"uploaded_by" json:"uploaded_by"`
	OriginalName string    `db:"original_name" json:"original_name"`
	Path         string    `db:"path" json:"-"`
	URL          string    `db:"url" json:"-"`
	Disk         string    `db:"disk" json:"disk"`
	MimeType     *string   `db:"mime_type" json:"mime_type"`
	Size         int64     `db:"size" json:"size"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	UploaderName *string   `db:"uploader_name" json:"uploader_name"`
	DownloadURL  string    `db:"-" json:"download_url"`
}

type ShortLinkClick struct {
	ID        int64     `db:"id" json:"id"`
	ClickedAt time.Time `db:"clicked_at" json:"clicked_at"`
	Referrer  *string   `db:"referrer" json:"referrer"`
	UserAgent *string   `db:"user_agent" json:"user_agent"`
}

type ShortLinkDaily struct {
	Day    string `db:"day" json:"day"`
	Count  int    `db:"count" json:"count"`
	Unique int    `db:"unique_clicks" json:"unique_clicks"`
}

type ShortLinkSource struct {
	Source string `db:"source" json:"source"`
	Count  int    `db:"count" json:"count"`
}

type ShortLinkStats struct {
	Total        int   `db:"total" json:"total"`
	Active       int   `db:"active" json:"active"`
	Inactive     int   `db:"inactive" json:"inactive"`
	Expired      int   `db:"expired" json:"expired"`
	Clicks       int64 `db:"clicks" json:"clicks"`
	UniqueClicks int64 `db:"unique_clicks" json:"unique_clicks"`
}

type ShortLinkFilter struct {
	Search  string
	Status  string
	Page    int
	PerPage int
}

type ShortLinkInput struct {
	Code        string
	TargetURL   string
	Title       string
	Description string
	ExpiresAt   string
	IsActive    bool
}

const shortCodeAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func generateShortCode(length int) (string, error) {
	if length < 4 {
		length = 7
	}
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = shortCodeAlphabet[int(random[i])%len(shortCodeAlphabet)]
	}
	return string(buf), nil
}

func (s *Store) ListShortLinks(filter ShortLinkFilter) ([]ShortLink, int, ShortLinkStats, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(l.code LIKE ? OR l.target_url LIKE ? OR COALESCE(l.title, '') LIKE ?
			OR COALESCE(l.description, '') LIKE ? OR u.name LIKE ? OR COALESCE(u.surname, '') LIKE ?)`)
		for i := 0; i < 6; i++ {
			args = append(args, like)
		}
	}
	timestamp := now()
	switch filter.Status {
	case "active":
		conditions = append(conditions, "l.is_active = 1 AND (l.expires_at IS NULL OR l.expires_at > ?)")
		args = append(args, timestamp)
	case "inactive":
		conditions = append(conditions, "l.is_active = 0")
	case "expired":
		conditions = append(conditions, "l.expires_at IS NOT NULL AND l.expires_at <= ?")
		args = append(args, timestamp)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	where := strings.Join(conditions, " AND ")
	base := ` FROM short_links l JOIN users u ON u.id = l.user_id WHERE ` + where
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*)`+base, args...); err != nil {
		return nil, 0, ShortLinkStats{}, err
	}
	query := `SELECT l.*, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS creator_name,
			(SELECT COUNT(DISTINCT c.visitor_hash) FROM short_link_clicks c
				WHERE c.short_link_id = l.id AND c.visitor_hash IS NOT NULL AND c.visitor_hash <> '') AS unique_clicks,
			(SELECT MAX(c.clicked_at) FROM short_link_clicks c WHERE c.short_link_id = l.id) AS last_click_at,
			(SELECT COUNT(*) FROM short_link_files f WHERE f.short_link_id = l.id) AS file_count` +
		base + ` ORDER BY l.created_at DESC, l.id DESC LIMIT ? OFFSET ?`
	items := []ShortLink{}
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	if err := s.DB.Select(&items, query, queryArgs...); err != nil {
		return nil, 0, ShortLinkStats{}, err
	}
	decorateShortLinks(items)
	stats, err := s.ShortLinkStats()
	return items, total, stats, err
}

func decorateShortLinks(items []ShortLink) {
	timestamp := now()
	for i := range items {
		items[i].Expired = items[i].ExpiresAt != nil && !items[i].ExpiresAt.After(timestamp)
	}
}

func (s *Store) ShortLinkStats() (ShortLinkStats, error) {
	var stats ShortLinkStats
	timestamp := now()
	err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COALESCE(SUM(is_active = 1 AND (expires_at IS NULL OR expires_at > ?)), 0) AS active,
		COALESCE(SUM(is_active = 0), 0) AS inactive,
		COALESCE(SUM(expires_at IS NOT NULL AND expires_at <= ?), 0) AS expired,
		COALESCE(SUM(click_count), 0) AS clicks
		FROM short_links`, timestamp, timestamp)
	if err != nil {
		return stats, err
	}
	_ = s.DB.Get(&stats.UniqueClicks, `SELECT COUNT(DISTINCT visitor_hash) FROM short_link_clicks
		WHERE visitor_hash IS NOT NULL AND visitor_hash <> ''`)
	return stats, nil
}

const shortLinkSelect = `SELECT l.*, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS creator_name,
	(SELECT COUNT(DISTINCT c.visitor_hash) FROM short_link_clicks c
		WHERE c.short_link_id = l.id AND c.visitor_hash IS NOT NULL AND c.visitor_hash <> '') AS unique_clicks,
	(SELECT MAX(c.clicked_at) FROM short_link_clicks c WHERE c.short_link_id = l.id) AS last_click_at,
	(SELECT COUNT(*) FROM short_link_files f WHERE f.short_link_id = l.id) AS file_count
	FROM short_links l JOIN users u ON u.id = l.user_id`

func (s *Store) FindShortLink(id int64) (ShortLink, error) {
	var item ShortLink
	err := s.DB.Get(&item, shortLinkSelect+` WHERE l.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.Expired = item.ExpiresAt != nil && !item.ExpiresAt.After(now())
	return item, nil
}

func (s *Store) FindShortLinkByCode(code string) (ShortLink, error) {
	var item ShortLink
	err := s.DB.Get(&item, `SELECT * FROM short_links WHERE code = ?`, strings.TrimSpace(code))
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.Expired = item.ExpiresAt != nil && !item.ExpiresAt.After(now())
	return item, nil
}

func (s *Store) CreateShortLink(userID int64, input ShortLinkInput) (ShortLink, error) {
	timestamp := now()
	code := strings.ToLower(strings.TrimSpace(input.Code))
	expires := nullableTime(input.ExpiresAt)
	if code == "" {
		var err error
		for attempt := 0; attempt < 10; attempt++ {
			code, err = generateShortCode(7)
			if err != nil {
				return ShortLink{}, err
			}
			result, execErr := s.DB.Exec(`
				INSERT INTO short_links (user_id, code, target_url, title, description, is_active, expires_at, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				userID, code, strings.TrimSpace(input.TargetURL), nullString(input.Title), nullString(input.Description),
				input.IsActive, expires, timestamp, timestamp)
			if execErr == nil {
				id, _ := result.LastInsertId()
				return s.FindShortLink(id)
			}
			if !strings.Contains(strings.ToLower(execErr.Error()), "duplicate") {
				return ShortLink{}, execErr
			}
		}
		return ShortLink{}, fmt.Errorf("não foi possível gerar um código único")
	}
	result, err := s.DB.Exec(`
		INSERT INTO short_links (user_id, code, target_url, title, description, is_active, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, code, strings.TrimSpace(input.TargetURL), nullString(input.Title), nullString(input.Description),
		input.IsActive, expires, timestamp, timestamp)
	if err != nil {
		return ShortLink{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindShortLink(id)
}

func (s *Store) UpdateShortLink(id int64, input ShortLinkInput) (ShortLink, error) {
	current, err := s.FindShortLink(id)
	if err != nil {
		return current, err
	}
	code := strings.ToLower(strings.TrimSpace(input.Code))
	if code == "" {
		code = current.Code
	}
	_, err = s.DB.Exec(`UPDATE short_links SET code = ?, target_url = ?, title = ?, description = ?,
		is_active = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
		code, strings.TrimSpace(input.TargetURL), nullString(input.Title), nullString(input.Description),
		input.IsActive, nullableTime(input.ExpiresAt), now(), id)
	if err != nil {
		return current, err
	}
	return s.FindShortLink(id)
}

func (s *Store) ToggleShortLink(id int64) (ShortLink, error) {
	current, err := s.FindShortLink(id)
	if err != nil {
		return current, err
	}
	_, err = s.DB.Exec(`UPDATE short_links SET is_active = ?, updated_at = ? WHERE id = ?`,
		!current.IsActive, now(), id)
	if err != nil {
		return current, err
	}
	return s.FindShortLink(id)
}

func (s *Store) DeleteShortLink(id int64) ([]string, error) {
	paths := []string{}
	_ = s.DB.Select(&paths, `SELECT path FROM short_link_files WHERE short_link_id = ?`, id)
	_, err := s.DB.Exec(`DELETE FROM short_links WHERE id = ?`, id)
	return paths, err
}

func (s *Store) ShortLinkFiles(linkID int64) ([]ShortLinkFile, error) {
	items := []ShortLinkFile{}
	err := s.DB.Select(&items, `SELECT f.*, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS uploader_name
		FROM short_link_files f LEFT JOIN users u ON u.id = f.uploaded_by
		WHERE f.short_link_id = ? ORDER BY f.created_at ASC, f.id ASC`, linkID)
	return items, err
}

func (s *Store) FindShortLinkFile(linkID, fileID int64) (ShortLinkFile, error) {
	var item ShortLinkFile
	err := s.DB.Get(&item, `SELECT * FROM short_link_files WHERE id = ? AND short_link_id = ?`, fileID, linkID)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}

func (s *Store) CreateShortLinkFile(linkID, userID int64, originalName, path, url, disk, mimeType string, size int64) (ShortLinkFile, error) {
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO short_link_files
		(short_link_id, uploaded_by, original_name, path, url, disk, mime_type, size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		linkID, userID, originalName, path, url, disk, nullString(mimeType), size, timestamp, timestamp)
	if err != nil {
		return ShortLinkFile{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindShortLinkFile(linkID, id)
}

func (s *Store) DeleteShortLinkFile(linkID, fileID int64) (ShortLinkFile, error) {
	file, err := s.FindShortLinkFile(linkID, fileID)
	if err != nil {
		return file, err
	}
	_, err = s.DB.Exec(`DELETE FROM short_link_files WHERE id = ? AND short_link_id = ?`, fileID, linkID)
	return file, err
}

func (s *Store) RecordShortLinkClick(id int64, visitorHash, referrer, userAgent string) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	timestamp := now()
	if _, err := tx.Exec(`INSERT INTO short_link_clicks
		(short_link_id, visitor_hash, referrer, user_agent, clicked_at) VALUES (?, ?, ?, ?, ?)`,
		id, nullString(visitorHash), nullString(referrer), nullString(userAgent), timestamp); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE short_links SET click_count = click_count + 1 WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ShortLinkClicks(linkID int64, limit int) ([]ShortLinkClick, error) {
	if limit <= 0 {
		limit = 20
	}
	items := []ShortLinkClick{}
	err := s.DB.Select(&items, `SELECT id, clicked_at, referrer, user_agent FROM short_link_clicks
		WHERE short_link_id = ? ORDER BY clicked_at DESC, id DESC LIMIT ?`, linkID, limit)
	return items, err
}

func (s *Store) ShortLinkDaily(linkID int64, days int) ([]ShortLinkDaily, error) {
	if days <= 0 {
		days = 14
	}
	from := time.Now().AddDate(0, 0, -(days - 1))
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	items := []ShortLinkDaily{}
	err := s.DB.Select(&items, `SELECT DATE_FORMAT(clicked_at, '%Y-%m-%d') AS day, COUNT(*) AS count,
		COUNT(DISTINCT NULLIF(visitor_hash, '')) AS unique_clicks
		FROM short_link_clicks WHERE short_link_id = ? AND clicked_at >= ?
		GROUP BY day ORDER BY day ASC`, linkID, from)
	return items, err
}

func (s *Store) ShortLinkSources(linkID int64, limit int) ([]ShortLinkSource, error) {
	if limit <= 0 {
		limit = 5
	}
	items := []ShortLinkSource{}
	err := s.DB.Select(&items, `SELECT COALESCE(NULLIF(referrer, ''), 'Direto') AS source, COUNT(*) AS count
		FROM short_link_clicks WHERE short_link_id = ?
		GROUP BY source ORDER BY count DESC LIMIT ?`, linkID, limit)
	return items, err
}

func (s *Store) ShortLinkOwnedBy(id, userID int64) (bool, error) {
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM short_links WHERE id = ? AND user_id = ?`, id, userID)
	return count > 0, err
}
