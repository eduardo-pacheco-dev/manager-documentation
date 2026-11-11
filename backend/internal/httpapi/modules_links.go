package httpapi

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"html/template"
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

type shortLinkPageFile struct {
	Name string
	Size string
	Type string
	URL  string
}

var shortLinkPublicTemplate = template.Must(template.New("short-link").Parse(`<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="theme-color" content="#161616">
<meta name="robots" content="noindex">
<title>{{.Title}}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{
  --cds-background:#f4f4f4;
  --cds-layer:#ffffff;
  --cds-layer-hover:#f4f4f4;
  --cds-text-primary:#161616;
  --cds-text-secondary:#525252;
  --cds-text-helper:#6f6f6f;
  --cds-border-subtle:#e0e0e0;
  --cds-border-strong:#8d8d8d;
  --cds-interactive:#0f62fe;
  --cds-interactive-hover:#0043ce;
  --cds-support-info:#0043ce;
  --cds-header:#161616;
  --cds-focus:#0f62fe;
  --cds-spacing-03:.5rem;
  --cds-spacing-05:1rem;
  --cds-spacing-07:2rem;
  --cds-spacing-09:3rem;
}
*{box-sizing:border-box}
html{background:var(--cds-background)}
body{
  margin:0;
  background:var(--cds-background);
  color:var(--cds-text-primary);
  font-family:"IBM Plex Sans","Segoe UI",system-ui,-apple-system,sans-serif;
  font-size:1rem;
  line-height:1.5;
  -webkit-font-smoothing:antialiased;
}
a{color:var(--cds-interactive)}
.cds-header{
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:1rem;
  height:3rem;
  padding:0 var(--cds-spacing-05);
  background:var(--cds-header);
  color:#fff;
}
.cds-header__brand{display:flex;align-items:center;gap:.75rem;font-size:.875rem;font-weight:600;letter-spacing:.01em}
.cds-header__mark{
  display:flex;align-items:center;justify-content:center;width:1.5rem;height:1.5rem;
  background:#fff;color:#161616;font-size:.75rem;font-weight:600;
}
.cds-header__meta{color:#c6c6c6;font-size:.75rem}
.shell{max-width:66rem;margin:0 auto;padding:var(--cds-spacing-07) var(--cds-spacing-05) var(--cds-spacing-09)}
.eyebrow{
  margin:0 0 .35rem;color:var(--cds-text-secondary);
  font-size:.75rem;font-weight:500;letter-spacing:.08em;text-transform:uppercase;
}
h1{
  margin:0 0 .5rem;
  font-size:clamp(1.75rem,4vw,2.625rem);
  font-weight:400;
  letter-spacing:0;
  line-height:1.15;
}
.lead{max-width:48rem;margin:0 0 1.25rem;color:var(--cds-text-secondary);font-size:1rem}
.tag-row{display:flex;flex-wrap:wrap;gap:.5rem;margin:0 0 2rem}
.cds-tag{
  display:inline-flex;align-items:center;gap:.35rem;
  min-height:1.5rem;padding:.15rem .55rem;
  border-radius:999px;background:#e0e0e0;color:#161616;
  font-size:.75rem;line-height:1.3;white-space:nowrap;
}
.cds-tag--blue{background:#d0e2ff;color:#002d9c}
.cds-tag--green{background:#defbe6;color:#0e6027}
.cds-tag--gray{background:#e0e0e0;color:#393939}
.panel{
  background:var(--cds-layer);
  border:1px solid var(--cds-border-subtle);
  padding:var(--cds-spacing-07);
  margin-bottom:var(--cds-spacing-05);
}
.panel:last-of-type{margin-bottom:0}
.panel h2{margin:0;font-size:1.25rem;font-weight:400;line-height:1.3}
.panel .panel__hint{margin:.25rem 0 1.25rem;color:var(--cds-text-helper);font-size:.8rem}
.target-row{display:flex;flex-wrap:wrap;align-items:center;gap:.75rem 1rem}
.cds-btn{
  display:inline-flex;align-items:center;justify-content:space-between;gap:2rem;
  min-height:3rem;padding:0 1rem;border:1px solid transparent;
  background:var(--cds-interactive);color:#fff;
  font-family:inherit;font-size:.875rem;font-weight:400;
  text-decoration:none;cursor:pointer;transition:background .12s ease;
}
.cds-btn:hover{background:var(--cds-interactive-hover);color:#fff}
.cds-btn:focus-visible{outline:2px solid var(--cds-focus);outline-offset:2px}
.cds-btn svg{flex-shrink:0}
.target-url{
  max-width:36rem;overflow:hidden;color:var(--cds-text-helper);
  font-size:.75rem;text-overflow:ellipsis;white-space:nowrap;
}
.file-list{list-style:none;margin:0;padding:0;border-top:1px solid var(--cds-border-subtle)}
.file-row{
  display:flex;align-items:center;gap:1rem;
  padding:1rem 0;border-bottom:1px solid var(--cds-border-subtle);
  transition:background .12s ease;
}
.file-row:hover{background:var(--cds-layer-hover)}
.file-icon{
  display:flex;align-items:center;justify-content:center;
  width:2.5rem;height:2.5rem;flex-shrink:0;background:#edf5ff;color:var(--cds-support-info);
}
.file-info{display:flex;flex-direction:column;gap:.15rem;min-width:0;flex:1}
.file-info strong{font-size:.875rem;font-weight:500;overflow-wrap:anywhere}
.file-info span{color:var(--cds-text-helper);font-size:.75rem}
.file-download{
  display:inline-flex;align-items:center;gap:.5rem;flex-shrink:0;
  min-height:2.5rem;padding:0 .9rem;border:1px solid var(--cds-interactive);
  color:var(--cds-interactive);font-size:.8rem;text-decoration:none;
  transition:background .12s ease,color .12s ease;
}
.file-download:hover{background:var(--cds-interactive);color:#fff}
.file-download:focus-visible{outline:2px solid var(--cds-focus);outline-offset:2px}
.empty-note{color:var(--cds-text-secondary);margin:.5rem 0 0;font-size:.875rem}
.notice{
  display:flex;gap:.75rem;margin-top:var(--cds-spacing-05);
  padding:1rem;border-left:3px solid var(--cds-interactive);background:#edf5ff;
  color:#0043ce;font-size:.8rem;
}
.footer{
  max-width:66rem;margin:0 auto;padding:0 var(--cds-spacing-05) var(--cds-spacing-07);
  color:var(--cds-text-helper);font-size:.72rem;
  display:flex;flex-wrap:wrap;gap:.5rem;justify-content:space-between;
}
.footer a{color:inherit}
@media (max-width:42rem){
  .shell{padding-top:var(--cds-spacing-05)}
  .panel{padding:1.25rem}
  .file-row{flex-wrap:wrap}
  .file-download{margin-left:3.5rem}
  .cds-header__meta{display:none}
}
</style>
</head>
<body>
<header class="cds-header">
  <div class="cds-header__brand"><span class="cds-header__mark">DC</span> Documentation Center</div>
  <span class="cds-header__meta">Compartilhamento de arquivos</span>
</header>

<main class="shell">
  <p class="eyebrow">Link compartilhado</p>
  <h1>{{.Title}}</h1>
  {{if .Description}}<p class="lead">{{.Description}}</p>{{end}}
  <div class="tag-row">
    <span class="cds-tag cds-tag--blue">{{.FileCount}} arquivo(s)</span>
    <span class="cds-tag cds-tag--gray">{{.TotalSize}}</span>
    {{if .Expires}}<span class="cds-tag cds-tag--green">Expira em {{.Expires}}</span>{{end}}
  </div>

  {{if .Target}}
  <section class="panel">
    <h2>Destino do link</h2>
    <p class="panel__hint">Este link também aponta para uma página externa.</p>
    <div class="target-row">
      <a class="cds-btn" href="{{.Target}}" target="_blank" rel="noopener noreferrer">
        Abrir destino
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M3 13 13 3M6 3h7v7" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>
      </a>
      <span class="target-url">{{.TargetHost}}</span>
    </div>
  </section>
  {{end}}

  <section class="panel">
    <h2>Arquivos para download</h2>
    <p class="panel__hint">Baixe os arquivos disponibilizados neste link.</p>
    {{if .Files}}
    <ul class="file-list">
      {{range .Files}}
      <li class="file-row">
        <span class="file-icon" aria-hidden="true">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M5 2.5h6l4 4v11H5v-15Z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><path d="M11 2.5v4h4" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><path d="M7.5 12h5M7.5 14.5h3.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
        </span>
        <div class="file-info">
          <strong>{{.Name}}</strong>
          <span>{{.Type}} · {{.Size}}</span>
        </div>
        <a class="file-download" href="{{.URL}}" download>
          Baixar
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M8 2v8m0 0 3-3m-3 3L5 7M3 13h10" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
        </a>
      </li>
      {{end}}
    </ul>
    {{else}}
    <p class="empty-note">Nenhum arquivo anexado a este link.</p>
    {{end}}
  </section>

  <div class="notice">
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="6.5" stroke="currentColor" stroke-width="1.2"/><path d="M8 7v4M8 4.8v.2" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>
    <span>Os arquivos são disponibilizados pelo proprietário do link. Em caso de dúvida, entre em contato com quem compartilhou.</span>
  </div>
</main>

<footer class="footer">
  <span>Documentation Center · Link /s/{{.Code}}</span>
  <span>Powered by IBM Carbon Design System</span>
</footer>
</body>
</html>`))

func shortLinkUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><meta charset="utf-8">
<title>Link indisponível</title><body style="font-family:sans-serif;padding:2rem">
<h1>Link indisponível</h1><p>Este link está desativado, expirou ou não existe.</p></body></html>`))
}

func (s *Server) shortLinkRedirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	link, err := s.Modules.FindShortLinkByCode(code)
	if err != nil || !link.IsActive || link.Expired {
		shortLinkUnavailable(w)
		return
	}
	files, _ := s.Modules.ShortLinkFiles(link.ID)
	if strings.TrimSpace(link.TargetURL) == "" && len(files) == 0 {
		shortLinkUnavailable(w)
		return
	}
	visitorHash := shortLinkVisitorHash(r, s.Config.AppKey)
	referrer := truncateString(r.Referer(), 1024)
	userAgent := truncateString(r.UserAgent(), 512)
	if err := s.Modules.RecordShortLinkClick(link.ID, visitorHash, referrer, userAgent); err != nil {
		log.Printf("short link click %d: %v", link.ID, err)
	}
	if len(files) == 0 {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		http.Redirect(w, r, link.TargetURL, http.StatusFound)
		return
	}
	pageFiles := make([]shortLinkPageFile, 0, len(files))
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
		extension := strings.TrimPrefix(strings.ToUpper(filepath.Ext(file.OriginalName)), ".")
		if extension == "" {
			extension = "ARQUIVO"
		}
		pageFiles = append(pageFiles, shortLinkPageFile{
			Name: file.OriginalName,
			Size: humanFileSize(file.Size),
			Type: extension,
			URL:  fmt.Sprintf("%s/files/%d", s.shortLinkURL(link.Code), file.ID),
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = shortLinkPublicTemplate.Execute(w, map[string]any{
		"Title": title, "Code": link.Code, "Description": derefString(link.Description),
		"Target": link.TargetURL, "TargetHost": shortLinkTargetHost(link.TargetURL),
		"Files": pageFiles, "FileCount": len(pageFiles), "TotalSize": humanFileSize(totalSize),
		"Expires": expires,
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
		shortLinkUnavailable(w)
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
