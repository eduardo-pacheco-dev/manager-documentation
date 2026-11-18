package modules

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

/* --------------------------------------------------------------------------
 * Forms module
 * ----------------------------------------------------------------------- */

// FormField is a single question inside a form section. The JSON names mirror
// the payload consumed by the React builder and by the public filling page.
type FormField struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Label           string            `json:"label"`
	Help            string            `json:"help,omitempty"`
	Required        bool              `json:"required,omitempty"`
	Options         []string          `json:"options,omitempty"`
	ScaleMin        int               `json:"scaleMin,omitempty"`
	ScaleMax        int               `json:"scaleMax,omitempty"`
	ScaleMinLabel   string            `json:"scaleMinLabel,omitempty"`
	ScaleMaxLabel   string            `json:"scaleMaxLabel,omitempty"`
	Value           string            `json:"value,omitempty"`
	Images          []string          `json:"images,omitempty"`
	ImageThumbnails []string          `json:"imageThumbnails,omitempty"`
	ImageDecisions  map[string]string `json:"imageDecisions,omitempty"`
	Files           []string          `json:"files,omitempty"`
	FileNames       []string          `json:"fileNames,omitempty"`
	Status          string            `json:"status,omitempty"`
}

// FormStep groups the questions of a form section.
type FormStep struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Fields []FormField `json:"fields"`
}

type Form struct {
	ID                      int64           `db:"id" json:"id"`
	UserID                  int64           `db:"user_id" json:"-"`
	ProjectID               *int64          `db:"project_id" json:"project_id"`
	ProjectCode             *string         `db:"project_code" json:"project_code"`
	ProjectName             *string         `db:"project_name" json:"project_name"`
	StationID               *int64          `db:"station_id" json:"station_id"`
	StationSiteID           *string         `db:"station_site_id" json:"station_site_id"`
	StationCode             *string         `db:"station_code" json:"station_code"`
	StationMunicipio        *string         `db:"station_municipio" json:"station_municipio"`
	StationEstado           *string         `db:"station_estado" json:"station_estado"`
	RadioLinkID             *int64          `db:"radio_link_id" json:"radio_link_id"`
	RadioLinkName           *string         `db:"radio_link_name" json:"radio_link_name"`
	RadioLinkCode           *string         `db:"radio_link_code" json:"radio_link_code"`
	Code                    string          `db:"code" json:"code"`
	Name                    string          `db:"name" json:"name"`
	Type                    string          `db:"type" json:"type"`
	Status                  string          `db:"status" json:"status"`
	AllowsMultipleResponses bool            `db:"allows_multiple_responses" json:"allows_multiple_responses"`
	PublicToken             string          `db:"public_token" json:"public_token"`
	Steps                   json.RawMessage `db:"steps" json:"steps"`
	Fields                  json.RawMessage `db:"fields" json:"fields"`
	CreatedAt               time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time       `db:"updated_at" json:"updated_at"`
	DeletedAt               *time.Time      `db:"deleted_at" json:"-"`
	ResponseCount           int             `db:"response_count" json:"response_count"`
	EvaluationCount         int             `db:"evaluation_count" json:"evaluation_count"`
}

type FormInput struct {
	ProjectID               *int64
	StationID               *int64
	RadioLinkID             *int64
	Name                    string
	Type                    string
	Status                  string
	AllowsMultipleResponses bool
	Steps                   json.RawMessage
	Fields                  json.RawMessage
}

type FormFilter struct {
	UserID      int64
	Search      string
	Status      string
	StationID   int64
	RadioLinkID int64
	Sort        string
	Direction   string
	Page        int
	PerPage     int
}

type FormStats struct {
	Total      int `db:"total" json:"total"`
	Pending    int `db:"pending" json:"pending"`
	InProgress int `db:"in_progress" json:"in_progress"`
	Filled     int `db:"filled" json:"filled"`
	Approved   int `db:"approved" json:"approved"`
	Rejected   int `db:"rejected" json:"rejected"`
}

