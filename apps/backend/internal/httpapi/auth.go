package httpapi

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/models"
)

type authPayload struct {
	Company              string `json:"company"`
	CompanyID            int64  `json:"company_id"`
	Name                 string `json:"name"`
	Surname              string `json:"surname"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	DeviceName           string `json:"device_name"`
	Remember             bool   `json:"remember"`
}

type authResponse struct {
	Data struct {
		User    models.PublicUser `json:"user"`
		Company models.Company    `json:"company"`
		Token   string            `json:"token,omitempty"`
		CSRF    string            `json:"csrf_token,omitempty"`
		Modules []moduleView      `json:"modules"`
	} `json:"data"`
}

// resolveLoginCompany accepts the company slug/ID sent by the login form. The
// default value keeps older integrations working with the seeded company.
func (s *Server) resolveLoginCompany(payload authPayload) (models.Company, error) {
	identifier := strings.TrimSpace(payload.Company)
	if payload.CompanyID > 0 {
		return s.Store.FindCompanyByID(payload.CompanyID)
	}
	if identifier == "" {
		identifier = "padrao"
	}
	return s.Store.FindCompanyByIdentifier(identifier)
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var payload authPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	email := normalizeEmail(payload.Email)
	if !s.Limiter.Allow(email + "|" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Muitas tentativas. Tente novamente em alguns minutos.", nil)
		return
	}
	company, companyErr := s.resolveLoginCompany(payload)
	if companyErr != nil {
		writeValidation(w, validationErrors{"company": "Empresa não encontrada."})
		return
	}
	if !company.IsActive {
		writeValidation(w, validationErrors{"company": "Esta empresa está desativada."})
		return
	}

	fields := validationErrors{}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "O nome é obrigatório.")
	} else if len([]rune(payload.Name)) > 255 {
		fields.add("name", "O nome deve ter no máximo 255 caracteres.")
	}
	if email == "" {
		fields.add("email", "O e-mail é obrigatório.")
	} else if !validateEmail(email) {
		fields.add("email", "Informe um e-mail válido.")
	}
	if payload.Password == "" {
		fields.add("password", "A senha é obrigatória.")
	} else if !validatePassword(payload.Password) {
		fields.add("password", "A senha deve ter pelo menos 8 caracteres.")
	} else if payload.Password != payload.PasswordConfirmation {
		fields.add("password_confirmation", "A confirmação de senha não confere.")
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}

	if _, err := s.Store.FindUserByCompanyEmail(company.ID, email); err == nil {
		writeValidation(w, validationErrors{"email": "Este e-mail já está em uso nesta empresa."})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar a conta.", nil)
		return
	}
	user := models.User{
		CompanyID:          company.ID,
		Name:               strings.TrimSpace(payload.Name),
		Surname:            ptr(payload.Surname),
		Email:              email,
		PasswordHash:       string(hash),
		IsAdmin:            false,
		IsActive:           true,
		Theme:              "system",
		EmailNotifications: true,
	}
	if err := s.Store.CreateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar a conta.", nil)
		return
	}

	if tenant, err := s.tenantFor(company.ID); err == nil {
		_ = tenant.Notifier.Notify(user, "Sua conta foi criada", "Bem-vindo ao Documentation Center. Acesse as configurações para personalizar sua experiência.", "/settings", "success")
	}

	s.writeAuthResponse(w, r, company, user, http.StatusCreated, payload.DeviceName, false)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var payload authPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	email := normalizeEmail(payload.Email)
	if !s.Limiter.Allow(email + "|" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "Muitas tentativas. Tente novamente em alguns minutos.", nil)
		return
	}

	fields := validationErrors{}
	if email == "" {
		fields.add("email", "O e-mail é obrigatório.")
	} else if !validateEmail(email) {
		fields.add("email", "Informe um e-mail válido.")
	}
	if payload.Password == "" {
		fields.add("password", "A senha é obrigatória.")
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}

	company, err := s.resolveLoginCompany(payload)
	if err != nil || !company.IsActive {
		writeError(w, http.StatusUnauthorized, "As credenciais informadas não correspondem a um usuário.", map[string]string{
			"email": "As credenciais informadas não correspondem a um usuário.",
		})
		return
	}
	user, err := s.Store.FindUserByCompanyEmail(company.ID, email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "As credenciais informadas não correspondem a um usuário.", map[string]string{
			"email": "As credenciais informadas não correspondem a um usuário.",
		})
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusForbidden, "Sua conta está desativada.", map[string]string{
			"email": "Sua conta está desativada.",
		})
		return
	}

	s.writeAuthResponse(w, r, company, user, http.StatusOK, payload.DeviceName, payload.Remember)
}

func (s *Server) writeAuthResponse(w http.ResponseWriter, r *http.Request, company models.Company, user models.User, status int, deviceName string, remember bool) {
	ttl := s.Config.SessionTTL
	if remember && ttl < 30*24*time.Hour {
		ttl = 30 * 24 * time.Hour
	}
	sessionStore, err := s.tenantFor(user.CompanyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível iniciar a sessão.", nil)
		return
	}
	token, csrf, err := sessionStore.Store.CreateSession(user.ID, ttl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível iniciar a sessão.", nil)
		return
	}
	setSessionCookie(w, s.Config, r, token, ttl)

	response := authResponse{}
	response.Data.User = user.Public()
	response.Data.Company = company
	response.Data.CSRF = csrf
	_, enabled := modulesForUser(user)
	response.Data.Modules = enabled

	if name := strings.TrimSpace(deviceName); name != "" {
		if apiToken, err := sessionStore.Store.CreateAPIToken(user.ID, name); err == nil {
			response.Data.Token = apiToken
		}
	}

	writeJSON(w, status, response)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	response := struct {
		Data struct {
			User    models.PublicUser `json:"user"`
			Company models.Company    `json:"company"`
			CSRF    string            `json:"csrf_token,omitempty"`
			Modules []moduleView      `json:"modules"`
		} `json:"data"`
	}{}
	response.Data.User = auth.User.Public()
	companyID := auth.TenantCompanyID
	if companyID == 0 {
		companyID = auth.User.CompanyID
	}
	if company, err := s.Store.FindCompanyByID(companyID); err == nil {
		response.Data.Company = company
	}
	response.Data.CSRF = auth.CSRFToken
	_, enabled := modulesForUser(auth.User)
	response.Data.Modules = enabled
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	if auth != nil {
		if auth.Method == "token" {
			_ = s.Store.DeleteAPIToken(auth.RawToken)
		} else {
			_ = s.Store.DeleteSession(auth.RawToken)
		}
	}
	clearSessionCookie(w, s.Config, r)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Sessão encerrada com sucesso."})
}

func (s *Server) showProfile(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	auth := authFrom(r)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"user": auth.User.Public()}})
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	auth := authFrom(r)
	var payload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	email := normalizeEmail(payload.Email)
	fields := validationErrors{}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "O nome é obrigatório.")
	} else if len([]rune(payload.Name)) > 255 {
		fields.add("name", "O nome deve ter no máximo 255 caracteres.")
	}
	if email == "" || !validateEmail(email) {
		fields.add("email", "Informe um e-mail válido.")
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	existing, err := s.Store.FindUserByEmail(email)
	if err == nil && existing.ID != auth.User.ID {
		writeValidation(w, validationErrors{"email": "Este e-mail já está em uso."})
		return
	}

	auth.User.Name = strings.TrimSpace(payload.Name)
	auth.User.Email = email
	if err := s.Store.UpdateUser(&auth.User); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o perfil.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Perfil atualizado com sucesso.",
		"data":    map[string]any{"user": auth.User.Public()},
	})
}
