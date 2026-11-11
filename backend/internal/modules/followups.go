package modules

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * FollowUp module
 * ----------------------------------------------------------------------- */

type FollowUpClient struct {
	ID            int64             `db:"id" json:"id"`
	Name          string            `db:"name" json:"name"`
	CreatedAt     time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time         `db:"updated_at" json:"updated_at"`
	ProjectCount  int               `db:"project_count" json:"project_count"`
	FollowUpCount int               `db:"follow_up_count" json:"follow_up_count"`
	Projects      []FollowUpProject `db:"-" json:"projects"`
}

type FollowUpProject struct {
	ID        int64     `db:"id" json:"id"`
	ClientID  int64     `db:"client_id" json:"client_id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type FollowUpAssignee struct {
	ID            int64     `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Email         *string   `db:"email" json:"email"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	FollowUpCount int       `db:"follow_up_count" json:"follow_up_count"`
}

type FollowUp struct {
	ID            int64            `db:"id" json:"id"`
	Title         string           `db:"title" json:"title"`
	Code          *string          `db:"code" json:"code"`
	Description   *string          `db:"description" json:"description"`
	ClientID      *int64           `db:"client_id" json:"client_id"`
	ProjectID     *int64           `db:"project_id" json:"project_id"`
	Status        string           `db:"status" json:"status"`
	Priority      string           `db:"priority" json:"priority"`
	DueDate       *string          `db:"due_date" json:"due_date"`
	AssigneeID    *int64           `db:"assignee_id" json:"assignee_id"`
	CreatedBy     *int64           `db:"created_by" json:"created_by"`
	CompletedAt   *time.Time       `db:"completed_at" json:"completed_at"`
	CreatedAt     time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time        `db:"updated_at" json:"updated_at"`
	ClientName    *string          `db:"client_name" json:"client_name"`
	ProjectName   *string          `db:"project_name" json:"project_name"`
	AssigneeName  *string          `db:"assignee_name" json:"assignee_name"`
	AssigneeEmail *string          `db:"assignee_email" json:"assignee_email"`
	CreatorName   *string          `db:"creator_name" json:"creator_name"`
	Updates       []FollowUpUpdate `db:"-" json:"updates"`
	Overdue       bool             `json:"overdue"`
	DueSoon       bool             `json:"due_soon"`
}

type FollowUpUpdate struct {
	ID         int64     `db:"id" json:"id"`
	FollowUpID int64     `db:"follow_up_id" json:"-"`
	UserID     *int64    `db:"user_id" json:"user_id"`
	Body       string    `db:"body" json:"body"`
	StatusFrom *string   `db:"status_from" json:"status_from"`
	StatusTo   *string   `db:"status_to" json:"status_to"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	AuthorName *string   `db:"author_name" json:"author_name"`
}

