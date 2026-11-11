package modules

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Demand dispatch module
 * ----------------------------------------------------------------------- */

type DemandUpdate struct {
	ID         int64     `db:"id" json:"id"`
	DemandID   int64     `db:"demand_id" json:"-"`
	UserID     *int64    `db:"user_id" json:"user_id"`
	Body       string    `db:"body" json:"body"`
	StatusFrom *string   `db:"status_from" json:"status_from"`
	StatusTo   *string   `db:"status_to" json:"status_to"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	AuthorName *string   `db:"author_name" json:"author_name"`
}

type DemandAttachment struct {
	ID             int64     `db:"id" json:"id"`
	DemandID       int64     `db:"demand_id" json:"-"`
	UploadedBy     *int64    `db:"uploaded_by" json:"uploaded_by"`
	OriginalName   string    `db:"original_name" json:"original_name"`
	Path           string    `db:"path" json:"-"`
	URL            string    `db:"url" json:"-"`
	Disk           string    `db:"disk" json:"disk"`
	MimeType       *string   `db:"mime_type" json:"mime_type"`
	Size           int64     `db:"size" json:"size"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
	UploaderName   *string   `db:"uploader_name" json:"uploader_name"`
	UploadedByName *string   `db:"uploaded_by_name" json:"uploaded_by_name"`
	DownloadURL    string    `db:"-" json:"download_url"`
}

type Demand struct {
	ID            int64              `db:"id" json:"id"`
	RequesterID   int64              `db:"requester_id" json:"requester_id"`
	AssigneeID    int64              `db:"assignee_id" json:"assignee_id"`
	Title         string             `db:"title" json:"title"`
	Description   *string            `db:"description" json:"description"`
	ClientName    *string            `db:"client_name" json:"client_name"`
	ClientEmail   *string            `db:"client_email" json:"client_email"`
	StationID     *int64             `db:"station_id" json:"station_id"`
	StationSiteID *string            `db:"station_site_id" json:"station_site_id"`
	StationCity   *string            `db:"station_municipio" json:"station_municipio"`
	StationState  *string            `db:"station_estado" json:"station_estado"`
	Operadora     *string            `db:"operadora" json:"operadora"`
	DemandType    *string            `db:"demand_type" json:"demand_type"`
	OCTIM         *string            `db:"oc_tim" json:"oc_tim"`
	ClientOS      *string            `db:"client_os" json:"client_os"`
	KeyNumber     *string            `db:"key_number" json:"key_number"`
	ActivityScope *string            `db:"activity_scope" json:"activity_scope"`
	Status        string             `db:"status" json:"status"`
	Priority      string             `db:"priority" json:"priority"`
	DueAt         *string            `db:"due_at" json:"due_at"`
	CompletedAt   *time.Time         `db:"completed_at" json:"completed_at"`
	CreatedAt     time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `db:"updated_at" json:"updated_at"`
	UploadToken   *string            `db:"upload_token" json:"-"`
	DownloadToken *string            `db:"download_token" json:"-"`
	UploadURL     string             `db:"-" json:"upload_url,omitempty"`
	DownloadURL   string             `db:"-" json:"download_url,omitempty"`
	RequesterName *string            `db:"requester_name" json:"requester_name"`
	AssigneeName  *string            `db:"assignee_name" json:"assignee_name"`
	AssigneeEmail *string            `db:"assignee_email" json:"assignee_email"`
	UpdateCount   int                `db:"update_count" json:"update_count"`
	Overdue       bool               `db:"-" json:"overdue"`
	Updates       []DemandUpdate     `db:"-" json:"updates"`
	Attachments   []DemandAttachment `db:"-" json:"attachments"`
}

type DemandFilter struct {
	Search     string
	Status     string
	Priority   string
	AssigneeID int64
	Scope      string
	Page       int
	PerPage    int
	UserID     int64
}

type DemandStats struct {
	Open       int `db:"open" json:"open"`
	InProgress int `db:"in_progress" json:"in_progress"`
	Waiting    int `db:"waiting" json:"waiting"`
	Completed  int `db:"completed" json:"completed"`
	Overdue    int `db:"overdue" json:"overdue"`
	Mine       int `db:"mine" json:"mine"`
	Created    int `db:"created" json:"created"`
}

