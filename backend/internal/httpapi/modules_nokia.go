package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type nokiaSitePayload struct {
	SiteID                 string `json:"site_id"`
	EndID                  string `json:"end_id"`
	OC                     string `json:"oc"`
	OSFamEntrega           string `json:"os_fam_entrega"`
	OSFamInstalacao        string `json:"os_fam_instalacao"`
	Regional               string `json:"regional"`
	Subcon                 string `json:"subcon"`
	ChecklistFamEntrega    bool   `json:"checklist_fam_entrega"`
	ChecklistFamInstalacao bool   `json:"checklist_fam_instalacao"`
	ChecklistFamPanoramica bool   `json:"checklist_fam_panoramica"`
	NokiaROMID             *int64 `json:"nokia_rom_id"`
}

func (p nokiaSitePayload) input() modules.NokiaSiteInput {
	return modules.NokiaSiteInput{
		SiteID: p.SiteID, EndID: p.EndID, OC: p.OC, OSFamEntrega: p.OSFamEntrega,
		OSFamInstalacao: p.OSFamInstalacao, Regional: p.Regional, Subcon: p.Subcon,
		ChecklistFamEntrega: p.ChecklistFamEntrega, ChecklistFamInstalacao: p.ChecklistFamInstalacao,
		ChecklistFamPanoramica: p.ChecklistFamPanoramica, NokiaROMID: p.NokiaROMID,
	}
}

func (s *Server) nokiaIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	items, total, stats, tasksTotal, err := s.Modules.ListNokiaSites(modules.NokiaSiteFilter{
		Search: query.Get("search"), Page: page, PerPage: 15,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os sites.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items, "stats": stats, "tasks_total": tasksTotal,
		"meta": paginationMeta(page, 15, total),
	})
}

func (s *Server) nokiaShow(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	site, err := s.Modules.FindNokiaSite(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Site não encontrado.", nil)
		return
	}
	for i := range site.Attachments {
		site.Attachments[i].URL = "/uploads/" + site.Attachments[i].Path
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"site": site}})
}

func (s *Server) nokiaCreate(w http.ResponseWriter, r *http.Request) {
	var payload nokiaSitePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.SiteID) == "" {
		writeValidation(w, validationErrors{"site_id": "Informe o Site ID."})
		return
	}
	site, err := s.Modules.CreateNokiaSite(payload.input())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeValidation(w, validationErrors{"oc": "Já existe um site com este OC."})
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível cadastrar o site.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Site cadastrado com sucesso.", "data": map[string]any{"site": site}})
}

func (s *Server) nokiaUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload nokiaSitePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	site, err := s.Modules.UpdateNokiaSite(id, payload.input())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeValidation(w, validationErrors{"oc": "Já existe um site com este OC."})
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o site.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Site atualizado com sucesso.", "data": map[string]any{"site": site}})
}

