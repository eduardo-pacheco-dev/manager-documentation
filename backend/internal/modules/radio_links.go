package modules

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Radio links module (point-to-point links between stations)
 * ----------------------------------------------------------------------- */

// RadioLink is a point-to-point radio connection between two stations.
type RadioLink struct {
	ID            int64                 `db:"id" json:"id"`
	Name          string                `db:"name" json:"name"`
	Code          *string               `db:"code" json:"code"`
	StationAID    int64                 `db:"station_a_id" json:"station_a_id"`
	StationBID    int64                 `db:"station_b_id" json:"station_b_id"`
	Status        string                `db:"status" json:"status"`
	Technology    *string               `db:"technology" json:"technology"`
	FrequencyMHz  *float64              `db:"frequency_mhz" json:"frequency_mhz"`
	CapacityMbps  *float64              `db:"capacity_mbps" json:"capacity_mbps"`
	DistanceKm    *float64              `db:"distance_km" json:"distance_km"`
	Notes         *string               `db:"notes" json:"notes"`
	CreatedAt     time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time             `db:"updated_at" json:"updated_at"`
	StationASite  *string               `db:"station_a_site_id" json:"station_a_site_id"`
	StationACode  *string               `db:"station_a_station_id" json:"station_a_station_id"`
	StationACity  *string               `db:"station_a_municipio" json:"station_a_municipio"`
	StationAState *string               `db:"station_a_estado" json:"station_a_estado"`
	StationAType  *string               `db:"station_a_tipo_de_elemento" json:"station_a_tipo_de_elemento"`
	StationBSite  *string               `db:"station_b_site_id" json:"station_b_site_id"`
	StationBCode  *string               `db:"station_b_station_id" json:"station_b_station_id"`
	StationBCity  *string               `db:"station_b_municipio" json:"station_b_municipio"`
	StationBState *string               `db:"station_b_estado" json:"station_b_estado"`
	StationBType  *string               `db:"station_b_tipo_de_elemento" json:"station_b_tipo_de_elemento"`
	Attachments   []RadioLinkAttachment `db:"-" json:"attachments"`
	Comments      []RadioLinkComment    `db:"-" json:"comments"`
}

type RadioLinkAttachment struct {
	ID           int64     `db:"id" json:"id"`
	RadioLinkID  int64     `db:"radio_link_id" json:"-"`
	UploadedBy   *int64    `db:"uploaded_by" json:"uploaded_by"`
	OriginalName string    `db:"original_name" json:"original_name"`
	Path         string    `db:"path" json:"path"`
	Disk         string    `db:"disk" json:"disk"`
	MimeType     *string   `db:"mime_type" json:"mime_type"`
	Size         int64     `db:"size" json:"size"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	UploaderName *string   `db:"uploader_name" json:"uploader_name"`
	URL          string    `db:"-" json:"url"`
}

type RadioLinkComment struct {
	ID          int64     `db:"id" json:"id"`
	RadioLinkID int64     `db:"radio_link_id" json:"-"`
	UserID      *int64    `db:"user_id" json:"user_id"`
	Body        string    `db:"body" json:"body"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	AuthorName  *string   `db:"author_name" json:"author_name"`
}

type RadioLinkStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Planned  int `json:"planned"`
	Inactive int `json:"inactive"`
	Stations int `json:"stations"`
}

type RadioLinkFilter struct {
	Search     string
	Status     string
	Technology string
	Page       int
	PerPage    int
}

const radioLinkSelect = `SELECT r.*,
	a.site_id AS station_a_site_id, a.station_id AS station_a_station_id,
	a.municipio AS station_a_municipio, a.estado AS station_a_estado,
	a.tipo_de_elemento AS station_a_tipo_de_elemento,
	b.site_id AS station_b_site_id, b.station_id AS station_b_station_id,
	b.municipio AS station_b_municipio, b.estado AS station_b_estado,
	b.tipo_de_elemento AS station_b_tipo_de_elemento
	FROM radio_links r
	JOIN stations a ON a.id = r.station_a_id
	JOIN stations b ON b.id = r.station_b_id`