type FollowUpSettings struct {
	ID                  int64     `db:"id" json:"-"`
	PerPage             int       `db:"per_page" json:"per_page"`
	DueSoonDays         int       `db:"due_soon_days" json:"due_soon_days"`
	WidgetLimit         int       `db:"widget_limit" json:"widget_limit"`
	EmailSummaryEnabled bool      `db:"email_summary_enabled" json:"email_summary_enabled"`
	EmailSubjectPrefix  string    `db:"email_subject_prefix" json:"email_subject_prefix"`
	SummaryPeriod       string    `db:"summary_period" json:"summary_period"`
	SummaryTimes        string    `db:"summary_times" json:"summary_times"`
	SummaryDays         string    `db:"summary_days" json:"summary_days"`
	SummaryRecipients   *string   `db:"summary_recipients" json:"summary_recipients"`
	SummaryCCRecipients *string   `db:"summary_cc_recipients" json:"summary_cc_recipients"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

type FollowUpFilter struct {
	Search    string
	Status    string
	Priority  string
	Assigned  string
	Sort      string
	Direction string
	Page      int
	PerPage   int
	UserEmail string
}

type FollowUpStats struct {
	Open      int `json:"open"`
	Overdue   int `json:"overdue"`
	Completed int `json:"completed"`
	Mine      int `json:"mine"`
}

func (s *Store) FollowUpSettings() (FollowUpSettings, error) {
	var settings FollowUpSettings
	err := s.DB.Get(&settings, `SELECT * FROM follow_up_settings WHERE id = 1`)
	return settings, err
}

func (s *Store) SaveFollowUpSettings(settings FollowUpSettings) error {
	_, err := s.DB.Exec(`UPDATE follow_up_settings SET per_page = ?, due_soon_days = ?, widget_limit = ?,
		email_summary_enabled = ?, email_subject_prefix = ?, summary_period = ?, summary_times = ?,
		summary_days = ?, summary_recipients = ?, summary_cc_recipients = ?, updated_at = ? WHERE id = 1`,
		settings.PerPage, settings.DueSoonDays, settings.WidgetLimit, settings.EmailSummaryEnabled,
		settings.EmailSubjectPrefix, settings.SummaryPeriod, settings.SummaryTimes, settings.SummaryDays,
		settings.SummaryRecipients, settings.SummaryCCRecipients, now())
	return err
}

func (s *Store) FollowUpClients() ([]FollowUpClient, error) {
	clients := []FollowUpClient{}
	if err := s.DB.Select(&clients, `SELECT c.*,
		(SELECT COUNT(*) FROM follow_up_projects p WHERE p.client_id = c.id) AS project_count,
		(SELECT COUNT(*) FROM follow_ups f WHERE f.client_id = c.id) AS follow_up_count
		FROM follow_up_clients c ORDER BY c.name`); err != nil {
		return nil, err
	}
	for i := range clients {
		clients[i].Projects = []FollowUpProject{}
		_ = s.DB.Select(&clients[i].Projects, `SELECT * FROM follow_up_projects WHERE client_id = ? ORDER BY name`, clients[i].ID)
	}
	return clients, nil
}

func (s *Store) CreateFollowUpClient(name string) (FollowUpClient, error) {
	result, err := s.DB.Exec(`INSERT INTO follow_up_clients (name, created_at, updated_at) VALUES (?, ?, ?)`,
		strings.TrimSpace(name), now(), now())
	if err != nil {
		return FollowUpClient{}, err
	}
	id, _ := result.LastInsertId()
	var client FollowUpClient
	err = s.DB.Get(&client, `SELECT * FROM follow_up_clients WHERE id = ?`, id)
	return client, err
}

func (s *Store) UpdateFollowUpClient(id int64, name string) error {
	_, err := s.DB.Exec(`UPDATE follow_up_clients SET name = ?, updated_at = ? WHERE id = ?`, strings.TrimSpace(name), now(), id)
	return err
}

func (s *Store) DeleteFollowUpClient(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM follow_up_clients WHERE id = ?`, id)
	return err
}

func (s *Store) CreateFollowUpProject(clientID int64, name string) (FollowUpProject, error) {
	result, err := s.DB.Exec(`INSERT INTO follow_up_projects (client_id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		clientID, strings.TrimSpace(name), now(), now())
	if err != nil {
		return FollowUpProject{}, err
	}
	id, _ := result.LastInsertId()
	var project FollowUpProject
	err = s.DB.Get(&project, `SELECT * FROM follow_up_projects WHERE id = ?`, id)
	return project, err
}

func (s *Store) UpdateFollowUpProject(id, clientID int64, name string) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE follow_up_projects SET client_id = ?, name = ?, updated_at = ? WHERE id = ?`,
		clientID, strings.TrimSpace(name), now(), id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE follow_ups SET client_id = ?, updated_at = ? WHERE project_id = ?`, clientID, now(), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteFollowUpProject(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM follow_up_projects WHERE id = ?`, id)
	return err
}

func (s *Store) FollowUpAssignees() ([]FollowUpAssignee, error) {
	assignees := []FollowUpAssignee{}
	err := s.DB.Select(&assignees, `SELECT a.*,
		(SELECT COUNT(*) FROM follow_ups f WHERE f.assignee_id = a.id) AS follow_up_count
		FROM follow_up_assignees a ORDER BY a.name`)
	return assignees, err
}

func (s *Store) CreateFollowUpAssignee(name, email string) (FollowUpAssignee, error) {
	result, err := s.DB.Exec(`INSERT INTO follow_up_assignees (name, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		strings.TrimSpace(name), nullString(strings.ToLower(email)), now(), now())
	if err != nil {
		return FollowUpAssignee{}, err
	}
	id, _ := result.LastInsertId()
	var assignee FollowUpAssignee
	err = s.DB.Get(&assignee, `SELECT * FROM follow_up_assignees WHERE id = ?`, id)
	return assignee, err
}

func (s *Store) DeleteFollowUpAssignee(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM follow_up_assignees WHERE id = ?`, id)
	return err
}

