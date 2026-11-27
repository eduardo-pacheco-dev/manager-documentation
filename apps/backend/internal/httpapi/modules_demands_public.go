package httpapi

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/modules"
)

type demandPublicFile struct {
	Name string `json:"name"`
	Size string `json:"size"`
	URL  string `json:"url"`
}

type demandPublicPageData struct {
	Title       string             `json:"title"`
	Heading     string             `json:"heading"`
	Intro       string             `json:"intro"`
	Description string             `json:"description"`
	Scope       string             `json:"scope"`
	DueDate     string             `json:"due_date"`
	ClientName  string             `json:"client_name"`
	IsUpload    bool               `json:"is_upload"`
	Files       []demandPublicFile `json:"files"`
	Uploaded    int                `json:"uploaded"`
	Error       string             `json:"error"`
}

func (s *Server) demandPublicInfo(w http.ResponseWriter, r *http.Request, token string, upload bool) {
	column := "download_token"
	if upload {
		column = "upload_token"
	}
	var ok bool
	s, r, ok = s.usePublicTenant(w, r, "demands", column, token)
	if !ok {
		return
	}
	var demand modules.Demand
	var err error
	if upload {
		demand, err = s.Modules.FindDemandByUploadToken(token)
	} else {
		demand, err = s.Modules.FindDemandByDownloadToken(token)
	}
	if err != nil || (upload && demand.Status == "cancelada") {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
		return
	}
	data := demandPublicPageData{
		Title: demand.Title, Heading: demand.Title,
		Intro:       "Você foi acionado(a) para enviar arquivos desta demanda.",
		Description: derefString(demand.Description), Scope: derefString(demand.ActivityScope),
		ClientName: derefString(demand.ClientName), IsUpload: upload,
		Files: []demandPublicFile{},
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
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) demandPublicUpload(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var ok bool
	s, r, ok = s.usePublicTenant(w, r, "demands", "upload_token", token)
	if !ok {
		return
	}
	demand, err := s.Modules.FindDemandByUploadToken(token)
	if err != nil || demand.Status == "cancelada" {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	if err := r.ParseMultipartForm(60 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido ou maior que o permitido.", nil)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "Selecione ao menos um arquivo.", nil)
		return
	}
	if len(files) > 10 {
		writeError(w, http.StatusBadRequest, "Envie no máximo 10 arquivos por vez.", nil)
		return
	}
	if len(demand.Attachments)+len(files) > 20 {
		writeError(w, http.StatusBadRequest, "Cada demanda aceita no máximo 20 arquivos.", nil)
		return
	}
	uploaderName := truncateString(r.FormValue("name"), 255)
	disk := s.Uploads.Provider()
	created := 0
	for _, header := range files {
		if header.Size > 50<<20 {
			writeError(w, http.StatusBadRequest, "Cada arquivo deve ter no máximo 50 MB.", nil)
			return
		}
		file, err := header.Open()
		if err != nil {
			continue
		}
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "demands")
		file.Close()
		if err != nil {
			writeError(w, http.StatusBadRequest, "Formato não permitido: "+header.Filename, nil)
			return
		}
		if _, err := s.Modules.CreateDemandAttachment(demand.ID, 0, header.Filename, result.Path, result.URL, disk,
			header.Header.Get("Content-Type"), header.Size, uploaderName); err == nil {
			created++
		}
	}
	if created == 0 {
		writeError(w, http.StatusBadRequest, "Não foi possível enviar os arquivos.", nil)
		return
	}
	author := uploaderName
	if author == "" {
		author = "Link público"
	}
	s.Modules.AddDemandSystemUpdate(demand.ID, fmt.Sprintf("%d arquivo(s) enviado(s) pelo link público por %s.", created, author))
	s.notifyDemandParticipants(0, demand, "Novo arquivo enviado pelo colaborador",
		fmt.Sprintf("%s · %d arquivo(s)", demand.Title, created), "info")
	writeJSON(w, http.StatusOK, map[string]any{"uploaded": created})
}

func (s *Server) demandPublicDownloadFile(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var ok bool
	s, r, ok = s.usePublicTenant(w, r, "demands", "download_token", token)
	if !ok {
		return
	}
	fileID, ok := parseStringID(chi.URLParam(r, "fileId"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	demand, err := s.Modules.FindDemandByDownloadToken(token)
	if err != nil {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
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
