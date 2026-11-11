package httpapi

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"

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
		log.Printf("nokia show %d: %v", id, err)
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
	_ = s.Modules.SyncNokiaSiteToStation(site.ID)
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
	_ = s.Modules.SyncNokiaSiteToStation(site.ID)
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
	if !map[string]bool{"tssr": true, "doc-d": true, "dwg": true, "invoice": true, "other": true}[category] {
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
		if header.Size > 40<<20 {
			writeValidation(w, validationErrors{"files": "Cada arquivo deve ter no máximo 40 MB."})
			return
		}
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

func (s *Server) nokiaStationSites(w http.ResponseWriter, r *http.Request) {
	includeLinked := r.URL.Query().Get("include_linked") == "1"
	options, err := s.Modules.StationSiteOptions(r.URL.Query().Get("search"), !includeLinked)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os sites das estações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": options})
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
	search := r.URL.Query().Get("search")
	items, _, _, _, err := s.Modules.ListNokiaSites(modules.NokiaSiteFilter{Search: search, Page: 1, PerPage: 10000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar os sites.", nil)
		return
	}
	tasks, _ := s.Modules.NokiaTasks()
	headers := nokiaExportHeaders(tasks)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := []string{
			item.SiteID, derefString(item.EndID), derefString(item.OC), derefString(item.ROMName),
			derefString(item.OSFamEntrega), derefString(item.OSFamInstalacao), derefString(item.Regional),
			derefString(item.Subcon), nokiaChecklistStatus(item.ChecklistFamEntregaStatus),
			nokiaChecklistStatus(item.ChecklistFamInstalacaoStatus), nokiaChecklistStatus(item.ChecklistFamPanoramicaStatus),
		}
		completedTasks := map[int64]bool{}
		for _, task := range item.Tasks {
			completedTasks[task.ID] = task.CompletedAt != nil
		}
		for _, task := range tasks {
			if completedTasks[task.ID] {
				row = append(row, "Concluída")
			} else {
				row = append(row, "Pendente")
			}
		}
		if item.InterruptionReason != nil {
			row = append(row, derefString(item.InterruptionReason), formatNokiaDateTime(item.InterruptionStartedAt),
				formatNokiaInt(item.InterruptionReturnDays), nokiaReturnDate(item.InterruptionStartedAt, item.InterruptionReturnDays))
		} else {
			row = append(row, "", "", "", "")
		}
		for _, field := range []string{"checklist_fam_entrega", "checklist_fam_instalacao", "checklist_fam_panoramica"} {
			row = append(row, nokiaRejectionSummary(item.Rejections, field))
		}
		row = append(row,
			derefString(item.MOSPlanAt), derefString(item.MOSRealAt), derefString(item.InstalacaoPlanAt),
			derefString(item.InstalacaoRealAt), derefString(item.IntegracaoPlanAt), derefString(item.IntegracaoRealAt),
			derefString(item.RFAPlanAt), derefString(item.RFARealAt), derefString(item.FamEntregaApprovedAt),
			derefString(item.FamInstalacaoApprovedAt), derefString(item.FamPanoramicaApprovedAt),
			item.CreatedAt.Format("02/01/2006 15:04"), item.UpdatedAt.Format("02/01/2006 15:04"),
		)
		rows = append(rows, row)
	}
	writeNokiaXLSX(w, "sites-nokia-"+time.Now().Format("2006-01-02")+".xlsx", "Sites", headers, rows, []int{18, 18, 16, 24, 20, 20, 16, 16})
}

func (s *Server) nokiaPendingExport(w http.ResponseWriter, r *http.Request) {
	metrics, err := s.Modules.NokiaMetrics()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível gerar o relatório de pendências.", nil)
		return
	}
	headers := []string{"Site ID", "OC", "Regional", "Subcon", "ROM", "Status do site", "Tipo", "Pendência", "Situação", "Aging (dias)"}
	rows := make([][]string, 0, len(metrics.PendingItems))
	for _, item := range metrics.PendingItems {
		rows = append(rows, []string{
			mapString(item, "site_id"), mapString(item, "oc"), mapString(item, "regional"), mapString(item, "subcon"),
			mapString(item, "rom"), mapString(item, "site_status"), mapString(item, "kind_label"),
			mapString(item, "description"), mapString(item, "detail"), mapString(item, "aging"),
		})
	}
	writeNokiaXLSX(w, "pendencias-nokia-"+time.Now().Format("2006-01-02")+".xlsx", "Pendências", headers, rows, []int{18, 16, 16, 16, 24, 18, 14, 30, 18, 14})
}