type FormResponse struct {
	ID               int64           `db:"id" json:"id"`
	FormID           int64           `db:"form_id" json:"form_id"`
	UserID           *int64          `db:"user_id" json:"user_id"`
	UserName         *string         `db:"user_name" json:"user_name"`
	Status           string          `db:"status" json:"status"`
	Steps            json.RawMessage `db:"steps" json:"steps"`
	SubmittedAt      *time.Time      `db:"submitted_at" json:"submitted_at"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
	EvaluationsCount int             `db:"evaluations_count" json:"evaluations_count"`
}

type FormEvaluation struct {
	ID             int64           `db:"id" json:"id"`
	FormID         int64           `db:"form_id" json:"form_id"`
	FormResponseID *int64          `db:"form_response_id" json:"form_response_id"`
	UserID         int64           `db:"user_id" json:"user_id"`
	UserName       *string         `db:"user_name" json:"user_name"`
	Status         string          `db:"status" json:"status"`
	Notes          *string         `db:"notes" json:"notes"`
	Changes        json.RawMessage `db:"changes" json:"changes"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

type FormTemplate struct {
	ID        int64           `db:"id" json:"id"`
	UserID    int64           `db:"user_id" json:"-"`
	Name      string          `db:"name" json:"name"`
	Type      string          `db:"type" json:"type"`
	Steps     json.RawMessage `db:"steps" json:"steps"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
}

// FormSettings holds the per-user preferences of the forms module.
type FormSettings struct {
	Types             []string `json:"types"`
	Statuses          []string `json:"statuses"`
	DefaultTemplateID *int64   `json:"default_template_id"`
	DefaultView       string   `json:"default_view"`
	PerPage           int      `json:"per_page"`
}

var DefaultFormTypes = []string{
	"Registro Fotográfico",
	"Checklist",
	"Vistoria",
	"Inscrição",
	"Laudo Técnico",
}

var DefaultFormStatuses = []string{
	"Pendente",
	"Em preenchimento",
	"Preenchido",
	"Aprovado",
	"Reprovado",
}

var FormViews = map[string]string{
	"table":   "Tabela",
	"cards":   "Cards",
	"compact": "Lista",
}

var FormPerPageOptions = []int{12, 24, 36}

const (
	FormDefaultView    = "table"
	FormDefaultPerPage = 12
)

func DefaultFormSettings() FormSettings {
	return FormSettings{
		Types:       append([]string{}, DefaultFormTypes...),
		Statuses:    append([]string{}, DefaultFormStatuses...),
		DefaultView: FormDefaultView,
		PerPage:     FormDefaultPerPage,
	}
}

func DecodeSteps(raw json.RawMessage) []FormStep {
	if len(raw) == 0 {
		return []FormStep{}
	}
	var steps []FormStep
	if err := json.Unmarshal(raw, &steps); err != nil {
		return []FormStep{}
	}
	if steps == nil {
		steps = []FormStep{}
	}
	return steps
}

func EncodeSteps(steps []FormStep) json.RawMessage {
	if steps == nil {
		steps = []FormStep{}
	}
	encoded, err := json.Marshal(steps)
	if err != nil {
		return json.RawMessage("[]")
	}
	return encoded
}

// NormalizeSteps guarantees the API always exposes a JSON array, even for rows
// created before the structure was filled or stored as SQL NULL.
func NormalizeSteps(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return json.RawMessage("[]")
	}
	return raw
}

// BlankSteps keeps the current structure but removes the answers of a previous
// response.
func BlankSteps(steps []FormStep) []FormStep {
	result := make([]FormStep, len(steps))
	for stepIndex, step := range steps {
		fields := make([]FormField, len(step.Fields))
		for fieldIndex, field := range step.Fields {
			field.Value = ""
			field.Images = nil
			field.ImageThumbnails = nil
			field.ImageDecisions = nil
			field.Files = nil
			field.FileNames = nil
			field.Status = "pendente"
			fields[fieldIndex] = field
		}
		step.Fields = fields
		result[stepIndex] = step
	}
	return result
}

// MergeSteps overlays the answers of a previous response on the current form
// structure, so questions added after the response are also displayed.
func MergeSteps(structure, previous []FormStep) []FormStep {
	previousFields := map[string]FormField{}
	for _, step := range previous {
		for _, field := range step.Fields {
			if field.ID != "" {
				previousFields[field.ID] = field
			}
		}
	}
	for stepIndex, step := range structure {
		for fieldIndex, field := range step.Fields {
			old, ok := previousFields[field.ID]
			if !ok {
				continue
			}
			field.Value = old.Value
			field.Images = old.Images
			field.ImageThumbnails = old.ImageThumbnails
			field.ImageDecisions = old.ImageDecisions
			field.Files = old.Files
			field.FileNames = old.FileNames
			if old.Status != "" {
				field.Status = old.Status
			}
			structure[stepIndex].Fields[fieldIndex] = field
		}
	}
	return structure
}

// ApplyAnswers writes the submitted values to the matching fields.
func ApplyAnswers(steps []FormStep, answers map[string]string, notApplicable map[string]bool) []FormStep {
	for stepIndex, step := range steps {
		for fieldIndex, field := range step.Fields {
			if field.ID == "" {
				continue
			}
			if field.Type == "image" {
				if notApplicable[field.ID] {
					field.Status = "nao_aplicavel"
				} else if field.Status == "nao_aplicavel" {
					if len(field.Images) > 0 {
						field.Status = "enviado"
					} else {
						field.Status = "pendente"
					}
				}
				steps[stepIndex].Fields[fieldIndex] = field
				continue
			}
			if field.Type == "file" {
				continue
			}
			value := strings.TrimSpace(answers[field.ID])
			field.Value = value
			if value != "" {
				field.Status = "enviado"
			} else {
				field.Status = "pendente"
			}
			steps[stepIndex].Fields[fieldIndex] = field
		}
	}
	return steps
}

func FormResultStatus(current string) string {
	switch current {
	case "", "Pendente", "Em preenchimento":
		return "Preenchido"
	default:
		return current
	}
}

// formSelect centralizes the joins and derived columns used by the form
// queries. A form may be linked to a project and to at most one asset
// (station or radio link).
const formSelect = `SELECT f.*, CONCAT('PRJ-', LPAD(p.id, 4, '0')) AS project_code, p.name AS project_name,
		st.site_id AS station_site_id, st.station_id AS station_code,
		st.municipio AS station_municipio, st.estado AS station_estado,
		rl.name AS radio_link_name, rl.code AS radio_link_code,
		(SELECT COUNT(*) FROM form_responses r WHERE r.form_id = f.id) AS response_count,
		(SELECT COUNT(*) FROM form_evaluations e WHERE e.form_id = f.id) AS evaluation_count
		FROM forms f
		LEFT JOIN projects p ON p.id = f.project_id
		LEFT JOIN stations st ON st.id = f.station_id
		LEFT JOIN radio_links rl ON rl.id = f.radio_link_id`

func (s *Store) ListForms(filter FormFilter) ([]Form, int, FormStats, []string, []string, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = FormDefaultPerPage
	}
	conditions := []string{"f.user_id = ?", "f.deleted_at IS NULL"}
	args := []any{filter.UserID}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, "(f.name LIKE ? OR f.code LIKE ?)")
		args = append(args, like, like)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		conditions = append(conditions, "f.status = ?")
		args = append(args, status)
	}
	if filter.StationID > 0 {
		conditions = append(conditions, "f.station_id = ?")
		args = append(args, filter.StationID)
	}
	if filter.RadioLinkID > 0 {
		conditions = append(conditions, "f.radio_link_id = ?")
		args = append(args, filter.RadioLinkID)
	}
	where := strings.Join(conditions, " AND ")

	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM forms f WHERE `+where, args...); err != nil {
		return nil, 0, FormStats{}, nil, nil, err
	}

	sortColumn := map[string]string{
		"code":       "f.code",
		"name":       "f.name",
		"type":       "f.type",
		"status":     "f.status",
		"created_at": "f.created_at",
		"updated_at": "f.updated_at",
	}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "f.updated_at"
	}
	direction := "DESC"
	if strings.EqualFold(filter.Direction, "asc") {
		direction = "ASC"
	}

	query := formSelect + ` WHERE ` + where + ` ORDER BY ` + sortColumn + ` ` + direction + `, f.id ` + direction + ` LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	forms := []Form{}
	if err := s.DB.Select(&forms, query, queryArgs...); err != nil {
		return nil, 0, FormStats{}, nil, nil, err
	}
	for index := range forms {
		forms[index].Steps = NormalizeSteps(forms[index].Steps)
	}

	var stats FormStats
	if err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COALESCE(SUM(status = 'Pendente'), 0) AS pending,
		COALESCE(SUM(status = 'Em preenchimento'), 0) AS in_progress,
		COALESCE(SUM(status = 'Preenchido'), 0) AS filled,
		COALESCE(SUM(status = 'Aprovado'), 0) AS approved,
		COALESCE(SUM(status = 'Reprovado'), 0) AS rejected
		FROM forms WHERE user_id = ? AND deleted_at IS NULL`, filter.UserID); err != nil {
		return nil, 0, stats, nil, nil, err
	}

	statuses := []string{}
	types := []string{}
	_ = s.DB.Select(&statuses, `SELECT DISTINCT status FROM forms WHERE user_id = ? AND deleted_at IS NULL AND status <> '' ORDER BY status`, filter.UserID)
	_ = s.DB.Select(&types, `SELECT DISTINCT type FROM forms WHERE user_id = ? AND deleted_at IS NULL AND type <> '' ORDER BY type`, filter.UserID)
	return forms, total, stats, statuses, types, nil
}

