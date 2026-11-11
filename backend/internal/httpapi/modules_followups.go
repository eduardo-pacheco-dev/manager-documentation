package httpapi

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

var followUpStatuses = map[string]bool{"pendente": true, "em_andamento": true, "concluido": true, "cancelado": true}
var followUpPriorities = map[string]bool{"baixa": true, "media": true, "alta": true, "urgente": true}

func (s *Server) followUpsIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	settings, _ := s.Modules.FollowUpSettings()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), settings.PerPage)
	if perPage <= 0 {
		perPage = 15
	}
	user := currentUser(r)
	filter := modules.FollowUpFilter{
		Search: query.Get("search"), Status: query.Get("status"), Priority: query.Get("priority"),
		Assigned: query.Get("assigned"), Sort: query.Get("sort"), Direction: query.Get("direction"),
		Page: page, PerPage: perPage, UserEmail: user.Email,
	}
	items, total, stats, err := s.Modules.ListFollowUps(filter)
	if err != nil {
		log.Printf("follow-ups list: %v", err)
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os follow-ups.", nil)
		return
	}
	clients, _ := s.Modules.FollowUpClients()
	assignees, _ := s.Modules.FollowUpAssignees()
	response := map[string]any{
		"data": items, "stats": stats, "clients": clients, "assignees": assignees,
		"meta": paginationMeta(page, perPage, total), "settings": settings,
	}
	if user.IsAdmin {
		response["settings"] = settings
	} else {
		response["settings"] = nil
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) followUpsShow(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	item, err := s.Modules.FindFollowUp(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Follow-up não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"follow_up": item}})
}