func (s *Server) nokiaTemplate(w http.ResponseWriter, r *http.Request) {
	tasks, _ := s.Modules.NokiaTasks()
	headers := nokiaTemplateHeaders(tasks)
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "Sites"
	f.SetSheetName("Sheet1", sheet)
	writeXLSXRow(f, sheet, 1, headers)
	styleNokiaSheet(f, sheet, len(headers), 1)
	example := []string{
		"SPEXEMPLO01", "END-00001", "OC-0001", "Nome do ROM", "OS-E-00001", "OS-I-00001", "TNO", "AFL",
		"Sim", "Não", "Não", "10/01/2026", "12/01/2026", "11/01/2026", "13/01/2026",
		"14/01/2026", "16/01/2026", "17/01/2026", "18/01/2026", "Sim", "Não", "Não",
	}
	for _, task := range tasks {
		example = append(example, "Não")
		_ = task
	}
	exSheet := "Exemplo"
	if _, err := f.NewSheet(exSheet); err == nil {
		writeXLSXRow(f, exSheet, 1, headers)
		writeXLSXRow(f, exSheet, 2, example)
		styleNokiaSheet(f, exSheet, len(headers), 1)
	}
	infoSheet := "Instruções"
	if _, err := f.NewSheet(infoSheet); err == nil {
		instructions := [][]string{
			{"Instruções de importação"},
			{"Preencha a aba Sites a partir da linha 2. A aba Exemplo é apenas referência."},
			{"Site ID e OC são obrigatórios."},
			{"O ROM deve ser o nome completo de um ROM cadastrado em Configurações Nokia."},
			{"Checklists e tarefas aceitam Sim/Não, 1/0 ou X."},
			{"Datas aceitam dd/mm/aaaa ou aaaa-mm-dd."},
			{"OC existente atualiza o site; OCs repetidos no arquivo são rejeitados."},
			{"A importação é atômica e aceita no máximo 500 sites."},
		}
		for index, row := range instructions {
			writeXLSXRow(f, infoSheet, index+1, row)
		}
		f.SetColWidth(infoSheet, "A", "A", 110)
	}
	writeWorkbook(w, f, "modelo-importacao-sites-nokia.xlsx")
}

