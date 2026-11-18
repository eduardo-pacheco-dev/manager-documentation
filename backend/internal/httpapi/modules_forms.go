package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

/* --------------------------------------------------------------------------
 * Forms module — authenticated endpoints
 * ----------------------------------------------------------------------- */

type formPayload struct {
	ProjectID               *int64             `json:"project_id"`
	StationID               *int64             `json:"station_id"`
	RadioLinkID             *int64             `json:"radio_link_id"`
	Name                    string             `json:"name"`
	Type                    string             `json:"type"`
	Status                  string             `json:"status"`
	AllowsMultipleResponses bool               `json:"allows_multiple_responses"`
	TemplateID              *int64             `json:"template_id"`
	Steps                   []modules.FormStep `json:"steps"`
}

func (s *Server) formsIndex(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	settings, err := s.Modules.FormSettings(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as configurações de formulários.", nil)
		return
	}
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), settings.PerPage)
	if perPage < 5 {
		perPage = 5
	}
	if perPage > 100 {
		perPage = 100
	}
	forms, total, stats, statuses, types, err := s.Modules.ListForms(modules.FormFilter{
		UserID:      user.ID,
		Search:      query.Get("search"),
		Status:      query.Get("status"),
		StationID:   int64(parsePositiveInt(query.Get("station_id"), 0)),
		RadioLinkID: int64(parsePositiveInt(query.Get("radio_link_id"), 0)),
		Sort:        query.Get("sort"),
		Direction:   query.Get("direction"),
		Page:        page,
		PerPage:     perPage,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os formulários.", nil)
		return
	}
	statuses = mergeFormOptions(settings.Statuses, statuses)
	types = mergeFormOptions(settings.Types, types)

	templates, _ := s.Modules.ListFormTemplates()
	writeJSON(w, http.StatusOK, map[string]any{
		"data":      forms,
		"stats":     stats,
		"statuses":  statuses,
		"types":     types,
		"templates": templates,
		"settings":  settings,
		"meta":      paginationMeta(page, perPage, total),
	})
}

func mergeFormOptions(configured, existing []string) []string {
	result := modules.NormalizeFormOptions(configured)
	seen := map[string]bool{}
	for _, value := range result {
		seen[strings.ToLower(value)] = true
	}
	for _, value := range existing {
		value = strings.TrimSpace(value)
		if value == "" || seen[strings.ToLower(value)] {
			continue
		}
		seen[strings.ToLower(value)] = true
		result = append(result, value)
	}
	return result
}

// validateFormAssets enforces that a form is linked to at most one asset and
// that the referenced station or radio link exists.
func (s *Server) validateFormAssets(payload formPayload) validationErrors {
	fields := validationErrors{}
	if payload.StationID != nil && payload.RadioLinkID != nil {
		fields.add("station_id", "Escolha uma estação ou um rádio link, não ambos.")
		return fields
	}
	if payload.StationID != nil {
		var count int
		if err := s.Modules.DB.Get(&count, `SELECT COUNT(*) FROM stations WHERE id = ?`, *payload.StationID); err != nil || count == 0 {
			fields.add("station_id", "A estação selecionada não existe.")
		}
	}
	if payload.RadioLinkID != nil {
		var count int
		if err := s.Modules.DB.Get(&count, `SELECT COUNT(*) FROM radio_links WHERE id = ?`, *payload.RadioLinkID); err != nil || count == 0 {
			fields.add("radio_link_id", "O rádio link selecionado não existe.")
		}
	}
	return fields
}

func (s *Server) formsShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	form, err := s.Modules.FindForm(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	responses, _ := s.Modules.ListFormResponses(form.ID)
	var responseID *int64
	if raw := r.URL.Query().Get("response"); raw != "" {
		if parsed, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && parsed > 0 {
			responseID = &parsed
		}
	}
	evaluations, _ := s.Modules.ListFormEvaluations(form.ID, responseID)
	templates, _ := s.Modules.ListFormTemplates()
	settings, _ := s.Modules.FormSettings(currentUser(r).ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"form":        form,
			"responses":   responses,
			"evaluations": evaluations,
			"templates":   templates,
			"settings":    settings,
		},
	})
}

