package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *Store {
	return &Store{DB: db}
}

/* --------------------------------------------------------------------------
 * Users
 * ----------------------------------------------------------------------- */

func (s *Store) FindUserByID(id int64) (models.User, error) {
	var user models.User
	err := s.DB.Get(&user, `SELECT * FROM users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (s *Store) FindUserByEmail(email string) (models.User, error) {
	var user models.User
	err := s.DB.Get(&user, `SELECT * FROM users WHERE email = ?`, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

func (s *Store) CreateUser(user *models.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	if user.Theme == "" {
		user.Theme = "system"
	}
	result, err := s.DB.Exec(`
		INSERT INTO users (
			name, surname, email, password_hash, is_admin, is_active, theme,
			enabled_modules, module_order, module_sections,
			dashboard_hidden_widgets, dashboard_widget_order, dashboard_widget_spans,
			email_notifications, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Name, user.Surname, user.Email, user.PasswordHash, user.IsAdmin, user.IsActive, user.Theme,
		user.EnabledModules, user.ModuleOrder, user.ModuleSections,
		user.DashboardHidden, user.DashboardOrder, user.DashboardSpans,
		user.EmailNotifications, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return err
	}
	user.ID, err = result.LastInsertId()
	return err
}

func (s *Store) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := s.DB.Exec(`
		UPDATE users SET
			name = ?, surname = ?, email = ?, password_hash = ?, is_admin = ?, is_active = ?,
			theme = ?, enabled_modules = ?, module_order = ?, module_sections = ?,
			dashboard_hidden_widgets = ?, dashboard_widget_order = ?, dashboard_widget_spans = ?,
			email_notifications = ?, updated_at = ?
		WHERE id = ?`,
		user.Name, user.Surname, user.Email, user.PasswordHash, user.IsAdmin, user.IsActive,
		user.Theme, user.EnabledModules, user.ModuleOrder, user.ModuleSections,
		user.DashboardHidden, user.DashboardOrder, user.DashboardSpans,
		user.EmailNotifications, user.UpdatedAt, user.ID,
	)
	return err
}

func (s *Store) UpdateUserPreferences(user *models.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := s.DB.Exec(`
		UPDATE users SET theme = ?, enabled_modules = ?, module_order = ?, module_sections = ?,
			dashboard_hidden_widgets = ?, dashboard_widget_order = ?, dashboard_widget_spans = ?,
			email_notifications = ?, updated_at = ?
		WHERE id = ?`,
		user.Theme, user.EnabledModules, user.ModuleOrder, user.ModuleSections,
		user.DashboardHidden, user.DashboardOrder, user.DashboardSpans,
		user.EmailNotifications, user.UpdatedAt, user.ID,
	)
	return err
}

func (s *Store) DeleteUser(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

type UserFilter struct {
	Search    string
	Profile   string
	Status    string
	Sort      string
	Direction string
	Page      int
	PerPage   int
}

type UserStats struct {
	Total    int `json:"total"`
	Admins   int `json:"admins"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
}

func (s *Store) CountUserStats() (UserStats, error) {
	var stats UserStats
	err := s.DB.Get(&stats, `SELECT
		COUNT(*) AS total,
		COALESCE(SUM(CASE WHEN is_admin = 1 THEN 1 ELSE 0 END), 0) AS admins,
		COALESCE(SUM(CASE WHEN is_active = 1 THEN 1 ELSE 0 END), 0) AS active,
		COALESCE(SUM(CASE WHEN is_active = 0 THEN 1 ELSE 0 END), 0) AS inactive
		FROM users`)
	return stats, err
}

func (s *Store) ListUsers(filter UserFilter) ([]models.User, int, error) {
	conditions := []string{"1 = 1"}
	args := []any{}

	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, "(name LIKE ? OR COALESCE(surname, '') LIKE ? OR email LIKE ?)")
		args = append(args, like, like, like)
	}
	switch filter.Profile {
	case "admin":
		conditions = append(conditions, "is_admin = 1")
	case "user":
		conditions = append(conditions, "is_admin = 0")
	}
	switch filter.Status {
	case "active":
		conditions = append(conditions, "is_active = 1")
	case "inactive":
		conditions = append(conditions, "is_active = 0")
	}

	sortColumn := map[string]string{
		"name":       "name",
		"email":      "email",
		"is_admin":   "is_admin",
		"created_at": "created_at",
	}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "created_at"
	}
	direction := "DESC"
	if strings.EqualFold(filter.Direction, "asc") {
		direction = "ASC"
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage != 10 && filter.PerPage != 25 && filter.PerPage != 50 {
		filter.PerPage = 10
	}

	where := strings.Join(conditions, " AND ")
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM users WHERE `+where, args...); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT * FROM users WHERE %s ORDER BY %s %s, id %s LIMIT ? OFFSET ?`,
		where, sortColumn, direction, direction)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

	users := []models.User{}
	if err := s.DB.Select(&users, query, args...); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (s *Store) RecentUsers(limit int) ([]models.User, error) {
	users := []models.User{}
	err := s.DB.Select(&users, `SELECT * FROM users ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	return users, err
}