var radioLinkOrder = ` ORDER BY r.updated_at DESC, r.id DESC`

func (s *Store) ListRadioLinks(filter RadioLinkFilter) ([]RadioLink, int, RadioLinkStats, []string, []string, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(r.name LIKE ? OR COALESCE(r.code, '') LIKE ? OR COALESCE(r.technology, '') LIKE ? OR a.site_id LIKE ? OR b.site_id LIKE ? OR COALESCE(a.station_id, '') LIKE ? OR COALESCE(b.station_id, '') LIKE ? OR COALESCE(a.municipio, '') LIKE ? OR COALESCE(b.municipio, '') LIKE ?)`)
		for i := 0; i < 9; i++ {
			args = append(args, like)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "r.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Technology != "" {
		conditions = append(conditions, "r.technology = ?")
		args = append(args, filter.Technology)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 15
	}
	where := strings.Join(conditions, " AND ")

	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM radio_links r
		JOIN stations a ON a.id = r.station_a_id
		JOIN stations b ON b.id = r.station_b_id WHERE `+where, args...); err != nil {
		return nil, 0, RadioLinkStats{}, nil, nil, err
	}

	query := radioLinkSelect + ` WHERE ` + where + radioLinkOrder + ` LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	links := []RadioLink{}
	if err := s.DB.Select(&links, query, queryArgs...); err != nil {
		return nil, 0, RadioLinkStats{}, nil, nil, err
	}

	var stats RadioLinkStats
	if err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COALESCE(SUM(status = 'ativo'), 0) AS active,
		COALESCE(SUM(status = 'planejado'), 0) AS planned,
		COALESCE(SUM(status = 'inativo'), 0) AS inactive,
		COUNT(DISTINCT station_a_id) + COUNT(DISTINCT station_b_id) AS stations
		FROM radio_links`); err != nil {
		return nil, 0, stats, nil, nil, err
	}
	statuses := []string{}
	technologies := []string{}
	_ = s.DB.Select(&statuses, `SELECT DISTINCT status FROM radio_links WHERE status <> '' ORDER BY status`)
	_ = s.DB.Select(&technologies, `SELECT DISTINCT technology FROM radio_links WHERE technology IS NOT NULL AND technology <> '' ORDER BY technology`)
	return links, total, stats, statuses, technologies, nil
}

func (s *Store) FindRadioLink(id int64) (RadioLink, error) {
	var link RadioLink
	err := s.DB.Get(&link, radioLinkSelect+` WHERE r.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return link, ErrNotFound
	}
	if err != nil {
		return link, err
	}
	link.Attachments = []RadioLinkAttachment{}
	link.Comments = []RadioLinkComment{}
	_ = s.DB.Select(&link.Attachments, `
		SELECT a.*, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS uploader_name
		FROM radio_link_attachments a LEFT JOIN users u ON u.id = a.uploaded_by
		WHERE a.radio_link_id = ? ORDER BY a.created_at DESC, a.id DESC`, id)
	for index := range link.Attachments {
		link.Attachments[index].URL = "/uploads/" + link.Attachments[index].Path
	}
	_ = s.DB.Select(&link.Comments, `
		SELECT c.*, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS author_name
		FROM radio_link_comments c LEFT JOIN users u ON u.id = c.user_id
		WHERE c.radio_link_id = ? ORDER BY c.created_at DESC, c.id DESC`, id)
	return link, nil
}

type RadioLinkInput struct {
	Name         string
	Code         string
	StationAID   int64
	StationBID   int64
	Status       string
	Technology   string
	FrequencyMHz *float64
	CapacityMbps *float64
	DistanceKm   *float64
	Notes        string
}

func (s *Store) CreateRadioLink(input RadioLinkInput) (RadioLink, error) {
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO radio_links (name, code, station_a_id, station_b_id, status, technology,
			frequency_mhz, capacity_mbps, distance_km, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Name), nullString(input.Code), input.StationAID, input.StationBID,
		input.Status, nullString(input.Technology), nullFloat(input.FrequencyMHz),
		nullFloat(input.CapacityMbps), nullFloat(input.DistanceKm), nullString(input.Notes),
		timestamp, timestamp)
	if err != nil {
		return RadioLink{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindRadioLink(id)
}

func (s *Store) UpdateRadioLink(id int64, input RadioLinkInput) (RadioLink, error) {
	result, err := s.DB.Exec(`
		UPDATE radio_links SET name = ?, code = ?, station_a_id = ?, station_b_id = ?, status = ?,
			technology = ?, frequency_mhz = ?, capacity_mbps = ?, distance_km = ?, notes = ?, updated_at = ?
		WHERE id = ?`,
		strings.TrimSpace(input.Name), nullString(input.Code), input.StationAID, input.StationBID,
		input.Status, nullString(input.Technology), nullFloat(input.FrequencyMHz),
		nullFloat(input.CapacityMbps), nullFloat(input.DistanceKm), nullString(input.Notes),
		now(), id)
	if err != nil {
		return RadioLink{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return RadioLink{}, ErrNotFound
	}
	return s.FindRadioLink(id)
}

func (s *Store) DeleteRadioLink(id int64) ([]string, error) {
	paths := []string{}
	_ = s.DB.Select(&paths, `SELECT path FROM radio_link_attachments WHERE radio_link_id = ?`, id)
	result, err := s.DB.Exec(`DELETE FROM radio_links WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, ErrNotFound
	}
	return paths, nil
}

