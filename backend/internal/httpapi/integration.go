package httpapi

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

/* --------------------------------------------------------------------------
 * Cross-module integration: related work, quick create and notifications
 * ----------------------------------------------------------------------- */

func (s *Server) workContext(w http.ResponseWriter, r *http.Request) {
	resourceType := strings.TrimSpace(r.URL.Query().Get("resource_type"))
	resourceID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("resource_id")), 10, 64)
	if err != nil || resourceID <= 0 {
		writeValidation(w, validationErrors{"resource_id": "Informe o recurso."})
		return
	}
	context, err := s.Modules.WorkContext(currentUser(r).ID, resourceType, resourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Recurso não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": context})
}

func (s *Server) createWorkLink(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Module       string `json:"module"`
		EntityID     int64  `json:"entity_id"`
		ResourceType string `json:"resource_type"`
		ResourceID   int64  `json:"resource_id"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.EntityID <= 0 || payload.ResourceID <= 0 ||
		!modules.WorkModules[payload.Module] || !modules.WorkResources[payload.ResourceType] {
		writeValidation(w, validationErrors{"module": "Informe um item e um recurso válidos."})
		return
	}
	link, err := s.Modules.CreateWorkLink(currentUser(r).ID, payload.Module, payload.EntityID, payload.ResourceType, payload.ResourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Item ou recurso não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Item vinculado.", "data": map[string]any{"link": link}})
}

func (s *Server) deleteWorkLink(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Vínculo não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteWorkLink(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusNotFound, "Vínculo não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Vínculo removido."})
}

type workItemPayload struct {
	ResourceType    string `json:"resource_type"`
	ResourceID      int64  `json:"resource_id"`
	Kind            string `json:"kind"`
	Title           string `json:"title"`
	Notes           string `json:"notes"`
	DueDate         string `json:"due_date"`
	Priority        string `json:"priority"`
	AssigneeID      *int64 `json:"assignee_id"`
	StartAt         string `json:"start_at"`
	EndAt           string `json:"end_at"`
	AllDay          bool   `json:"is_all_day"`
	DurationMinutes int    `json:"duration_minutes"`
}

func (s *Server) createWorkItem(w http.ResponseWriter, r *http.Request) {
	var payload workItemPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	user := currentUser(r)
	if strings.TrimSpace(payload.Title) == "" {
		writeValidation(w, validationErrors{"title": "Informe o título."})
		return
	}
	resource, err := s.Modules.WorkResource(payload.ResourceType, payload.ResourceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Recurso não encontrado.", nil)
		return
	}
	entity := map[string]any{}
	var entityID int64
	var module string
	switch payload.Kind {
	case "task":
		module = "task"
		listID, err := s.defaultTodoList(user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível preparar a lista de tarefas.", nil)
			return
		}
		task, err := s.Modules.CreateTodoTask(user.ID, modules.TaskInput{
			TodoListID: listID, Title: payload.Title, Notes: payload.Notes, DueAt: payload.DueDate,
		})
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar a tarefa.", nil)
			return
		}
		entityID = task.ID
		entity = map[string]any{"task": task}
	case "follow_up":
		module = "follow_up"
		status := "pendente"
		priority := payload.Priority
		if priority == "" {
			priority = "media"
		}
		item, err := s.Modules.CreateFollowUp(user.ID, modules.FollowUpInput{
			Title: payload.Title, Description: payload.Notes, Status: status, Priority: priority,
			DueDate: payload.DueDate, AssigneeID: payload.AssigneeID,
		})
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar o follow-up.", nil)
			return
		}
		entityID = item.ID
		entity = map[string]any{"follow_up": item}
		s.notifyFollowUpAssignment(user.ID, item)
	case "agenda_event":
		module = "agenda_event"
		event, err := s.createQuickAgendaEvent(user.ID, payload, resource)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar o evento.", nil)
			return
		}
		entityID = event.ID
		entity = map[string]any{"event": event}
	default:
		writeValidation(w, validationErrors{"kind": "Escolha tarefa, follow-up ou evento."})
		return
	}
	link, err := s.Modules.CreateWorkLink(user.ID, module, entityID, payload.ResourceType, payload.ResourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Item criado, mas não foi possível vinculá-lo.", nil)
		return
	}
	entity["link"] = link
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Item criado e vinculado ao recurso.",
		"data":    entity,
	})
}

func (s *Server) defaultTodoList(userID int64) (int64, error) {
	var listID int64
	err := s.Modules.DB.Get(&listID, `SELECT id FROM todo_lists
		WHERE user_id = ? AND deleted_at IS NULL ORDER BY position ASC, id ASC LIMIT 1`, userID)
	if err == nil {
		return listID, nil
	}
	list, err := s.Modules.CreateTodoList(userID, "Geral", "indigo", "check-circle")
	if err != nil {
		return 0, err
	}
	return list.ID, nil
}

func (s *Server) createQuickAgendaEvent(userID int64, payload workItemPayload, resource modules.WorkResource) (modules.AgendaEvent, error) {
	if err := s.Modules.EnsureAgendaCalendars(userID); err != nil {
		return modules.AgendaEvent{}, err
	}
	calendars, err := s.Modules.AgendaCalendars(userID)
	if err != nil {
		return modules.AgendaEvent{}, err
	}
	if len(calendars) == 0 {
		return modules.AgendaEvent{}, modules.ErrValidation
	}
	calendarID := int64(0)
	for _, calendar := range calendars {
		if calendar.Role == "reader" || calendar.Role == "free_busy_reader" {
			continue
		}
		if calendarID == 0 || calendar.IsDefault {
			calendarID = calendar.ID
		}
		if calendar.IsDefault {
			break
		}
	}
	if calendarID == 0 {
		return modules.AgendaEvent{}, modules.ErrForbidden
	}
	start := time.Now().Add(time.Hour)
	if parsed, err := parseDateTime(payload.StartAt); err == nil {
		start = parsed
	}
	duration := time.Duration(payload.DurationMinutes) * time.Minute
	if duration <= 0 {
		duration = time.Hour
	}
	end := start.Add(duration)
	if parsed, err := parseDateTime(payload.EndAt); err == nil {
		end = parsed
	}
	if end.Before(start) {
		end = start.Add(duration)
	}
	location := strings.TrimSpace(strings.Join([]string{resource.Label, resource.Subtitle}, " — "))
	description := strings.TrimSpace(payload.Notes)
	if description == "" {
		description = "Compromisso criado a partir de " + resource.URL + "."
	}
	return s.Modules.CreateAgendaEvent(userID, modules.AgendaEventInput{
		CalendarID: calendarID, Title: payload.Title, Description: description, Location: location,
		StartAt: start.Format("2006-01-02T15:04"), EndAt: end.Format("2006-01-02T15:04"),
		IsAllDay: payload.AllDay, Status: "confirmed", Visibility: "default", Transparency: "opaque",
	})
}

/* --------------------------------------------------------------------------
 * Event-driven notifications
 * ----------------------------------------------------------------------- */

func (s *Server) notifyUserByEmail(actorID int64, email, title, body, link, level string) {
	if strings.TrimSpace(email) == "" {
		return
	}
	user, err := s.Store.FindUserByEmail(email)
	if err != nil || !user.IsActive || user.ID == actorID {
		return
	}
	if err := s.Notifier.Notify(user, title, body, link, level); err != nil {
		log.Printf("notification to %s: %v", email, err)
	}
}

func (s *Server) notifyFollowUpAssignment(actorID int64, item modules.FollowUp) {
	if item.AssigneeEmail == nil {
		return
	}
	body := fmt.Sprintf("%s · Prioridade %s", item.Title, followUpPriorityLabel(item.Priority))
	if item.DueDate != nil && *item.DueDate != "" {
		body += " · Prazo " + *item.DueDate
	}
	s.notifyUserByEmail(actorID, *item.AssigneeEmail, "Follow-up atribuído a você", body,
		fmt.Sprintf("/follow-ups?followup=%d", item.ID), "info")
}

func (s *Server) notifyFollowUpStatusChange(actorID int64, item modules.FollowUp, previousStatus string) {
	if item.Status == previousStatus {
		return
	}
	level := "info"
	if item.Status == "concluido" {
		level = "success"
	}
	title := "Follow-up atualizado"
	if item.Status == "concluido" {
		title = "Follow-up concluído"
	}
	body := fmt.Sprintf("%s · %s → %s", item.Title, followUpStatusLabel(previousStatus), followUpStatusLabel(item.Status))
	link := fmt.Sprintf("/follow-ups?followup=%d", item.ID)
	if item.AssigneeEmail != nil {
		s.notifyUserByEmail(actorID, *item.AssigneeEmail, title, body, link, level)
	}
	if item.CreatedBy != nil && *item.CreatedBy != actorID {
		if creator, err := s.Store.FindUserByID(*item.CreatedBy); err == nil && creator.IsActive {
			if spamErr := s.Notifier.Notify(creator, title, body, link, level); spamErr != nil {
				log.Printf("notification follow-up %d: %v", item.ID, spamErr)
			}
		}
	}
}

const followUpStatusLabels = "pendente|em_andamento|concluido|cancelado"

func (s *Server) notifyFollowUpComment(actorID int64, item modules.FollowUp, body string) {
	preview := strings.TrimSpace(body)
	if len([]rune(preview)) > 120 {
		preview = string([]rune(preview)[:120]) + "…"
	}
	title := "Nova atualização em follow-up"
	message := item.Title + " · " + preview
	link := fmt.Sprintf("/follow-ups?followup=%d", item.ID)
	if item.AssigneeEmail != nil {
		s.notifyUserByEmail(actorID, *item.AssigneeEmail, title, message, link, "info")
	}
	if item.CreatedBy != nil && *item.CreatedBy != actorID {
		if creator, err := s.Store.FindUserByID(*item.CreatedBy); err == nil && creator.IsActive {
			_ = s.Notifier.Notify(creator, title, message, link, "info")
		}
	}
}

func (s *Server) notifyAgendaInvite(actorID int64, event modules.AgendaEvent) {
	link := fmt.Sprintf("/agenda?event=%d", event.ID)
	title := "Novo evento na agenda"
	body := event.Title + " · " + event.StartAt.Local().Format("02/01/2006 15:04")
	seen := map[int64]bool{actorID: true}
	for _, attendee := range event.Attendees {
		if attendee.UserID != nil {
			if seen[*attendee.UserID] {
				continue
			}
			seen[*attendee.UserID] = true
			user, err := s.Store.FindUserByID(*attendee.UserID)
			if err != nil || !user.IsActive {
				continue
			}
			_ = s.Notifier.Notify(user, title, body, link, "info")
			continue
		}
		s.notifyUserByEmail(actorID, attendee.Email, title, body, link, "info")
	}
}

func (s *Server) notifyNokiaRejection(actorID, siteID int64, reason string) {
	var row struct {
		SiteID   string  `db:"site_id"`
		ROMEmail *string `db:"rom_email"`
	}
	err := s.Modules.DB.Get(&row, `SELECT s.site_id, r.email AS rom_email
		FROM nokia_sites s LEFT JOIN nokia_roms r ON r.id = s.nokia_rom_id WHERE s.id = ?`, siteID)
	if err != nil || row.ROMEmail == nil {
		return
	}
	body := fmt.Sprintf("Site %s · %s", row.SiteID, strings.TrimSpace(reason))
	s.notifyUserByEmail(actorID, *row.ROMEmail, "Site Nokia rejeitado", body,
		fmt.Sprintf("/nokia/%d", siteID), "warning")
}

/* --------------------------------------------------------------------------
 * Scheduled reminders (cross-module work center)
 * ----------------------------------------------------------------------- */

// StartIntegrationWorker runs cross-module reminders immediately and then on
// every interval. The returned function stops the worker.
func (s *Server) StartIntegrationWorker(interval time.Duration) func() {
	stop := make(chan struct{})
	go func() {
		s.runWorkReminders()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				s.runWorkReminders()
			}
		}
	}()
	return func() { close(stop) }
}

func (s *Server) runWorkReminders() {
	users := []models.User{}
	if err := s.Modules.DB.Select(&users, `SELECT * FROM users WHERE is_active = 1`); err != nil {
		log.Printf("reminders: users: %v", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	week := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	for _, user := range users {
		type taskReminder struct {
			ID    int64  `db:"id"`
			Title string `db:"title"`
			DueAt string `db:"due_at"`
		}
		tasks := []taskReminder{}
		if err := s.Modules.DB.Select(&tasks, `SELECT id, title, due_at FROM todo_tasks
			WHERE user_id = ? AND completed_at IS NULL AND due_at IS NOT NULL AND due_at <= ?
			ORDER BY due_at ASC`, user.ID, week); err == nil {
			for _, task := range tasks {
				level := "warning"
				title := "Tarefa vencendo"
				if task.DueAt < today {
					level = "danger"
					title = "Tarefa atrasada"
				}
				s.notifyOnce(user, fmt.Sprintf("task-due:%d:%s", task.ID, today), title,
					fmt.Sprintf("%s · vence em %s", task.Title, task.DueAt),
					fmt.Sprintf("/tasks?task=%d", task.ID), level)
			}
		}
		type followReminder struct {
			ID      int64  `db:"id"`
			Title   string `db:"title"`
			DueDate string `db:"due_date"`
		}
		follows := []followReminder{}
		if err := s.Modules.DB.Select(&follows, `SELECT f.id, f.title, f.due_date
			FROM follow_ups f JOIN follow_up_assignees a ON a.id = f.assignee_id
			WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ?
				AND f.due_date IS NOT NULL AND f.due_date <= ? ORDER BY f.due_date ASC`,
			strings.ToLower(user.Email), week); err == nil {
			for _, item := range follows {
				level := "warning"
				title := "Follow-up vencendo"
				if item.DueDate < today {
					level = "danger"
					title = "Follow-up atrasado"
				}
				s.notifyOnce(user, fmt.Sprintf("follow-up-due:%d:%s", item.ID, today), title,
					fmt.Sprintf("%s · prazo %s", item.Title, item.DueDate),
					fmt.Sprintf("/follow-ups?followup=%d", item.ID), level)
			}
		}
		type eventReminder struct {
			ID      int64     `db:"id"`
			Title   string    `db:"title"`
			StartAt time.Time `db:"start_at"`
		}
		events := []eventReminder{}
		if err := s.Modules.DB.Select(&events, `SELECT DISTINCT e.id, e.title, e.start_at
			FROM agenda_events e
			LEFT JOIN agenda_event_attendees a ON a.agenda_event_id = e.id
			WHERE e.status <> 'cancelled' AND e.start_at BETWEEN ? AND ?
				AND (e.user_id = ? OR a.user_id = ? OR a.email = ?)
			ORDER BY e.start_at ASC`, time.Now(), time.Now().Add(24*time.Hour), user.ID, user.ID, user.Email); err == nil {
			for _, event := range events {
				s.notifyOnce(user, fmt.Sprintf("event-soon:%d:%s", event.ID, today), "Compromisso em breve",
					fmt.Sprintf("%s · %s", event.Title, event.StartAt.Local().Format("02/01/2006 15:04")),
					fmt.Sprintf("/agenda?event=%d", event.ID), "info")
			}
		}
	}
}

func (s *Server) notifyOnce(user models.User, key, title, body, link, level string) {
	if _, err := s.Notifier.NotifyOnce(user, key, title, body, link, level); err != nil {
		log.Printf("reminder notification for %s: %v", user.Email, err)
	}
}