func (s *Server) nokiaImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione uma planilha CSV ou XLSX."})
		return
	}
	defer file.Close()
	matrix, err := readNokiaImportMatrix(file, header.Filename)
	if err != nil || len(matrix) < 2 {
		writeValidation(w, validationErrors{"file": "O arquivo está vazio ou inválido."})
		return
	}
	headers := matrix[0]
	index := map[string]int{}
	for i, value := range headers {
		index[normalizeNokiaHeader(value)] = i
	}
	if _, ok := index["site_id"]; !ok {
		writeValidation(w, validationErrors{"file": "O arquivo precisa conter a coluna Site ID."})
		return
	}
	if _, ok := index["oc"]; !ok {
		writeValidation(w, validationErrors{"file": "O arquivo precisa conter a coluna OC."})
		return
	}
	tasks, _ := s.Modules.NokiaTasks()
	roms, _ := s.Modules.NokiaROMs()
	romByName := map[string]int64{}
	for _, rom := range roms {
		romByName[strings.ToLower(rom.FullName())] = rom.ID
	}
	taskByHeader := map[string]modules.NokiaTask{}
	for _, task := range tasks {
		taskByHeader[normalizeNokiaHeader("Tarefa - "+task.Name)] = task
		taskByHeader[normalizeNokiaHeader("Atividade - "+task.Name)] = task
	}
	type importRow struct {
		input      modules.NokiaSiteInput
		dates      map[string]string
		checklists map[string]string
		tasks      map[int64]bool
		line       int
	}
	rows := make([]importRow, 0, len(matrix)-1)
	errorsList := []string{}
	seenOC := map[string]int{}
	for line, row := range matrix[1:] {
		get := func(names ...string) string {
			for _, name := range names {
				if col, ok := index[normalizeNokiaHeader(name)]; ok && col < len(row) {
					return strings.TrimSpace(row[col])
				}
			}
			return ""
		}
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		siteID := get("site_id")
		oc := get("oc")
		lineNumber := line + 2
		if siteID == "" {
			errorsList = append(errorsList, fmt.Sprintf("Linha %d: informe o Site ID.", lineNumber))
			continue
		}
		if oc == "" {
			errorsList = append(errorsList, fmt.Sprintf("Linha %d: informe o OC.", lineNumber))
			continue
		}
		if previous, exists := seenOC[strings.ToLower(oc)]; exists {
			errorsList = append(errorsList, fmt.Sprintf("Linha %d: o OC %s já foi usado na linha %d.", lineNumber, oc, previous))
			continue
		}
		seenOC[strings.ToLower(oc)] = lineNumber
		romName := get("rom", "rom_nome", "nome_rom", "responsavel")
		var romID *int64
		if romName != "" {
			if id, ok := romByName[strings.ToLower(romName)]; ok {
				romID = &id
			} else {
				errorsList = append(errorsList, fmt.Sprintf("Linha %d: ROM %s não encontrado. Cadastre-o em Configurações Nokia.", lineNumber, romName))
				continue
			}
		}
		checklists := map[string]string{}
		for headerKey, field := range map[string]string{
			"fam_entrega": "checklist_fam_entrega", "fam_instalacao": "checklist_fam_instalacao", "fam_panoramica": "checklist_fam_panoramica",
		} {
			if _, present := index[headerKey]; !present {
				continue
			}
			value := get(headerKey)
			if value == "" {
				continue
			}
			if parseNokiaBool(value) {
				checklists[field] = "sent"
			} else {
				checklists[field] = "pending"
			}
		}
		dates := map[string]string{}
		for headerKey, field := range map[string]string{
			"mos_plan": "mos_plan_at", "mos_real": "mos_real_at", "instalacao_plan": "instalacao_plan_at",
			"instalacao_real": "instalacao_real_at", "integracao_plan": "integracao_plan_at",
			"integracao_real": "integracao_real_at", "rfa_plan": "rfa_plan_at", "rfa_real": "rfa_real_at",
			"fam_entrega_aprovado": "fam_entrega_approved_at", "fam_instalacao_aprovado": "fam_instalacao_approved_at",
			"fam_panoramica_aprovado": "fam_panoramica_approved_at", "fam_entrega_aprovada": "fam_entrega_approved_at",
			"fam_instalacao_aprovada": "fam_instalacao_approved_at", "fam_panoramica_aprovada": "fam_panoramica_approved_at",
		} {
			if _, present := index[headerKey]; !present {
				continue
			}
			raw := get(headerKey)
			if raw == "" {
				continue
			}
			parsed, ok := normalizeNokiaImportDate(raw)
			if !ok {
				errorsList = append(errorsList, fmt.Sprintf("Linha %d: %q não é uma data válida.", lineNumber, raw))
				continue
			}
			dates[field] = parsed
		}
		taskValues := map[int64]bool{}
		for headerKey, task := range taskByHeader {
			if _, present := index[headerKey]; !present {
				continue
			}
			value := get(headerKey)
			if value == "" {
				continue
			}
			taskValues[task.ID] = parseNokiaBool(value)
		}
		rows = append(rows, importRow{
			input: modules.NokiaSiteInput{
				SiteID: siteID, EndID: get("end_id"), OC: oc, OSFamEntrega: get("os_fam_entrega"),
				OSFamInstalacao: get("os_fam_instalacao"), Regional: get("regional"), Subcon: get("subcon"),
				ChecklistFamEntrega:    parseNokiaBool(get("fam_entrega")),
				ChecklistFamInstalacao: parseNokiaBool(get("fam_instalacao")),
				ChecklistFamPanoramica: parseNokiaBool(get("fam_panoramica")),
				NokiaROMID:             romID,
			},
			dates: dates, checklists: checklists, tasks: taskValues, line: lineNumber,
		})
	}
	if len(rows) == 0 && len(errorsList) == 0 {
		writeValidation(w, validationErrors{"file": "O arquivo não possui nenhum site para importar."})
		return
	}
	if len(rows) > 500 {
		writeValidation(w, validationErrors{"file": "A importação aceita no máximo 500 sites por arquivo."})
		return
	}
	if len(errorsList) > 0 {
		message := strings.Join(firstN(errorsList, 5), " ")
		if len(errorsList) > 5 {
			message += fmt.Sprintf(" E mais %d erro(s).", len(errorsList)-5)
		}
		writeValidation(w, validationErrors{"file": message})
		return
	}
	created, updated := 0, 0
	for _, row := range rows {
		existing, findErr := s.Modules.FindNokiaSiteByOC(row.input.OC)
		var site modules.NokiaSite
		var saveErr error
		if findErr == nil {
			input := row.input
			if input.EndID == "" {
				input.EndID = derefString(existing.EndID)
			}
			if input.OSFamEntrega == "" {
				input.OSFamEntrega = derefString(existing.OSFamEntrega)
			}
			if input.OSFamInstalacao == "" {
				input.OSFamInstalacao = derefString(existing.OSFamInstalacao)
			}
			if input.Regional == "" {
				input.Regional = derefString(existing.Regional)
			}
			if input.Subcon == "" {
				input.Subcon = derefString(existing.Subcon)
			}
			if input.NokiaROMID == nil {
				input.NokiaROMID = existing.NokiaROMID
			}
			site, saveErr = s.Modules.UpdateNokiaSiteBase(existing.ID, input)
			if saveErr == nil {
				updated++
			}
		} else {
			site, saveErr = s.Modules.CreateNokiaSite(row.input)
			if saveErr == nil {
				created++
			}
		}
		if saveErr != nil {
			writeError(w, http.StatusUnprocessableEntity, "Não foi possível importar os sites.", nil)
			return
		}
		for field, status := range row.checklists {
			_, _ = s.Modules.UpdateNokiaChecklist(site.ID, field, status)
		}
		if len(row.dates) > 0 {
			_, _ = s.Modules.UpdateNokiaDates(site.ID, row.dates)
		}
		for taskID, completed := range row.tasks {
			_ = s.Modules.SetNokiaTaskState(site.ID, taskID, completed)
		}
		_ = s.Modules.SyncNokiaSiteToStation(site.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("%d site(s) criado(s) e %d atualizado(s) com sucesso.", created, updated),
	})
}

