package httpapi

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type demandPayload struct {
	AssigneeID  int64  `json:"assignee_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	DueAt       string `json:"due_at"`
}

func (p demandPayload) input() modules.DemandInput {
	return modules.DemandInput{
		AssigneeID: p.AssigneeID, Title: p.Title, Description: p.Description,
		Status: p.Status, Priority: p.Priority, DueAt: p.DueAt,
	}
}

func validateDemand(payload demandPayload) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.Title) == "" {
		fields["title"] = "Informe o título da demanda."
	} else if len([]rune(payload.Title)) > 255 {
		fields["title"] = "O título deve ter no máximo 255 caracteres."
	}
	if payload.AssigneeID <= 0 {
		fields["assignee_id"] = "Selecione o colaborador responsável."
	}
	if payload.Priority != "" && !validDemandPriority(payload.Priority) {
		fields["priority"] = "Prioridade inválida."
	}
	if payload.Status != "" && !validDemandStatus(payload.Status) {
		fields["status"] = "Status inválido."
	}
	if strings.TrimSpace(payload.DueAt) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(payload.DueAt)); err != nil {
			fields["due_at"] = "Informe uma data válida."
		}
	}
	if len([]rune(payload.Description)) > 10000 {
		fields["description"] = "A descrição deve ter no máximo 10.000 caracteres."
	}
	return fields
}

func validDemandStatus(value string) bool {
	for _, status := range modules.DemandStatuses {
		if status == value {
			return true
		}
	}
	return false
}

func validDemandPriority(value string) bool {
	for _, priority := range modules.DemandPriorities {
		if priority == value {
			return true
		}
	}
	return false
}

func (s *Server) demandAssigneeExists(id int64) bool {
	var count int
	if err := s.Modules.DB.Get(&count, `SELECT COUNT(*) FROM users WHERE id = ? AND is_active = 1`, id); err != nil {
		return false
	}
	return count > 0
}

func canManageDemand(admin bool, userID int64, demand modules.Demand) bool {
	return admin || demand.RequesterID == userID
}

func canParticipateDemand(admin bool, userID int64, demand modules.Demand) bool {
	return admin || demand.RequesterID == userID || demand.AssigneeID == userID
}

func (s *Server) demandsIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	assigneeID, _ := strconv.ParseInt(strings.TrimSpace(query.Get("assignee_id")), 10, 64)
	user := currentUser(r)
	items, total, stats, err := s.Modules.ListDemands(modules.DemandFilter{
		Search: strings.TrimSpace(query.Get("search")), Status: strings.TrimSpace(query.Get("status")),
		Priority: strings.TrimSpace(query.Get("priority")), AssigneeID: assigneeID,
		Scope: strings.TrimSpace(query.Get("scope")), Page: page, PerPage: perPage, UserID: user.ID,
	})
	if err != nil {
		log.Printf("demands list: %v", err)
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as demandas.", nil)
		return
	}
	collaborators, _ := s.Modules.DemandCollaborators()
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items, "stats": stats, "collaborators": collaborators,
		"meta": paginationMeta(page, perPage, total),
		"filters": map[string]any{
			"search": strings.TrimSpace(query.Get("search")), "status": strings.TrimSpace(query.Get("status")),
			"priority": strings.TrimSpace(query.Get("priority")), "assignee_id": assigneeID,
			"scope": strings.TrimSpace(query.Get("scope")),
		},
	})
}

func (s *Server) demandsCreate(w http.ResponseWriter, r *http.Request) {
	var payload demandPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateDemand(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if !s.demandAssigneeExists(payload.AssigneeID) {
		writeValidation(w, validationErrors{"assignee_id": "Colaborador não encontrado ou inativo."})
		return
	}
	if payload.Status == "" {
		payload.Status = "aberta"
	}
	if payload.Priority == "" {
		payload.Priority = "media"
	}
	demand, err := s.Modules.CreateDemand(currentUser(r).ID, payload.input())
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar a demanda.", nil)
		return
	}
	s.notifyDemandParticipants(currentUser(r).ID, demand, "Nova demanda atribuída a você",
		demand.Title+" · Prazo "+derefString(demand.DueAt), "info")
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Demanda criada e encaminhada.", "data": map[string]any{"demand": demand}})
}

func (s *Server) demandsShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	s.presentDemand(&demand)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"demand": demand}})
}

func (s *Server) demandsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canManageDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Apenas o solicitante ou administrador pode editar esta demanda.", nil)
		return
	}
	var payload demandPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateDemand(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if !s.demandAssigneeExists(payload.AssigneeID) {
		writeValidation(w, validationErrors{"assignee_id": "Colaborador não encontrado ou inativo."})
		return
	}
	previousAssignee := demand.AssigneeID
	previousStatus := demand.Status
	updated, err := s.Modules.UpdateDemand(id, user.ID, payload.input())
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível atualizar a demanda.", nil)
		return
	}
	if updated.AssigneeID != previousAssignee || updated.Status != previousStatus {
		s.notifyDemandParticipants(user.ID, updated, "Demanda atualizada", updated.Title, "info")
	}
	s.presentDemand(&updated)
	writeJSON(w, http.StatusOK, map[string]any{"message": "Demanda atualizada com sucesso.", "data": map[string]any{"demand": updated}})
}

func (s *Server) demandsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canManageDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Apenas o solicitante ou administrador pode excluir esta demanda.", nil)
		return
	}
	paths, err := s.Modules.DeleteDemand(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir a demanda.", nil)
		return
	}
	for _, path := range paths {
		s.Uploads.DeleteLocal(path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Demanda excluída com sucesso."})
}

func (s *Server) demandsStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canParticipateDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Você não participa desta demanda.", nil)
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !validDemandStatus(payload.Status) {
		writeValidation(w, validationErrors{"status": "Status inválido."})
		return
	}
	updated, err := s.Modules.SetDemandStatus(id, user.ID, payload.Status)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível alterar o status.", nil)
		return
	}
	s.notifyDemandParticipants(user.ID, updated, "Status da demanda atualizado",
		updated.Title+" · "+modules.DemandStatusLabel(updated.Status), "info")
	s.presentDemand(&updated)
	writeJSON(w, http.StatusOK, map[string]any{"message": "Status atualizado.", "data": map[string]any{"demand": updated}})
}

func (s *Server) demandsAddUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canParticipateDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Você não participa desta demanda.", nil)
		return
	}
	var payload struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len([]rune(strings.TrimSpace(payload.Body))) < 2 {
		writeValidation(w, validationErrors{"body": "Escreva ao menos 2 caracteres."})
		return
	}
	update, err := s.Modules.AddDemandUpdate(id, user.ID, payload.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível publicar o comentário.", nil)
		return
	}
	s.notifyDemandParticipants(user.ID, demand, "Nova atualização na demanda",
		demand.Title+" · "+truncateString(payload.Body, 120), "info")
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Comentário publicado.", "data": map[string]any{"update": update}})
}

func (s *Server) demandsCollaborators(w http.ResponseWriter, r *http.Request) {
	items, err := s.Modules.DemandCollaborators()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os colaboradores.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) demandsUploadAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canParticipateDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Você não participa desta demanda.", nil)
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
	if len(demand.Attachments)+len(files) > 20 {
		writeValidation(w, validationErrors{"files": "Cada demanda aceita no máximo 20 arquivos."})
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
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "demands")
		file.Close()
		if err != nil {
			writeValidation(w, validationErrors{"files": fmt.Sprintf("Formato não permitido: %s", header.Filename)})
			return
		}
		if _, err := s.Modules.CreateDemandAttachment(id, user.ID, header.Filename, result.Path, result.URL, disk,
			header.Header.Get("Content-Type"), header.Size); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": fmt.Sprintf("%d arquivo(s) anexado(s).", created)})
}

func (s *Server) demandsDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	attachmentID, ok := parseID(r, "attachmentId")
	if !ok {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canParticipateDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Você não participa desta demanda.", nil)
		return
	}
	attachment, err := s.Modules.FindDemandAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	if attachment.Disk == "local" {
		http.Redirect(w, r, "/uploads/"+attachment.Path, http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, attachment.URL, http.StatusTemporaryRedirect)
}

func (s *Server) demandsDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	attachmentID, ok := parseID(r, "attachmentId")
	if !ok {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	demand, err := s.Modules.FindDemand(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Demanda não encontrada.", nil)
		return
	}
	user := currentUser(r)
	if !canParticipateDemand(user.IsAdmin, user.ID, demand) {
		writeError(w, http.StatusForbidden, "Você não participa desta demanda.", nil)
		return
	}
	attachment, err := s.Modules.DeleteDemandAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Arquivo não encontrado.", nil)
		return
	}
	if attachment.Disk == "local" {
		s.Uploads.DeleteLocal(attachment.Path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Arquivo removido."})
}

func (s *Server) demandsExport(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	assigneeID, _ := strconv.ParseInt(strings.TrimSpace(query.Get("assignee_id")), 10, 64)
	items, _, _, err := s.Modules.ListDemands(modules.DemandFilter{
		Search: strings.TrimSpace(query.Get("search")), Status: strings.TrimSpace(query.Get("status")),
		Priority: strings.TrimSpace(query.Get("priority")), AssigneeID: assigneeID,
		Scope: strings.TrimSpace(query.Get("scope")), Page: 1, PerPage: 10000, UserID: currentUser(r).ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar as demandas.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="demandas-%s.csv"`, time.Now().Format("2006-01-02")))
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"ID", "Título", "Solicitante", "Responsável", "Status", "Prioridade", "Prazo", "Atrasada", "Criada em", "Concluída em"})
	for _, item := range items {
		overdue := "Não"
		if item.Overdue {
			overdue = "Sim"
		}
		completed := ""
		if item.CompletedAt != nil {
			completed = item.CompletedAt.Format("02/01/2006 15:04")
		}
		_ = writer.Write([]string{
			strconv.FormatInt(item.ID, 10), item.Title, derefString(item.RequesterName), derefString(item.AssigneeName),
			modules.DemandStatusLabel(item.Status), item.Priority, derefString(item.DueAt), overdue,
			item.CreatedAt.Format("02/01/2006 15:04"), completed,
		})
	}
	writer.Flush()
}

func (s *Server) presentDemand(demand *modules.Demand) {
	demand.Overdue = demand.DueAt != nil && demand.Status != "concluida" && demand.Status != "cancelada" &&
		*demand.DueAt < time.Now().Format("2006-01-02")
	for i := range demand.Attachments {
		demand.Attachments[i].DownloadURL = fmt.Sprintf("/api/v1/demands/%d/attachments/%d", demand.ID, demand.Attachments[i].ID)
	}
}

func (s *Server) notifyDemandParticipants(actorID int64, demand modules.Demand, title, body, level string) {
	link := fmt.Sprintf("/demands/%d", demand.ID)
	for _, userID := range []int64{demand.AssigneeID, demand.RequesterID} {
		if userID <= 0 || userID == actorID {
			continue
		}
		user, err := s.Store.FindUserByID(userID)
		if err != nil || !user.IsActive {
			continue
		}
		if err := s.Notifier.Notify(user, title, body, link, level); err != nil {
			log.Printf("demand notification %d: %v", userID, err)
		}
	}
}
