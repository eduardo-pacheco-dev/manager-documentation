package httpapi

import (
	_ "embed"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
)

func (s *Server) showSettings(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	allModules, enabledModules := modulesForUser(user)
	allModules = settingsModules(allModules)
	enabledModules = settingsModules(enabledModules)

	timezone := s.Config.Timezone
	if stored, err := s.Store.GetSetting("timezone"); err == nil && stored != "" {
		if _, locationErr := time.LoadLocation(stored); locationErr == nil {
			timezone = stored
		}
	}

	response := map[string]any{
		"data": map[string]any{
			"user":                user.Public(),
			"modules":             allModules,
			"enabled_modules":     enabledModules,
			"theme":               user.Theme,
			"email_notifications": user.EmailNotifications,
			"timezone":            timezone,
			"timezones":           timezones(),
			"is_admin":            user.IsAdmin,
		},
	}

	if user.IsAdmin {
		storage, err := s.Store.GetStorageSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível carregar as configurações de armazenamento.", nil)
			return
		}
		response["data"].(map[string]any)["storage"] = s.storageView(storage)
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) updateTheme(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Theme string `json:"theme"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	theme := strings.ToLower(strings.TrimSpace(payload.Theme))
	if theme != "system" && theme != "light" && theme != "dark" {
		writeValidation(w, validationErrors{"theme": "Selecione um tema válido."})
		return
	}
	user := currentUser(r)
	user.Theme = theme
	if err := s.Store.UpdateUserPreferences(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar o tema.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Preferência de tema salva com sucesso.",
		"data":    map[string]any{"theme": theme},
	})
}

func (s *Server) updateModules(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Modules  []string          `json:"modules"`
		Order    []string          `json:"order"`
		Sections map[string]string `json:"sections"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	user := currentUser(r)
	available := models.AvailableModules(user)
	availableKeys := map[string]bool{}
	for _, module := range available {
		availableKeys[module.Key] = true
	}
	toggleable := map[string]bool{}
	for _, module := range models.ToggleableModules(user) {
		toggleable[module.Key] = true
	}

	fields := validationErrors{}
	selected := make([]string, 0, len(payload.Modules))
	seen := map[string]bool{}
	for _, key := range payload.Modules {
		if !toggleable[key] {
			fields.add("modules", "Um ou mais módulos selecionados não estão disponíveis para você.")
			continue
		}
		if !seen[key] {
			selected = append(selected, key)
			seen[key] = true
		}
	}
	if len(payload.Order) > 0 {
		orderSeen := map[string]bool{}
		for _, key := range payload.Order {
			if !availableKeys[key] {
				fields.add("order", "A ordem informada contém módulos inválidos.")
			}
			if orderSeen[key] {
				fields.add("order", "A ordem não pode conter módulos repetidos.")
			}
			orderSeen[key] = true
		}
	}
	for key, section := range payload.Sections {
		if !availableKeys[key] {
			fields.add("sections", "As seções informadas contêm módulos inválidos.")
		}
		if section != models.SectionSidebar && section != models.SectionModules {
			fields.add("sections", "Seção inválida.")
		}
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}

	user.EnabledModules = models.NewStringList(selected)
	if len(payload.Order) > 0 {
		order := make([]string, 0, len(payload.Order))
		for _, key := range payload.Order {
			if availableKeys[key] {
				order = append(order, key)
			}
		}
		user.ModuleOrder = models.NewStringList(order)
	}
	if len(payload.Sections) > 0 {
		sections := map[string]string{}
		for key, section := range payload.Sections {
			if availableKeys[key] && (section == models.SectionSidebar || section == models.SectionModules) {
				sections[key] = section
			}
		}
		user.ModuleSections = models.NewStringMap(sections)
	}
	if err := s.Store.UpdateUserPreferences(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar os módulos.", nil)
		return
	}
	all, enabled := modulesForUser(user)
	all = settingsModules(all)
	enabled = settingsModules(enabled)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Módulos atualizados com sucesso.",
		"data":    map[string]any{"modules": all, "enabled_modules": enabled},
	})
}

func (s *Server) updateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		EmailNotifications *bool `json:"email_notifications"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.EmailNotifications == nil {
		writeValidation(w, validationErrors{"email_notifications": "Informe a preferência de notificações."})
		return
	}
	user := currentUser(r)
	user.EmailNotifications = *payload.EmailNotifications
	if err := s.Store.UpdateUserPreferences(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar as preferências.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Preferências de notificação atualizadas com sucesso.",
		"data":    map[string]any{"email_notifications": user.EmailNotifications},
	})
}

func (s *Server) updateTimezone(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Timezone string `json:"timezone"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	timezone := strings.TrimSpace(payload.Timezone)
	if timezone == "" {
		writeValidation(w, validationErrors{"timezone": "Informe um fuso horário."})
		return
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		writeValidation(w, validationErrors{"timezone": "Fuso horário inválido."})
		return
	}
	if err := s.Store.SetSetting("timezone", timezone); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar o fuso horário.", nil)
		return
	}
	time.Local = location
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Fuso horário do sistema atualizado com sucesso.",
		"data":    map[string]any{"timezone": timezone},
	})
}