func nokiaExportHeaders(tasks []modules.NokiaTask) []string {
	headers := []string{
		"Site ID", "End ID", "OC", "ROM", "OS FAM Entrega", "OS FAM Instalação", "Regional", "Subcon",
		"Checklist - FAM Entrega", "Checklist - FAM Instalação", "Checklist - FAM Panorâmica",
	}
	for _, task := range tasks {
		headers = append(headers, "Tarefa - "+task.Name)
	}
	headers = append(headers, "Interrupção - Motivo", "Interrupção - Início", "Interrupção - Prazo (dias)", "Interrupção - Retorno previsto")
	for _, label := range []string{"FAM Entrega", "FAM Instalação", "FAM Panorâmica"} {
		headers = append(headers, "Rejeição - "+label)
	}
	headers = append(headers,
		"MOS Plan", "MOS Real", "Instalação Plan", "Instalação Real", "Integração Plan", "Integração Real",
		"RFA Plan", "RFA Real", "FAM Entrega Aprovado", "FAM Instalação Aprovado", "FAM Panorâmica Aprovado",
		"Criado em", "Atualizado em",
	)
	return headers
}

func nokiaTemplateHeaders(tasks []modules.NokiaTask) []string {
	headers := []string{
		"Site ID *", "End ID", "OC *", "ROM", "OS FAM Entrega", "OS FAM Instalação", "Regional", "Subcon",
		"FAM Entrega", "FAM Instalação", "FAM Panorâmica", "MOS Plan", "MOS Real", "Instalação Plan",
		"Instalação Real", "Integração Plan", "Integração Real", "RFA Plan", "RFA Real",
		"FAM Entrega Aprovado", "FAM Instalação Aprovado", "FAM Panorâmica Aprovado",
	}
	for _, task := range tasks {
		headers = append(headers, "Tarefa - "+task.Name)
	}
	return headers
}

func writeNokiaXLSX(w http.ResponseWriter, filename, sheet string, headers []string, rows [][]string, widths []int) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	f.SetSheetName("Sheet1", sheet)
	writeXLSXRow(f, sheet, 1, headers)
	for index, row := range rows {
		writeXLSXRow(f, sheet, index+2, row)
	}
	styleNokiaSheet(f, sheet, len(headers), len(rows)+1)
	for i, width := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, float64(width))
	}
	writeWorkbook(w, f, filename)
}

func writeXLSXRow(f *excelize.File, sheet string, row int, values []string) {
	for col, value := range values {
		cell, _ := excelize.CoordinatesToCellName(col+1, row)
		_ = f.SetCellValue(sheet, cell, value)
	}
}