func (s *Server) formsCreate(w http.ResponseWriter, r *http.Request) {
	var payload formPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	fields := validationErrors{}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "Informe o nome do formulário.")
	}
	if len([]rune(payload.Name)) > 255 {
		fields.add("name", "O nome deve ter no máximo 255 caracteres.")
	}
	for name, message := range s.validateFormAssets(payload) {
		fields.add(name, message)
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if payload.TemplateID != nil {
		template, err := s.Modules.FindFormTemplate(*payload.TemplateID)
		if err != nil {
			writeValidation(w, validationErrors{"template_id": "O template selecionado não existe."})
			return
		}
		payload.Steps = modules.DecodeSteps(template.Steps)
		if strings.TrimSpace(payload.Type) == "" {
			payload.Type = template.Type
		}
	}
	steps, _ := json.Marshal(payload.Steps)
	form, err := s.Modules.CreateForm(currentUser(r).ID, modules.FormInput{
		ProjectID:               payload.ProjectID,
		StationID:               payload.StationID,
		RadioLinkID:             payload.RadioLinkID,
		Name:                    payload.Name,
		Type:                    payload.Type,
		Status:                  payload.Status,
		AllowsMultipleResponses: payload.AllowsMultipleResponses,
		Steps:                   steps,
		Fields:                  json.RawMessage("[]"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o formulário.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Formulário criado com sucesso.",
		"data":    map[string]any{"form": form},
	})
}

func (s *Server) formsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	var payload formPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	fields := validationErrors{}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "Informe o nome do formulário.")
	}
	if strings.TrimSpace(payload.Type) == "" {
		fields.add("type", "Selecione o tipo do formulário.")
	}
	if strings.TrimSpace(payload.Status) == "" {
		fields.add("status", "Selecione o status do formulário.")
	}
	for name, message := range s.validateFormAssets(payload) {
		fields.add(name, message)
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	form, err := s.Modules.UpdateForm(currentUser(r).ID, id, modules.FormInput{
		ProjectID:               payload.ProjectID,
		StationID:               payload.StationID,
		RadioLinkID:             payload.RadioLinkID,
		Name:                    payload.Name,
		Type:                    payload.Type,
		Status:                  payload.Status,
		AllowsMultipleResponses: payload.AllowsMultipleResponses,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Formulário atualizado com sucesso.",
		"data":    map[string]any{"form": form},
	})
}