func (s *Store) ListFollowUps(filter FollowUpFilter) ([]FollowUp, int, FollowUpStats, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(f.title LIKE ? OR COALESCE(f.code, '') LIKE ? OR COALESCE(f.description, '') LIKE ? OR f.status LIKE ? OR f.priority LIKE ? OR COALESCE(f.due_date, '') LIKE ? OR c.name LIKE ? OR p.name LIKE ? OR a.name LIKE ? OR a.email LIKE ?)`)
		for i := 0; i < 10; i++ {
			args = append(args, like)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "f.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		conditions = append(conditions, "f.priority = ?")
		args = append(args, filter.Priority)
	}
	if filter.Assigned == "me" {
		conditions = append(conditions, "a.email = ?")
		args = append(args, filter.UserEmail)
	} else if filter.Assigned != "" {
		conditions = append(conditions, "f.assignee_id = ?")
		args = append(args, filter.Assigned)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 15
	}
	where := strings.Join(conditions, " AND ")
	base := ` FROM follow_ups f
		LEFT JOIN follow_up_clients c ON c.id = f.client_id
		LEFT JOIN follow_up_projects p ON p.id = f.project_id
		LEFT JOIN follow_up_assignees a ON a.id = f.assignee_id
		LEFT JOIN users u ON u.id = f.created_by
		WHERE ` + where
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*)`+base, args...); err != nil {
		return nil, 0, FollowUpStats{}, err
	}
	order := "CASE WHEN f.due_date IS NULL THEN 1 ELSE 0 END, f.due_date"
	switch filter.Sort {
	case "title":
		order = "f.title"
	case "code":
		order = "COALESCE(f.code, '')"
	case "status":
		order = "CASE f.status WHEN 'pendente' THEN 0 WHEN 'em_andamento' THEN 1 WHEN 'concluido' THEN 2 ELSE 3 END"
	case "priority":
		order = "CASE f.priority WHEN 'baixa' THEN 0 WHEN 'media' THEN 1 WHEN 'alta' THEN 2 ELSE 3 END"
	case "assignee":
		order = "COALESCE(a.name, '')"
	case "completed_at":
		order = "f.completed_at"
	case "due_date":
		order = "CASE WHEN f.due_date IS NULL THEN 1 ELSE 0 END, f.due_date"
	}
	direction := "ASC"
	if strings.EqualFold(filter.Direction, "desc") {
		direction = "DESC"
	}
	query := `SELECT f.*, c.name AS client_name, p.name AS project_name, a.name AS assignee_name,
		a.email AS assignee_email, TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS creator_name` +
		base + ` ORDER BY ` + order + ` ` + direction + `, f.id ` + direction + ` LIMIT ? OFFSET ?`
	items := []FollowUp{}
	if err := s.DB.Select(&items, query, append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)...); err != nil {
		return nil, 0, FollowUpStats{}, err
	}
	s.decorateFollowUps(items)
	stats, err := s.FollowUpStats(filter.UserEmail)
	return items, total, stats, err
}

func (s *Store) decorateFollowUps(items []FollowUp) {
	today := time.Now().Format("2006-01-02")
	for i := range items {
		if items[i].DueDate != nil && items[i].Status != "concluido" && items[i].Status != "cancelado" {
			items[i].Overdue = *items[i].DueDate < today
			items[i].DueSoon = !items[i].Overdue && *items[i].DueDate <= time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		}
	}
}

func (s *Store) FollowUpStats(userEmail string) (FollowUpStats, error) {
	var stats FollowUpStats
	err := s.DB.Get(&stats, `SELECT
		COALESCE(SUM(CASE WHEN status NOT IN ('concluido','cancelado') THEN 1 ELSE 0 END), 0) AS open,
		COALESCE(SUM(CASE WHEN status NOT IN ('concluido','cancelado') AND due_date IS NOT NULL AND due_date < ? THEN 1 ELSE 0 END), 0) AS overdue,
		COALESCE(SUM(CASE WHEN status = 'concluido' THEN 1 ELSE 0 END), 0) AS completed,
		COALESCE(SUM(CASE WHEN status NOT IN ('concluido','cancelado') AND assignee_id IN
			(SELECT id FROM follow_up_assignees WHERE email = ?) THEN 1 ELSE 0 END), 0) AS mine
		FROM follow_ups`, time.Now().Format("2006-01-02"), userEmail)
	return stats, err
}

