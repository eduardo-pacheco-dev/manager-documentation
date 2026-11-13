package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

func collaboratorError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, modules.ErrValidation):
		message := strings.TrimSpace(strings.TrimPrefix(err.Error(), modules.ErrValidation.Error()+":"))
		if message == "" {
			message = fallback
		}
		writeError(w, http.StatusUnprocessableEntity, message, nil)
	case errors.Is(err, modules.ErrNotFound):
		writeError(w, http.StatusNotFound, fallback, nil)
	default:
		writeError(w, http.StatusInternalServerError, fallback, nil)
	}
}

func (s *Server) collaboratorsIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 15)
	if perPage > 200 {
		perPage = 200
	}
	filter := modules.CollaboratorFilter{
		Search: query.Get("search"), Status: query.Get("status"),
		Sort: query.Get("sort"), Direction: query.Get("direction"),
		Page: page, PerPage: perPage,
	}
	items, total, stats, err := s.Modules.ListCollaborators(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os colaboradores.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"stats": stats,
		"meta":  paginationMeta(page, filter.PerPage, total),
		"filters": map[string]any{
			"search": filter.Search, "status": filter.Status, "sort": filter.Sort,
			"direction": filter.Direction, "per_page": filter.PerPage,
		},
	})
}

func (s *Server) collaboratorsShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Colaborador não encontrado.", nil)
		return
	}
	collaborator, err := s.Modules.FindCollaborator(id)
	if err != nil {
		collaboratorError(w, err, "Colaborador não encontrado.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"collaborator": collaborator}})
}

type collaboratorPayload struct {
	Name                  string  `json:"name"`
	SocialName            string  `json:"social_name"`
	Email                 string  `json:"email"`
	PersonalEmail         string  `json:"personal_email"`
	Phone                 string  `json:"phone"`
	Mobile                string  `json:"mobile"`
	Role                  string  `json:"role"`
	Department            string  `json:"department"`
	ManagerName           string  `json:"manager_name"`
	Document              string  `json:"document"`
	CPF                   string  `json:"cpf"`
	RG                    string  `json:"rg"`
	RGIssuer              string  `json:"rg_issuer"`
	CTPS                  string  `json:"ctps"`
	CTPSSeries            string  `json:"ctps_series"`
	PIS                   string  `json:"pis"`
	VoterRegistration     string  `json:"voter_registration"`
	MilitaryRegistration  string  `json:"military_registration"`
	CNH                   string  `json:"cnh"`
	CNHCategory           string  `json:"cnh_category"`
	CNHExpiry             string  `json:"cnh_expiry"`
	BirthDate             string  `json:"birth_date"`
	Gender                string  `json:"gender"`
	MaritalStatus         string  `json:"marital_status"`
	Nationality           string  `json:"nationality"`
	Birthplace            string  `json:"birthplace"`
	MotherName            string  `json:"mother_name"`
	FatherName            string  `json:"father_name"`
	ZipCode               string  `json:"zip_code"`
	Street                string  `json:"street"`
	AddressNumber         string  `json:"address_number"`
	Complement            string  `json:"complement"`
	Neighborhood          string  `json:"neighborhood"`
	City                  string  `json:"city"`
	State                 string  `json:"state"`
	Country               string  `json:"country"`
	EmergencyContactName  string  `json:"emergency_contact_name"`
	EmergencyContactPhone string  `json:"emergency_contact_phone"`
	ContractType          string  `json:"contract_type"`
	WorkRegime            string  `json:"work_regime"`
	WorkLocation          string  `json:"work_location"`
	HireDate              string  `json:"hire_date"`
	TerminationDate       string  `json:"termination_date"`
	WeeklyHours           float64 `json:"weekly_hours"`
	Salary                float64 `json:"salary"`
	BankName              string  `json:"bank_name"`
	BankAgency            string  `json:"bank_agency"`
	BankAccount           string  `json:"bank_account"`
	BankAccountType       string  `json:"bank_account_type"`
	PixKey                string  `json:"pix_key"`
	CostHour              float64 `json:"cost_hour"`
	MaxUnits              int     `json:"max_units"`
	Color                 string  `json:"color"`
	IsActive              *bool   `json:"is_active"`
	Notes                 string  `json:"notes"`
}

func (p collaboratorPayload) input(active bool) modules.CollaboratorInput {
	return modules.CollaboratorInput{
		Name: p.Name, SocialName: p.SocialName, Email: p.Email, PersonalEmail: p.PersonalEmail,
		Phone: p.Phone, Mobile: p.Mobile, Role: p.Role, Department: p.Department, ManagerName: p.ManagerName,
		Document: p.Document, CPF: p.CPF, RG: p.RG, RGIssuer: p.RGIssuer, CTPS: p.CTPS,
		CTPSSeries: p.CTPSSeries, PIS: p.PIS, VoterRegistration: p.VoterRegistration,
		MilitaryRegistration: p.MilitaryRegistration, CNH: p.CNH, CNHCategory: p.CNHCategory,
		CNHExpiry: p.CNHExpiry, BirthDate: p.BirthDate, Gender: p.Gender, MaritalStatus: p.MaritalStatus,
		Nationality: p.Nationality, Birthplace: p.Birthplace, MotherName: p.MotherName, FatherName: p.FatherName,
		ZipCode: p.ZipCode, Street: p.Street, AddressNumber: p.AddressNumber, Complement: p.Complement,
		Neighborhood: p.Neighborhood, City: p.City, State: p.State, Country: p.Country,
		EmergencyContactName: p.EmergencyContactName, EmergencyContactPhone: p.EmergencyContactPhone,
		ContractType: p.ContractType, WorkRegime: p.WorkRegime, WorkLocation: p.WorkLocation,
		HireDate: p.HireDate, TerminationDate: p.TerminationDate, WeeklyHours: p.WeeklyHours,
		Salary: p.Salary, BankName: p.BankName, BankAgency: p.BankAgency, BankAccount: p.BankAccount,
		BankAccountType: p.BankAccountType, PixKey: p.PixKey, CostHour: p.CostHour, MaxUnits: p.MaxUnits,
		Color: p.Color, IsActive: active, Notes: p.Notes,
	}
}

func validateCollaboratorPayload(payload collaboratorPayload) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "O nome é obrigatório.")
	}
	if email := strings.TrimSpace(payload.Email); email != "" && !validateEmail(email) {
		fields.add("email", "Informe um e-mail corporativo válido.")
	}
	if email := strings.TrimSpace(payload.PersonalEmail); email != "" && !validateEmail(email) {
		fields.add("personal_email", "Informe um e-mail pessoal válido.")
	}
	if payload.CostHour < 0 {
		fields.add("cost_hour", "O custo por hora não pode ser negativo.")
	}
	if payload.Salary < 0 {
		fields.add("salary", "O salário não pode ser negativo.")
	}
	if payload.WeeklyHours < 0 || payload.WeeklyHours > 168 {
		fields.add("weekly_hours", "A jornada semanal deve ficar entre 0 e 168 horas.")
	}
	if payload.MaxUnits < 0 || payload.MaxUnits > 100 {
		fields.add("max_units", "A capacidade deve ficar entre 0 e 100%.")
	}
	if cpf := digits(payload.CPF); cpf != "" && len(cpf) != 11 {
		fields.add("cpf", "O CPF deve conter 11 dígitos.")
	}
	for field, value := range map[string]string{
		"birth_date": payload.BirthDate, "cnh_expiry": payload.CNHExpiry,
		"hire_date": payload.HireDate, "termination_date": payload.TerminationDate,
	} {
		if value := strings.TrimSpace(value); value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				fields.add(field, "Informe uma data válida.")
			}
		}
	}
	return fields
}

