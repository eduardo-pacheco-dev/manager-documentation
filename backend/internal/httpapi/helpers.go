package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
)

const sessionCookie = "dc_session"

type errorResponse struct {
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string, fields map[string]string) {
	writeJSON(w, status, errorResponse{Message: message, Errors: fields})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "Não foi possível interpretar os dados enviados.", nil)
		return false
	}
	return true
}

type validationErrors map[string]string

func (v validationErrors) add(field, message string) {
	if _, exists := v[field]; !exists {
		v[field] = message
	}
}

func (v validationErrors) empty() bool {
	return len(v) == 0
}

func writeValidation(w http.ResponseWriter, fields validationErrors) {
	writeError(w, http.StatusUnprocessableEntity, "Existem campos inválidos.", fields)
}

func validateEmail(value string) bool {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)
	if err != nil {
		return false
	}
	return strings.EqualFold(address.Address, value) && strings.Contains(value, "@")
}

func validatePassword(value string) bool {
	return len([]rune(value)) >= 8
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseID(r *http.Request, name string) (int64, bool) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

func setSessionCookie(w http.ResponseWriter, cfg config.Config, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
	})
}

func clearSessionCookie(w http.ResponseWriter, cfg config.Config) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func paginationMeta(page, perPage, total int) map[string]any {
	lastPage := 1
	if perPage > 0 {
		lastPage = (total + perPage - 1) / perPage
	}
	if lastPage < 1 {
		lastPage = 1
	}
	return map[string]any{
		"current_page": page,
		"last_page":    lastPage,
		"per_page":     perPage,
		"total":        total,
		"from":         min((page-1)*perPage+1, max(total, 0)),
		"to":           min(page*perPage, total),
	}
}

type moduleView struct {
	Key            string `json:"key"`
	Label          string `json:"label"`
	Icon           string `json:"icon"`
	Path           string `json:"path"`
	Section        string `json:"section"`
	DefaultSection string `json:"default_section"`
	Toggleable     bool   `json:"toggleable"`
	AdminOnly      bool   `json:"admin_only"`
	Primary        bool   `json:"primary"`
	Enabled        bool   `json:"enabled"`
}

func modulesForUser(user models.User) ([]moduleView, []moduleView) {
	available := models.AvailableModules(user)
	enabledSet := map[string]bool{}
	for _, module := range models.EnabledModules(user) {
		enabledSet[module.Key] = true
	}
	all := make([]moduleView, 0, len(available))
	for _, module := range available {
		all = append(all, moduleView{
			Key:            module.Key,
			Label:          module.Label,
			Icon:           module.Icon,
			Path:           module.Path,
			Section:        models.ModuleSection(user, module),
			DefaultSection: module.DefaultSection,
			Toggleable:     module.Toggleable,
			AdminOnly:      module.AdminOnly,
			Primary:        module.Primary,
			Enabled:        enabledSet[module.Key],
		})
	}
	enabled := make([]moduleView, 0, len(all))
	for _, module := range all {
		if module.Enabled {
			enabled = append(enabled, module)
		}
	}
	return all, enabled
}

func formatDate(t time.Time) string {
	return t.Local().Format("2006-01-02T15:04:05-07:00")
}

func ptr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func fmtInt(value int) string {
	return strconv.Itoa(value)
}

func greetingFor(user models.User) string {
	return fmt.Sprintf("%s, %s!", greetingPrefix(), user.FirstName())
}

func greetingPrefix() string {
	switch hour := time.Now().Hour(); {
	case hour < 12:
		return "Bom dia"
	case hour < 18:
		return "Boa tarde"
	default:
		return "Boa noite"
	}
}

func modelsStringList(values []string) models.NullableStringList {
	return models.NewStringList(values)
}

func modelsIntMap(values map[string]int) models.NullableIntMap {
	return models.NewIntMap(values)
}

func parseDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime %q", value)
}
