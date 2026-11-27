package httpapi

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skip2/go-qrcode"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/modules"
)

var shortCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

var reservedShortCodes = map[string]bool{
	"api": true, "api-docs": true, "s": true, "login": true, "register": true, "dashboard": true,
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
	if target != "" {
		if len(target) > 2048 {
			fields["target_url"] = "A URL deve ter no máximo 2048 caracteres."
		} else if parsed, err := url.Parse(target); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			fields["target_url"] = "Use uma URL válida com http:// ou https://."
		}
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
	files, _ := s.Modules.ShortLinkFiles(id)
	for i := range files {
		files[i].DownloadURL = fmt.Sprintf("%s/files/%d", s.shortLinkURL(link.Code), files[i].ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"link": link, "short_url": s.shortLinkURL(link.Code),
		"clicks": clicks, "daily": daily, "sources": sources, "files": files,
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
	paths, err := s.Modules.DeleteShortLink(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o link.", nil)
		return
	}
	for _, path := range paths {
		s.Uploads.DeleteLocal(path)
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

func (s *Server) linksFiles(w http.ResponseWriter, r *http.Request) {
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
	files, err := s.Modules.ShortLinkFiles(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os arquivos.", nil)
		return
	}
	for i := range files {
		files[i].DownloadURL = fmt.Sprintf("%s/files/%d", s.shortLinkURL(link.Code), files[i].ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": files})
}

func (s *Server) linksUploadFiles(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusForbidden, "Você não pode anexar arquivos a este link.", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	if err := r.ParseMultipartForm(60 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido ou maior que o permitido.", nil)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeValidation(w, validationErrors{"files": "Selecione ao menos um arquivo."})
		return
	}
	if len(files) > 10 {
		writeValidation(w, validationErrors{"files": "Envie no máximo 10 arquivos por vez."})
		return
	}
	existing, _ := s.Modules.ShortLinkFiles(id)
	if len(existing)+len(files) > 20 {
		writeValidation(w, validationErrors{"files": "Cada link aceita no máximo 20 arquivos."})
		return
	}
	disk := s.Uploads.Provider()
	created := 0
	for _, header := range files {
		if header.Size > 50<<20 {
			writeValidation(w, validationErrors{"files": "Cada arquivo deve ter no máximo 50 MB."})
			return
		}
		file, err := header.Open()
		if err != nil {
			continue
		}
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "links")
		file.Close()
		if err != nil {
			writeValidation(w, validationErrors{"files": fmt.Sprintf("Formato não permitido: %s", header.Filename)})
			return
		}
		if _, err := s.Modules.CreateShortLinkFile(id, user.ID, header.Filename, result.Path, result.URL, disk,
			header.Header.Get("Content-Type"), header.Size); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": fmt.Sprintf("%d arquivo(s) anexado(s).", created)})
}

func (s *Server) linksDeleteFile(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	fileID, ok := parseID(r, "fileId")
	if !ok {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindShortLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Link não encontrado.", nil)
		return
	}
	user := currentUser(r)
	if !s.canManageShortLink(user.ID, user.IsAdmin, link) {
		writeError(w, http.StatusForbidden, "Você não pode remover este arquivo.", nil)
		return
	}
	file, err := s.Modules.DeleteShortLinkFile(id, fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	if file.Disk == "local" {
		s.Uploads.DeleteLocal(file.Path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Arquivo removido."})
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

type shortLinkPublicFile struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (s *Server) shortLinkPublic(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	link, err := s.Modules.FindShortLinkByCode(code)
	if err != nil || !link.IsActive || link.Expired {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
		return
	}
	files, _ := s.Modules.ShortLinkFiles(link.ID)
	if strings.TrimSpace(link.TargetURL) == "" && len(files) == 0 {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
		return
	}

	visitorHash := shortLinkVisitorHash(r, s.Config.AppKey)
	referrer := truncateString(r.Referer(), 1024)
	userAgent := truncateString(r.UserAgent(), 512)
	if err := s.Modules.RecordShortLinkClick(link.ID, visitorHash, referrer, userAgent); err != nil {
		log.Printf("short link click %d: %v", link.ID, err)
	}

	w.Header().Set("Cache-Control", "no-store")
	if len(files) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"redirect_url": link.TargetURL})
		return
	}

	pageFiles := make([]shortLinkPublicFile, 0, len(files))
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
		extension := strings.TrimPrefix(strings.ToUpper(filepath.Ext(file.OriginalName)), ".")
		if extension == "" {
			extension = "ARQUIVO"
		}
		pageFiles = append(pageFiles, shortLinkPublicFile{
			Name: file.OriginalName,
			Size: humanFileSize(file.Size),
			Type: extension,
			URL:  fmt.Sprintf("/s/%s/files/%d", link.Code, file.ID),
		})
	}
	title := link.Code
	if link.Title != nil && strings.TrimSpace(*link.Title) != "" {
		title = *link.Title
	}
	expires := ""
	if link.ExpiresAt != nil {
		expires = link.ExpiresAt.Local().Format("02/01/2006 15:04")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code": link.Code, "title": title, "description": derefString(link.Description),
		"target_url": link.TargetURL, "target_host": shortLinkTargetHost(link.TargetURL),
		"files": pageFiles, "file_count": len(pageFiles), "total_size": humanFileSize(totalSize),
		"expires": expires,
	})
}

func shortLinkTargetHost(target string) string {
	parsed, err := url.Parse(target)
	if err != nil || parsed.Host == "" {
		return target
	}
	host := parsed.Host
	if parsed.Path != "" && parsed.Path != "/" {
		host += parsed.Path
	}
	return host
}

func (s *Server) shortLinkFileDownload(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	fileID, ok := parseStringID(chi.URLParam(r, "fileId"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	link, err := s.Modules.FindShortLinkByCode(code)
	if err != nil || !link.IsActive || link.Expired {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
		return
	}
	file, err := s.Modules.FindShortLinkFile(link.ID, fileID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if file.Disk == "local" {
		target := filepath.Join(s.Config.UploadDir, filepath.FromSlash(file.Path))
		if _, err := os.Stat(target); err != nil {
			http.NotFound(w, r)
			return
		}
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalName})
		w.Header().Set("Content-Disposition", disposition)
		if file.MimeType != nil && strings.TrimSpace(*file.MimeType) != "" {
			w.Header().Set("Content-Type", *file.MimeType)
		}
		http.ServeFile(w, r, target)
		return
	}
	http.Redirect(w, r, file.URL, http.StatusFound)
}

func parseStringID(value string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return id, err == nil && id > 0
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