func styleNokiaSheet(f *excelize.File, sheet string, columns, lastRow int) {
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1E293B"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "E2E8F0"}, {Type: "right", Style: 1, Color: "E2E8F0"},
			{Type: "top", Style: 1, Color: "E2E8F0"}, {Type: "bottom", Style: 1, Color: "E2E8F0"},
		},
	})
	bodyStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Style: 1, Color: "E2E8F0"}, {Type: "right", Style: 1, Color: "E2E8F0"},
			{Type: "top", Style: 1, Color: "E2E8F0"}, {Type: "bottom", Style: 1, Color: "E2E8F0"},
		},
	})
	lastCol, _ := excelize.ColumnNumberToName(columns)
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle)
	if lastRow > 1 {
		_ = f.SetCellStyle(sheet, "A2", fmt.Sprintf("%s%d", lastCol, lastRow), bodyStyle)
	}
	_ = f.SetRowHeight(sheet, 1, 30)
	_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	_ = f.AutoFilter(sheet, "A1:"+lastCol+"1", nil)
}

func writeWorkbook(w http.ResponseWriter, f *excelize.File, filename string) {
	buffer, err := f.WriteToBuffer()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível gerar a planilha.", nil)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, buffer)
}

func readNokiaImportMatrix(file io.Reader, filename string) ([][]string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".xlsx":
		workbook, err := excelize.OpenReader(file)
		if err != nil {
			return nil, err
		}
		defer func() { _ = workbook.Close() }()
		sheets := workbook.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("workbook has no sheets")
		}
		return workbook.GetRows(sheets[0])
	case ".xls":
		return nil, fmt.Errorf("xls not supported")
	default:
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
		delimiter := detectNokiaDelimiter(string(content))
		reader := csv.NewReader(bytes.NewReader(content))
		reader.Comma = delimiter
		reader.FieldsPerRecord = -1
		return reader.ReadAll()
	}
}

func detectNokiaDelimiter(content string) rune {
	line := content
	if index := strings.IndexAny(content, "\r\n"); index >= 0 {
		line = content[:index]
	}
	counts := map[rune]int{';': strings.Count(line, ";"), ',': strings.Count(line, ","), '\t': strings.Count(line, "\t")}
	best := ';'
	bestCount := -1
	for delimiter, count := range counts {
		if count > bestCount {
			best, bestCount = delimiter, count
		}
	}
	return best
}

func normalizeNokiaHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
	).Replace(value)
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

func parseNokiaBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "sim", "s", "x", "yes", "y", "true", "on", "concluido", "concluida", "aprovado", "aprovada", "ok":
		return true
	default:
		return false
	}
}

func normalizeNokiaImportDate(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	for _, layout := range []string{"02/01/2006", "02-01-2006", "2006-01-02", "02/01/06", "2006/01/02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed.Format("2006-01-02"), true
		}
	}
	if serial, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64); err == nil && serial > 0 {
		if parsed, err := excelize.ExcelDateToTime(serial, false); err == nil {
			return parsed.Format("2006-01-02"), true
		}
	}
	return "", false
}

func nokiaChecklistStatus(value string) string {
	switch value {
	case "sent":
		return "Aprovado"
	case "analysis":
		return "Análise TIM"
	case "rejected":
		return "Rejeitado"
	default:
		return "Pendente"
	}
}

func formatNokiaDateTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("02/01/2006 15:04")
}

func formatNokiaInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func nokiaReturnDate(started *time.Time, days *int) string {
	if started == nil || days == nil {
		return ""
	}
	return started.AddDate(0, 0, *days).Format("02/01/2006")
}

func nokiaRejectionSummary(rejections []modules.NokiaRejection, field string) string {
	parts := []string{}
	index := 0
	for _, rejection := range rejections {
		if rejection.ChecklistField != field {
			continue
		}
		index++
		returnDate := rejection.StartedAt.AddDate(0, 0, rejection.ReturnDays).Format("02/01/2006")
		status := "Ativa"
		if rejection.ResolvedAt != nil {
			status = "Encerrada"
		}
		fieldReturn := "Não"
		if rejection.RequiresFieldReturn {
			fieldReturn = "Sim"
		}
		parts = append(parts, fmt.Sprintf("#%d %s — %d dias — %s — retorno a campo: %s — %s",
			index, rejection.Reason, rejection.ReturnDays, returnDate, fieldReturn, status))
	}
	return strings.Join(parts, "; ")
}

func mapString(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func firstN(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}

func chiURLParam(r *http.Request, name string) string {
	return strings.TrimSpace(chi.URLParam(r, name))
}