type storagePayload struct {
	Provider                  string `json:"provider"`
	MaxUploadMB               int64  `json:"max_upload_mb"`
	AwsAccessKey              string `json:"aws_access_key"`
	AwsSecretKey              string `json:"aws_secret_key"`
	AwsRegion                 string `json:"aws_region"`
	AwsBucket                 string `json:"aws_bucket"`
	AwsEndpoint               string `json:"aws_endpoint"`
	AwsPublicURL              string `json:"aws_public_url"`
	AwsUsePathStyle           bool   `json:"aws_use_path_style"`
	CloudflareAccountID       string `json:"cloudflare_account_id"`
	CloudflareAccessKeyID     string `json:"cloudflare_access_key_id"`
	CloudflareSecretAccessKey string `json:"cloudflare_secret_access_key"`
	CloudflareBucket          string `json:"cloudflare_bucket"`
	CloudflarePublicURL       string `json:"cloudflare_public_url"`
}

func (s *Server) updateStorage(w http.ResponseWriter, r *http.Request) {
	var payload storagePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	provider := strings.ToLower(strings.TrimSpace(payload.Provider))
	if provider != "local" && provider != "aws" && provider != "cloudflare" {
		writeValidation(w, validationErrors{"provider": "Provedor de armazenamento inválido."})
		return
	}
	if payload.MaxUploadMB < 1 || payload.MaxUploadMB > 200 {
		writeValidation(w, validationErrors{"max_upload_mb": "O limite deve estar entre 1 e 200 MB."})
		return
	}

	existing, err := s.Store.GetStorageSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as configurações atuais.", nil)
		return
	}
	fields := validationErrors{}
	if provider == "aws" {
		if strings.TrimSpace(payload.AwsBucket) == "" {
			fields.add("aws_bucket", "Informe o bucket.")
		}
		if strings.TrimSpace(payload.AwsRegion) == "" {
			fields.add("aws_region", "Informe a região.")
		}
		if strings.TrimSpace(payload.AwsAccessKey) == "" && !existing.HasAWSAccess() {
			fields.add("aws_access_key", "Informe a chave de acesso.")
		}
		if strings.TrimSpace(payload.AwsSecretKey) == "" && !existing.HasAWSSecret() {
			fields.add("aws_secret_key", "Informe a chave secreta.")
		}
	}
	if provider == "cloudflare" {
		if strings.TrimSpace(payload.CloudflareAccountID) == "" {
			fields.add("cloudflare_account_id", "Informe o Account ID.")
		}
		if strings.TrimSpace(payload.CloudflareBucket) == "" {
			fields.add("cloudflare_bucket", "Informe o bucket.")
		}
		if strings.TrimSpace(payload.CloudflareAccessKeyID) == "" && !existing.HasCFAccess() {
			fields.add("cloudflare_access_key_id", "Informe a chave de acesso.")
		}
		if strings.TrimSpace(payload.CloudflareSecretAccessKey) == "" && !existing.HasCFSecret() {
			fields.add("cloudflare_secret_access_key", "Informe a chave secreta.")
		}
	}
	for field, value := range map[string]string{
		"aws_endpoint":          payload.AwsEndpoint,
		"aws_public_url":        payload.AwsPublicURL,
		"cloudflare_public_url": payload.CloudflarePublicURL,
	} {
		if strings.TrimSpace(value) != "" && !validHTTPURL(value) {
			fields.add(field, "Informe uma URL válida.")
		}
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}

	settings := existing
	settings.Provider = provider
	settings.MaxUploadMB = payload.MaxUploadMB
	settings.AwsRegion = strings.TrimSpace(payload.AwsRegion)
	if settings.AwsRegion == "" {
		settings.AwsRegion = "us-east-1"
	}
	settings.AwsBucket = ptr(payload.AwsBucket)
	settings.AwsEndpoint = ptr(payload.AwsEndpoint)
	settings.AwsPublicURL = ptr(payload.AwsPublicURL)
	settings.AwsUsePathStyle = payload.AwsUsePathStyle
	settings.CloudflareAccountID = ptr(payload.CloudflareAccountID)
	settings.CloudflareBucket = ptr(payload.CloudflareBucket)
	settings.CloudflarePublicURL = ptr(payload.CloudflarePublicURL)

	if strings.TrimSpace(payload.AwsAccessKey) != "" {
		encrypted, err := s.Cipher.Encrypt(strings.TrimSpace(payload.AwsAccessKey))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível proteger as credenciais.", nil)
			return
		}
		settings.AwsAccessKey = ptr(encrypted)
	}
	if strings.TrimSpace(payload.AwsSecretKey) != "" {
		encrypted, err := s.Cipher.Encrypt(strings.TrimSpace(payload.AwsSecretKey))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível proteger as credenciais.", nil)
			return
		}
		settings.AwsSecretKey = ptr(encrypted)
	}
	if strings.TrimSpace(payload.CloudflareAccessKeyID) != "" {
		encrypted, err := s.Cipher.Encrypt(strings.TrimSpace(payload.CloudflareAccessKeyID))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível proteger as credenciais.", nil)
			return
		}
		settings.CloudflareAccessKeyID = ptr(encrypted)
	}
	if strings.TrimSpace(payload.CloudflareSecretAccessKey) != "" {
		encrypted, err := s.Cipher.Encrypt(strings.TrimSpace(payload.CloudflareSecretAccessKey))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível proteger as credenciais.", nil)
			return
		}
		settings.CloudflareSecretAccessKey = ptr(encrypted)
	}

	if err := s.Store.SaveStorageSettings(&settings); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar as configurações de armazenamento.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Configurações de armazenamento salvas com sucesso.",
		"data":    map[string]any{"storage": s.storageView(settings)},
	})
}

