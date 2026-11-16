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
	ID                              int64     `db:"id" json:"id"`
	ResourceID                      string    `db:"resource_id" json:"resource_id"`
	Name                            string    `db:"name" json:"name"`
	Gender                          *string   `db:"gender" json:"gender"`
	Nationality                     *string   `db:"nationality" json:"nationality"`
	UsGreenCard                     *string   `db:"us_green_card" json:"us_green_card"`
	Subcontractor                   *string   `db:"subcontractor" json:"subcontractor"`
	CertificateType                 *string   `db:"certificate_type" json:"certificate_type"`
	IDNumber                        *string   `db:"id_number" json:"id_number"`
	BirthYear                       *string   `db:"birth_year" json:"birth_year"`
	Tel                             *string   `db:"tel" json:"tel"`
	PersonalMailbox                 *string   `db:"personal_mailbox" json:"personal_mailbox"`
	Remarks                         *string   `db:"remarks" json:"remarks"`
	RestrictedPartyScreeningResults *string   `db:"restricted_party_screening_results" json:"restricted_party_screening_results"`
	ServiceField                    *string   `db:"service_field" json:"service_field"`
	OutsourcingType                 *string   `db:"outsourcing_type" json:"outsourcing_type"`
	PersonnelType                   *string   `db:"personnel_type" json:"personnel_type"`
	ServiceDepartmentTwoLevel       *string   `db:"service_department_two_level" json:"service_department_two_level"`
	ServiceDepartmentThreeLevel     *string   `db:"service_department_three_level" json:"service_department_three_level"`
	ServiceDepartmentFourLevel      *string   `db:"service_department_four_level" json:"service_department_four_level"`
	PersonnelStatus                 *string   `db:"personnel_status" json:"personnel_status"`
	OutsourcingAccount              *string   `db:"outsourcing_account" json:"outsourcing_account"`
	AccountStatus                   *string   `db:"account_status" json:"account_status"`
	ZTEMail                         *string   `db:"zte_mail" json:"zte_mail"`
	AcademicQualifications          *string   `db:"academic_qualifications" json:"academic_qualifications"`
	LearningMode                    *string   `db:"learning_mode" json:"learning_mode"`
	GraduationTime                  *string   `db:"graduation_time" json:"graduation_time"`
	GraduationSchool                *string   `db:"graduation_school" json:"graduation_school"`
	Major                           *string   `db:"major" json:"major"`
	CreatedBy                       *string   `db:"created_by" json:"created_by"`
	LastUpdatedBy                   *string   `db:"last_updated_by" json:"last_updated_by"`
	PasswordEncrypted               *string   `db:"password_encrypted" json:"-"`
	HasPassword                     bool      `db:"-" json:"has_password"`
	CreatedAt                       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                       time.Time `db:"updated_at" json:"updated_at"`
}

type IepmsUserFilter struct {
	Search       string
	Status       string
	ServiceField string
	Page         int
	PerPage      int
}

type IepmsUserStats struct {
	Total         int `json:"total"`
	Normal        int `json:"normal"`
	Subcontractor int `json:"subcontractor"`
	Accounts      int `json:"accounts"`
}

