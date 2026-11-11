package httpapi

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

func (s *Server) agendaIndex(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if err := s.Modules.EnsureAgendaCalendars(user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível preparar a agenda.", nil)
		return
	}
	calendars, err := s.Modules.AgendaCalendars(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os calendários.", nil)
		return
	}
	from, to := agendaRange(r)
	ids := []int64{}
	for _, raw := range r.URL.Query()["calendars[]"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		for _, raw := range r.URL.Query()["calendars"] {
			if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
	}
	occurrences, err := s.Modules.AgendaEventsInRange(user.ID, ids, from, to, r.URL.Query().Get("search"))
	if err != nil {
		log.Printf("agenda index: %v", err)
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os eventos.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"calendars": calendars, "occurrences": occurrences, "from": from, "to": to,
			"colors": modules.AgendaColors,
		},
	})
}

func agendaRange(r *http.Request) (time.Time, time.Time) {
	loc := time.Local
	reference := time.Now().In(loc)
	if raw := r.URL.Query().Get("date"); raw != "" {
		if parsed, err := time.ParseInLocation("2006-01-02", raw, loc); err == nil {
			reference = parsed
		}
	}
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "month"
	}
	switch view {
	case "week":
		offset := (int(reference.Weekday()) + 6) % 7
		start := reference.AddDate(0, 0, -offset)
		return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc), start.AddDate(0, 0, 7)
	case "day":
		start := time.Date(reference.Year(), reference.Month(), reference.Day(), 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 0, 1)
	case "list":
		start := time.Date(reference.Year(), reference.Month(), reference.Day(), 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 0, 30)
	default:
		first := time.Date(reference.Year(), reference.Month(), 1, 0, 0, 0, 0, loc)
		offset := (int(first.Weekday()) + 6) % 7
		start := first.AddDate(0, 0, -offset)
		return start, start.AddDate(0, 0, 42)
	}
}

type agendaEventPayload struct {
	CalendarID      int64                         `json:"calendar_id"`
	Title           string                        `json:"title"`
	Description     string                        `json:"description"`
	Location        string                        `json:"location"`
	MeetingURL      string                        `json:"meeting_url"`
	StartAt         string                        `json:"start_at"`
	EndAt           string                        `json:"end_at"`
	IsAllDay        bool                          `json:"is_all_day"`
	Timezone        string                        `json:"timezone"`
	Status          string                        `json:"status"`
	Visibility      string                        `json:"visibility"`
	Transparency    string                        `json:"transparency"`
	Color           string                        `json:"color"`
	Recurrence      *modules.AgendaRecurrence     `json:"recurrence"`
	RecurrenceUntil string                        `json:"recurrence_until"`
	Attendees       []string                      `json:"attendees"`
	Reminders       []modules.AgendaReminderInput `json:"reminders"`
	CanInvite       bool                          `json:"attendees_can_invite_others"`
	CanModify       bool                          `json:"attendees_can_modify"`
	CanSeeOthers    bool                          `json:"attendees_can_see_other_attendees"`
	Attachments     []modules.AgendaAttachment    `json:"attachments"`
	Scope           string                        `json:"scope"`
	RecurrenceID    string                        `json:"recurrence_id"`
}

func (p agendaEventPayload) input() modules.AgendaEventInput {
	return modules.AgendaEventInput{
		CalendarID: p.CalendarID, Title: p.Title, Description: p.Description, Location: p.Location,
		MeetingURL: p.MeetingURL, StartAt: p.StartAt, EndAt: p.EndAt, IsAllDay: p.IsAllDay,
		Timezone: p.Timezone, Status: p.Status, Visibility: p.Visibility, Transparency: p.Transparency,
		Color: p.Color, Recurrence: p.Recurrence, RecurrenceUntil: p.RecurrenceUntil,
		Attendees: p.Attendees, Reminders: p.Reminders, CanInvite: p.CanInvite, CanModify: p.CanModify,
		CanSeeOthers: p.CanSeeOthers, Attachments: p.Attachments,
	}
}