func (s *Server) nokiaDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	paths, err := s.Modules.DeleteNokiaSite(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o site.", nil)
		return
	}
	for _, path := range paths {
		s.Uploads.DeleteLocal(path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Site excluído com sucesso."})
}

func (s *Server) nokiaUpdateChecklist(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	field := chiURLParam(r, "field")
	if _, ok := modules.NokiaChecklists[field]; !ok {
		writeError(w, http.StatusNotFound, "Checklist não encontrado.", nil)
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	valid := map[string]bool{"pending": true, "analysis": true, "rejected": true, "sent": true}
	if !valid[payload.Status] {
		writeValidation(w, validationErrors{"status": "Status inválido."})
		return
	}
	site, err := s.Modules.UpdateNokiaChecklist(id, field, payload.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o checklist.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": modules.NokiaChecklists[field] + " atualizado.", "data": map[string]any{"site": site}})
}

func (s *Server) nokiaUpdateDates(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload map[string]string
	if !decodeJSON(w, r, &payload) {
		return
	}
	site, err := s.Modules.UpdateNokiaDates(id, payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar as datas.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Datas atualizadas com sucesso.", "data": map[string]any{"site": site}})
}

func (s *Server) nokiaUpdateInterruption(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Reason string `json:"reason"`
		Days   int    `json:"return_days"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Reason) == "" {
		writeValidation(w, validationErrors{"reason": "Informe o motivo da parada."})
		return
	}
	if payload.Days != 7 && payload.Days != 14 && payload.Days != 21 && payload.Days != 28 {
		writeValidation(w, validationErrors{"return_days": "O prazo deve ser 7, 14, 21 ou 28 dias."})
		return
	}
	if err := s.Modules.SetNokiaInterruption(id, payload.Reason, payload.Days); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível registrar a interrupção.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Interrupção de obra registrada."})
}

func (s *Server) nokiaClearInterruption(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.ClearNokiaInterruption(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível encerrar a interrupção.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Interrupção de obra encerrada."})
}

func (s *Server) nokiaToggleTask(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	taskID, _ := parseID(r, "taskId")
	if err := s.Modules.ToggleNokiaTask(id, taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tarefa atualizada."})
}

func (s *Server) nokiaCreateRejection(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		FAMs          []string `json:"fams"`
		Reason        string   `json:"reason"`
		ReturnDays    int      `json:"return_days"`
		RequiresField bool     `json:"requires_field_return"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if len(payload.FAMs) == 0 || strings.TrimSpace(payload.Reason) == "" {
		writeValidation(w, validationErrors{"reason": "Selecione ao menos uma FAM e informe o motivo."})
		return
	}
	if payload.ReturnDays != 7 && payload.ReturnDays != 14 && payload.ReturnDays != 21 && payload.ReturnDays != 28 {
		writeValidation(w, validationErrors{"return_days": "O prazo deve ser 7, 14, 21 ou 28 dias."})
		return
	}
	if err := s.Modules.CreateNokiaRejection(id, payload.FAMs, payload.Reason, payload.ReturnDays, payload.RequiresField); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível registrar a rejeição.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Rejeição registrada."})
}

func (s *Server) nokiaResolveRejection(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	rejectionID, _ := parseID(r, "rejectionId")
	if err := s.Modules.ResolveNokiaRejection(id, rejectionID); err != nil {
		writeError(w, http.StatusNotFound, "Rejeição não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Rejeição encerrada."})
}

func (s *Server) nokiaUploadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := r.ParseMultipartForm(40 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	category := r.FormValue("category")
	if category != "tssr" && category != "doc-d" && category != "dwg" {
		writeValidation(w, validationErrors{"category": "Categoria inválida."})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeValidation(w, validationErrors{"files": "Selecione ao menos um arquivo."})
		return
	}
	created := 0
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			continue
		}
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "nokia")
		file.Close()
		if err != nil {
			writeValidation(w, validationErrors{"files": "Formato de arquivo não permitido."})
			return
		}
		if err := s.Modules.CreateNokiaAttachment(id, category, header.Filename, result.Path, header.Header.Get("Content-Type"), header.Size); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": fmt.Sprintf("%d anexo(s) enviado(s).", created)})
}

func (s *Server) nokiaDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	attachment, err := s.Modules.FindNokiaAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo não encontrado.", nil)
		return
	}
	http.Redirect(w, r, "/uploads/"+attachment.Path, http.StatusTemporaryRedirect)
}

func (s *Server) nokiaDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	path, err := s.Modules.DeleteNokiaAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo não encontrado.", nil)
		return
	}
	s.Uploads.DeleteLocal(path)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Anexo removido com sucesso."})
}

func (s *Server) nokiaCreateComment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.CreateNokiaComment(id, currentUser(r).ID, payload.Body); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível publicar o comentário.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Comentário publicado."})
}

func (s *Server) nokiaDeleteComment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	commentID, _ := parseID(r, "commentId")
	comment, err := s.Modules.FindNokiaComment(id, commentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Comentário não encontrado.", nil)
		return
	}
	auth := authFrom(r)
	if !auth.User.IsAdmin && (comment.UserID == nil || *comment.UserID != auth.User.ID) {
		writeError(w, http.StatusForbidden, "Você não pode remover este comentário.", nil)
		return
	}
	if err := s.Modules.DeleteNokiaComment(id, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível remover o comentário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Comentário removido."})
}

func (s *Server) nokiaROMList(w http.ResponseWriter, r *http.Request) {
	roms, err := s.Modules.NokiaROMs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os ROMs.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": roms})
}

