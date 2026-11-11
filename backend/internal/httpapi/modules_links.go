package httpapi

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skip2/go-qrcode"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

var shortCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

var reservedShortCodes = map[string]bool{
	"api": true, "s": true, "login": true, "register": true, "dashboard": true,
	"links": true, "uploads": true, "healthz": true, "docs": true, "settings": true,
	"users": true, "profile": true, "stations": true, "nokia": true, "agenda": true,
	"tasks": true, "follow-ups": true, "notifications": true, "dev-docs": true, "assets": true,
}

type shortLinkPayload struct {
	Code        string `json:"code"`
	TargetURL   string `json:"target_url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ExpiresAt   string `json:"expires_at"`
	IsActive    *bool  `json:"is_active"`
}

func (p shortLinkPayload) input() modules.ShortLinkInput {
	active := true
	if p.IsActive != nil {
		active = *p.IsActive
	}
	return modules.ShortLinkInput{
		Code: p.Code, TargetURL: p.TargetURL, Title: p.Title, Description: p.Description,
		ExpiresAt: p.ExpiresAt, IsActive: active,
	}
}

func validateShortLink(payload shortLinkPayload) validationErrors {
	fields := validationErrors{}
	target := strings.TrimSpace(payload.TargetURL)
	if target == "" {
		fields["target_url"] = "Informe a URL de destino."
	} else if len(target) > 2048 {
		fields["target_url"] = "A URL deve ter no máximo 2048 caracteres."
	} else if parsed, err := url.Parse(target); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		fields["target_url"] = "Use uma URL válida com http:// ou https://."
	}
	code := strings.ToLower(strings.TrimSpace(payload.Code))
	if code != "" {
		if !shortCodePattern.MatchString(code) {
			fields["code"] = "Use de 3 a 32 caracteres: letras minúsculas, números, hífen ou sublinhado."
		} else if reservedShortCodes[code] {
			fields["code"] = "Este código é reservado pelo sistema."
		}
	}
	if len([]rune(strings.TrimSpace(payload.Title))) > 255 {
		fields["title"] = "O título deve ter no máximo 255 caracteres."
	}
	if strings.TrimSpace(payload.ExpiresAt) != "" {
		if _, err := parseDateTime(payload.ExpiresAt); err != nil {
			fields["expires_at"] = "Informe uma data/hora válida."
		}
	}
	return fields
}

func (s *Server) shortLinkURL(code string) string {
	return strings.TrimRight(s.Config.PublicURL, "/") + "/s/" + code
}

func (s *Server) presentShortLink(link *modules.ShortLink) {
	link.Expired = link.ExpiresAt != nil && !link.ExpiresAt.After(time.Now())
	link.ShortURL = s.shortLinkURL(link.Code)
}

func (s *Server) canManageShortLink(userID int64, admin bool, link modules.ShortLink) bool {
	return admin || link.UserID == userID
}

func (s *Server) linksIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	status := strings.TrimSpace(query.Get("status"))
	if status != "" && status != "active" && status != "inactive" && status != "expired" {
		writeValidation(w, validationErrors{"status": "Status inválido."})
		return
	}
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	items, total, stats, err := s.Modules.ListShortLinks(modules.ShortLinkFilter{
		Search: strings.TrimSpace(query.Get("search")), Status: status, Page: page, PerPage: perPage,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os links.", nil)
		return
	}
	for i := range items {
		s.presentShortLink(&items[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  items,
		"stats": stats,
		"meta":  paginationMeta(page, perPage, total),
		"filters": map[string]any{
			"search": strings.TrimSpace(query.Get("search")), "status": status,
		},
	})
}

func (s *Server) linksCreate(w http.ResponseWriter, r *http.Request) {
	var payload shortLinkPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateShortLink(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	link, err := s.Modules.CreateShortLink(currentUser(r).ID, payload.input())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			writeValidation(w, validationErrors{"code": "Este código já está em uso."})
			return
		}
		log.Printf("short link create: %v", err)
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar o link.", nil)
		return
	}
	s.presentShortLink(&link)
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Link curto criado com sucesso.",
		"data":    map[string]any{"link": link, "short_url": s.shortLinkURL(link.Code)},
	})
}

func (s *Server) linksShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindShortLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	s.presentShortLink(&link)
	clicks, _ := s.Modules.ShortLinkClicks(id, 30)
	daily, _ := s.Modules.ShortLinkDaily(id, 14)
	sources, _ := s.Modules.ShortLinkSources(id, 5)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"link": link, "short_url": s.shortLinkURL(link.Code),
		"clicks": clicks, "daily": daily, "sources": sources,
	}})
}

func (s *Server) linksUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindShortLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	user := currentUser(r)
	if !s.canManageShortLink(user.ID, user.IsAdmin, link) {
		writeError(w, http.StatusForbidden, "Você não pode editar este link.", nil)
		return
	}
	var payload shortLinkPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateShortLink(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	updated, err := s.Modules.UpdateShortLink(id, payload.input())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			writeValidation(w, validationErrors{"code": "Este código já está em uso."})
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível atualizar o link.", nil)
		return
	}
	s.presentShortLink(&updated)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Link atualizado com sucesso.",
		"data":    map[string]any{"link": updated, "short_url": s.shortLinkURL(updated.Code)},
	})
}

func (s *Server) linksDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindShortLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	user := currentUser(r)
	if !s.canManageShortLink(user.ID, user.IsAdmin, link) {
		writeError(w, http.StatusForbidden, "Você não pode excluir este link.", nil)
		return
	}
	if err := s.Modules.DeleteShortLink(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o link.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Link excluído com sucesso."})
}

func (s *Server) linksToggle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindShortLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	user := currentUser(r)
	if !s.canManageShortLink(user.ID, user.IsAdmin, link) {
		writeError(w, http.StatusForbidden, "Você não pode alterar este link.", nil)
		return
	}
	updated, err := s.Modules.ToggleShortLink(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível alterar o link.", nil)
		return
	}
	s.presentShortLink(&updated)
	message := "Link ativado."
	if !updated.IsActive {
		message = "Link desativado."
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": message, "data": map[string]any{"link": updated}})
}

func (s *Server) linksQRCode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindShortLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	png, err := qrcode.Encode(s.shortLinkURL(link.Code), qrcode.Medium, 512)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível gerar o QR Code.", nil)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s.png"`, link.Code))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func (s *Server) linksExport(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	items, _, _, err := s.Modules.ListShortLinks(modules.ShortLinkFilter{
		Search: strings.TrimSpace(query.Get("search")), Status: strings.TrimSpace(query.Get("status")),
		Page: 1, PerPage: 10000,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar os links.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="links-%s.csv"`, time.Now().Format("2006-01-02")))
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"Código", "URL curta", "Destino", "Título", "Cliques", "Visitantes únicos", "Ativo", "Expira em", "Criador", "Criado em"})
	for _, item := range items {
		active := "Não"
		if item.IsActive && !item.Expired {
			active = "Sim"
		}
		expires := ""
		if item.ExpiresAt != nil {
			expires = item.ExpiresAt.Format("02/01/2006 15:04")
		}
		_ = writer.Write([]string{
			item.Code, s.shortLinkURL(item.Code), item.TargetURL, derefString(item.Title),
			strconv.FormatInt(item.ClickCount, 10), strconv.FormatInt(item.UniqueClicks, 10),
			active, expires, derefString(item.CreatorName), item.CreatedAt.Format("02/01/2006 15:04"),
		})
	}
	writer.Flush()
}

func (s *Server) shortLinkRedirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	link, err := s.Modules.FindShortLinkByCode(code)
	if err != nil || !link.IsActive || link.Expired {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><meta charset="utf-8">
<title>Link indisponível</title><body style="font-family:sans-serif;padding:2rem">
<h1>Link indisponível</h1><p>Este link está desativado, expirou ou não existe.</p></body></html>`))
		return
	}
	visitorHash := shortLinkVisitorHash(r, s.Config.AppKey)
	referrer := truncateString(r.Referer(), 1024)
	userAgent := truncateString(r.UserAgent(), 512)
	if err := s.Modules.RecordShortLinkClick(link.ID, visitorHash, referrer, userAgent); err != nil {
		log.Printf("short link click %d: %v", link.ID, err)
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	http.Redirect(w, r, link.TargetURL, http.StatusFound)
}

func shortLinkVisitorHash(r *http.Request, salt string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(host + "|" + salt))
	return hex.EncodeToString(sum[:])
}

func truncateString(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return string(runes)
}