func (s *Server) agendaCreateEvent(w http.ResponseWriter, r *http.Request) {
	var payload agendaEventPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Title) == "" || payload.CalendarID == 0 {
		writeValidation(w, validationErrors{"title": "Informe o título e o calendário do evento."})
		return
	}
	event, err := s.Modules.CreateAgendaEvent(currentUser(r).ID, payload.input())
	if err != nil {
		log.Printf("agenda create event: %v", err)
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível criar o evento.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Evento criado.", "data": map[string]any{"event": event}})
}

func (s *Server) agendaUpdateEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload agendaEventPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	var recurrenceID *time.Time
	if parsed, err := parseDateTime(payload.RecurrenceID); err == nil {
		recurrenceID = &parsed
	}
	event, err := s.Modules.UpdateAgendaEvent(currentUser(r).ID, id, payload.input(), payload.Scope, recurrenceID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível atualizar o evento.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Evento atualizado.", "data": map[string]any{"event": event}})
}

func (s *Server) agendaDeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Scope        string `json:"scope"`
		RecurrenceID string `json:"recurrence_id"`
	}
	_ = decodeJSON(w, r, &payload)
	var recurrenceID *time.Time
	if parsed, err := parseDateTime(payload.RecurrenceID); err == nil {
		recurrenceID = &parsed
	}
	if err := s.Modules.DeleteAgendaEvent(currentUser(r).ID, id, payload.Scope, recurrenceID); err != nil {
		writeError(w, http.StatusForbidden, "Não foi possível excluir o evento.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Evento excluído."})
}

func (s *Server) agendaShowEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	event, err := s.Modules.FindAgendaEvent(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Evento não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"event": event}})
}

func (s *Server) agendaDuplicateEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	event, err := s.Modules.DuplicateAgendaEvent(currentUser(r).ID, id)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível duplicar o evento.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Evento duplicado.", "data": map[string]any{"event": event}})
}

func (s *Server) agendaRespondEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	valid := map[string]bool{"accepted": true, "declined": true, "tentative": true, "needs_action": true}
	if !valid[payload.Status] {
		writeValidation(w, validationErrors{"status": "Resposta inválida."})
		return
	}
	if err := s.Modules.RespondAgendaEvent(currentUser(r).ID, id, payload.Status, payload.Comment); err != nil {
		writeError(w, http.StatusForbidden, "Você não é participante deste evento.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Resposta registrada."})
}