func (s *Store) FindFollowUp(id int64) (FollowUp, error) {
	var item FollowUp
	err := s.DB.Get(&item, `SELECT f.*, c.name AS client_name, p.name AS project_name,
		a.name AS assignee_name, a.email AS assignee_email,
		TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS creator_name
		FROM follow_ups f
		LEFT JOIN follow_up_clients c ON c.id = f.client_id
		LEFT JOIN follow_up_projects p ON p.id = f.project_id
		LEFT JOIN follow_up_assignees a ON a.id = f.assignee_id
		LEFT JOIN users u ON u.id = f.created_by
		WHERE f.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	items := []FollowUp{item}
	s.decorateFollowUps(items)
	item = items[0]
	item.Updates = []FollowUpUpdate{}
	_ = s.DB.Select(&item.Updates, `SELECT u.*, TRIM(CONCAT(us.name, ' ', COALESCE(us.surname, ''))) AS author_name
		FROM follow_up_updates u LEFT JOIN users us ON us.id = u.user_id
		WHERE u.follow_up_id = ? ORDER BY u.created_at DESC, u.id DESC`, id)
	return item, nil
}

type FollowUpInput struct {
	Title       string
	Code        string
	Description string
	ClientID    *int64
	ProjectID   *int64
	Status      string
	Priority    string
	DueDate     string
	CompletedAt string
	AssigneeID  *int64
}

func (s *Store) CreateFollowUp(userID int64, input FollowUpInput) (FollowUp, error) {
	timestamp := now()
	completedAt := completionValue(input.Status, input.CompletedAt, nil)
	result, err := s.DB.Exec(`
		INSERT INTO follow_ups (title, code, description, client_id, project_id, status, priority,
			due_date, assignee_id, created_by, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Title), nullString(input.Code), nullString(input.Description),
		input.ClientID, input.ProjectID, input.Status, input.Priority, nullDate(input.DueDate),
		input.AssigneeID, userID, completedAt, timestamp, timestamp)
	if err != nil {
		return FollowUp{}, err
	}
	id, _ := result.LastInsertId()
	_, _ = s.DB.Exec(`INSERT INTO follow_up_updates (follow_up_id, user_id, body, status_from, status_to, created_at, updated_at)
		VALUES (?, ?, ?, NULL, ?, ?, ?)`, id, userID, "Follow-up criado.", input.Status, timestamp, timestamp)
	return s.FindFollowUp(id)
}

func completionValue(status, submitted string, existing *time.Time) any {
	if status != "concluido" {
		return nil
	}
	if parsed, err := parseDateTime(submitted); err == nil {
		return parsed
	}
	if existing != nil {
		return *existing
	}
	return now()
}

func (s *Store) UpdateFollowUp(id int64, userID int64, input FollowUpInput) (FollowUp, error) {
	current, err := s.FindFollowUp(id)
	if err != nil {
		return current, err
	}
	completedAt := completionValue(input.Status, input.CompletedAt, current.CompletedAt)
	_, err = s.DB.Exec(`
		UPDATE follow_ups SET title = ?, code = ?, description = ?, client_id = ?, project_id = ?,
			status = ?, priority = ?, due_date = ?, assignee_id = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.Title), nullString(input.Code), nullString(input.Description),
		input.ClientID, input.ProjectID, input.Status, input.Priority, nullDate(input.DueDate),
		input.AssigneeID, completedAt, now(), id)
	if err != nil {
		return current, err
	}
	if current.Status != input.Status {
		_, _ = s.DB.Exec(`INSERT INTO follow_up_updates (follow_up_id, user_id, body, status_from, status_to, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, id, userID,
			"Status alterado de "+followUpStatusLabel(current.Status)+" para "+followUpStatusLabel(input.Status)+".",
			current.Status, input.Status, now(), now())
	}
	return s.FindFollowUp(id)
}

