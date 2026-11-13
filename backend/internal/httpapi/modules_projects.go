package httpapi

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

func projectError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, modules.ErrValidation):
		message := strings.TrimSpace(strings.TrimPrefix(err.Error(), modules.ErrValidation.Error()+":"))
		if message == "" {
			message = fallback
		}
		writeError(w, http.StatusUnprocessableEntity, message, nil)
	case errors.Is(err, modules.ErrNotFound):
		writeError(w, http.StatusNotFound, fallback, nil)
	default:
		writeError(w, http.StatusInternalServerError, fallback, nil)
	}
}

func (s *Server) projectsIndex(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("archived") == "1"
	projects, err := s.Modules.ListProjects(includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os projetos.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"projects": projects}})
}

type projectPayload struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	Currency    string  `json:"currency"`
	Budget      float64 `json:"budget"`
	WorkingDays string  `json:"working_days"`
	HoursPerDay float64 `json:"hours_per_day"`
	FlowMode    string  `json:"flow_mode"`
}

func (p projectPayload) input() modules.ProjectInput {
	return modules.ProjectInput{
		Name: p.Name, Description: p.Description, Status: p.Status, StartDate: p.StartDate,
		EndDate: p.EndDate, Currency: p.Currency, Budget: p.Budget, WorkingDays: p.WorkingDays,
		HoursPerDay: p.HoursPerDay, FlowMode: p.FlowMode,
	}
}

func (s *Server) projectsCreate(w http.ResponseWriter, r *http.Request) {
	var payload projectPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	project, err := s.Modules.CreateProject(currentUser(r).ID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível criar o projeto.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Projeto criado com sucesso.", "data": map[string]any{"project": project}})
}

func (s *Server) projectsShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	bundle, err := s.Modules.ProjectBundle(id)
	if err != nil {
		projectError(w, err, "Projeto não encontrado.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": bundle})
}

func (s *Server) projectsUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload projectPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	project, err := s.Modules.UpdateProject(id, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível atualizar o projeto.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Projeto atualizado.", "data": map[string]any{"project": project}})
}

func (s *Server) projectsDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteProject(id); err != nil {
		projectError(w, err, "Não foi possível excluir o projeto.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Projeto excluído."})
}