type Collaborator struct {
	ID    int64  `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`
	Email string `db:"email" json:"email"`
}

var DemandStatuses = []string{"aberta", "em_andamento", "aguardando", "concluida", "cancelada"}
var DemandPriorities = []string{"baixa", "media", "alta", "urgente"}

var DemandStatusLabels = map[string]string{
	"aberta": "Aberta", "em_andamento": "Em andamento", "aguardando": "Aguardando",
	"concluida": "Concluída", "cancelada": "Cancelada",
}

func DemandStatusLabel(value string) string {
	if label, ok := DemandStatusLabels[value]; ok {
		return label
	}
	return value
}

func validDemandStatus(value string) bool {
	for _, status := range DemandStatuses {
		if status == value {
			return true
		}
	}
	return false
}

func validDemandPriority(value string) bool {
	for _, priority := range DemandPriorities {
		if priority == value {
			return true
		}
	}
	return false
}

type DemandStationOption struct {
	ID         int64   `db:"id" json:"id"`
	SiteID     string  `db:"site_id" json:"site_id"`
	StationID  *string `db:"station_id" json:"station_id"`
	EnderecoID *string `db:"endereco_id" json:"endereco_id"`
	Logradouro *string `db:"logradouro" json:"logradouro"`
	Municipio  *string `db:"municipio" json:"municipio"`
	Estado     *string `db:"estado" json:"estado"`
}

func (s *Store) SearchDemandStations(search string, limit int) ([]DemandStationOption, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	search = strings.TrimSpace(search)
	items := []DemandStationOption{}
	query := `SELECT id, site_id, station_id, endereco_id, logradouro, municipio, estado
		FROM stations`
	args := []any{}
	if search != "" {
		like := "%" + search + "%"
		query += ` WHERE (site_id LIKE ? OR COALESCE(station_id, '') LIKE ? OR COALESCE(endereco_id, '') LIKE ?
			OR COALESCE(logradouro, '') LIKE ? OR COALESCE(bairro, '') LIKE ? OR COALESCE(municipio, '') LIKE ?)`
		args = append(args, like, like, like, like, like, like)
	}
	if search != "" {
		query += ` ORDER BY site_id ASC LIMIT ?`
	} else {
		query += ` ORDER BY updated_at DESC LIMIT ?`
	}
	args = append(args, limit)
	err := s.DB.Select(&items, query, args...)
	return items, err
}

func (s *Store) DemandCollaborators() ([]Collaborator, error) {
	items := []Collaborator{}
	err := s.DB.Select(&items, `SELECT id, TRIM(CONCAT(name, ' ', COALESCE(surname, ''))) AS name, email
		FROM users WHERE is_active = 1 ORDER BY name ASC, surname ASC`)
	return items, err
}

func (s *Store) ListDemands(filter DemandFilter) ([]Demand, int, DemandStats, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(d.title LIKE ? OR COALESCE(d.description, '') LIKE ?
			OR TRIM(CONCAT(r.name, ' ', COALESCE(r.surname, ''))) LIKE ?
			OR TRIM(CONCAT(a.name, ' ', COALESCE(a.surname, ''))) LIKE ?)`)
		for i := 0; i < 4; i++ {
			args = append(args, like)
		}
	}
	if validDemandStatus(filter.Status) {
		conditions = append(conditions, "d.status = ?")
		args = append(args, filter.Status)
	}
	if validDemandPriority(filter.Priority) {
		conditions = append(conditions, "d.priority = ?")
		args = append(args, filter.Priority)
	}
	if filter.AssigneeID > 0 {
		conditions = append(conditions, "d.assignee_id = ?")
		args = append(args, filter.AssigneeID)
	}
	switch filter.Scope {
	case "mine":
		conditions = append(conditions, "d.assignee_id = ?")
		args = append(args, filter.UserID)
	case "created":
		conditions = append(conditions, "d.requester_id = ?")
		args = append(args, filter.UserID)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	where := strings.Join(conditions, " AND ")
	base := ` FROM demands d
		JOIN users r ON r.id = d.requester_id
		JOIN users a ON a.id = d.assignee_id
		WHERE ` + where
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*)`+base, args...); err != nil {
		return nil, 0, DemandStats{}, err
	}
	query := `SELECT d.*,
		TRIM(CONCAT(r.name, ' ', COALESCE(r.surname, ''))) AS requester_name,
		TRIM(CONCAT(a.name, ' ', COALESCE(a.surname, ''))) AS assignee_name,
		a.email AS assignee_email,
		(SELECT COUNT(*) FROM demand_updates u WHERE u.demand_id = d.id) AS update_count` +
		base + ` ORDER BY
			CASE d.status WHEN 'aberta' THEN 0 WHEN 'em_andamento' THEN 1 WHEN 'aguardando' THEN 2
				WHEN 'concluida' THEN 3 ELSE 4 END ASC,
			CASE WHEN d.due_at IS NULL THEN 1 ELSE 0 END ASC, d.due_at ASC, d.updated_at DESC, d.id DESC
			LIMIT ? OFFSET ?`
	items := []Demand{}
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	if err := s.DB.Select(&items, query, queryArgs...); err != nil {
		return nil, 0, DemandStats{}, err
	}
	decorateDemands(items)
	stats, err := s.DemandStats(filter.UserID)
	return items, total, stats, err
}