func (s *Server) agendaMoveEvent(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		StartAt      string `json:"start_at"`
		EndAt        string `json:"end_at"`
		Scope        string `json:"scope"`
		RecurrenceID string `json:"recurrence_id"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	var recurrenceID *time.Time
	if parsed, err := parseDateTime(payload.RecurrenceID); err == nil {
		recurrenceID = &parsed
	}
	if err := s.Modules.MoveAgendaEvent(currentUser(r).ID, id, payload.StartAt, payload.EndAt, payload.Scope, recurrenceID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível reagendar o evento.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Evento reagendado."})
}

type agendaCalendarPayload struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Color       string                        `json:"color"`
	Timezone    string                        `json:"timezone"`
	IsDefault   bool                          `json:"is_default"`
	Reminders   []modules.AgendaReminderInput `json:"default_reminders"`
}

func (s *Server) agendaCreateCalendar(w http.ResponseWriter, r *http.Request) {
	var payload agendaCalendarPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeValidation(w, validationErrors{"name": "Informe o nome do calendário."})
		return
	}
	calendar, err := s.Modules.CreateAgendaCalendar(currentUser(r).ID, modules.AgendaCalendarInput{
		Name: payload.Name, Description: payload.Description, Color: payload.Color, Timezone: payload.Timezone,
		IsDefault: payload.IsDefault, Reminders: payload.Reminders,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o calendário.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Calendário criado.", "data": map[string]any{"calendar": calendar}})
}

func (s *Server) agendaUpdateCalendar(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload agendaCalendarPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	err := s.Modules.UpdateAgendaCalendar(currentUser(r).ID, id, modules.AgendaCalendarInput{
		Name: payload.Name, Description: payload.Description, Color: payload.Color, Timezone: payload.Timezone,
		IsDefault: payload.IsDefault, Reminders: payload.Reminders,
	})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível atualizar o calendário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Calendário atualizado."})
}

func (s *Server) agendaDeleteCalendar(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.DeleteAgendaCalendar(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusForbidden, "Não foi possível excluir o calendário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Calendário excluído."})
}

func (s *Server) agendaCalendarVisibility(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Visible *bool `json:"is_visible"`
	}
	if !decodeJSON(w, r, &payload) || payload.Visible == nil {
		writeValidation(w, validationErrors{"is_visible": "Informe a visibilidade."})
		return
	}
	if err := s.Modules.SetAgendaCalendarVisibility(currentUser(r).ID, id, *payload.Visible); err != nil {
		writeError(w, http.StatusForbidden, "Não foi possível alterar a visibilidade.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Visibilidade atualizada.", "data": map[string]any{"is_visible": *payload.Visible}})
}

func (s *Server) agendaCalendarDefault(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if err := s.Modules.MakeAgendaCalendarDefault(currentUser(r).ID, id); err != nil {
		writeError(w, http.StatusForbidden, "Não foi possível definir o calendário padrão.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Calendário padrão atualizado."})
}

func (s *Server) agendaShareCalendar(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	target, err := s.Store.FindUserByEmail(payload.Email)
	if err != nil {
		writeValidation(w, validationErrors{"email": "Usuário não encontrado."})
		return
	}
	if err := s.Modules.ShareAgendaCalendar(currentUser(r).ID, id, target.ID, payload.Role); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível compartilhar o calendário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Calendário compartilhado com " + target.Email + "."})
}

func (s *Server) agendaUnshareCalendar(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	shareID, _ := parseID(r, "shareId")
	if err := s.Modules.UnshareAgendaCalendar(currentUser(r).ID, id, shareID); err != nil {
		writeError(w, http.StatusNotFound, "Compartilhamento não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Compartilhamento removido."})
}

func (s *Server) agendaExportICS(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	from := time.Now().AddDate(-1, 0, 0)
	to := time.Now().AddDate(2, 0, 0)
	ids := []int64{}
	if raw := r.URL.Query().Get("calendar"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	occurrences, err := s.Modules.AgendaEventsInRange(user.ID, ids, from, to, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar a agenda.", nil)
		return
	}
	var builder strings.Builder
	builder.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Documentation Center//Agenda//PT-BR\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n")
	for _, occurrence := range occurrences {
		builder.WriteString("BEGIN:VEVENT\r\n")
		builder.WriteString(fmt.Sprintf("UID:agenda-event-%d@documentation-center\r\n", occurrence.ID))
		builder.WriteString("DTSTAMP:" + time.Now().UTC().Format("20060102T150405Z") + "\r\n")
		if occurrence.IsAllDay {
			builder.WriteString("DTSTART;VALUE=DATE:" + occurrence.Start.Format("20060102") + "\r\n")
			builder.WriteString("DTEND;VALUE=DATE:" + occurrence.End.Format("20060102") + "\r\n")
		} else {
			builder.WriteString("DTSTART:" + occurrence.Start.UTC().Format("20060102T150405Z") + "\r\n")
			builder.WriteString("DTEND:" + occurrence.End.UTC().Format("20060102T150405Z") + "\r\n")
		}
		builder.WriteString("SUMMARY:" + icsEscape(occurrence.Title) + "\r\n")
		if occurrence.Description != nil {
			builder.WriteString("DESCRIPTION:" + icsEscape(*occurrence.Description) + "\r\n")
		}
		if occurrence.Location != nil {
			builder.WriteString("LOCATION:" + icsEscape(*occurrence.Location) + "\r\n")
		}
		if occurrence.MeetingURL != nil {
			builder.WriteString("URL:" + *occurrence.MeetingURL + "\r\n")
		}
		builder.WriteString("STATUS:" + strings.ToUpper(occurrence.Status) + "\r\n")
		builder.WriteString("END:VEVENT\r\n")
	}
	builder.WriteString("END:VCALENDAR\r\n")
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="agenda-%s.ics"`, time.Now().Format("20060102-150405")))
	_, _ = w.Write([]byte(builder.String()))
}