func digits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func (s *Server) collaboratorsCreate(w http.ResponseWriter, r *http.Request) {
	var payload collaboratorPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateCollaboratorPayload(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	active := true
	if payload.IsActive != nil {
		active = *payload.IsActive
	}
	collaborator, err := s.Modules.CreateCollaborator(payload.input(active))
	if err != nil {
		collaboratorError(w, err, "Não foi possível criar o colaborador.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Colaborador criado com sucesso.",
		"data":    map[string]any{"collaborator": collaborator},
	})
}

func (s *Server) collaboratorsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Colaborador não encontrado.", nil)
		return
	}
	var payload collaboratorPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateCollaboratorPayload(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	active := true
	if payload.IsActive != nil {
		active = *payload.IsActive
	} else if existing, err := s.Modules.FindCollaborator(id); err == nil {
		active = existing.IsActive
	}
	collaborator, err := s.Modules.UpdateCollaborator(id, payload.input(active))
	if err != nil {
		collaboratorError(w, err, "Não foi possível atualizar o colaborador.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Colaborador atualizado.",
		"data":    map[string]any{"collaborator": collaborator},
	})
}

func (s *Server) collaboratorsToggle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Colaborador não encontrado.", nil)
		return
	}
	collaborator, err := s.Modules.ToggleCollaboratorActive(id)
	if err != nil {
		collaboratorError(w, err, "Não foi possível alterar o status do colaborador.")
		return
	}
	message := "Colaborador ativado."
	if !collaborator.IsActive {
		message = "Colaborador desativado."
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": message,
		"data":    map[string]any{"collaborator": collaborator},
	})
}

func (s *Server) collaboratorsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Colaborador não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteCollaborator(id); err != nil {
		collaboratorError(w, err, "Não foi possível excluir o colaborador.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Colaborador excluído."})
}
