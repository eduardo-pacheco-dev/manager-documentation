package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type iepmsProjectPayload struct {
	Name string `json:"name"`
}

type iepmsUserProjectPayload struct {
	ProjectID int64   `json:"project_id"`
	ExpiresAt *string `json:"expires_at"`
}

type iepmsUserProjectsPayload struct {
	Projects []iepmsUserProjectPayload `json:"projects"`
}

func (s *Server) iepmsProjectsIndex(w http.ResponseWriter, r *http.Request) {
	projects, err := s.Modules.ListIepmsProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os projetos do iEPMS.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": projects})
}

func (s *Server) iepmsProjectsCreate(w http.ResponseWriter, r *http.Request) {
	var payload iepmsProjectPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	name := strings.TrimSpace(payload.Name)
	fields := validationErrors{}
	if name == "" {
		fields.add("name", "Informe o nome do projeto.")
	}
	if len([]rune(name)) > 255 {
		fields.add("name", "O nome deve ter no máximo 255 caracteres.")
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	project, err := s.Modules.CreateIepmsProject(name)
	if err != nil {
		if isDuplicateKey(err) {
			writeValidation(w, validationErrors{"name": "Já existe um projeto com este nome."})
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível cadastrar o projeto.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Projeto cadastrado com sucesso.",
		"data":    project,
	})
}

func (s *Server) iepmsProjectsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteIepmsProject(id); err != nil {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Projeto removido com sucesso."})
}

func (s *Server) iepmsUserProjectsShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	if _, err := s.Modules.FindIepmsUser(id); err != nil {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	projects, err := s.Modules.ListIepmsUserProjects(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os projetos do usuário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"projects": projects}})
}

func (s *Server) iepmsUserProjectsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	if _, err := s.Modules.FindIepmsUser(id); err != nil {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}

	var payload iepmsUserProjectsPayload
	if !decodeJSON(w, r, &payload) {
		return
	}

	available, err := s.Modules.ListIepmsProjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os projetos do iEPMS.", nil)
		return
	}
	known := make(map[int64]bool, len(available))
	for _, project := range available {
		known[project.ID] = true
	}

	fields := validationErrors{}
	seen := make(map[int64]bool, len(payload.Projects))
	items := make([]modules.IepmsUserProjectInput, 0, len(payload.Projects))
	for _, entry := range payload.Projects {
		if entry.ProjectID <= 0 || !known[entry.ProjectID] {
			fields.add("projects", "Selecione apenas projetos cadastrados.")
			continue
		}
		if seen[entry.ProjectID] {
			fields.add("projects", "Cada projeto pode aparecer apenas uma vez no checklist.")
			continue
		}
		seen[entry.ProjectID] = true

		var expires *string
		if entry.ExpiresAt != nil && strings.TrimSpace(*entry.ExpiresAt) != "" {
			value := strings.TrimSpace(*entry.ExpiresAt)
			if _, err := time.Parse("2006-01-02", value); err != nil {
				fields.add("expires_at", "Informe a validade no formato AAAA-MM-DD.")
				continue
			}
			expires = &value
		}
		items = append(items, modules.IepmsUserProjectInput{ProjectID: entry.ProjectID, ExpiresAt: expires})
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}

	if err := s.Modules.ReplaceIepmsUserProjects(id, items); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar o checklist de projetos.", nil)
		return
	}

	projects, err := s.Modules.ListIepmsUserProjects(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os projetos do usuário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Checklist de projetos atualizado com sucesso.",
		"data":    map[string]any{"projects": projects},
	})
}