func (s *Store) ListIepmsUsers(filter IepmsUserFilter) ([]IepmsUser, int, IepmsUserStats, []string, []string, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(resource_id LIKE ? OR name LIKE ? OR COALESCE(subcontractor, '') LIKE ? OR COALESCE(id_number, '') LIKE ? OR COALESCE(tel, '') LIKE ? OR COALESCE(personal_mailbox, '') LIKE ? OR COALESCE(zte_mail, '') LIKE ? OR COALESCE(outsourcing_account, '') LIKE ? OR COALESCE(service_department_two_level, '') LIKE ? OR COALESCE(service_department_three_level, '') LIKE ? OR COALESCE(service_department_four_level, '') LIKE ? OR COALESCE(major, '') LIKE ?)`)
		for i := 0; i < 12; i++ {
			args = append(args, like)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "personnel_status = ?")
		args = append(args, filter.Status)
	}
	if filter.ServiceField != "" {
		conditions = append(conditions, "service_field = ?")
		args = append(args, filter.ServiceField)
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
		return nil, 0, IepmsUserStats{}, nil, nil, err
	}
	query := `SELECT * FROM iepms_users WHERE ` + where + ` ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	users := []IepmsUser{}
	if err := s.DB.Select(&users, query, queryArgs...); err != nil {
		return nil, 0, IepmsUserStats{}, nil, nil, err
	}
	for index := range users {
		users[index].HasPassword = hasPassword(users[index].PasswordEncrypted)
	}
	var stats IepmsUserStats
	if err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COALESCE(SUM(personnel_status = 'Normal'), 0) AS normal,
		COALESCE(SUM(COALESCE(subcontractor, '') <> ''), 0) AS subcontractor,
		COALESCE(SUM(COALESCE(outsourcing_account, '') <> ''), 0) AS accounts
		FROM iepms_users`); err != nil {
		return nil, 0, stats, nil, nil, err
	}
	statuses := []string{}
	serviceFields := []string{}
	_ = s.DB.Select(&statuses, `SELECT DISTINCT personnel_status FROM iepms_users WHERE personnel_status IS NOT NULL AND personnel_status <> '' ORDER BY personnel_status`)
	_ = s.DB.Select(&serviceFields, `SELECT DISTINCT service_field FROM iepms_users WHERE service_field IS NOT NULL AND service_field <> '' ORDER BY service_field`)
	return users, total, stats, statuses, serviceFields, nil
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
	ResourceID                      string
	Name                            string
	Gender                          string
	Nationality                     string
	UsGreenCard                     string
	Subcontractor                   string
	CertificateType                 string
	IDNumber                        string
	BirthYear                       string
	Tel                             string
	PersonalMailbox                 string
	Remarks                         string
	RestrictedPartyScreeningResults string
	ServiceField                    string
	OutsourcingType                 string
	PersonnelType                   string
	ServiceDepartmentTwoLevel       string
	ServiceDepartmentThreeLevel     string
	ServiceDepartmentFourLevel      string
	PersonnelStatus                 string
	OutsourcingAccount              string
	AccountStatus                   string
	ZTEMail                         string
	AcademicQualifications          string
	LearningMode                    string
	GraduationTime                  string
	GraduationSchool                string
	Major                           string
	CreatedBy                       string
	LastUpdatedBy                   string
	PasswordEncrypted               string
	PasswordChanged                 bool
}

const iepmsUserInsert = `INSERT INTO iepms_users (
	resource_id, name, gender, nationality, us_green_card, subcontractor, certificate_type,
	id_number, birth_year, tel, personal_mailbox, remarks, restricted_party_screening_results,
	service_field, outsourcing_type, personnel_type, service_department_two_level,
	service_department_three_level, service_department_four_level, personnel_status,
	outsourcing_account, account_status, zte_mail, academic_qualifications, learning_mode,
	graduation_time, graduation_school, major, created_by, last_updated_by, password_encrypted,
	created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *Store) CreateIepmsUser(input IepmsUserInput) (IepmsUser, error) {
	timestamp := now()
	result, err := s.DB.Exec(iepmsUserInsert,
		strings.TrimSpace(input.ResourceID), strings.TrimSpace(input.Name), nullString(input.Gender),
		nullString(input.Nationality), nullString(input.UsGreenCard), nullString(input.Subcontractor),
		nullString(input.CertificateType), nullString(input.IDNumber), nullString(input.BirthYear),
		nullString(input.Tel), nullString(input.PersonalMailbox), nullString(input.Remarks),
		nullString(input.RestrictedPartyScreeningResults), nullString(input.ServiceField),
		nullString(input.OutsourcingType), nullString(input.PersonnelType),
		nullString(input.ServiceDepartmentTwoLevel), nullString(input.ServiceDepartmentThreeLevel),
		nullString(input.ServiceDepartmentFourLevel), nullString(input.PersonnelStatus),
		nullString(input.OutsourcingAccount), nullString(input.AccountStatus), nullString(input.ZTEMail),
		nullString(input.AcademicQualifications), nullString(input.LearningMode), nullString(input.GraduationTime),
		nullString(input.GraduationSchool), nullString(input.Major), nullString(input.CreatedBy),
		nullString(input.LastUpdatedBy), nullString(input.PasswordEncrypted), timestamp, timestamp)
	if err != nil {
		return IepmsUser{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindIepmsUser(id)
}

func (s *Store) UpdateIepmsUser(id int64, input IepmsUserInput) (IepmsUser, error) {
	_, err := s.DB.Exec(`UPDATE iepms_users SET
		resource_id = ?, name = ?, gender = ?, nationality = ?, us_green_card = ?, subcontractor = ?,
		certificate_type = ?, id_number = ?, birth_year = ?, tel = ?, personal_mailbox = ?, remarks = ?,
		restricted_party_screening_results = ?, service_field = ?, outsourcing_type = ?, personnel_type = ?,
		service_department_two_level = ?, service_department_three_level = ?, service_department_four_level = ?,
		personnel_status = ?, outsourcing_account = ?, account_status = ?, zte_mail = ?,
		academic_qualifications = ?, learning_mode = ?, graduation_time = ?, graduation_school = ?, major = ?,
		created_by = ?, last_updated_by = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.ResourceID), strings.TrimSpace(input.Name), nullString(input.Gender),
		nullString(input.Nationality), nullString(input.UsGreenCard), nullString(input.Subcontractor),
		nullString(input.CertificateType), nullString(input.IDNumber), nullString(input.BirthYear),
		nullString(input.Tel), nullString(input.PersonalMailbox), nullString(input.Remarks),
		nullString(input.RestrictedPartyScreeningResults), nullString(input.ServiceField),
		nullString(input.OutsourcingType), nullString(input.PersonnelType),
		nullString(input.ServiceDepartmentTwoLevel), nullString(input.ServiceDepartmentThreeLevel),
		nullString(input.ServiceDepartmentFourLevel), nullString(input.PersonnelStatus),
		nullString(input.OutsourcingAccount), nullString(input.AccountStatus), nullString(input.ZTEMail),
		nullString(input.AcademicQualifications), nullString(input.LearningMode), nullString(input.GraduationTime),
		nullString(input.GraduationSchool), nullString(input.Major), nullString(input.CreatedBy),
		nullString(input.LastUpdatedBy), now(), id)
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