func (s *Store) FindForm(userID, id int64) (Form, error) {
	var form Form
	err := s.DB.Get(&form, formSelect+`
		WHERE f.id = ? AND f.user_id = ? AND f.deleted_at IS NULL`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return form, ErrNotFound
	}
	form.Steps = NormalizeSteps(form.Steps)
	return form, err
}

func (s *Store) FindFormByToken(token string) (Form, error) {
	var form Form
	err := s.DB.Get(&form, formSelect+`
		WHERE f.public_token = ? AND f.deleted_at IS NULL`, strings.TrimSpace(token))
	if errors.Is(err, sql.ErrNoRows) {
		return form, ErrNotFound
	}
	form.Steps = NormalizeSteps(form.Steps)
	return form, err
}

func (s *Store) generateFormCode(userID int64) string {
	var sequence int
	_ = s.DB.Get(&sequence, `SELECT COUNT(*) FROM forms WHERE user_id = ?`, userID)
	sequence++
	for {
		code := fmt.Sprintf("FRM-%04d", sequence)
		var exists int
		_ = s.DB.Get(&exists, `SELECT COUNT(*) FROM forms WHERE user_id = ? AND code = ?`, userID, code)
		if exists == 0 {
			return code
		}
		sequence++
	}
}

func (s *Store) CreateForm(userID int64, input FormInput) (Form, error) {
	timestamp := now()
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "Pendente"
	}
	typ := strings.TrimSpace(input.Type)
	if typ == "" {
		typ = DefaultFormTypes[0]
	}
	steps := NormalizeSteps(input.Steps)
	fields := input.Fields
	if len(fields) == 0 {
		fields = json.RawMessage("[]")
	}
	result, err := s.DB.Exec(`INSERT INTO forms
		(user_id, project_id, station_id, radio_link_id, code, name, type, status, allows_multiple_responses, public_token, steps, fields, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, input.ProjectID, input.StationID, input.RadioLinkID, s.generateFormCode(userID), strings.TrimSpace(input.Name), typ, status,
		input.AllowsMultipleResponses, strings.ReplaceAll(uuid.NewString(), "-", ""), steps, fields, timestamp, timestamp)
	if err != nil {
		return Form{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindForm(userID, id)
}

func (s *Store) UpdateForm(userID, id int64, input FormInput) (Form, error) {
	status := strings.TrimSpace(input.Status)
	_, err := s.DB.Exec(`UPDATE forms SET project_id = ?, station_id = ?, radio_link_id = ?, name = ?, type = ?, status = COALESCE(NULLIF(?, ''), status),
		allows_multiple_responses = ?, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		input.ProjectID, input.StationID, input.RadioLinkID, strings.TrimSpace(input.Name), strings.TrimSpace(input.Type), status,
		input.AllowsMultipleResponses, now(), id, userID)
	if err != nil {
		return Form{}, err
	}
	return s.FindForm(userID, id)
}