func (s *Server) projectsToggleArchive(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	project, err := s.Modules.ToggleProjectArchived(id)
	if err != nil {
		projectError(w, err, "Não foi possível arquivar o projeto.")
		return
	}
	message := "Projeto arquivado."
	if !project.Archived {
		message = "Projeto restaurado."
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": message, "data": map[string]any{"project": project}})
}

type projectTaskPayload struct {
	ParentID        int64   `json:"parent_id"`
	Name            string  `json:"name"`
	Notes           string  `json:"notes"`
	TaskType        string  `json:"task_type"`
	StartDate       string  `json:"start_date"`
	EndDate         string  `json:"end_date"`
	DurationDays    int     `json:"duration_days"`
	Progress        int     `json:"progress"`
	Priority        string  `json:"priority"`
	EffortHours     float64 `json:"effort_hours"`
	FixedCost       float64 `json:"fixed_cost"`
	AssigneeID      int64   `json:"assignee_id"`
	PipelineStageID int64   `json:"pipeline_stage_id"`
	Position        int     `json:"position"`
}

func (p projectTaskPayload) input() modules.ProjectTaskInput {
	return modules.ProjectTaskInput{
		ParentID: p.ParentID, Name: p.Name, Notes: p.Notes, TaskType: p.TaskType,
		StartDate: p.StartDate, EndDate: p.EndDate, DurationDays: p.DurationDays, Progress: p.Progress,
		Priority: p.Priority, EffortHours: p.EffortHours, FixedCost: p.FixedCost,
		AssigneeID: p.AssigneeID, PipelineStageID: p.PipelineStageID, Position: p.Position,
	}
}

func (s *Server) projectTaskCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload projectTaskPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	task, err := s.Modules.CreateProjectTask(projectID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível criar a tarefa.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Tarefa criada.", "data": map[string]any{"task": task}})
}

func (s *Server) projectTaskUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	taskID, ok := parseID(r, "taskId")
	if !ok {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	var payload projectTaskPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	task, err := s.Modules.UpdateProjectTask(projectID, taskID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível atualizar a tarefa.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Tarefa atualizada.", "data": map[string]any{"task": task}})
}

func (s *Server) projectTaskDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	taskID, ok := parseID(r, "taskId")
	if !ok {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	if err := s.Modules.DeleteProjectTask(projectID, taskID); err != nil {
		projectError(w, err, "Não foi possível excluir a tarefa.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tarefa excluída."})
}

func (s *Server) projectTaskMove(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	taskID, ok := parseID(r, "taskId")
	if !ok {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	var payload struct {
		Direction string `json:"direction"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.MoveProjectTask(projectID, taskID, payload.Direction); err != nil {
		projectError(w, err, "Não foi possível mover a tarefa.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tarefa movida."})
}

func (s *Server) projectTaskMoveStage(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	taskID, ok := parseID(r, "taskId")
	if !ok {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	var payload struct {
		StageID int64 `json:"stage_id"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	task, err := s.Modules.MoveProjectTaskToStage(projectID, taskID, payload.StageID)
	if err != nil {
		projectError(w, err, "Não foi possível mover o card.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Card movido.", "data": map[string]any{"task": task}})
}

type projectStagePayload struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	WIPLimit int    `json:"wip_limit"`
	IsDone   bool   `json:"is_done"`
}

func (p projectStagePayload) input() modules.ProjectPipelineStageInput {
	return modules.ProjectPipelineStageInput{
		Name: p.Name, Color: p.Color, WIPLimit: p.WIPLimit, IsDone: p.IsDone,
	}
}

func (s *Server) projectStageCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload projectStagePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	stage, err := s.Modules.CreatePipelineStage(projectID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível criar a etapa.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Etapa criada.", "data": map[string]any{"stage": stage}})
}

func (s *Server) projectStageUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	stageID, ok := parseID(r, "stageId")
	if !ok {
		writeError(w, http.StatusNotFound, "Etapa não encontrada.", nil)
		return
	}
	var payload projectStagePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	stage, err := s.Modules.UpdatePipelineStage(projectID, stageID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível atualizar a etapa.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Etapa atualizada.", "data": map[string]any{"stage": stage}})
}

func (s *Server) projectStageMove(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	stageID, ok := parseID(r, "stageId")
	if !ok {
		writeError(w, http.StatusNotFound, "Etapa não encontrada.", nil)
		return
	}
	var payload struct {
		Direction string `json:"direction"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.MovePipelineStage(projectID, stageID, payload.Direction); err != nil {
		projectError(w, err, "Não foi possível mover a etapa.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Etapa movida."})
}

func (s *Server) projectStageDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	stageID, ok := parseID(r, "stageId")
	if !ok {
		writeError(w, http.StatusNotFound, "Etapa não encontrada.", nil)
		return
	}
	if err := s.Modules.DeletePipelineStage(projectID, stageID); err != nil {
		projectError(w, err, "Não foi possível excluir a etapa.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Etapa excluída."})
}

func (s *Server) projectDependencyCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload struct {
		PredecessorID int64  `json:"predecessor_id"`
		SuccessorID   int64  `json:"successor_id"`
		Type          string `json:"type"`
		LagDays       int    `json:"lag_days"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	dependency, err := s.Modules.CreateProjectDependency(projectID, payload.PredecessorID, payload.SuccessorID, payload.Type, payload.LagDays)
	if err != nil {
		projectError(w, err, "Não foi possível criar a dependência.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Dependência criada.", "data": map[string]any{"dependency": dependency}})
}

func (s *Server) projectDependencyDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	dependencyID, ok := parseID(r, "dependencyId")
	if !ok {
		writeError(w, http.StatusNotFound, "Dependência não encontrada.", nil)
		return
	}
	if err := s.Modules.DeleteProjectDependency(projectID, dependencyID); err != nil {
		projectError(w, err, "Não foi possível excluir a dependência.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Dependência excluída."})
}

type projectResourcePayload struct {
	CollaboratorID int64   `json:"collaborator_id"`
	Name           string  `json:"name"`
	Role           string  `json:"role"`
	Email          string  `json:"email"`
	CostHour       float64 `json:"cost_hour"`
	MaxUnits       int     `json:"max_units"`
	Color          string  `json:"color"`
}

func (p projectResourcePayload) input() modules.ProjectResourceInput {
	return modules.ProjectResourceInput{
		CollaboratorID: p.CollaboratorID, Name: p.Name, Role: p.Role, Email: p.Email,
		CostHour: p.CostHour, MaxUnits: p.MaxUnits, Color: p.Color,
	}
}

func (s *Server) projectResourceCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload projectResourcePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	resource, err := s.Modules.CreateProjectResource(projectID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível criar o recurso.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Recurso criado.", "data": map[string]any{"resource": resource}})
}

func (s *Server) projectResourceUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	resourceID, ok := parseID(r, "resourceId")
	if !ok {
		writeError(w, http.StatusNotFound, "Recurso não encontrado.", nil)
		return
	}
	var payload projectResourcePayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	resource, err := s.Modules.UpdateProjectResource(projectID, resourceID, payload.input())
	if err != nil {
		projectError(w, err, "Não foi possível atualizar o recurso.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Recurso atualizado.", "data": map[string]any{"resource": resource}})
}

func (s *Server) projectResourceDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	resourceID, ok := parseID(r, "resourceId")
	if !ok {
		writeError(w, http.StatusNotFound, "Recurso não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteProjectResource(projectID, resourceID); err != nil {
		projectError(w, err, "Não foi possível excluir o recurso.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Recurso excluído."})
}

func (s *Server) projectAssignmentCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload struct {
		TaskID       int64   `json:"task_id"`
		ResourceID   int64   `json:"resource_id"`
		Units        int     `json:"units"`
		PlannedHours float64 `json:"planned_hours"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	assignment, err := s.Modules.CreateProjectAssignment(projectID, modules.ProjectAssignmentInput{
		TaskID: payload.TaskID, ResourceID: payload.ResourceID, Units: payload.Units, PlannedHours: payload.PlannedHours,
	})
	if err != nil {
		projectError(w, err, "Não foi possível alocar o recurso.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Recurso alocado.", "data": map[string]any{"assignment": assignment}})
}

func (s *Server) projectAssignmentDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	assignmentID, ok := parseID(r, "assignmentId")
	if !ok {
		writeError(w, http.StatusNotFound, "Alocação não encontrada.", nil)
		return
	}
	if err := s.Modules.DeleteProjectAssignment(projectID, assignmentID); err != nil {
		projectError(w, err, "Não foi possível remover a alocação.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Alocação removida."})
}

func (s *Server) projectHolidayCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload struct {
		Date string `json:"date"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	holiday, err := s.Modules.CreateProjectHoliday(projectID, payload.Date, payload.Name)
	if err != nil {
		projectError(w, err, "Não foi possível cadastrar o feriado.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Feriado cadastrado.", "data": map[string]any{"holiday": holiday}})
}

func (s *Server) projectHolidayDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	holidayID, ok := parseID(r, "holidayId")
	if !ok {
		writeError(w, http.StatusNotFound, "Feriado não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteProjectHoliday(projectID, holidayID); err != nil {
		projectError(w, err, "Não foi possível excluir o feriado.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Feriado excluído."})
}

func (s *Server) projectBaselineCreate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	baseline, err := s.Modules.CreateProjectBaseline(projectID, payload.Name)
	if err != nil {
		projectError(w, err, "Não foi possível criar a linha de base.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Linha de base criada.", "data": map[string]any{"baseline": baseline}})
}

func (s *Server) projectBaselineDelete(w http.ResponseWriter, r *http.Request) {
	projectID, _ := parseID(r, "id")
	baselineID, ok := parseID(r, "baselineId")
	if !ok {
		writeError(w, http.StatusNotFound, "Linha de base não encontrada.", nil)
		return
	}
	if err := s.Modules.DeleteProjectBaseline(projectID, baselineID); err != nil {
		projectError(w, err, "Não foi possível excluir a linha de base.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Linha de base excluída."})
}

func (s *Server) projectsExport(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Projeto não encontrado.", nil)
		return
	}
	bundle, err := s.Modules.ProjectBundle(id)
	if err != nil {
		projectError(w, err, "Projeto não encontrado.")
		return
	}
	filename := fmt.Sprintf("projeto-%d.csv", bundle.Project.ID)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{
		"WBS", "Tarefa", "Tipo", "Início", "Fim", "Duração (dias)", "Progresso (%)",
		"Prioridade", "Responsável", "Recursos", "Custo", "Crítica", "Início base", "Fim base", "Variação (dias)",
	})
	for _, task := range bundle.Tasks {
		taskType := map[string]string{"task": "Tarefa", "summary": "Resumo", "milestone": "Marco"}[task.TaskType]
		critical := "Não"
		if task.IsCritical {
			critical = "Sim"
		}
		variance := task.EndVariance
		_ = writer.Write([]string{
			task.WBS, task.Name, taskType, task.ScheduledFrom, task.ScheduledTo,
			fmt.Sprintf("%d", task.EffectiveDays), fmt.Sprintf("%d", task.Progress), task.Priority,
			task.AssigneeName, strings.Join(task.Resources, ", "), fmt.Sprintf("%.2f", task.TotalCost),
			critical, derefString(task.BaselineStart), derefString(task.BaselineEnd), fmt.Sprintf("%d", variance),
		})
	}
	writer.Flush()
}

var _ = chi.URLParam
