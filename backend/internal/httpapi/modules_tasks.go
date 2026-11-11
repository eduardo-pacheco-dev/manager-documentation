package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

func taskView(value string) string {
	switch value {
	case "my_day", "important", "planned", "all", "completed", "today", "week", "list":
		return value
	default:
		return "my_day"
	}
}

func (s *Server) tasksIndex(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	query := r.URL.Query()
	view := taskView(query.Get("view"))
	var listID int64
	if activeList := query.Get("list"); activeList != "" {
		parsed, err := strconv.ParseInt(activeList, 10, 64)
		if err != nil {
			writeError(w, http.StatusNotFound, "Lista não encontrada.", nil)
			return
		}
		if _, err := s.Modules.FindTodoList(user.ID, parsed); err != nil {
			writeError(w, http.StatusNotFound, "Lista não encontrada.", nil)
			return
		}
		listID = parsed
		view = "list"
	}
	lists, err := s.Modules.TodoLists(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as listas.", nil)
		return
	}
	today := time.Now().Format("2006-01-02")
	weekEnd := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	counts, err := s.Modules.TaskCounts(user.ID, today, weekEnd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as tarefas.", nil)
		return
	}
	sortValue := query.Get("sort")
	tasks, err := s.Modules.TodoTasks(user.ID, modules.TaskFilter{
		View: view, ListID: listID, Search: query.Get("search"), Sort: sortValue, Today: today, WeekEnd: weekEnd,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as tarefas.", nil)
		return
	}
	completed := []modules.TodoTask{}
	if view != "completed" {
		completed, _ = s.Modules.TodoTasks(user.ID, modules.TaskFilter{
			View: "completed", ListID: listID, Search: query.Get("search"), Sort: sortValue, Today: today, WeekEnd: weekEnd,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"lists": lists, "tasks": tasks, "completed": completed, "counts": counts,
			"view": view, "list_id": listID, "sort": sortValue, "search": query.Get("search"),
			"today": today,
		},
	})
}

type taskPayload struct {
	Title       string `json:"title"`
	TodoListID  int64  `json:"todo_list_id"`
	Notes       string `json:"notes"`
	DueAt       string `json:"due_at"`
	RemindAt    string `json:"remind_at"`
	RepeatType  string `json:"repeat_type"`
	IsImportant bool   `json:"is_important"`
	MyDay       bool   `json:"my_day"`
}

func (s *Server) tasksCreate(w http.ResponseWriter, r *http.Request) {
	var payload taskPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Title) == "" {
		writeValidation(w, validationErrors{"title": "O título é obrigatório."})
		return
	}
	if payload.TodoListID == 0 {
		writeValidation(w, validationErrors{"todo_list_id": "Selecione uma lista."})
		return
	}
	user := currentUser(r)
	task, err := s.Modules.CreateTodoTask(user.ID, modules.TaskInput{
		TodoListID: payload.TodoListID, Title: payload.Title, Notes: payload.Notes, DueAt: payload.DueAt,
		RemindAt: payload.RemindAt, RepeatType: payload.RepeatType, IsImportant: payload.IsImportant, MyDay: payload.MyDay,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Tarefa criada com sucesso.", "data": map[string]any{"task": task}})
}

func (s *Server) tasksUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	var payload taskPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	user := currentUser(r)
	task, err := s.Modules.UpdateTodoTask(user.ID, id, modules.TaskInput{
		TodoListID: payload.TodoListID, Title: payload.Title, Notes: payload.Notes, DueAt: payload.DueAt,
		RemindAt: payload.RemindAt, RepeatType: payload.RepeatType, IsImportant: payload.IsImportant, MyDay: payload.MyDay,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível atualizar a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Tarefa atualizada.", "data": map[string]any{"task": task}})
}

func (s *Server) tasksShow(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	task, err := s.Modules.FindTodoTask(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"task": task}})
}

func (s *Server) tasksDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	if err := s.Modules.DeleteTodoTask(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tarefa excluída."})
}

func (s *Server) tasksToggle(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	task, message, err := s.Modules.ToggleTodoTask(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	if message == "" {
		if task.CompletedAt != nil {
			message = "Tarefa concluída."
		} else {
			message = "Tarefa reaberta."
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": message, "data": map[string]any{"task": task}})
}

func (s *Server) tasksToggleImportant(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	task, err := s.Modules.ToggleTodoImportant(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Importância atualizada.", "data": map[string]any{"task": task}})
}

func (s *Server) tasksToggleMyDay(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	task, err := s.Modules.ToggleTodoMyDay(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Meu Dia atualizado.", "data": map[string]any{"task": task}})
}

func (s *Server) tasksMove(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		TodoListID int64  `json:"todo_list_id"`
		Direction  string `json:"direction"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	message, err := s.Modules.MoveTodoTask(currentUser(r).ID, id, payload.TodoListID, payload.Direction)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível mover a tarefa.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": message})
}

func (s *Server) tasksCreateList(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Icon  string `json:"icon"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeValidation(w, validationErrors{"name": "O nome da lista é obrigatório."})
		return
	}
	list, err := s.Modules.CreateTodoList(currentUser(r).ID, payload.Name, payload.Color, payload.Icon)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar a lista.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Lista criada.", "data": map[string]any{"list": list}})
}

func (s *Server) tasksUpdateList(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Icon  string `json:"icon"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.UpdateTodoList(currentUser(r).ID, id, payload.Name, payload.Color, payload.Icon); err != nil {
		writeError(w, http.StatusNotFound, "Lista não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Lista atualizada."})
}

func (s *Server) tasksDeleteList(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteTodoList(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusNotFound, "Lista não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Lista excluída."})
}

func (s *Server) tasksCreateStep(w http.ResponseWriter, r *http.Request) {
	taskID, _ := parseID(r, "id")
	var payload struct {
		Title string `json:"title"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Title) == "" {
		writeValidation(w, validationErrors{"title": "Informe o título da etapa."})
		return
	}
	step, err := s.Modules.CreateTodoStep(currentUser(r).ID, taskID, payload.Title)
	if err != nil {
		writeError(w, http.StatusNotFound, "Tarefa não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Etapa adicionada.", "data": map[string]any{"step": step}})
}

func (s *Server) tasksUpdateStep(w http.ResponseWriter, r *http.Request) {
	taskID, _ := parseID(r, "id")
	stepID, _ := parseID(r, "stepId")
	var payload struct {
		Title     string `json:"title"`
		Completed *bool  `json:"completed"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := s.Modules.UpdateTodoStep(currentUser(r).ID, taskID, stepID, payload.Title, payload.Completed); err != nil {
		writeError(w, http.StatusNotFound, "Etapa não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Etapa atualizada."})
}

func (s *Server) tasksDeleteStep(w http.ResponseWriter, r *http.Request) {
	taskID, _ := parseID(r, "id")
	stepID, _ := parseID(r, "stepId")
	if err := s.Modules.DeleteTodoStep(currentUser(r).ID, taskID, stepID); err != nil {
		writeError(w, http.StatusNotFound, "Etapa não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Etapa removida."})
}

func (s *Server) tasksExport(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	query := r.URL.Query()
	today := time.Now().Format("2006-01-02")
	tasks, err := s.Modules.TodoTasks(user.ID, modules.TaskFilter{
		View: taskView(query.Get("view")), Search: query.Get("search"), Sort: query.Get("sort"),
		Today: today, WeekEnd: time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar as tarefas.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="tarefas.csv"`)
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"Tarefa", "Lista", "Status", "Importante", "Meu Dia", "Vencimento", "Lembrete", "Repetição", "Etapas", "Anotações", "Concluída em", "Criada em"})
	for _, task := range tasks {
		status := "Pendente"
		if task.CompletedAt != nil {
			status = "Concluída"
		}
		important := "Não"
		if task.IsImportant {
			important = "Sim"
		}
		myDay := "Não"
		if task.MyDayOn != nil && *task.MyDayOn == today {
			myDay = "Sim"
		}
		completedSteps := 0
		for _, step := range task.Steps {
			if step.CompletedAt != nil {
				completedSteps++
			}
		}
		steps := ""
		if len(task.Steps) > 0 {
			steps = fmt.Sprintf("%d/%d", completedSteps, len(task.Steps))
		}
		remind := ""
		if task.RemindAt != nil {
			remind = task.RemindAt.Format("02/01/2006 15:04")
		}
		completed := ""
		if task.CompletedAt != nil {
			completed = task.CompletedAt.Format("02/01/2006 15:04")
		}
		_ = writer.Write([]string{task.Title, task.ListName, status, important, myDay, derefString(task.DueAt), remind, task.RepeatType, steps, derefString(task.Notes), completed, task.CreatedAt.Format("02/01/2006 15:04")})
	}
	writer.Flush()
}

var _ = chi.URLParam