type followUpPayload struct {
	Title       string `json:"title"`
	Code        string `json:"code"`
	Description string `json:"description"`
	ClientID    *int64 `json:"client_id"`
	ProjectID   *int64 `json:"project_id"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	DueDate     string `json:"due_date"`
	CompletedAt string `json:"completed_at"`
	AssigneeID  *int64 `json:"assignee_id"`
}

func validateFollowUp(payload followUpPayload) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.Title) == "" {
		fields.add("title", "O título é obrigatório.")
	} else if len([]rune(payload.Title)) > 150 {
		fields.add("title", "O título deve ter no máximo 150 caracteres.")
	}
	if !followUpStatuses[payload.Status] {
		fields.add("status", "Selecione um status válido.")
	}
	if !followUpPriorities[payload.Priority] {
		fields.add("priority", "Selecione uma prioridade válida.")
	}
	if len([]rune(payload.Code)) > 50 {
		fields.add("code", "O código deve ter no máximo 50 caracteres.")
	}
	if len([]rune(payload.Description)) > 5000 {
		fields.add("description", "A descrição deve ter no máximo 5000 caracteres.")
	}
	return fields
}

func (s *Server) followUpsCreate(w http.ResponseWriter, r *http.Request) {
	var payload followUpPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateFollowUp(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if payload.ProjectID != nil {
		var projectClient int64
		if err := s.Modules.DB.Get(&projectClient, `SELECT client_id FROM follow_up_projects WHERE id = ?`, *payload.ProjectID); err != nil ||
			payload.ClientID == nil || *payload.ClientID != projectClient {
			writeValidation(w, validationErrors{"project_id": "O projeto selecionado não pertence ao cliente informado."})
			return
		}
	}
	item, err := s.Modules.CreateFollowUp(currentUser(r).ID, modules.FollowUpInput{
		Title: payload.Title, Code: payload.Code, Description: payload.Description, ClientID: payload.ClientID,
		ProjectID: payload.ProjectID, Status: payload.Status, Priority: payload.Priority, DueDate: payload.DueDate,
		CompletedAt: payload.CompletedAt, AssigneeID: payload.AssigneeID,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar o follow-up.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Follow-up criado com sucesso.", "data": map[string]any{"follow_up": item}})
}

func (s *Server) followUpsUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload followUpPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateFollowUp(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if payload.ProjectID != nil {
		var projectClient int64
		if err := s.Modules.DB.Get(&projectClient, `SELECT client_id FROM follow_up_projects WHERE id = ?`, *payload.ProjectID); err != nil ||
			payload.ClientID == nil || *payload.ClientID != projectClient {
			writeValidation(w, validationErrors{"project_id": "O projeto selecionado não pertence ao cliente informado."})
			return
		}
	}
	item, err := s.Modules.UpdateFollowUp(id, currentUser(r).ID, modules.FollowUpInput{
		Title: payload.Title, Code: payload.Code, Description: payload.Description, ClientID: payload.ClientID,
		ProjectID: payload.ProjectID, Status: payload.Status, Priority: payload.Priority, DueDate: payload.DueDate,
		CompletedAt: payload.CompletedAt, AssigneeID: payload.AssigneeID,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível atualizar o follow-up.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Follow-up atualizado com sucesso.", "data": map[string]any{"follow_up": item}})
}

func (s *Server) followUpsDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteFollowUp(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o follow-up.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Follow-up excluído com sucesso."})
}

func (s *Server) followUpsAddUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len([]rune(strings.TrimSpace(payload.Body))) < 2 {
		writeValidation(w, validationErrors{"body": "A atualização precisa ter pelo menos 2 caracteres."})
		return
	}
	update, err := s.Modules.AddFollowUpUpdate(id, currentUser(r).ID, payload.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível registrar a atualização.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Atualização registrada com sucesso.", "data": map[string]any{"update": update}})
}

func (s *Server) followUpsDeleteUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	updateID, _ := parseID(r, "updateId")
	update, err := s.Modules.FindFollowUpUpdate(id, updateID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Atualização não encontrada.", nil)
		return
	}
	auth := authFrom(r)
	if !auth.User.IsAdmin && (update.UserID == nil || *update.UserID != auth.User.ID) {
		writeError(w, http.StatusForbidden, "Você não pode remover esta atualização.", nil)
		return
	}
	if err := s.Modules.DeleteFollowUpUpdate(id, updateID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível remover a atualização.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Atualização removida com sucesso."})
}

func (s *Server) followUpsUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var payload modules.FollowUpSettings
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.PerPage < 5 || payload.PerPage > 100 {
		writeValidation(w, validationErrors{"per_page": "Use um valor entre 5 e 100."})
		return
	}
	if payload.DueSoonDays < 1 || payload.DueSoonDays > 60 {
		writeValidation(w, validationErrors{"due_soon_days": "Use um valor entre 1 e 60."})
		return
	}
	if payload.WidgetLimit < 1 || payload.WidgetLimit > 20 {
		writeValidation(w, validationErrors{"widget_limit": "Use um valor entre 1 e 20."})
		return
	}
	if err := s.Modules.SaveFollowUpSettings(payload); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar as configurações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Configurações atualizadas com sucesso."})
}

func (s *Server) followUpsCreateAssignee(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeValidation(w, validationErrors{"name": "Informe o nome do responsável."})
		return
	}
	assignee, err := s.Modules.CreateFollowUpAssignee(payload.Name, payload.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o responsável.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Responsável adicionado.", "data": map[string]any{"assignee": assignee}})
}

func (s *Server) followUpsDeleteAssignee(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteFollowUpAssignee(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível remover o responsável.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Responsável removido."})
}

func (s *Server) followUpsCreateClient(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ClientID    *int64 `json:"client_id"`
		ClientName  string `json:"client_name"`
		ProjectName string `json:"project_name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	var client modules.FollowUpClient
	var err error
	if payload.ClientID != nil {
		client.ID = *payload.ClientID
	} else {
		if strings.TrimSpace(payload.ClientName) == "" {
			writeValidation(w, validationErrors{"client_name": "Informe o nome do cliente."})
			return
		}
		client, err = s.Modules.CreateFollowUpClient(payload.ClientName)
		if err != nil {
			writeValidation(w, validationErrors{"client_name": "Já existe um cliente com este nome."})
			return
		}
	}
	if strings.TrimSpace(payload.ProjectName) != "" && client.ID > 0 {
		if _, err := s.Modules.CreateFollowUpProject(client.ID, payload.ProjectName); err != nil {
			writeValidation(w, validationErrors{"project_name": "Não foi possível criar o projeto."})
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Cliente/projeto cadastrado com sucesso."})
}

func (s *Server) followUpsUpdateClient(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		ClientName string `json:"client_name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.UpdateFollowUpClient(id, payload.ClientName); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o cliente.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Cliente atualizado com sucesso."})
}

func (s *Server) followUpsDeleteClient(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteFollowUpClient(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o cliente.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Cliente removido com sucesso."})
}

func (s *Server) followUpsCreateProject(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ClientID    int64  `json:"client_id"`
		ProjectName string `json:"project_name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.ClientID == 0 || strings.TrimSpace(payload.ProjectName) == "" {
		writeValidation(w, validationErrors{"project_name": "Informe o cliente e o projeto."})
		return
	}
	if _, err := s.Modules.CreateFollowUpProject(payload.ClientID, payload.ProjectName); err != nil {
		writeValidation(w, validationErrors{"project_name": "Este cliente já possui um projeto com este nome."})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Projeto cadastrado com sucesso."})
}

func (s *Server) followUpsUpdateProject(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		ClientID    int64  `json:"client_id"`
		ProjectName string `json:"project_name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.UpdateFollowUpProject(id, payload.ClientID, payload.ProjectName); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o projeto.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Projeto atualizado com sucesso."})
}

func (s *Server) followUpsDeleteProject(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteFollowUpProject(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o projeto.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Projeto removido com sucesso."})
}

func (s *Server) followUpsExport(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	user := currentUser(r)
	items, _, _, err := s.Modules.ListFollowUps(modules.FollowUpFilter{
		Search: query.Get("search"), Status: query.Get("status"), Priority: query.Get("priority"),
		Assigned: query.Get("assigned"), Page: 1, PerPage: 1000, UserEmail: user.Email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar os follow-ups.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="follow-ups-`+time.Now().Format("2006-01-02")+`.csv"`)
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"Código", "Título", "Descrição", "Cliente", "Projeto", "Status", "Prioridade", "Responsável", "Prazo", "Atrasado", "Concluído em", "Criado em"})
	for _, item := range items {
		_ = writer.Write([]string{
			derefString(item.Code), item.Title, derefString(item.Description), derefString(item.ClientName),
			derefString(item.ProjectName), followUpStatusLabel(item.Status), followUpPriorityLabel(item.Priority),
			derefString(item.AssigneeName), derefString(item.DueDate), boolLabel(item.Overdue),
			formatOptionalDate(item.CompletedAt), item.CreatedAt.Format("02/01/2006 15:04"),
		})
	}
	writer.Flush()
}