func (s *Server) formsUpdateSteps(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	var payload struct {
		Steps []modules.FormStep `json:"steps"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateFormSteps(payload.Steps); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	form, err := s.Modules.UpdateFormSteps(currentUser(r).ID, id, payload.Steps)
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message":  "Formulário atualizado com sucesso.",
		"saved_at": time.Now().Format(time.RFC3339),
		"data":     map[string]any{"form": form},
	})
}

func validateFormSteps(steps []modules.FormStep) validationErrors {
	fields := validationErrors{}
	stepIDs := map[string]bool{}
	fieldIDs := map[string]bool{}
	for stepIndex, step := range steps {
		if strings.TrimSpace(step.ID) == "" {
			fields.add(fmt.Sprintf("steps.%d.id", stepIndex), "Seção sem identificador.")
			continue
		}
		if stepIDs[step.ID] {
			fields.add(fmt.Sprintf("steps.%d.id", stepIndex), "Seção duplicada.")
		}
		stepIDs[step.ID] = true
		for fieldIndex, field := range step.Fields {
			if strings.TrimSpace(field.ID) == "" {
				fields.add(fmt.Sprintf("steps.%d.fields.%d.id", stepIndex, fieldIndex), "Pergunta sem identificador.")
				continue
			}
			if fieldIDs[field.ID] {
				fields.add(fmt.Sprintf("steps.%d.fields.%d.id", stepIndex, fieldIndex), "Pergunta duplicada.")
			}
			fieldIDs[field.ID] = true
			if strings.TrimSpace(field.Label) == "" {
				fields.add(fmt.Sprintf("steps.%d.fields.%d.label", stepIndex, fieldIndex), "Informe o título da pergunta.")
			}
			switch field.Type {
			case "text", "textarea", "number", "date", "time", "radio", "checkbox", "dropdown", "scale", "image", "file":
			default:
				fields.add(fmt.Sprintf("steps.%d.fields.%d.type", stepIndex, fieldIndex), "Tipo de pergunta inválido.")
			}
		}
	}
	return fields
}

func (s *Server) formsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteForm(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Formulário excluído com sucesso."})
}

func (s *Server) formsEvaluationsIndex(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	form, err := s.Modules.FindForm(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	var responseID *int64
	if raw := r.URL.Query().Get("response_id"); raw != "" {
		if parsed, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && parsed > 0 {
			responseID = &parsed
		}
	}
	evaluations, err := s.Modules.ListFormEvaluations(form.ID, responseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as avaliações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": evaluations})
}

type formEvaluationPayload struct {
	FormResponseID *int64                       `json:"form_response_id"`
	Status         string                       `json:"status"`
	Notes          string                       `json:"notes"`
	Decisions      map[string]string            `json:"decisions"`
	ImageDecisions map[string]map[string]string `json:"image_decisions"`
}

func (s *Server) formsEvaluate(w http.ResponseWriter, r *http.Request) {
	s.handleFormEvaluation(w, r, true)
}

func (s *Server) formsEvaluationAutosave(w http.ResponseWriter, r *http.Request) {
	s.handleFormEvaluation(w, r, false)
}

func (s *Server) handleFormEvaluation(w http.ResponseWriter, r *http.Request, final bool) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	var payload formEvaluationPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	form, err := s.Modules.FindForm(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Formulário não encontrado.", nil)
		return
	}
	if payload.FormResponseID != nil && *payload.FormResponseID > 0 {
		response, err := s.Modules.FindFormResponse(form.ID, *payload.FormResponseID)
		if err != nil {
			writeError(w, http.StatusNotFound, "Resposta não encontrada para este formulário.", nil)
			return
		}
		_ = response
	}
	fields := validationErrors{}
	if final {
		if payload.Status != "Aprovado" && payload.Status != "Reprovado" {
			fields.add("status", "Selecione o resultado geral da avaliação.")
		}
		if payload.Status == "Reprovado" && strings.TrimSpace(payload.Notes) == "" {
			fields.add("notes", "Informe o motivo da reprovação.")
		}
		if len([]rune(payload.Notes)) > 2000 {
			fields.add("notes", "O parecer deve ter no máximo 2.000 caracteres.")
		}
	} else if payload.Status != "" && payload.Status != "Aprovado" && payload.Status != "Reprovado" {
		fields.add("status", "Resultado inválido.")
	}
	validDecisions := map[string]bool{"aprovado": true, "reprovado": true, "pendente": true, "nao_aplicavel": true}
	for fieldID, decision := range payload.Decisions {
		if decision != "" && !validDecisions[decision] {
			fields.add("decisions."+fieldID, "Decisão inválida.")
		}
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}

	evaluation, err := s.Modules.ApplyFormEvaluation(form.ID, payload.FormResponseID, currentUser(r).ID,
		payload.Status, payload.Notes, payload.Decisions, payload.ImageDecisions, final)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível registrar a avaliação.", nil)
		return
	}
	updated, _ := s.Modules.FindForm(currentUser(r).ID, form.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"saved_at":   time.Now().Format("15:04"),
		"message":    "Avaliação registrada com sucesso.",
		"evaluation": evaluation,
		"data":       map[string]any{"form": updated},
	})
}

/* --------------------------------------------------------------------------
 * Templates
 * ----------------------------------------------------------------------- */

func (s *Server) formTemplatesIndex(w http.ResponseWriter, r *http.Request) {
	templates, err := s.Modules.ListFormTemplates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os templates.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": templates})
}

func (s *Server) formTemplatesCreate(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		FormID *int64 `json:"form_id"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeValidation(w, validationErrors{"name": "Informe o nome do template."})
		return
	}
	steps := json.RawMessage("[]")
	if payload.FormID != nil {
		form, err := s.Modules.FindForm(currentUser(r).ID, *payload.FormID)
		if err != nil {
			writeValidation(w, validationErrors{"form_id": "Formulário não encontrado."})
			return
		}
		steps = form.Steps
		if strings.TrimSpace(payload.Type) == "" {
			payload.Type = form.Type
		}
	}
	template, err := s.Modules.CreateFormTemplate(currentUser(r).ID, payload.Name, payload.Type, steps)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar o template.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Template salvo com sucesso.",
		"data":    map[string]any{"template": template},
	})
}