func followUpStatusLabel(value string) string {
	labels := map[string]string{"pendente": "Pendente", "em_andamento": "Em andamento", "concluido": "Concluído", "cancelado": "Cancelado"}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}

func (s *Store) DeleteFollowUp(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM follow_ups WHERE id = ?`, id)
	return err
}

func (s *Store) AddFollowUpUpdate(followUpID, userID int64, body string) (FollowUpUpdate, error) {
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO follow_up_updates (follow_up_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, followUpID, userID, strings.TrimSpace(body), timestamp, timestamp)
	if err != nil {
		return FollowUpUpdate{}, err
	}
	id, _ := result.LastInsertId()
	var update FollowUpUpdate
	err = s.DB.Get(&update, `SELECT u.*, TRIM(CONCAT(us.name, ' ', COALESCE(us.surname, ''))) AS author_name
		FROM follow_up_updates u LEFT JOIN users us ON us.id = u.user_id WHERE u.id = ?`, id)
	return update, err
}

func (s *Store) FindFollowUpUpdate(followUpID, updateID int64) (FollowUpUpdate, error) {
	var update FollowUpUpdate
	err := s.DB.Get(&update, `SELECT * FROM follow_up_updates WHERE id = ? AND follow_up_id = ?`, updateID, followUpID)
	if errors.Is(err, sql.ErrNoRows) {
		return update, ErrNotFound
	}
	return update, err
}

func (s *Store) DeleteFollowUpUpdate(followUpID, updateID int64) error {
	_, err := s.DB.Exec(`DELETE FROM follow_up_updates WHERE id = ? AND follow_up_id = ?`, updateID, followUpID)
	return err
}

func (s *Store) FollowUpOverdueWidget(userEmail string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 5
	}
	items := []FollowUp{}
	err := s.DB.Select(&items, `SELECT f.*, c.name AS client_name, p.name AS project_name,
		a.name AS assignee_name, a.email AS assignee_email,
		TRIM(CONCAT(u.name, ' ', COALESCE(u.surname, ''))) AS creator_name
		FROM follow_ups f
		LEFT JOIN follow_up_clients c ON c.id = f.client_id
		LEFT JOIN follow_up_projects p ON p.id = f.project_id
		LEFT JOIN follow_up_assignees a ON a.id = f.assignee_id
		LEFT JOIN users u ON u.id = f.created_by
		WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ?
		ORDER BY CASE WHEN f.due_date IS NULL THEN 1 ELSE 0 END, f.due_date ASC LIMIT ?`, userEmail, limit)
	if err != nil {
		return nil, err
	}
	s.decorateFollowUps(items)
	var openCount, overdueCount int
	_ = s.DB.Get(&openCount, `SELECT COUNT(*) FROM follow_ups f LEFT JOIN follow_up_assignees a ON a.id = f.assignee_id
		WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ?`, userEmail)
	_ = s.DB.Get(&overdueCount, `SELECT COUNT(*) FROM follow_ups f LEFT JOIN follow_up_assignees a ON a.id = f.assignee_id
		WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ? AND f.due_date < ?`, userEmail, time.Now().Format("2006-01-02"))
	return map[string]any{"items": items, "open_count": openCount, "overdue_count": overdueCount}, nil
}

func (s *Store) FollowUpChartsWidget() (map[string]any, error) {
	stats, _ := s.FollowUpStats("")
	var total int
	_ = s.DB.Get(&total, `SELECT COUNT(*) FROM follow_ups`)
	statusRows := []struct {
		Status string `db:"status" json:"status"`
		Count  int    `db:"count" json:"count"`
	}{}
	_ = s.DB.Select(&statusRows, `SELECT status, COUNT(*) AS count FROM follow_ups GROUP BY status`)
	priorityRows := []struct {
		Priority string `db:"priority" json:"priority"`
		Count    int    `db:"count" json:"count"`
	}{}
	_ = s.DB.Select(&priorityRows, `SELECT priority, COUNT(*) AS count FROM follow_ups GROUP BY priority`)
	return map[string]any{
		"total": total, "open": stats.Open, "overdue": stats.Overdue, "completed": stats.Completed,
		"statuses": statusRows, "priorities": priorityRows,
	}, nil
}