func (s *Store) CreateRadioLinkAttachment(linkID, userID int64, originalName, path, disk, mimeType string, size int64) (RadioLinkAttachment, error) {
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO radio_link_attachments (radio_link_id, uploaded_by, original_name, path, disk, mime_type, size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		linkID, userID, originalName, path, disk, nullString(mimeType), size, timestamp, timestamp)
	if err != nil {
		return RadioLinkAttachment{}, err
	}
	id, _ := result.LastInsertId()
	var attachment RadioLinkAttachment
	err = s.DB.Get(&attachment, `SELECT * FROM radio_link_attachments WHERE id = ?`, id)
	attachment.URL = "/uploads/" + attachment.Path
	return attachment, err
}

func (s *Store) FindRadioLinkAttachment(linkID, attachmentID int64) (RadioLinkAttachment, error) {
	var attachment RadioLinkAttachment
	err := s.DB.Get(&attachment, `SELECT * FROM radio_link_attachments WHERE id = ? AND radio_link_id = ?`, attachmentID, linkID)
	if errors.Is(err, sql.ErrNoRows) {
		return attachment, ErrNotFound
	}
	return attachment, err
}

func (s *Store) DeleteRadioLinkAttachment(linkID, attachmentID int64) (string, error) {
	attachment, err := s.FindRadioLinkAttachment(linkID, attachmentID)
	if err != nil {
		return "", err
	}
	_, err = s.DB.Exec(`DELETE FROM radio_link_attachments WHERE id = ?`, attachmentID)
	return attachment.Path, err
}

func (s *Store) CreateRadioLinkComment(linkID, userID int64, body string) (RadioLinkComment, error) {
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO radio_link_comments (radio_link_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, linkID, userID, strings.TrimSpace(body), timestamp, timestamp)
	if err != nil {
		return RadioLinkComment{}, err
	}
	id, _ := result.LastInsertId()
	var comment RadioLinkComment
	err = s.DB.Get(&comment, `
		SELECT c.*, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS author_name
		FROM radio_link_comments c LEFT JOIN users u ON u.id = c.user_id WHERE c.id = ?`, id)
	return comment, err
}

func (s *Store) FindRadioLinkComment(linkID, commentID int64) (RadioLinkComment, error) {
	var comment RadioLinkComment
	err := s.DB.Get(&comment, `SELECT * FROM radio_link_comments WHERE id = ? AND radio_link_id = ?`, commentID, linkID)
	if errors.Is(err, sql.ErrNoRows) {
		return comment, ErrNotFound
	}
	return comment, err
}

func (s *Store) DeleteRadioLinkComment(linkID, commentID int64) error {
	_, err := s.DB.Exec(`DELETE FROM radio_link_comments WHERE id = ? AND radio_link_id = ?`, commentID, linkID)
	return err
}

func nullFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