func (s *Server) formTemplatesDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Template não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteFormTemplate(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o template.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Template excluído com sucesso."})
}

/* --------------------------------------------------------------------------
 * Settings
 * ----------------------------------------------------------------------- */

func (s *Server) formsSettingsShow(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Modules.FormSettings(currentUser(r).ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as configurações de formulários.", nil)
		return
	}
	templates, _ := s.Modules.ListFormTemplates()
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"settings":         settings,
			"templates":        templates,
			"views":            modules.FormViews,
			"per_page_options": modules.FormPerPageOptions,
		},
	})
}

func (s *Server) formsSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Types             []string `json:"types"`
		Statuses          []string `json:"statuses"`
		DefaultTemplateID *int64   `json:"default_template_id"`
		DefaultView       string   `json:"default_view"`
		PerPage           int      `json:"per_page"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	types := modules.NormalizeFormOptions(payload.Types)
	statuses := modules.NormalizeFormOptions(payload.Statuses)
	fields := validationErrors{}
	if len(types) == 0 {
		fields.add("types", "Informe ao menos um tipo de formulário.")
	}
	if len(statuses) == 0 {
		fields.add("statuses", "Informe ao menos um status de formulário.")
	}
	if _, ok := modules.FormViews[payload.DefaultView]; !ok {
		fields.add("default_view", "Selecione a visualização padrão.")
	}
	validPerPage := false
	for _, option := range modules.FormPerPageOptions {
		if payload.PerPage == option {
			validPerPage = true
			break
		}
	}
	if !validPerPage {
		fields.add("per_page", "A quantidade de itens por página é inválida.")
	}
	if payload.DefaultTemplateID != nil {
		if _, err := s.Modules.FindFormTemplate(*payload.DefaultTemplateID); err != nil {
			fields.add("default_template_id", "O template selecionado não existe.")
		}
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	settings := modules.FormSettings{
		Types:             types,
		Statuses:          statuses,
		DefaultTemplateID: payload.DefaultTemplateID,
		DefaultView:       payload.DefaultView,
		PerPage:           payload.PerPage,
	}
	if err := s.Modules.SaveFormSettings(currentUser(r).ID, settings); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível salvar as configurações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Configurações salvas com sucesso.",
		"data":    map[string]any{"settings": settings},
	})
}

func (s *Server) formsSettingsExport(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Modules.FormSettings(currentUser(r).ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as configurações.", nil)
		return
	}
	templateName := ""
	if settings.DefaultTemplateID != nil {
		if template, err := s.Modules.FindFormTemplate(*settings.DefaultTemplateID); err == nil {
			templateName = template.Name
		}
	}
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()
	sheet := "Configurações"
	file.SetSheetName("Sheet1", sheet)
	writeXLSXRow(file, sheet, 1, []string{"Configuração", "Valor"})
	row := 2
	for _, typ := range settings.Types {
		writeXLSXRow(file, sheet, row, []string{"Tipo de formulário", typ})
		row++
	}
	for _, status := range settings.Statuses {
		writeXLSXRow(file, sheet, row, []string{"Status de formulário", status})
		row++
	}
	viewLabel := modules.FormViews[settings.DefaultView]
	writeXLSXRow(file, sheet, row, []string{"Visualização padrão", viewLabel})
	row++
	writeXLSXRow(file, sheet, row, []string{"Itens por página", strconv.Itoa(settings.PerPage)})
	row++
	writeXLSXRow(file, sheet, row, []string{"Template padrão", templateName})
	styleNokiaSheet(file, sheet, 2, row)
	_ = file.SetColWidth(sheet, "A", "A", 30)
	_ = file.SetColWidth(sheet, "B", "B", 46)
	writeWorkbook(w, file, "configuracoes-formularios.xlsx")
}

func (s *Server) formsSettingsImport(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione uma planilha para importar."})
		return
	}
	defer file.Close()
	matrix, err := readNokiaImportMatrix(file, header.Filename)
	if err != nil || len(matrix) < 2 {
		writeValidation(w, validationErrors{"file": "O arquivo está vazio ou inválido."})
		return
	}
	var types, statuses []string
	view := ""
	perPage := 0
	templateName := ""
	for _, row := range matrix[1:] {
		if len(row) < 2 {
			continue
		}
		key := normalizeNokiaHeader(row[0])
		value := strings.TrimSpace(row[1])
		switch key {
		case "tipo", "tipos", "tipo_de_formulario", "tipos_de_formulario", "tipo_do_formulario":
			types = append(types, value)
		case "status", "statuses", "status_de_formulario", "status_de_formularios", "status_do_formulario":
			statuses = append(statuses, value)
		case "visualizacao", "visualizacao_padrao", "visualizacao_do_formulario", "view":
			view = value
		case "itens_por_pagina", "itens_por_pagina_do_formulario", "itens_pagina", "per_page":
			perPage, _ = strconv.Atoi(strings.Fields(value + " 0")[0])
		case "template", "template_padrao", "modelo", "modelo_padrao":
			templateName = value
		}
	}
	types = modules.NormalizeFormOptions(types)
	statuses = modules.NormalizeFormOptions(statuses)
	fields := validationErrors{}
	if len(types) == 0 {
		fields.add("file", "Informe ao menos um tipo de formulário no template.")
	}
	if len(statuses) == 0 {
		fields.add("file", "Informe ao menos um status de formulário no template.")
	}
	normalizedView := ""
	switch normalizeNokiaHeader(view) {
	case "table", "tabela":
		normalizedView = "table"
	case "cards", "card":
		normalizedView = "cards"
	case "compact", "lista", "list":
		normalizedView = "compact"
	default:
		fields.add("file", "A visualização padrão deve ser Tabela, Cards ou Lista.")
	}
	validPerPage := false
	for _, option := range modules.FormPerPageOptions {
		if perPage == option {
			validPerPage = true
			break
		}
	}
	if !validPerPage {
		fields.add("file", "A quantidade de itens por página deve ser 12, 24 ou 36.")
	}
	var templateID *int64
	if templateName != "" {
		templates, _ := s.Modules.ListFormTemplates()
		for _, template := range templates {
			if strings.EqualFold(template.Name, templateName) || strconv.FormatInt(template.ID, 10) == templateName {
				id := template.ID
				templateID = &id
				break
			}
		}
		if templateID == nil {
			fields.add("file", fmt.Sprintf("O template padrão %q não foi encontrado.", templateName))
		}
	}
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	settings := modules.FormSettings{
		Types:             types,
		Statuses:          statuses,
		DefaultTemplateID: templateID,
		DefaultView:       normalizedView,
		PerPage:           perPage,
	}
	if err := s.Modules.SaveFormSettings(currentUser(r).ID, settings); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível importar as configurações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Template de configurações importado com sucesso.",
		"data":    map[string]any{"settings": settings},
	})
}
