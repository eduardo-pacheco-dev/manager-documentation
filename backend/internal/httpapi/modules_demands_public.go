package httpapi

import (
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type demandPublicFile struct {
	Name string
	Size string
	URL  string
}

type demandPublicPageData struct {
	Title       string
	Heading     string
	Intro       string
	Description string
	Scope       string
	DueDate     string
	ClientName  string
	IsUpload    bool
	Files       []demandPublicFile
	Uploaded    int
	Error       string
}

var demandPublicTemplate = template.Must(template.New("demand-public").Parse(`<!doctype html>
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
:root{--bg:#f4f4f4;--layer:#fff;--text:#161616;--muted:#525252;--helper:#6f6f6f;--border:#e0e0e0;--blue:#0f62fe;--blue-hover:#0043ce;--danger:#da1e28;--success:#0e6027;--success-bg:#defbe6}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--text);font-family:"IBM Plex Sans","Segoe UI",system-ui,sans-serif;line-height:1.5;-webkit-font-smoothing:antialiased}
.cds-header{display:flex;align-items:center;justify-content:space-between;height:3rem;padding:0 1rem;background:#161616;color:#fff}
.brand{display:flex;align-items:center;gap:.75rem;font-size:.875rem;font-weight:600}
.mark{display:flex;align-items:center;justify-content:center;width:1.5rem;height:1.5rem;background:#fff;color:#161616;font-size:.75rem;font-weight:600}
.header-meta{color:#c6c6c6;font-size:.75rem}
.shell{max-width:66rem;margin:0 auto;padding:2rem 1rem 3rem}
.eyebrow{margin:0 0 .35rem;color:var(--muted);font-size:.75rem;font-weight:500;letter-spacing:.08em;text-transform:uppercase}
h1{margin:0 0 .5rem;font-size:clamp(1.75rem,4vw,2.5rem);font-weight:400;line-height:1.15}
h2{margin:0;font-size:1.2rem;font-weight:400}
.lead{max-width:48rem;margin:0 0 1rem;color:var(--muted)}
.tags{display:flex;flex-wrap:wrap;gap:.5rem;margin:0 0 1.5rem}
.tag{display:inline-flex;align-items:center;min-height:1.5rem;padding:.15rem .55rem;border-radius:999px;background:#e0e0e0;color:#161616;font-size:.75rem}
.tag--blue{background:#d0e2ff;color:#002d9c}
.tag--green{background:#defbe6;color:#0e6027}
.panel{background:var(--layer);border:1px solid var(--border);padding:2rem;margin-bottom:1rem}
.panel h2{font-weight:400}
.panel .hint{margin:.25rem 0 1.25rem;color:var(--helper);font-size:.8rem}
.alert{padding:1rem;margin-bottom:1rem;border-left:3px solid var(--success);background:var(--success-bg);color:var(--success);font-size:.85rem}
.alert--error{border-color:var(--danger);background:#fff1f1;color:#a2191f}
.description{padding:.85rem;border-left:3px solid var(--blue);background:#f4f4f4;color:var(--muted);font-size:.85rem;white-space:pre-wrap}
.meta{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.75rem;margin-top:1rem}
.meta div{padding:.6rem;background:#f4f4f4}
.meta span{display:block;color:var(--helper);font-size:.7rem}
.meta strong{font-size:.82rem;font-weight:500}
form{display:flex;flex-direction:column;gap:1rem}
label{display:flex;flex-direction:column;gap:.35rem;color:var(--muted);font-size:.75rem}
input[type=text],input[type=file]{width:100%;padding:.75rem;border:0;border-bottom:1px solid #8d8d8d;background:#f4f4f4;color:var(--text);font-family:inherit;font-size:.85rem}
input[type=text]:focus,input[type=file]:focus{outline:2px solid var(--blue);outline-offset:-2px}
.btn{display:inline-flex;align-items:center;justify-content:space-between;gap:2rem;min-height:3rem;padding:0 1rem;border:1px solid transparent;background:var(--blue);color:#fff;font-family:inherit;font-size:.875rem;cursor:pointer;text-decoration:none}
.btn:hover{background:var(--blue-hover)}
.file-list{list-style:none;margin:0;padding:0;border-top:1px solid var(--border)}
.file-row{display:flex;align-items:center;gap:1rem;padding:1rem 0;border-bottom:1px solid var(--border)}
.file-icon{display:flex;align-items:center;justify-content:center;width:2.5rem;height:2.5rem;flex-shrink:0;background:#edf5ff;color:#0043ce}
.file-info{display:flex;flex-direction:column;min-width:0;flex:1}
.file-info strong{font-size:.875rem;font-weight:500;overflow-wrap:anywhere}
.file-info span{color:var(--helper);font-size:.75rem}
.download{display:inline-flex;align-items:center;min-height:2.5rem;padding:0 .9rem;border:1px solid var(--blue);color:var(--blue);font-size:.8rem;text-decoration:none;white-space:nowrap}
.download:hover{background:var(--blue);color:#fff}
.empty{color:var(--muted);font-size:.85rem}
.footer{max-width:66rem;margin:0 auto;padding:0 1rem 2rem;color:var(--helper);font-size:.72rem;display:flex;flex-wrap:wrap;gap:.5rem;justify-content:space-between}
@media (max-width:42rem){.shell{padding-top:1rem}.panel{padding:1.25rem}.meta{grid-template-columns:1fr}.header-meta{display:none}.file-row{flex-wrap:wrap}.download{margin-left:3.5rem}}
</style>
</head>
<body>
<header class="cds-header">
  <div class="brand"><span class="mark">DC</span> Documentation Center</div>
  <span class="header-meta">{{if .IsUpload}}Envio de arquivos{{else}}Download de arquivos{{end}}</span>
</header>
<main class="shell">
  <p class="eyebrow">Demanda</p>
  <h1>{{.Heading}}</h1>
  <p class="lead">{{.Intro}}</p>
  <div class="tags">
    {{if .DueDate}}<span class="tag tag--blue">Prazo {{.DueDate}}</span>{{end}}
    {{if .ClientName}}<span class="tag">Cliente: {{.ClientName}}</span>{{end}}
    {{if .IsUpload}}<span class="tag tag--green">Link de envio</span>{{else}}<span class="tag tag--blue">Link de download</span>{{end}}
  </div>

  {{if .Uploaded}}<div class="alert">{{.Uploaded}} arquivo(s) enviado(s) com sucesso. O responsável pela demanda foi notificado.</div>{{end}}
  {{if .Error}}<div class="alert alert--error">{{.Error}}</div>{{end}}

  {{if .Description}}
  <section class="panel">
    <h2>Descrição</h2>
    <p class="hint">Resumo da demanda.</p>
    <div class="description">{{.Description}}</div>
    <div class="meta">
      {{if .DueDate}}<div><span>Prazo</span><strong>{{.DueDate}}</strong></div>{{end}}
      {{if .ClientName}}<div><span>Cliente</span><strong>{{.ClientName}}</strong></div>{{end}}
    </div>
  </section>
  {{end}}

  {{if .Scope}}
  <section class="panel">
    <h2>Escopo da atividade</h2>
    <p class="hint">Detalhamento operacional solicitado.</p>
    <div class="description">{{.Scope}}</div>
  </section>
  {{end}}

  {{if .IsUpload}}
  <section class="panel">
    <h2>Enviar arquivos</h2>
    <p class="hint">Envie documentos, evidências, fotos ou planilhas relacionados à demanda. Até 20 arquivos por demanda, com no máximo 50 MB cada.</p>
    <form method="post" enctype="multipart/form-data">
      <label>Seu nome (opcional)
        <input type="text" name="name" maxlength="255" placeholder="Nome do colaborador">
      </label>
      <label>Arquivos
        <input type="file" name="files" multiple required>
      </label>
      <button class="btn" type="submit">Enviar arquivos</button>
    </form>
  </section>
  {{else}}
  <section class="panel">
    <h2>Arquivos para download</h2>
    <p class="hint">Baixe os arquivos disponibilizados nesta demanda.</p>
    {{if .Files}}
    <ul class="file-list">{{range .Files}}
      <li class="file-row">
        <span class="file-icon" aria-hidden="true"><svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M5 2.5h6l4 4v11H5v-15Z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><path d="M11 2.5v4h4" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/></svg></span>
        <div class="file-info"><strong>{{.Name}}</strong><span>{{.Size}}</span></div>
        <a class="download" href="{{.URL}}" download>Baixar</a>
      </li>
    {{end}}</ul>
    {{else}}<p class="empty">Nenhum arquivo disponível para download.</p>{{end}}
  </section>
  {{end}}

  <p class="empty">Este link é exclusivo para esta demanda. Não o compartilhe com pessoas não envolvidas.</p>
</main>
<footer class="footer"><span>Documentation Center · Demanda</span><span>Powered by IBM Carbon Design System</span></footer>
</body>
</html>`))

func demandUnavailablePage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusGone)
	_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><meta charset="utf-8"><title>Link indisponível</title>