func icsEscape(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\r\n", "\\n", "\n", "\\n", ";", "\\;", ",", "\\,")
	return replacer.Replace(value)
}

func (s *Server) agendaImportICS(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	calendarID, err := strconv.ParseInt(r.FormValue("calendar_id"), 10, 64)
	if err != nil || calendarID == 0 {
		writeValidation(w, validationErrors{"calendar_id": "Selecione um calendário."})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione um arquivo .ics."})
		return
	}
	defer file.Close()
	events, err := parseICS(file)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Não foi possível interpretar o arquivo ICS.", nil)
		return
	}
	created := 0
	for _, event := range events {
		event.CalendarID = calendarID
		input := event.input()
		if _, err := s.Modules.CreateAgendaEvent(user.ID, input); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("%d evento(s) importado(s).", created), "data": map[string]any{"imported": created}})
}

func parseICS(reader io.Reader) ([]agendaEventPayload, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var events []agendaEventPayload
	var current *agendaEventPayload
	lastKey := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, " ") && current != nil {
			switch lastKey {
			case "DESCRIPTION":
				current.Description += strings.TrimSpace(line)
			case "LOCATION":
				current.Location += strings.TrimSpace(line)
			case "SUMMARY":
				current.Title += strings.TrimSpace(line)
			}
			continue
		}
		switch {
		case line == "BEGIN:VEVENT":
			current = &agendaEventPayload{}
		case line == "END:VEVENT":
			if current != nil {
				if current.Title == "" {
					current.Title = "Evento importado"
				}
				if current.StartAt == "" {
					current.StartAt = time.Now().Format(time.RFC3339)
				}
				if current.EndAt == "" {
					current.EndAt = time.Now().Add(time.Hour).Format(time.RFC3339)
				}
				events = append(events, *current)
			}
			current = nil
		case current != nil:
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.ToUpper(strings.Split(parts[0], ";")[0])
			value := parts[1]
			lastKey = key
			switch key {
			case "SUMMARY":
				current.Title = value
			case "DESCRIPTION":
				current.Description = value
			case "LOCATION":
				current.Location = value
			case "URL":
				current.MeetingURL = value
			case "DTSTART":
				if parsed, ok := parseICSTime(value); ok {
					current.StartAt = parsed.Format(time.RFC3339)
					current.IsAllDay = len(value) == 8
				}
			case "DTEND":
				if parsed, ok := parseICSTime(value); ok {
					current.EndAt = parsed.Format(time.RFC3339)
					current.IsAllDay = len(value) == 8
				}
			case "RRULE":
				recurrence := modules.AgendaRecurrence{Interval: 1}
				for _, part := range strings.Split(value, ";") {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) != 2 {
						continue
					}
					switch kv[0] {
					case "FREQ":
						recurrence.Frequency = strings.ToLower(kv[1])
					case "INTERVAL":
						recurrence.Interval, _ = strconv.Atoi(kv[1])
					case "COUNT":
						recurrence.Count, _ = strconv.Atoi(kv[1])
					case "UNTIL":
						if parsed, ok := parseICSTime(kv[1]); ok {
							recurrence.Until = parsed.Format("2006-01-02")
						}
					}
				}
				if recurrence.Frequency != "" {
					current.Recurrence = &recurrence
				}
			case "ATTENDEE":
				if index := strings.Index(strings.ToLower(value), "mailto:"); index >= 0 {
					current.Attendees = append(current.Attendees, strings.TrimSpace(value[index+7:]))
				}
			}
		}
	}
	return events, scanner.Err()
}

func parseICSTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"20060102T150405Z", "20060102T1504Z", "20060102T150405", "20060102"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

var _ = sort.Slice
var _ = chi.URLParam