func decorateDemands(items []Demand) {
	today := time.Now().Format("2006-01-02")
	for i := range items {
		item := &items[i]
		if item.DueAt != nil && item.Status != "concluida" && item.Status != "cancelada" && *item.DueAt < today {
			item.Overdue = true
		}
	}
}

func (s *Store) DemandStats(userID int64) (DemandStats, error) {
	var stats DemandStats
	today := time.Now().Format("2006-01-02")
	err := s.DB.Get(&stats, `SELECT
		COALESCE(SUM(status IN ('aberta','em_andamento','aguardando')), 0) AS open,
		COALESCE(SUM(status = 'em_andamento'), 0) AS in_progress,
		COALESCE(SUM(status = 'aguardando'), 0) AS waiting,
		COALESCE(SUM(status = 'concluida'), 0) AS completed,
		COALESCE(SUM(status NOT IN ('concluida','cancelada') AND due_at IS NOT NULL AND due_at < ?), 0) AS overdue,
		COALESCE(SUM(assignee_id = ? AND status NOT IN ('concluida','cancelada')), 0) AS mine,
		COALESCE(SUM(requester_id = ?), 0) AS created
		FROM demands`, today, userID, userID)
	return stats, err
}

const demandSelect = `SELECT d.*,
	TRIM(CONCAT(r.name, ' ', COALESCE(r.surname, ''))) AS requester_name,
	TRIM(CONCAT(a.name, ' ', COALESCE(a.surname, ''))) AS assignee_name,
	a.email AS assignee_email,
	s.site_id AS station_site_id, s.municipio AS station_municipio, s.estado AS station_estado,
	(SELECT COUNT(*) FROM demand_updates u WHERE u.demand_id = d.id) AS update_count
	FROM demands d
	JOIN users r ON r.id = d.requester_id
	JOIN users a ON a.id = d.assignee_id
	LEFT JOIN stations s ON s.id = d.station_id`

