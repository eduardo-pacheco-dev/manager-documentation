package modules

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * iEPMS users module
 * ----------------------------------------------------------------------- */

// IepmsUser is a person registered in the iEPMS application. The fields mirror
// the Excel export/import layout used by the iEPMS back office.
type IepmsUser struct {
	ID                 int64     `db:"id" json:"id"`
	ResourceID         string    `db:"resource_id" json:"resource_id"`
	Name               string    `db:"name" json:"name"`
	Gender             *string   `db:"gender" json:"gender"`
	Nationality        *string   `db:"nationality" json:"nationality"`
	CertificateType    *string   `db:"certificate_type" json:"certificate_type"`
	IDNumber           *string   `db:"id_number" json:"id_number"`
	BirthYear          *string   `db:"birth_year" json:"birth_year"`
	Tel                *string   `db:"tel" json:"tel"`
	PersonalMailbox    *string   `db:"personal_mailbox" json:"personal_mailbox"`
	OutsourcingAccount *string   `db:"outsourcing_account" json:"outsourcing_account"`
	LastUpdatedBy      *string   `db:"last_updated_by" json:"last_updated_by"`
	Status             string    `db:"status" json:"status"`
	PasswordEncrypted  *string   `db:"password_encrypted" json:"-"`
	HasPassword        bool      `db:"-" json:"has_password"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

type IepmsUserFilter struct {
	Search  string
	Status  string
	Page    int
	PerPage int
}

type IepmsUserStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
}

func (s *Store) ListIepmsUsers(filter IepmsUserFilter) ([]IepmsUser, int, IepmsUserStats, []string, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(resource_id LIKE ? OR name LIKE ? OR COALESCE(id_number, '') LIKE ? OR COALESCE(tel, '') LIKE ? OR COALESCE(personal_mailbox, '') LIKE ? OR COALESCE(outsourcing_account, '') LIKE ?)`)
		for i := 0; i < 6; i++ {
			args = append(args, like)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM iepms_users WHERE `+where, args...); err != nil {
		return nil, 0, IepmsUserStats{}, nil, err
	}
	query := `SELECT * FROM iepms_users WHERE ` + where + ` ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	users := []IepmsUser{}
	if err := s.DB.Select(&users, query, queryArgs...); err != nil {
		return nil, 0, IepmsUserStats{}, nil, err
	}
	for index := range users {
		users[index].HasPassword = hasPassword(users[index].PasswordEncrypted)
	}
	var stats IepmsUserStats
	if err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COALESCE(SUM(status = 'ativo'), 0) AS active,
		COALESCE(SUM(status = 'inativado'), 0) AS inactive
		FROM iepms_users`); err != nil {
		return nil, 0, stats, nil, err
	}
	statuses := []string{}
	_ = s.DB.Select(&statuses, `SELECT DISTINCT status FROM iepms_users WHERE status <> '' ORDER BY status`)
	return users, total, stats, statuses, nil
}

func (s *Store) FindIepmsUser(id int64) (IepmsUser, error) {
	var user IepmsUser
	err := s.DB.Get(&user, `SELECT * FROM iepms_users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	user.HasPassword = hasPassword(user.PasswordEncrypted)
	return user, err
}

func hasPassword(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

type IepmsUserInput struct {
	ResourceID         string
	Name               string
	Gender             string
	Nationality        string
	CertificateType    string
	IDNumber           string
	BirthYear          string
	Tel                string
	PersonalMailbox    string
	OutsourcingAccount string
	LastUpdatedBy      string
	Status             string
	PasswordEncrypted  string
	PasswordChanged    bool
}

const iepmsUserInsert = `INSERT INTO iepms_users (
	resource_id, name, gender, nationality, certificate_type,
	id_number, birth_year, tel, personal_mailbox,
	outsourcing_account, last_updated_by, status,
	password_encrypted, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *Store) CreateIepmsUser(input IepmsUserInput) (IepmsUser, error) {
	timestamp := now()
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "ativo"
	}
	result, err := s.DB.Exec(iepmsUserInsert,
		strings.TrimSpace(input.ResourceID), strings.TrimSpace(input.Name), nullString(input.Gender),
		nullString(input.Nationality), nullString(input.CertificateType), nullString(input.IDNumber),
		nullString(input.BirthYear), nullString(input.Tel), nullString(input.PersonalMailbox),
		nullString(input.OutsourcingAccount), nullString(input.LastUpdatedBy), status,
		nullString(input.PasswordEncrypted), timestamp, timestamp)
	if err != nil {
		return IepmsUser{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindIepmsUser(id)
}

func (s *Store) UpdateIepmsUser(id int64, input IepmsUserInput) (IepmsUser, error) {
	_, err := s.DB.Exec(`UPDATE iepms_users SET
		resource_id = ?, name = ?, gender = ?, nationality = ?,
		certificate_type = ?, id_number = ?, birth_year = ?, tel = ?, personal_mailbox = ?,
		outsourcing_account = ?, last_updated_by = ?,
		status = COALESCE(NULLIF(?, ''), status), updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.ResourceID), strings.TrimSpace(input.Name), nullString(input.Gender),
		nullString(input.Nationality), nullString(input.CertificateType), nullString(input.IDNumber),
		nullString(input.BirthYear), nullString(input.Tel), nullString(input.PersonalMailbox),
		nullString(input.OutsourcingAccount), nullString(input.LastUpdatedBy),
		strings.TrimSpace(input.Status), now(), id)
	if err != nil {
		return IepmsUser{}, err
	}
	if input.PasswordChanged {
		if _, err := s.DB.Exec(`UPDATE iepms_users SET password_encrypted = ? WHERE id = ?`,
			nullString(input.PasswordEncrypted), id); err != nil {
			return IepmsUser{}, err
		}
	}
	return s.FindIepmsUser(id)
}

// UpsertIepmsUser creates a user or updates the existing record with the same
// Resource ID. It reports whether a new record was created.
func (s *Store) UpsertIepmsUser(input IepmsUserInput) (bool, error) {
	var id int64
	err := s.DB.Get(&id, `SELECT id FROM iepms_users WHERE resource_id = ?`, strings.TrimSpace(input.ResourceID))
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.CreateIepmsUser(input)
		return true, err
	}
	if err != nil {
		return false, err
	}
	_, err = s.UpdateIepmsUser(id, input)
	return false, err
}

func (s *Store) DeleteIepmsUser(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM iepms_users WHERE id = ?`, id)
	return err
}

// FindIepmsUserPassword returns the encrypted password for the given user.
func (s *Store) FindIepmsUserPassword(id int64) (string, error) {
	var encrypted *string
	err := s.DB.Get(&encrypted, `SELECT password_encrypted FROM iepms_users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if encrypted == nil {
		return "", nil
	}
	return *encrypted, nil
}

func (s *Store) IepmsUserByResourceID(resourceID string) (IepmsUser, error) {
	var user IepmsUser
	err := s.DB.Get(&user, `SELECT * FROM iepms_users WHERE resource_id = ?`, strings.TrimSpace(resourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	user.HasPassword = hasPassword(user.PasswordEncrypted)
	return user, err
}