func (s *Server) updatePassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CurrentPassword      string `json:"current_password"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"password_confirmation"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	user := currentUser(r)
	fields := validationErrors{}
	if payload.CurrentPassword == "" {
		fields.add("current_password", "Informe a senha atual.")
	} else if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.CurrentPassword)) != nil {
		fields.add("current_password", "A senha atual não confere.")
	}
	if payload.Password == "" {
		fields.add("password", "Informe a nova senha.")
	} else if !validatePassword(payload.Password) {
		fields.add("password", "A nova senha deve ter pelo menos 8 caracteres.")
	} else if payload.Password == payload.CurrentPassword {
		fields.add("password", "A nova senha deve ser diferente da senha atual.")
	} else if payload.Password != payload.PasswordConfirmation {
		fields.add("password_confirmation", "A confirmação de senha não confere.")
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a senha.", nil)
		return
	}
	user.PasswordHash = string(hash)
	if err := s.Store.UpdateUser(&user); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a senha.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Senha atualizada com sucesso."})
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		DeletePassword string `json:"delete_password"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	user := currentUser(r)
	if payload.DeletePassword == "" {
		writeValidation(w, validationErrors{"delete_password": "Informe sua senha para excluir a conta."})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.DeletePassword)) != nil {
		writeValidation(w, validationErrors{"delete_password": "A senha informada não confere."})
		return
	}
	if err := s.Store.DeleteUser(user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir a conta.", nil)
		return
	}
	clearSessionCookie(w, s.Config, r)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Sua conta foi excluída com sucesso."})
}

func (s *Server) storageView(settings models.StorageSettings) map[string]any {
	return map[string]any{
		"provider":                  settings.Provider,
		"max_upload_mb":             settings.MaxUploadMB,
		"aws_region":                settings.AwsRegion,
		"aws_bucket":                derefString(settings.AwsBucket),
		"aws_endpoint":              derefString(settings.AwsEndpoint),
		"aws_public_url":            derefString(settings.AwsPublicURL),
		"aws_use_path_style":        settings.AwsUsePathStyle,
		"aws_access_key_set":        settings.HasAWSAccess(),
		"aws_secret_key_set":        settings.HasAWSSecret(),
		"cloudflare_account_id":     derefString(settings.CloudflareAccountID),
		"cloudflare_bucket":         derefString(settings.CloudflareBucket),
		"cloudflare_public_url":     derefString(settings.CloudflarePublicURL),
		"cloudflare_access_key_set": settings.HasCFAccess(),
		"cloudflare_secret_key_set": settings.HasCFSecret(),
		"configured":                settings.Configured(settings.Provider),
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func validHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func settingsModules(modules []moduleView) []moduleView {
	result := make([]moduleView, 0, len(modules))
	for _, module := range modules {
		if module.SystemLink {
			continue
		}
		result = append(result, module)
	}
	return result
}

func timezones() []string {
	lines := strings.Split(strings.TrimSpace(timezoneData), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

//go:embed timezones.txt
var timezoneData string