func (s *Store) UserCount() (int, error) {
	var count int
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM users`)
	return count, err
}

/* --------------------------------------------------------------------------
 * Dev docs
 * ----------------------------------------------------------------------- */

type DocFilter struct {
	Search   string
	Category string
	Page     int
}

type DocStats struct {
	Total      int `json:"total"`
	Published  int `json:"published"`
	Drafts     int `json:"drafts"`
	Categories int `json:"categories"`
}

func (s *Store) DocStats() (DocStats, error) {
	var stats DocStats
	err := s.DB.Get(&stats, `SELECT
		COUNT(*) AS total,
		COALESCE(SUM(CASE WHEN is_published = 1 THEN 1 ELSE 0 END), 0) AS published,
		COALESCE(SUM(CASE WHEN is_published = 0 THEN 1 ELSE 0 END), 0) AS drafts,
		COUNT(DISTINCT category) AS categories
		FROM dev_docs`)
	return stats, err
}

func (s *Store) ListDocs(filter DocFilter) ([]models.DevDocSummary, int, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, "(d.title LIKE ? OR COALESCE(d.summary, '') LIKE ? OR d.content LIKE ?)")
		args = append(args, like, like, like)
	}
	if filter.Category != "" {
		conditions = append(conditions, "d.category = ?")
		args = append(args, filter.Category)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	const perPage = 12
	where := strings.Join(conditions, " AND ")

	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM dev_docs d WHERE `+where, args...); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT d.id, d.title, d.slug, d.category, d.summary, d.is_published, d.updated_at,
			u.full_name AS author_name
		FROM dev_docs d
		LEFT JOIN (
			SELECT id, TRIM(name || ' ' || COALESCE(surname, '')) AS full_name FROM users
		) u ON u.id = d.user_id
		WHERE ` + where + `
		ORDER BY d.updated_at DESC, d.id DESC
		LIMIT ? OFFSET ?`
	args = append(args, perPage, (filter.Page-1)*perPage)

	docs := []models.DevDocSummary{}
	if err := s.DB.Select(&docs, query, args...); err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

// DocNavigation returns every page ordered by title for the wiki sidebar.
func (s *Store) DocNavigation() ([]models.DevDocSummary, error) {
	docs := []models.DevDocSummary{}
	err := s.DB.Select(&docs, `
		SELECT d.id, d.title, d.slug, d.category, d.summary, d.is_published, d.updated_at,
			u.full_name AS author_name
		FROM dev_docs d
		LEFT JOIN (
			SELECT id, TRIM(name || ' ' || COALESCE(surname, '')) AS full_name FROM users
		) u ON u.id = d.user_id
		ORDER BY d.title COLLATE NOCASE ASC`)
	return docs, err
}

func (s *Store) FindDoc(id int64) (models.DevDoc, error) {
	var doc models.DevDoc
	err := s.DB.Get(&doc, `
		SELECT d.*, u.full_name AS author_name
		FROM dev_docs d
		LEFT JOIN (
			SELECT id, TRIM(name || ' ' || COALESCE(surname, '')) AS full_name FROM users
		) u ON u.id = d.user_id
		WHERE d.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return doc, ErrNotFound
	}
	return doc, err
}

func (s *Store) FindDocBySlug(slug string) (models.DevDoc, error) {
	var doc models.DevDoc
	err := s.DB.Get(&doc, `SELECT * FROM dev_docs WHERE slug = ?`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return doc, ErrNotFound
	}
	return doc, err
}

func (s *Store) CreateDoc(doc *models.DevDoc) error {
	now := time.Now().UTC()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	result, err := s.DB.Exec(`
		INSERT INTO dev_docs (user_id, title, slug, category, summary, content, is_published, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.UserID, doc.Title, doc.Slug, doc.Category, doc.Summary, doc.Content,
		doc.IsPublished, doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return err
	}
	doc.ID, err = result.LastInsertId()
	return err
}

func (s *Store) UpdateDoc(doc *models.DevDoc) error {
	doc.UpdatedAt = time.Now().UTC()
	_, err := s.DB.Exec(`
		UPDATE dev_docs SET title = ?, category = ?, summary = ?, content = ?, is_published = ?, updated_at = ?
		WHERE id = ?`,
		doc.Title, doc.Category, doc.Summary, doc.Content, doc.IsPublished, doc.UpdatedAt, doc.ID,
	)
	return err
}

func (s *Store) DeleteDoc(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM dev_docs WHERE id = ?`, id)
	return err
}

func (s *Store) UniqueSlug(base string) (string, error) {
	if base == "" {
		base = "documento"
	}
	candidate := base
	for suffix := 2; suffix < 1000; suffix++ {
		var count int
		if err := s.DB.Get(&count, `SELECT COUNT(*) FROM dev_docs WHERE slug = ?`, candidate); err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix()), nil
}