<body style="font-family:system-ui,sans-serif;background:#f4f4f4;color:#161616;padding:3rem 1rem">
<main style="max-width:640px;margin:0 auto;background:#fff;border:1px solid #e0e0e0;padding:2rem">
<h1 style="font-weight:400">Link indisponível</h1><p style="color:#525252">Este link expirou, foi desativado ou não existe.</p></main></body></html>`))
}

func (s *Server) demandPublicPage(w http.ResponseWriter, r *http.Request, token string, upload bool) {
	var demand modules.Demand
	var err error
	if upload {
		demand, err = s.Modules.FindDemandByUploadToken(token)
	} else {
		demand, err = s.Modules.FindDemandByDownloadToken(token)
	}
	if err != nil || (upload && demand.Status == "cancelada") {
		demandUnavailablePage(w)
		return
	}
	title := demand.Title
	client := derefString(demand.ClientName)
	data := demandPublicPageData{
		Title: title, Heading: title,
		Intro:       "Você foi acionado(a) para enviar arquivos desta demanda.",
		Description: derefString(demand.Description), Scope: derefString(demand.ActivityScope),
		ClientName: client, IsUpload: upload,
	}
	if !upload {
		data.Intro = "Os arquivos desta demanda estão disponíveis para download."
	}
	if demand.DueAt != nil {
		if parsed, err := time.Parse("2006-01-02", *demand.DueAt); err == nil {
			data.DueDate = parsed.Format("02/01/2006")
		} else {
			data.DueDate = *demand.DueAt
		}
	}
	if upload {
		data.Intro = "Utilize o formulário abaixo para enviar os arquivos solicitados pelo responsável da demanda."
	} else {
		for _, file := range demand.Attachments {
			data.Files = append(data.Files, demandPublicFile{
				Name: file.OriginalName,
				Size: humanFileSize(file.Size),
				URL:  fmt.Sprintf("/d/%s/files/%d", token, file.ID),
			})
		}
	}
	if uploaded := r.URL.Query().Get("uploaded"); uploaded != "" {
		data.Uploaded, _ = strconv.Atoi(uploaded)
	}
	if message := r.URL.Query().Get("error"); message != "" {
		data.Error = message
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = demandPublicTemplate.Execute(w, data)
}

func (s *Server) demandPublicUploadPage(w http.ResponseWriter, r *http.Request) {
	s.demandPublicPage(w, r, chi.URLParam(r, "token"), true)
}

func (s *Server) demandPublicDownloadPage(w http.ResponseWriter, r *http.Request) {
	s.demandPublicPage(w, r, chi.URLParam(r, "token"), false)
}

func (s *Server) demandPublicUpload(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	demand, err := s.Modules.FindDemandByUploadToken(token)
	if err != nil || demand.Status == "cancelada" {
		demandUnavailablePage(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	if err := r.ParseMultipartForm(60 << 20); err != nil {
		http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Arquivo inválido ou maior que o permitido."), http.StatusSeeOther)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Selecione ao menos um arquivo."), http.StatusSeeOther)
		return
	}
	if len(files) > 10 {
		http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Envie no máximo 10 arquivos por vez."), http.StatusSeeOther)
		return
	}
	if len(demand.Attachments)+len(files) > 20 {
		http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Cada demanda aceita no máximo 20 arquivos."), http.StatusSeeOther)
		return
	}
	uploaderName := truncateString(r.FormValue("name"), 255)
	disk := s.Uploads.Provider()
	created := 0
	for _, header := range files {
		if header.Size > 50<<20 {
			http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Cada arquivo deve ter no máximo 50 MB."), http.StatusSeeOther)
			return
		}
		file, err := header.Open()
		if err != nil {
			continue
		}
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "demands")
		file.Close()
		if err != nil {
			http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Formato não permitido: "+header.Filename), http.StatusSeeOther)
			return
		}
		if _, err := s.Modules.CreateDemandAttachment(demand.ID, 0, header.Filename, result.Path, result.URL, disk,
			header.Header.Get("Content-Type"), header.Size, uploaderName); err == nil {
			created++
		}
	}
	if created == 0 {
		http.Redirect(w, r, "/u/"+token+"?error="+url.QueryEscape("Não foi possível enviar os arquivos."), http.StatusSeeOther)
		return
	}
	author := uploaderName
	if author == "" {
		author = "Link público"
	}
	s.Modules.AddDemandSystemUpdate(demand.ID, fmt.Sprintf("%d arquivo(s) enviado(s) pelo link público por %s.", created, author))
	s.notifyDemandParticipants(0, demand, "Novo arquivo enviado pelo colaborador",
		fmt.Sprintf("%s · %d arquivo(s)", demand.Title, created), "info")
	http.Redirect(w, r, fmt.Sprintf("/u/%s?uploaded=%d", token, created), http.StatusSeeOther)
}

func (s *Server) demandPublicDownloadFile(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	fileID, ok := parseStringID(chi.URLParam(r, "fileId"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	demand, err := s.Modules.FindDemandByDownloadToken(token)
	if err != nil {
		demandUnavailablePage(w)
		return
	}
	attachment, err := s.Modules.FindDemandAttachment(demand.ID, fileID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if attachment.Disk == "local" {
		target := filepath.Join(s.Config.UploadDir, filepath.FromSlash(attachment.Path))
		if _, err := os.Stat(target); err != nil {
			http.NotFound(w, r)
			return
		}
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.OriginalName})
		w.Header().Set("Content-Disposition", disposition)
		if attachment.MimeType != nil && strings.TrimSpace(*attachment.MimeType) != "" {
			w.Header().Set("Content-Type", *attachment.MimeType)
		}
		http.ServeFile(w, r, target)
		return
	}
	http.Redirect(w, r, attachment.URL, http.StatusFound)
}