func (s *Store) UpdateFormSteps(userID, id int64, steps []FormStep) (Form, error) {
	_, err := s.DB.Exec(`UPDATE forms SET steps = ?, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		EncodeSteps(steps), now(), id, userID)
	if err != nil {
		return Form{}, err
	}
	return s.FindForm(userID, id)
}

func (s *Store) DeleteForm(userID, id int64) error {
	result, err := s.DB.Exec(`UPDATE forms SET deleted_at = ?, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		now(), now(), id, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListFormResponses(formID int64) ([]FormResponse, error) {
	responses := []FormResponse{}
	err := s.DB.Select(&responses, `SELECT r.*, u.name AS user_name,
		(SELECT COUNT(*) FROM form_evaluations e WHERE e.form_response_id = r.id) AS evaluations_count
		FROM form_responses r LEFT JOIN users u ON u.id = r.user_id
		WHERE r.form_id = ? ORDER BY COALESCE(r.submitted_at, r.created_at) DESC, r.id DESC`, formID)
	for index := range responses {
		responses[index].Steps = NormalizeSteps(responses[index].Steps)
	}
	return responses, err
}

func (s *Store) FindFormResponse(formID, id int64) (FormResponse, error) {
	var response FormResponse
	err := s.DB.Get(&response, `SELECT r.*, u.name AS user_name,
		(SELECT COUNT(*) FROM form_evaluations e WHERE e.form_response_id = r.id) AS evaluations_count
		FROM form_responses r LEFT JOIN users u ON u.id = r.user_id
		WHERE r.id = ? AND r.form_id = ?`, id, formID)
	if errors.Is(err, sql.ErrNoRows) {
		return response, ErrNotFound
	}
	response.Steps = NormalizeSteps(response.Steps)
	return response, err
}

func (s *Store) LatestFormResponse(formID int64) (*FormResponse, error) {
	var response FormResponse
	err := s.DB.Get(&response, `SELECT r.*, u.name AS user_name,
		(SELECT COUNT(*) FROM form_evaluations e WHERE e.form_response_id = r.id) AS evaluations_count
		FROM form_responses r LEFT JOIN users u ON u.id = r.user_id
		WHERE r.form_id = ? AND r.submitted_at IS NOT NULL
		ORDER BY r.submitted_at DESC, r.id DESC LIMIT 1`, formID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	response.Steps = NormalizeSteps(response.Steps)
	return &response, nil
}

func (s *Store) FindFormResponseByID(formID, id int64) (*FormResponse, error) {
	response, err := s.FindFormResponse(formID, id)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &response, nil
}

type FormResponseInput struct {
	UserID      *int64
	Status      string
	Steps       []FormStep
	SubmittedAt *time.Time
}

func (s *Store) CreateFormResponse(formID int64, input FormResponseInput) (FormResponse, error) {
	timestamp := now()
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "Preenchido"
	}
	result, err := s.DB.Exec(`INSERT INTO form_responses (form_id, user_id, status, steps, submitted_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, formID, input.UserID, status, EncodeSteps(input.Steps), input.SubmittedAt, timestamp, timestamp)
	if err != nil {
		return FormResponse{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindFormResponse(formID, id)
}

func (s *Store) UpdateFormResponse(formID, id int64, status string, steps []FormStep, submittedAt *time.Time) error {
	_, err := s.DB.Exec(`UPDATE form_responses SET status = ?, steps = ?, submitted_at = ?, updated_at = ?
		WHERE id = ? AND form_id = ?`, status, EncodeSteps(steps), submittedAt, now(), id, formID)
	return err
}

func (s *Store) UpdateFormStatusAndSteps(formID int64, status string, steps []FormStep) error {
	_, err := s.DB.Exec(`UPDATE forms SET status = ?, steps = ?, updated_at = ? WHERE id = ?`, status, EncodeSteps(steps), now(), formID)
	return err
}

func (s *Store) ListFormEvaluations(formID int64, responseID *int64) ([]FormEvaluation, error) {
	evaluations := []FormEvaluation{}
	query := `SELECT e.*, u.name AS user_name FROM form_evaluations e LEFT JOIN users u ON u.id = e.user_id
		WHERE e.form_id = ?`
	args := []any{formID}
	if responseID != nil && *responseID > 0 {
		query += ` AND e.form_response_id = ?`
		args = append(args, *responseID)
	}
	query += ` ORDER BY e.created_at DESC, e.id DESC`
	err := s.DB.Select(&evaluations, query, args...)
	return evaluations, err
}

type FormChange struct {
	Type  string  `json:"type"`
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Index *int    `json:"index,omitempty"`
	From  *string `json:"from"`
	To    string  `json:"to"`
}

func (s *Store) ApplyFormEvaluation(formID int64, responseID *int64, userID int64, status, notes string, decisions map[string]string, imageDecisions map[string]map[string]string, register bool) (*FormEvaluation, error) {
	form, err := s.FindFormByOwner(formID)
	if err != nil {
		return nil, err
	}
	var response *FormResponse
	if responseID != nil && *responseID > 0 {
		found, findErr := s.FindFormResponse(formID, *responseID)
		if findErr != nil {
			return nil, findErr
		}
		response = &found
	}

	var before []FormStep
	if response != nil {
		before = DecodeSteps(response.Steps)
	} else {
		before = DecodeSteps(form.Steps)
	}
	steps := DecodeSteps(EncodeSteps(before))
	changes := []FormChange{}

	for stepIndex, step := range steps {
		for fieldIndex, field := range step.Fields {
			decision := strings.TrimSpace(decisions[field.ID])
			if decision != "" && decision != field.Status {
				from := field.Status
				changes = append(changes, FormChange{Type: "field", ID: field.ID, Label: field.Label, From: &from, To: decision})
				steps[stepIndex].Fields[fieldIndex].Status = decision
			}
			if imageMap, ok := imageDecisions[field.ID]; ok && field.Type == "image" {
				current := field.ImageDecisions
				if current == nil {
					current = map[string]string{}
				}
				for index, value := range imageMap {
					if value != "aprovado" && value != "reprovado" {
						continue
					}
					parsedIndex, parseErr := parsePositiveIndex(index)
					if parseErr != nil || parsedIndex >= len(field.Images) {
						continue
					}
					from := current[index]
					if from == value {
						continue
					}
					idx := parsedIndex
					changes = append(changes, FormChange{Type: "image", ID: field.ID, Label: field.Label, Index: &idx, From: &from, To: value})
					current[index] = value
				}
				steps[stepIndex].Fields[fieldIndex].ImageDecisions = current
			}
		}
	}

	final := register && (status == "Aprovado" || (status == "Reprovado" && strings.TrimSpace(notes) != ""))
	if response != nil {
		responseStatus := response.Status
		if final {
			responseStatus = status
		}
		submittedAt := response.SubmittedAt
		if err := s.UpdateFormResponse(formID, response.ID, responseStatus, steps, submittedAt); err != nil {
			return nil, err
		}
		if s.isLatestFormResponse(formID, response.ID) {
			formStatus := form.Status
			if final {
				formStatus = status
			}
			_ = s.UpdateFormStatusAndSteps(formID, formStatus, steps)
		}
	} else {
		formStatus := form.Status
		if final {
			formStatus = status
		}
		if err := s.UpdateFormStatusAndSteps(formID, formStatus, steps); err != nil {
			return nil, err
		}
	}

	if !final {
		return nil, nil
	}

	encodedChanges, _ := json.Marshal(changes)
	result, err := s.DB.Exec(`INSERT INTO form_evaluations
		(form_id, form_response_id, user_id, status, notes, changes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		formID, responseID, userID, status, nullString(notes), encodedChanges, now(), now())
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	evaluation := FormEvaluation{}
	if err := s.DB.Get(&evaluation, `SELECT e.*, u.name AS user_name FROM form_evaluations e
		LEFT JOIN users u ON u.id = e.user_id WHERE e.id = ?`, id); err != nil {
		return nil, err
	}
	return &evaluation, nil
}

func (s *Store) FindFormByOwner(id int64) (Form, error) {
	var form Form
	err := s.DB.Get(&form, `SELECT * FROM forms WHERE id = ? AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return form, ErrNotFound
	}
	form.Steps = NormalizeSteps(form.Steps)
	return form, err
}

func (s *Store) isLatestFormResponse(formID, responseID int64) bool {
	var latest int64
	err := s.DB.Get(&latest, `SELECT id FROM form_responses WHERE form_id = ? AND submitted_at IS NOT NULL
		ORDER BY submitted_at DESC, id DESC LIMIT 1`, formID)
	return err == nil && latest == responseID
}

func parsePositiveIndex(value string) (int, error) {
	var index int
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &index)
	if err != nil || index < 0 {
		return 0, fmt.Errorf("invalid index")
	}
	return index, nil
}

func (s *Store) ListFormTemplates() ([]FormTemplate, error) {
	templates := []FormTemplate{}
	err := s.DB.Select(&templates, `SELECT * FROM form_templates ORDER BY name, id`)
	return templates, err
}

func (s *Store) FindFormTemplate(id int64) (FormTemplate, error) {
	var template FormTemplate
	err := s.DB.Get(&template, `SELECT * FROM form_templates WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return template, ErrNotFound
	}
	return template, err
}

func (s *Store) CreateFormTemplate(userID int64, name, typ string, steps json.RawMessage) (FormTemplate, error) {
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO form_templates (user_id, name, type, steps, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, userID, strings.TrimSpace(name), strings.TrimSpace(typ), steps, timestamp, timestamp)
	if err != nil {
		return FormTemplate{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindFormTemplate(id)
}

func (s *Store) DeleteFormTemplate(userID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM form_templates WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) FormSettings(userID int64) (FormSettings, error) {
	settings := DefaultFormSettings()
	var raw []byte
	err := s.DB.Get(&raw, `SELECT settings FROM form_user_settings WHERE user_id = ?`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &settings)
	}
	if len(settings.Types) == 0 {
		settings.Types = DefaultFormTypes
	}
	if len(settings.Statuses) == 0 {
		settings.Statuses = DefaultFormStatuses
	}
	if settings.DefaultView == "" {
		settings.DefaultView = FormDefaultView
	}
	if settings.PerPage == 0 {
		settings.PerPage = FormDefaultPerPage
	}
	return settings, nil
}

func (s *Store) SaveFormSettings(userID int64, settings FormSettings) error {
	encoded, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT INTO form_user_settings (user_id, settings, updated_at) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE settings = VALUES(settings), updated_at = VALUES(updated_at)`,
		userID, encoded, now())
	return err
}

// NormalizeFormOptions trims, removes empty values and duplicates from a list.
func NormalizeFormOptions(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}