func (s *Server) nokiaSettings(w http.ResponseWriter, r *http.Request) {
	roms, _ := s.Modules.NokiaROMs()
	tasks, _ := s.Modules.NokiaTasks()
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"roms": roms, "tasks": tasks}})
}

func (s *Server) nokiaCreateROM(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.FirstName) == "" || strings.TrimSpace(payload.LastName) == "" || strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Phone) == "" {
		writeValidation(w, validationErrors{"first_name": "Preencha nome, sobrenome, e-mail e telefone do ROM."})
		return
	}
	rom, err := s.Modules.CreateNokiaROM(payload.FirstName, payload.LastName, payload.Email, payload.Phone)
	if err != nil {
		writeValidation(w, validationErrors{"email": "Já existe um ROM com este e-mail."})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "ROM cadastrado com sucesso.", "data": map[string]any{"rom": rom}})
}

func (s *Server) nokiaUpdateROM(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.UpdateNokiaROM(id, payload.FirstName, payload.LastName, payload.Email, payload.Phone); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o ROM.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ROM atualizado com sucesso."})
}

func (s *Server) nokiaDeleteROM(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteNokiaROM(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o ROM.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ROM excluído com sucesso."})
}

func (s *Server) nokiaCreateTask(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	task, err := s.Modules.CreateNokiaTask(payload.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Tarefa cadastrada com sucesso.", "data": map[string]any{"task": task}})
}

func (s *Server) nokiaUpdateTask(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.UpdateNokiaTask(id, payload.Name); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tarefa atualizada com sucesso."})
}

func (s *Server) nokiaMoveTask(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Direction string `json:"direction"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.MoveNokiaTask(id, payload.Direction); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível mover a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Ordem das tarefas atualizada."})
}

func (s *Server) nokiaDeleteTask(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteNokiaTask(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tarefa excluída com sucesso."})
}

func (s *Server) nokiaExport(w http.ResponseWriter, r *http.Request) {
	items, _, _, _, err := s.Modules.ListNokiaSites(modules.NokiaSiteFilter{Page: 1, PerPage: 10000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar os sites.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sites-nokia-%s.csv"`, time.Now().Format("2006-01-02")))
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"Site ID", "End ID", "OC", "ROM", "OS FAM Entrega", "OS FAM Instalação", "Regional", "Subcon", "FAM Entrega", "FAM Instalação", "FAM Panorâmica", "MOS Plan", "MOS Real", "Instalação Plan", "Instalação Real", "Integração Plan", "Integração Real", "RFA Plan", "RFA Real"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.SiteID, derefString(item.EndID), derefString(item.OC), derefString(item.ROMName),
			derefString(item.OSFamEntrega), derefString(item.OSFamInstalacao), derefString(item.Regional), derefString(item.Subcon),
			item.ChecklistFamEntregaStatus, item.ChecklistFamInstalacaoStatus, item.ChecklistFamPanoramicaStatus,
			derefString(item.MOSPlanAt), derefString(item.MOSRealAt), derefString(item.InstalacaoPlanAt),
			derefString(item.InstalacaoRealAt), derefString(item.IntegracaoPlanAt), derefString(item.IntegracaoRealAt),
			derefString(item.RFAPlanAt), derefString(item.RFARealAt),
		})
	}
	writer.Flush()
}

func (s *Server) nokiaTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="modelo-importacao-sites-nokia.csv"`)
	_, _ = w.Write([]byte("Site ID *;End ID;OC *;ROM;OS FAM Entrega;OS FAM Instalação;Regional;Subcon\n"))
}

func (s *Server) nokiaImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
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
		siteID := get("Site ID", "site_id")
		oc := get("OC", "oc")
		if siteID == "" {
			continue
		}
		if oc == "" {
			writeValidation(w, validationErrors{"file": "Todos os sites precisam de um OC."})
			return
		}
		payload := modules.NokiaSiteInput{
			SiteID: siteID, EndID: get("End ID"), OC: oc, OSFamEntrega: get("OS FAM Entrega"),
			OSFamInstalacao: get("OS FAM Instalação"), Regional: get("Regional"), Subcon: get("Subcon"),
		}
		if _, err := s.Modules.CreateNokiaSite(payload); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("%d site(s) importado(s).", created)})
}

func chiURLParam(r *http.Request, name string) string {
	return strings.TrimSpace(chi.URLParam(r, name))
}
