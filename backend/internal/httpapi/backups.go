package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/backup"
)

// maxImportBytes limits the size of an uploaded backup archive (4 GiB).
const maxImportBytes = int64(4) << 30

// backupsIndex lists the available backups and the active configuration.
func (s *Server) backupsIndex(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Backups.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível listar os backups.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     entries,
		"settings": s.Backups.Settings(),
	})
}

// backupsCreate generates a new backup synchronously.
func (s *Server) backupsCreate(w http.ResponseWriter, r *http.Request) {
	extendWriteDeadline(w, 30*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	entry, err := s.Backups.Create(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o backup: "+err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Backup criado com sucesso.",
		"data":    entry,
	})
}

// backupsImport stores an uploaded backup archive. When the "restore" form
// field is true, the archive is restored right after being imported.
func (s *Server) backupsImport(w http.ResponseWriter, r *http.Request) {
	extendWriteDeadline(w, 2*time.Hour)
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Não foi possível ler o arquivo enviado.", nil)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione um arquivo .zip de backup."})
		return
	}
	defer file.Close()

	entry, err := s.Backups.Import(r.Context(), file, header.Filename)
	if err != nil {
		var limit *http.MaxBytesError
		switch {
		case errors.Is(err, backup.ErrInvalidArchive):
			writeValidation(w, validationErrors{"file": "Arquivo inválido: " + err.Error()})
		case errors.As(err, &limit):
			writeError(w, http.StatusRequestEntityTooLarge, "O arquivo excede o limite de 4 GB.", nil)
		default:
			writeError(w, http.StatusInternalServerError, "Não foi possível importar o backup: "+err.Error(), nil)
		}
		return
	}

	restore := strings.EqualFold(strings.TrimSpace(r.FormValue("restore")), "true")
	if restore {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if err := s.Backups.Restore(ctx, entry.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "Backup importado, mas a restauração falhou: "+err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"message": "Backup importado e restaurado com sucesso.",
			"data":    entry,
		})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Backup importado com sucesso.",
		"data":    entry,
	})
}

// backupsDownload streams a backup archive to the administrator.
func (s *Server) backupsDownload(w http.ResponseWriter, r *http.Request) {
	path, err := s.Backups.Path(chi.URLParam(r, "name"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Backup não encontrado.", nil)
		return
	}
	extendWriteDeadline(w, 30*time.Minute)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(path)))
	http.ServeFile(w, r, path)
}

// backupsDelete removes a backup file.
func (s *Server) backupsDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.Backups.Delete(name); err != nil {
		if errors.Is(err, backup.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Backup não encontrado.", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível remover o backup.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Backup removido com sucesso."})
}

// backupsRestore restores the database and local uploads from a backup. The
// caller must confirm the backup name because the operation is destructive.
func (s *Server) backupsRestore(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var payload struct {
		Confirm string `json:"confirm"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Confirm) != name {
		writeValidation(w, validationErrors{"confirm": "Digite o nome do backup para confirmar a restauração."})
		return
	}
	extendWriteDeadline(w, 30*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := s.Backups.Restore(ctx, name); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível restaurar o backup: "+err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Backup restaurado com sucesso."})
}

// extendWriteDeadline disables the global HTTP write timeout for long-running
// operations such as dumping or restoring a large database.
func extendWriteDeadline(w http.ResponseWriter, deadline time.Duration) {
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(deadline))
}
