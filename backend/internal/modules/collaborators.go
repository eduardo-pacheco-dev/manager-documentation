package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Collaborators module
 *
 * Central registry of people that can be allocated to projects as resources.
 * ----------------------------------------------------------------------- */

type Collaborator struct {
	ID                    int64     `db:"id" json:"id"`
	Name                  string    `db:"name" json:"name"`
	SocialName            *string   `db:"social_name" json:"social_name"`
	Email                 *string   `db:"email" json:"email"`
	PersonalEmail         *string   `db:"personal_email" json:"personal_email"`
	Phone                 *string   `db:"phone" json:"phone"`
	Mobile                *string   `db:"mobile" json:"mobile"`
	Role                  *string   `db:"role" json:"role"`
	Department            *string   `db:"department" json:"department"`
	ManagerName           *string   `db:"manager_name" json:"manager_name"`
	Document              *string   `db:"document" json:"document"`
	CPF                   *string   `db:"cpf" json:"cpf"`
	RG                    *string   `db:"rg" json:"rg"`
	RGIssuer              *string   `db:"rg_issuer" json:"rg_issuer"`
	CTPS                  *string   `db:"ctps" json:"ctps"`
	CTPSSeries            *string   `db:"ctps_series" json:"ctps_series"`
	PIS                   *string   `db:"pis" json:"pis"`
	VoterRegistration     *string   `db:"voter_registration" json:"voter_registration"`
	MilitaryRegistration  *string   `db:"military_registration" json:"military_registration"`
	CNH                   *string   `db:"cnh" json:"cnh"`
	CNHCategory           *string   `db:"cnh_category" json:"cnh_category"`
	CNHExpiry             *string   `db:"cnh_expiry" json:"cnh_expiry"`
	BirthDate             *string   `db:"birth_date" json:"birth_date"`
	Gender                *string   `db:"gender" json:"gender"`
	MaritalStatus         *string   `db:"marital_status" json:"marital_status"`
	Nationality           *string   `db:"nationality" json:"nationality"`
	Birthplace            *string   `db:"birthplace" json:"birthplace"`
	MotherName            *string   `db:"mother_name" json:"mother_name"`
	FatherName            *string   `db:"father_name" json:"father_name"`
	ZipCode               *string   `db:"zip_code" json:"zip_code"`
	Street                *string   `db:"street" json:"street"`
	AddressNumber         *string   `db:"address_number" json:"address_number"`
	Complement            *string   `db:"complement" json:"complement"`
	Neighborhood          *string   `db:"neighborhood" json:"neighborhood"`
	City                  *string   `db:"city" json:"city"`
	State                 *string   `db:"state" json:"state"`
	Country               *string   `db:"country" json:"country"`
	EmergencyContactName  *string   `db:"emergency_contact_name" json:"emergency_contact_name"`
	EmergencyContactPhone *string   `db:"emergency_contact_phone" json:"emergency_contact_phone"`
	ContractType          *string   `db:"contract_type" json:"contract_type"`
	WorkRegime            *string   `db:"work_regime" json:"work_regime"`
	WorkLocation          *string   `db:"work_location" json:"work_location"`
	HireDate              *string   `db:"hire_date" json:"hire_date"`
	TerminationDate       *string   `db:"termination_date" json:"termination_date"`
	WeeklyHours           float64   `db:"weekly_hours" json:"weekly_hours"`
	Salary                float64   `db:"salary" json:"salary"`
	BankName              *string   `db:"bank_name" json:"bank_name"`
	BankAgency            *string   `db:"bank_agency" json:"bank_agency"`
	BankAccount           *string   `db:"bank_account" json:"bank_account"`
	BankAccountType       *string   `db:"bank_account_type" json:"bank_account_type"`
	PixKey                *string   `db:"pix_key" json:"pix_key"`
	CostHour              float64   `db:"cost_hour" json:"cost_hour"`
	MaxUnits              int       `db:"max_units" json:"max_units"`
	Color                 string    `db:"color" json:"color"`
	IsActive              bool      `db:"is_active" json:"is_active"`
	Notes                 *string   `db:"notes" json:"notes"`
	ProjectCount          int       `db:"project_count" json:"project_count"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
}

type CollaboratorInput struct {
	Name                  string
	SocialName            string
	Email                 string
	PersonalEmail         string
	Phone                 string
	Mobile                string
	Role                  string
	Department            string
	ManagerName           string
	Document              string
	CPF                   string
	RG                    string
	RGIssuer              string
	CTPS                  string
	CTPSSeries            string
	PIS                   string
	VoterRegistration     string
	MilitaryRegistration  string
	CNH                   string
	CNHCategory           string
	CNHExpiry             string
	BirthDate             string
	Gender                string
	MaritalStatus         string
	Nationality           string
	Birthplace            string
	MotherName            string
	FatherName            string
	ZipCode               string
	Street                string
	AddressNumber         string
	Complement            string
	Neighborhood          string
	City                  string
	State                 string
	Country               string
	EmergencyContactName  string
	EmergencyContactPhone string
	ContractType          string
	WorkRegime            string
	WorkLocation          string
	HireDate              string
	TerminationDate       string
	WeeklyHours           float64
	Salary                float64
	BankName              string
	BankAgency            string
	BankAccount           string
	BankAccountType       string
	PixKey                string
	CostHour              float64
	MaxUnits              int
	Color                 string
	IsActive              bool
	Notes                 string
}

type CollaboratorFilter struct {
	Search    string
	Status    string
	Sort      string
	Direction string
	Page      int
	PerPage   int
}

type CollaboratorStats struct {
	Total     int `db:"total" json:"total"`
	Active    int `db:"active" json:"active"`
	Inactive  int `db:"inactive" json:"inactive"`
	Allocated int `db:"allocated" json:"allocated"`
}

const collaboratorSelect = `
	SELECT c.*, (SELECT COUNT(*) FROM project_resources r WHERE r.collaborator_id = c.id) AS project_count
	FROM collaborators c`

func (s *Store) ListCollaborators(filter CollaboratorFilter) ([]Collaborator, int, CollaboratorStats, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(c.name LIKE ? OR COALESCE(c.social_name, '') LIKE ? OR COALESCE(c.email, '') LIKE ?
			OR COALESCE(c.personal_email, '') LIKE ? OR COALESCE(c.role, '') LIKE ? OR COALESCE(c.department, '') LIKE ?
			OR COALESCE(c.phone, '') LIKE ? OR COALESCE(c.mobile, '') LIKE ? OR COALESCE(c.document, '') LIKE ?
			OR COALESCE(c.cpf, '') LIKE ? OR COALESCE(c.rg, '') LIKE ? OR COALESCE(c.city, '') LIKE ?
			OR COALESCE(c.state, '') LIKE ? OR COALESCE(c.contract_type, '') LIKE ? OR COALESCE(c.manager_name, '') LIKE ?)`)
		for i := 0; i < 15; i++ {
			args = append(args, like)
		}
	}
	switch filter.Status {
	case "active":
		conditions = append(conditions, "c.is_active = 1")
	case "inactive":
		conditions = append(conditions, "c.is_active = 0")
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 15
	}
	if filter.PerPage > 200 {
		filter.PerPage = 200
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM collaborators c WHERE `+where, args...); err != nil {
		return nil, 0, CollaboratorStats{}, err
	}
	order := "c.name"
	switch filter.Sort {
	case "role":
		order = "COALESCE(c.role, '')"
	case "department":
		order = "COALESCE(c.department, '')"
	case "cost_hour":
		order = "c.cost_hour"
	case "projects":
		order = "project_count"
	case "created_at":
		order = "c.created_at"
	}
	direction := "ASC"
	if strings.EqualFold(filter.Direction, "desc") {
		direction = "DESC"
	}
	query := collaboratorSelect + ` WHERE ` + where +
		` ORDER BY ` + order + ` ` + direction + `, c.id ` + direction + ` LIMIT ? OFFSET ?`
	items := []Collaborator{}
	if err := s.DB.Select(&items, query, append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)...); err != nil {
		return nil, 0, CollaboratorStats{}, err
	}
	stats, err := s.CollaboratorStats()
	return items, total, stats, err
}

func (s *Store) CollaboratorStats() (CollaboratorStats, error) {
	var stats CollaboratorStats
	err := s.DB.Get(&stats, `
		SELECT COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN c.is_active = 1 THEN 1 ELSE 0 END), 0) AS active,
			COALESCE(SUM(CASE WHEN c.is_active = 0 THEN 1 ELSE 0 END), 0) AS inactive,
			COALESCE(SUM(CASE WHEN EXISTS (
				SELECT 1 FROM project_resources r WHERE r.collaborator_id = c.id
			) THEN 1 ELSE 0 END), 0) AS allocated
		FROM collaborators c`)
	return stats, err
}

func (s *Store) FindCollaborator(id int64) (Collaborator, error) {
	var collaborator Collaborator
	err := s.DB.Get(&collaborator, collaboratorSelect+` WHERE c.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return collaborator, ErrNotFound
	}
	return collaborator, err
}

