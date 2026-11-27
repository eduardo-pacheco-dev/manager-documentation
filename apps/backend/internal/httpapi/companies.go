package httpapi

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/models"
)

var companySlugRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func databaseCompanyID(r *http.Request) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
}

type companyPayload struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	IsActive      *bool  `json:"is_active"`
	AdminName     string `json:"admin_name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

func (s *Server) listCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := s.Store.ListCompanies()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível listar as empresas.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": companies})
}

func (s *Server) createCompany(w http.ResponseWriter, r *http.Request) {
	var payload companyPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	slug := strings.ToLower(strings.TrimSpace(payload.Slug))
	name := strings.TrimSpace(payload.Name)
	fields := validationErrors{}
	if name == "" {
		fields.add("name", "O nome é obrigatório.")
	}
	if slug == "" {
		fields.add("slug", "O identificador é obrigatório.")
	} else if !companySlugRE.MatchString(slug) || len(slug) > 64 {
		fields.add("slug", "Use letras minúsculas, números e hífens (máx. 64 caracteres).")
	}
	hasAdmin := strings.TrimSpace(payload.AdminEmail) != "" || strings.TrimSpace(payload.AdminPassword) != ""
	if hasAdmin {
		if !validateEmail(normalizeEmail(payload.AdminEmail)) {
			fields.add("admin_email", "Informe um e-mail válido para o administrador.")
		}
		if !validatePassword(payload.AdminPassword) {
			fields.add("admin_password", "A senha deve ter pelo menos 8 caracteres.")
		}
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if _, err := s.Store.FindCompanyBySlug(slug); err == nil {
		writeValidation(w, validationErrors{"slug": "Este identificador já está em uso."})
		return
	}

	company := models.Company{Name: name, Slug: slug, IsActive: payload.IsActive == nil || *payload.IsActive}
	var admin *models.User
	if hasAdmin {
		hash, err := bcrypt.GenerateFromPassword([]byte(payload.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível criar o administrador.", nil)
			return
		}
		admin = &models.User{
			Name:               strings.TrimSpace(payload.AdminName),
			Email:              normalizeEmail(payload.AdminEmail),
			PasswordHash:       string(hash),
			IsAdmin:            true,
			IsActive:           true,
			Theme:              "system",
			EmailNotifications: true,
		}
		if admin.Name == "" {
			admin.Name = "Administrador"
		}
	}
	if err := s.Store.CreateCompanyWithAdmin(&company, admin); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar a empresa.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": company})
}

func (s *Server) updateCompany(w http.ResponseWriter, r *http.Request) {
	id, err := databaseCompanyID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Empresa inválida.", nil)
		return
	}
	company, err := s.Store.FindCompanyByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Empresa não encontrada.", nil)
		return
	}
	var payload companyPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	slug := strings.ToLower(strings.TrimSpace(payload.Slug))
	name := strings.TrimSpace(payload.Name)
	fields := validationErrors{}
	if name == "" {
		fields.add("name", "O nome é obrigatório.")
	}
	if !companySlugRE.MatchString(slug) || len(slug) > 64 {
		fields.add("slug", "Use letras minúsculas, números e hífens (máx. 64 caracteres).")
	}
	if other, err := s.Store.FindCompanyBySlug(slug); err == nil && other.ID != company.ID {
		fields.add("slug", "Este identificador já está em uso.")
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if company.ID == 1 || company.Slug == "padrao" {
		if slug != "padrao" || (payload.IsActive != nil && !*payload.IsActive) {
			writeValidation(w, validationErrors{"company": "A empresa padrão não pode ser renomeada nem desativada."})
			return
		}
	}
	company.Name = name
	company.Slug = slug
	if payload.IsActive != nil {
		company.IsActive = *payload.IsActive
	}
	if err := s.Store.UpdateCompany(&company); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a empresa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": company})
}
