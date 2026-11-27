package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/store"
)

type userPayload struct {
	Name                 string `json:"name"`
	Surname              string `json:"surname"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	IsAdmin              *bool  `json:"is_admin"`
	CompanyID            int64  `json:"company_id"`
	IsSuperAdmin         *bool  `json:"is_super_admin"`
}

// manageUserTenant lets a super-admin manage users in a company other than the
// active one. Regular admins are always pinned to their own company and cannot
// promote anyone to super-admin.
func (s *Server) manageUserTenant(w http.ResponseWriter, r *http.Request, companyID int64) (*Server, bool) {
	tenant, ok := s.useTenant(w, r)
	if !ok {
		return nil, false
	}
	auth := authFrom(r)
	if !auth.User.IsSuperAdmin || companyID == 0 || companyID == auth.TenantCompanyID {
		return tenant, true
	}
	company, err := s.Store.FindCompanyByID(companyID)
	if err != nil || !company.IsActive {
		writeValidation(w, validationErrors{"company_id": "Empresa inválida ou inativa."})
		return nil, false
	}
	target, err := s.tenantFor(companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível acessar a empresa.", nil)
		return nil, false
	}
	return target, true
}

func tenantUserCompanyID(r *http.Request, requested int64) int64 {
	auth := authFrom(r)
	if auth.User.IsSuperAdmin && requested > 0 {
		return requested
	}
	return auth.TenantCompanyID
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 10)
	if perPage != 10 && perPage != 25 && perPage != 50 {
		perPage = 10
	}
	filter := store.UserFilter{
		Search:    strings.TrimSpace(query.Get("search")),
		Profile:   query.Get("profile"),
		Status:    query.Get("status"),
		Sort:      query.Get("sort"),
		Direction: query.Get("direction"),
		Page:      page,
		PerPage:   perPage,
	}
	if filter.Sort == "" {
		filter.Sort = "created_at"
	}
	if filter.Direction != "asc" && filter.Direction != "desc" {
		if filter.Sort == "created_at" {
			filter.Direction = "desc"
		} else {
			filter.Direction = "asc"
		}
	}

	users, total, err := s.Store.ListUsers(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os usuários.", nil)
		return
	}
	stats, err := s.Store.CountUserStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as estatísticas.", nil)
		return
	}

	public := make([]models.PublicUser, 0, len(users))
	for _, user := range users {
		public = append(public, user.Public())
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  public,
		"stats": stats,
		"meta":  paginationMeta(page, perPage, total),
		"filters": map[string]any{
			"search":    filter.Search,
			"profile":   filter.Profile,
			"status":    filter.Status,
			"sort":      filter.Sort,
			"direction": filter.Direction,
			"per_page":  perPage,
		},
	})
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var payload userPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	tenant, ok := s.manageUserTenant(w, r, payload.CompanyID)
	if !ok {
		return
	}
	isSuperAdmin := authFrom(r).User.IsSuperAdmin
	if !isSuperAdmin && payload.IsSuperAdmin != nil && *payload.IsSuperAdmin {
		writeValidation(w, validationErrors{"is_super_admin": "Somente super-administradores podem promover usuários."})
		return
	}
	email := normalizeEmail(payload.Email)
	fields := validateUserPayload(payload, email, true, 0)
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if existing, err := tenant.Store.FindUserByCompanyEmail(tenantUserCompanyID(r, payload.CompanyID), email); err == nil && existing.ID > 0 {
		writeValidation(w, validationErrors{"email": "Este e-mail já está em uso."})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o usuário.", nil)
		return
	}
	user := models.User{
		Name:               strings.TrimSpace(payload.Name),
		Surname:            ptr(payload.Surname),
		Email:              email,
		PasswordHash:       string(hash),
		IsAdmin:            payload.IsAdmin != nil && *payload.IsAdmin,
		IsSuperAdmin:       isSuperAdmin && payload.IsSuperAdmin != nil && *payload.IsSuperAdmin,
		IsActive:           true,
		Theme:              "system",
		EmailNotifications: true,
	}
	if user.IsSuperAdmin {
		user.IsAdmin = true
	}
	if err := tenant.Store.CreateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o usuário.", nil)
		return
	}
	_ = tenant.Notifier.Notify(user, "Sua conta foi criada", "Um administrador criou sua conta no Documentation Center.", "/settings", "success")
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Usuário criado com sucesso.",
		"data":    map[string]any{"user": user.Public()},
	})
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário não encontrado.", nil)
		return
	}
	var payload userPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	tenant, ok := s.manageUserTenant(w, r, payload.CompanyID)
	if !ok {
		return
	}
	isSuperAdmin := authFrom(r).User.IsSuperAdmin
	if !isSuperAdmin && payload.IsSuperAdmin != nil && *payload.IsSuperAdmin {
		writeValidation(w, validationErrors{"is_super_admin": "Somente super-administradores podem promover usuários."})
		return
	}
	user, err := tenant.Store.FindUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Usuário não encontrado.", nil)
		return
	}
	if isSuperAdmin && payload.CompanyID > 0 && payload.CompanyID != user.CompanyID {
		writeValidation(w, validationErrors{"company_id": "Não é possível mover usuários entre empresas."})
		return
	}
	email := normalizeEmail(payload.Email)
	fields := validateUserPayload(payload, email, false, id)
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	companyID := tenantUserCompanyID(r, payload.CompanyID)
	if companyID == 0 {
		companyID = user.CompanyID
	}
	if existing, err := tenant.Store.FindUserByCompanyEmail(companyID, email); err == nil && existing.ID != id {
		writeValidation(w, validationErrors{"email": "Este e-mail já está em uso."})
		return
	}

	if strings.TrimSpace(payload.Password) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o usuário.", nil)
			return
		}
		user.PasswordHash = string(hash)
	}
	user.Name = strings.TrimSpace(payload.Name)
	user.Surname = ptr(payload.Surname)
	user.Email = email
	if payload.IsAdmin != nil {
		user.IsAdmin = *payload.IsAdmin
	}
	if isSuperAdmin && payload.IsSuperAdmin != nil {
		if user.ID == authFrom(r).User.ID && !*payload.IsSuperAdmin {
			writeValidation(w, validationErrors{"is_super_admin": "Você não pode remover o próprio acesso de super-administrador."})
			return
		}
		user.IsSuperAdmin = *payload.IsSuperAdmin
		if user.IsSuperAdmin {
			user.IsAdmin = true
		}
	}
	if err := tenant.Store.UpdateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o usuário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Usuário atualizado com sucesso.",
		"data":    map[string]any{"user": user.Public()},
	})
}

func (s *Server) toggleUser(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário não encontrado.", nil)
		return
	}
	auth := authFrom(r)
	if auth.User.ID == id {
		writeValidation(w, validationErrors{"user": "Você não pode desativar a própria conta."})
		return
	}
	user, err := s.Store.FindUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Usuário não encontrado.", nil)
		return
	}
	user.IsActive = !user.IsActive
	if !user.IsActive {
		_ = s.Store.DeleteUserSessions(user.ID)
	}
	if err := s.Store.UpdateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível alterar o status do usuário.", nil)
		return
	}
	message := "Usuário desativado com sucesso."
	if user.IsActive {
		message = "Usuário ativado com sucesso."
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": message,
		"data":    map[string]any{"user": user.Public()},
	})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário não encontrado.", nil)
		return
	}
	if authFrom(r).User.ID == id {
		writeValidation(w, validationErrors{"user": "Você não pode excluir a própria conta."})
		return
	}
	if _, err := s.Store.FindUserByID(id); err != nil {
		writeError(w, http.StatusNotFound, "Usuário não encontrado.", nil)
		return
	}
	if err := s.Store.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o usuário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Usuário excluído com sucesso."})
}

func validateUserPayload(payload userPayload, email string, requirePassword bool, currentID int64) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "O nome é obrigatório.")
	} else if len([]rune(payload.Name)) > 255 {
		fields.add("name", "O nome deve ter no máximo 255 caracteres.")
	}
	if len([]rune(payload.Surname)) > 255 {
		fields.add("surname", "O sobrenome deve ter no máximo 255 caracteres.")
	}
	if email == "" {
		fields.add("email", "O e-mail é obrigatório.")
	} else if !validateEmail(email) {
		fields.add("email", "Informe um e-mail válido.")
	}
	if requirePassword || strings.TrimSpace(payload.Password) != "" {
		if payload.Password == "" {
			fields.add("password", "A senha é obrigatória.")
		} else if !validatePassword(payload.Password) {
			fields.add("password", "A senha deve ter pelo menos 8 caracteres.")
		} else if payload.Password != payload.PasswordConfirmation {
			fields.add("password_confirmation", "A confirmação de senha não confere.")
		}
	}
	return fields
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