func (s *Store) FindCollaboratorByEmail(email string) (Collaborator, error) {
	var collaborator Collaborator
	err := s.DB.Get(&collaborator, collaboratorSelect+` WHERE c.email = ?`, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, sql.ErrNoRows) {
		return collaborator, ErrNotFound
	}
	return collaborator, err
}

func (s *Store) CreateCollaborator(input CollaboratorInput) (Collaborator, error) {
	input, err := s.normalizeCollaborator(0, input)
	if err != nil {
		return Collaborator{}, err
	}
	timestamp := now()
	columns := input.columns()
	values := input.values()
	placeholders := make([]string, len(values))
	for index := range placeholders {
		placeholders[index] = "?"
	}
	result, err := s.DB.Exec(`
		INSERT INTO collaborators (`+strings.Join(columns, ", ")+`, created_at, updated_at)
		VALUES (`+strings.Join(placeholders, ", ")+`, ?, ?)`,
		append(values, timestamp, timestamp)...)
	if err != nil {
		return Collaborator{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindCollaborator(id)
}

func (s *Store) UpdateCollaborator(id int64, input CollaboratorInput) (Collaborator, error) {
	if _, err := s.FindCollaborator(id); err != nil {
		return Collaborator{}, err
	}
	input, err := s.normalizeCollaborator(id, input)
	if err != nil {
		return Collaborator{}, err
	}
	columns := input.columns()
	values := input.values()
	assignments := make([]string, len(columns))
	for index, column := range columns {
		assignments[index] = column + " = ?"
	}
	_, err = s.DB.Exec(`
		UPDATE collaborators SET `+strings.Join(assignments, ", ")+`, updated_at = ?
		WHERE id = ?`, append(values, now(), id)...)
	if err != nil {
		return Collaborator{}, err
	}
	return s.FindCollaborator(id)
}

func (input CollaboratorInput) columns() []string {
	return []string{
		"name", "social_name", "email", "personal_email", "phone", "mobile",
		"role", "department", "manager_name", "document", "cpf", "rg", "rg_issuer",
		"ctps", "ctps_series", "pis", "voter_registration", "military_registration",
		"cnh", "cnh_category", "cnh_expiry", "birth_date", "gender", "marital_status",
		"nationality", "birthplace", "mother_name", "father_name", "zip_code", "street",
		"address_number", "complement", "neighborhood", "city", "state", "country",
		"emergency_contact_name", "emergency_contact_phone", "contract_type", "work_regime",
		"work_location", "hire_date", "termination_date", "weekly_hours", "salary",
		"bank_name", "bank_agency", "bank_account", "bank_account_type", "pix_key",
		"cost_hour", "max_units", "color", "is_active", "notes",
	}
}

func (input CollaboratorInput) values() []any {
	return []any{
		input.Name, nullString(input.SocialName), nullString(input.Email), nullString(input.PersonalEmail),
		nullString(input.Phone), nullString(input.Mobile), nullString(input.Role), nullString(input.Department),
		nullString(input.ManagerName), nullString(input.Document), nullString(input.CPF), nullString(input.RG),
		nullString(input.RGIssuer), nullString(input.CTPS), nullString(input.CTPSSeries), nullString(input.PIS),
		nullString(input.VoterRegistration), nullString(input.MilitaryRegistration), nullString(input.CNH),
		nullString(input.CNHCategory), nullString(input.CNHExpiry), nullString(input.BirthDate),
		nullString(input.Gender), nullString(input.MaritalStatus), nullString(input.Nationality),
		nullString(input.Birthplace), nullString(input.MotherName), nullString(input.FatherName),
		nullString(input.ZipCode), nullString(input.Street), nullString(input.AddressNumber),
		nullString(input.Complement), nullString(input.Neighborhood), nullString(input.City),
		nullString(input.State), nullString(input.Country), nullString(input.EmergencyContactName),
		nullString(input.EmergencyContactPhone), nullString(input.ContractType), nullString(input.WorkRegime),
		nullString(input.WorkLocation), nullString(input.HireDate), nullString(input.TerminationDate),
		input.WeeklyHours, input.Salary, nullString(input.BankName), nullString(input.BankAgency),
		nullString(input.BankAccount), nullString(input.BankAccountType), nullString(input.PixKey),
		input.CostHour, input.MaxUnits, input.Color, input.IsActive, nullString(input.Notes),
	}
}

func (s *Store) ToggleCollaboratorActive(id int64) (Collaborator, error) {
	collaborator, err := s.FindCollaborator(id)
	if err != nil {
		return Collaborator{}, err
	}
	_, err = s.DB.Exec(`UPDATE collaborators SET is_active = ?, updated_at = ? WHERE id = ?`,
		!collaborator.IsActive, now(), id)
	if err != nil {
		return Collaborator{}, err
	}
	return s.FindCollaborator(id)
}

func (s *Store) DeleteCollaborator(id int64) error {
	if _, err := s.FindCollaborator(id); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM collaborators WHERE id = ?`, id)
	return err
}

func (s *Store) normalizeCollaborator(id int64, input CollaboratorInput) (CollaboratorInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return input, fmt.Errorf("%w: nome é obrigatório", ErrValidation)
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.PersonalEmail = strings.ToLower(strings.TrimSpace(input.PersonalEmail))
	for _, field := range []*string{
		&input.SocialName, &input.Phone, &input.Mobile, &input.Role, &input.Department, &input.ManagerName,
		&input.Document, &input.RG, &input.RGIssuer, &input.CTPS, &input.CTPSSeries, &input.VoterRegistration,
		&input.MilitaryRegistration, &input.CNH, &input.CNHCategory, &input.Gender, &input.MaritalStatus,
		&input.Nationality, &input.Birthplace, &input.MotherName, &input.FatherName, &input.ZipCode,
		&input.Street, &input.AddressNumber, &input.Complement, &input.Neighborhood, &input.City,
		&input.State, &input.Country, &input.EmergencyContactName, &input.EmergencyContactPhone,
		&input.WorkLocation, &input.BankName, &input.BankAgency, &input.BankAccount,
		&input.BankAccountType, &input.PixKey, &input.Notes,
	} {
		*field = strings.TrimSpace(*field)
	}
	input.CPF = digitsOnly(input.CPF)
	input.PIS = digitsOnly(input.PIS)
	input.BirthDate = normalizeDate(input.BirthDate)
	input.CNHExpiry = normalizeDate(input.CNHExpiry)
	input.HireDate = normalizeDate(input.HireDate)
	input.TerminationDate = normalizeDate(input.TerminationDate)
	input.ContractType = strings.ToLower(strings.TrimSpace(input.ContractType))
	input.WorkRegime = strings.ToLower(strings.TrimSpace(input.WorkRegime))
	input.Color = strings.TrimSpace(input.Color)
	if input.Color == "" {
		input.Color = "blue"
	}
	if input.MaxUnits <= 0 {
		input.MaxUnits = 100
	}
	input.MaxUnits = clampInt(input.MaxUnits, 1, 100)
	if input.CostHour < 0 {
		input.CostHour = 0
	}
	input.CostHour = round2(input.CostHour)
	if input.WeeklyHours <= 0 {
		input.WeeklyHours = 44
	}
	input.WeeklyHours = round2(input.WeeklyHours)
	if input.Salary < 0 {
		input.Salary = 0
	}
	input.Salary = round2(input.Salary)
	if input.HireDate != "" && input.TerminationDate != "" && input.TerminationDate < input.HireDate {
		return input, fmt.Errorf("%w: a data de desligamento não pode ser anterior à admissão", ErrValidation)
	}
	if input.Email != "" {
		var existingID int64
		err := s.DB.Get(&existingID, `SELECT id FROM collaborators WHERE email = ?`, input.Email)
		if err == nil && existingID != id {
			return input, fmt.Errorf("%w: já existe um colaborador com este e-mail", ErrValidation)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return input, err
		}
	}
	if input.CPF != "" {
		var existingID int64
		err := s.DB.Get(&existingID, `SELECT id FROM collaborators WHERE cpf = ?`, input.CPF)
		if err == nil && existingID != id {
			return input, fmt.Errorf("%w: já existe um colaborador com este CPF", ErrValidation)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return input, err
		}
	}
	return input, nil
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