func (s *Store) FindDemand(id int64) (Demand, error) {
	var item Demand
	err := s.DB.Get(&item, demandSelect+` WHERE d.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.Overdue = item.DueAt != nil && item.Status != "concluida" && item.Status != "cancelada" && *item.DueAt < time.Now().Format("2006-01-02")
	item.Updates = []DemandUpdate{}
	_ = s.DB.Select(&item.Updates, `SELECT u.*, TRIM(CONCAT(us.name, ' ', COALESCE(us.surname, ''))) AS author_name
		FROM demand_updates u LEFT JOIN users us ON us.id = u.user_id
		WHERE u.demand_id = ? ORDER BY u.created_at DESC, u.id DESC`, id)
	item.Attachments = []DemandAttachment{}
	_ = s.DB.Select(&item.Attachments, `SELECT at.*, TRIM(CONCAT(us.name, ' ', COALESCE(us.surname, ''))) AS uploader_name
		FROM demand_attachments at LEFT JOIN users us ON us.id = at.uploaded_by
		WHERE at.demand_id = ? ORDER BY at.created_at DESC, at.id DESC`, id)
	return item, nil
}

type DemandInput struct {
	AssigneeID    int64
	Title         string
	Description   string
	ClientName    string
	ClientEmail   string
	StationID     *int64
	Operadora     string
	DemandType    string
	OCTIM         string
	ClientOS      string
	KeyNumber     string
	ActivityScope string
	Status        string
	Priority      string
	DueAt         string
}

func (s *Store) CreateDemand(requesterID int64, input DemandInput) (Demand, error) {
	if input.Status == "" {
		input.Status = "aberta"
	}
	if input.Priority == "" {
		input.Priority = "media"
	}
	uploadToken, err := demandPublicToken()
	if err != nil {
		return Demand{}, err
	}
	downloadToken, err := demandPublicToken()
	if err != nil {
		return Demand{}, err
	}
	timestamp := now()
	var completedAt any
	if input.Status == "concluida" {
		completedAt = timestamp
	}
	result, err := s.DB.Exec(`INSERT INTO demands
		(requester_id, assignee_id, title, description, client_name, client_email,
		 station_id, operadora, demand_type, oc_tim, client_os, key_number, activity_scope,
		 upload_token, download_token, status, priority, due_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		requesterID, input.AssigneeID, strings.TrimSpace(input.Title), nullString(input.Description),
		nullString(input.ClientName), nullString(strings.ToLower(strings.TrimSpace(input.ClientEmail))),
		input.StationID, nullString(input.Operadora), nullString(input.DemandType), nullString(input.OCTIM),
		nullString(input.ClientOS), nullString(input.KeyNumber), nullString(input.ActivityScope),
		uploadToken, downloadToken, input.Status, input.Priority, nullDate(input.DueAt), completedAt, timestamp, timestamp)
	if err != nil {
		return Demand{}, err
	}
	id, _ := result.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO demand_updates (demand_id, user_id, body, status_from, status_to, created_at, updated_at)
		VALUES (?, ?, ?, NULL, ?, ?, ?)`, id, requesterID, "Demanda criada e encaminhada.", input.Status, timestamp, timestamp)
	return s.FindDemand(id)
}

func demandPublicToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func (s *Store) FindDemandByUploadToken(token string) (Demand, error) {
	return s.findDemandByToken("upload_token", token)
}

func (s *Store) FindDemandByDownloadToken(token string) (Demand, error) {
	return s.findDemandByToken("download_token", token)
}

func (s *Store) findDemandByToken(column, token string) (Demand, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Demand{}, ErrNotFound
	}
	var id int64
	if err := s.DB.Get(&id, `SELECT id FROM demands WHERE `+column+` = ?`, token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Demand{}, ErrNotFound
		}
		return Demand{}, err
	}
	return s.FindDemand(id)
}

func (s *Store) AddDemandSystemUpdate(id int64, body string) {
	timestamp := now()
	_, _ = s.DB.Exec(`INSERT INTO demand_updates (demand_id, user_id, body, created_at, updated_at)
		VALUES (?, NULL, ?, ?, ?)`, id, strings.TrimSpace(body), timestamp, timestamp)
}

func (s *Store) UpdateDemand(id, userID int64, input DemandInput) (Demand, error) {
	current, err := s.FindDemand(id)
	if err != nil {
		return current, err
	}
	if input.Status == "" {
		input.Status = current.Status
	}
	if input.Priority == "" {
		input.Priority = current.Priority
	}
	completedAt := current.CompletedAt
	if input.Status == "concluida" {
		if completedAt == nil {
			timestamp := now()
			completedAt = &timestamp
		}
	} else {
		completedAt = nil
	}
	_, err = s.DB.Exec(`UPDATE demands SET assignee_id = ?, title = ?, description = ?,
		client_name = ?, client_email = ?, station_id = ?, operadora = ?, demand_type = ?,
		oc_tim = ?, client_os = ?, key_number = ?, activity_scope = ?,
		status = ?, priority = ?, due_at = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
		input.AssigneeID, strings.TrimSpace(input.Title), nullString(input.Description),
		nullString(input.ClientName), nullString(strings.ToLower(strings.TrimSpace(input.ClientEmail))),
		input.StationID, nullString(input.Operadora), nullString(input.DemandType), nullString(input.OCTIM),
		nullString(input.ClientOS), nullString(input.KeyNumber), nullString(input.ActivityScope),
		input.Status, input.Priority, nullDate(input.DueAt), completedAt, now(), id)
	if err != nil {
		return current, err
	}
	if current.Status != input.Status {
		_, _ = s.DB.Exec(`INSERT INTO demand_updates (demand_id, user_id, body, status_from, status_to, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, id, userID,
			"Status alterado de "+DemandStatusLabel(current.Status)+" para "+DemandStatusLabel(input.Status)+".",
			current.Status, input.Status, now(), now())
	}
	return s.FindDemand(id)
}

func (s *Store) SetDemandStatus(id, userID int64, status string) (Demand, error) {
	current, err := s.FindDemand(id)
	if err != nil {
		return current, err
	}
	if current.Status == status {
		return current, nil
	}
	var completedAt any
	if status == "concluida" {
		completedAt = now()
	}
	_, err = s.DB.Exec(`UPDATE demands SET status = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
		status, completedAt, now(), id)
	if err != nil {
		return current, err
	}
	_, _ = s.DB.Exec(`INSERT INTO demand_updates (demand_id, user_id, body, status_from, status_to, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, id, userID,
		"Status alterado de "+DemandStatusLabel(current.Status)+" para "+DemandStatusLabel(status)+".",
		current.Status, status, now(), now())
	return s.FindDemand(id)
}

func (s *Store) AddDemandUpdate(id, userID int64, body string) (DemandUpdate, error) {
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO demand_updates (demand_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, id, userID, strings.TrimSpace(body), timestamp, timestamp)
	if err != nil {
		return DemandUpdate{}, err
	}
	updateID, _ := result.LastInsertId()
	var update DemandUpdate
	err = s.DB.Get(&update, `SELECT u.*, TRIM(CONCAT(us.name, ' ', COALESCE(us.surname, ''))) AS author_name
		FROM demand_updates u LEFT JOIN users us ON us.id = u.user_id WHERE u.id = ?`, updateID)
	return update, err
}

func (s *Store) DeleteDemand(id int64) ([]string, error) {
	paths := []string{}
	_ = s.DB.Select(&paths, `SELECT path FROM demand_attachments WHERE demand_id = ?`, id)
	_, err := s.DB.Exec(`DELETE FROM demands WHERE id = ?`, id)
	return paths, err
}

func (s *Store) CreateDemandAttachment(demandID, userID int64, originalName, path, url, disk, mimeType string, size int64, uploadedByName string) (DemandAttachment, error) {
	timestamp := now()
	var uploadedBy any
	if userID > 0 {
		uploadedBy = userID
	}
	result, err := s.DB.Exec(`INSERT INTO demand_attachments
		(demand_id, uploaded_by, uploaded_by_name, original_name, path, url, disk, mime_type, size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		demandID, uploadedBy, nullString(uploadedByName), originalName, path, url, disk, nullString(mimeType), size, timestamp, timestamp)
	if err != nil {
		return DemandAttachment{}, err
	}
	id, _ := result.LastInsertId()
	var attachment DemandAttachment
	err = s.DB.Get(&attachment, `SELECT * FROM demand_attachments WHERE id = ?`, id)
	return attachment, err
}

func (s *Store) FindDemandAttachment(demandID, attachmentID int64) (DemandAttachment, error) {
	var attachment DemandAttachment
	err := s.DB.Get(&attachment, `SELECT * FROM demand_attachments WHERE id = ? AND demand_id = ?`, attachmentID, demandID)
	if errors.Is(err, sql.ErrNoRows) {
		return attachment, ErrNotFound
	}
	return attachment, err
}

func (s *Store) DeleteDemandAttachment(demandID, attachmentID int64) (DemandAttachment, error) {
	attachment, err := s.FindDemandAttachment(demandID, attachmentID)
	if err != nil {
		return attachment, err
	}
	_, err = s.DB.Exec(`DELETE FROM demand_attachments WHERE id = ? AND demand_id = ?`, attachmentID, demandID)
	return attachment, err
}

func (s *Store) DemandMineWidget(userID int64, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}
	items := []Demand{}
	err := s.DB.Select(&items, demandSelect+` WHERE d.assignee_id = ?
		AND d.status NOT IN ('concluida','cancelada')
		ORDER BY CASE WHEN d.due_at IS NULL THEN 1 ELSE 0 END, d.due_at ASC, d.id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	decorateDemands(items)
	stats, _ := s.DemandStats(userID)
	return map[string]any{"items": items, "stats": stats}, nil
}