func followUpPriorityLabel(value string) string {
	labels := map[string]string{"baixa": "Baixa", "media": "Média", "alta": "Alta", "urgente": "Urgente"}
	return labels[value]
}

func followUpStatusLabel(value string) string {
	labels := map[string]string{"pendente": "Pendente", "em_andamento": "Em andamento", "concluido": "Concluído", "cancelado": "Cancelado"}
	if label, ok := labels[value]; ok {
		return label
	}
	return value
}

func boolLabel(value bool) string {
	if value {
		return "Sim"
	}
	return "Não"
}

func formatOptionalDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("02/01/2006")
}

func (s *Server) followUpsImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione um arquivo CSV."})
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		writeValidation(w, validationErrors{"file": "O arquivo está vazio ou inválido."})
		return
	}
	headers := map[string]int{}
	for index, value := range rows[0] {
		headers[normalizeHeader(value)] = index
	}
	if _, ok := headers["titulo"]; !ok {
		if _, ok := headers["title"]; !ok {
			writeValidation(w, validationErrors{"file": "O arquivo precisa conter a coluna Título (ou title)."})
			return
		}
	}
	user := currentUser(r)
	created := 0
	for _, row := range rows[1:] {
		get := func(names ...string) string {
			for _, name := range names {
				if index, ok := headers[normalizeHeader(name)]; ok && index < len(row) {
					return strings.TrimSpace(row[index])
				}
			}
			return ""
		}
		title := get("Título", "title")
		if title == "" {
			continue
		}
		status := get("Status", "status")
		if !followUpStatuses[status] {
			status = "pendente"
		}
		priority := get("Prioridade", "priority")
		if !followUpPriorities[priority] {
			priority = "media"
		}
		due := normalizeImportDate(get("Prazo", "due_date"))
		_, err := s.Modules.CreateFollowUp(user.ID, modules.FollowUpInput{
			Title: title, Code: get("Código", "code"), Description: get("Descrição", "description"),
			Status: status, Priority: priority, DueDate: due,
		})
		if err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("%d follow-up(s) importado(s).", created)})
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	previous := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			previous = false
		} else if !previous {
			builder.WriteByte('_')
			previous = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func normalizeImportDate(value string) string {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"02/01/2006", "2006-01-02", "02-01-2006", "02/01/06"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return value
}

func (s *Server) followUpsEmailOne(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Recipients string `json:"recipients"`
		Subject    string `json:"subject"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	item, err := s.Modules.FindFollowUp(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Follow-up não encontrado.", nil)
		return
	}
	settings, _ := s.Modules.FollowUpSettings()
	subject := strings.TrimSpace(payload.Subject)
	if subject == "" {
		subject = settings.EmailSubjectPrefix + ": " + item.Title
	}
	body := "Follow-up: " + item.Title + "\nStatus: " + followUpStatusLabel(item.Status) +
		"\nPrioridade: " + followUpPriorityLabel(item.Priority) + "\nPrazo: " + derefString(item.DueDate)
	for _, recipient := range splitEmails(payload.Recipients) {
		_ = s.Notifier.Send(recipient, subject, body)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Follow-up enviado por e-mail."})
}

func (s *Server) followUpsEmailTable(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Recipients string `json:"recipients"`
		Subject    string `json:"subject"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	settings, _ := s.Modules.FollowUpSettings()
	subject := strings.TrimSpace(payload.Subject)
	if subject == "" {
		subject = settings.EmailSubjectPrefix + " — " + time.Now().Format("02/01/2006")
	}
	body := "Tabela de follow-ups enviada pelo Documentation Center."
	for _, recipient := range splitEmails(payload.Recipients) {
		_ = s.Notifier.Send(recipient, subject, body)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tabela enviada por e-mail."})
}

func splitEmails(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t' })
	result := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && !seen[part] {
			result = append(result, part)
			seen[part] = true
		}
	}
	return result
}

var _ = strconv.Atoi
var _ = io.EOF
