package modules

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type AgendaEventInput struct {
	CalendarID      int64
	Title           string
	Description     string
	Location        string
	MeetingURL      string
	StartAt         string
	EndAt           string
	IsAllDay        bool
	Timezone        string
	Status          string
	Visibility      string
	Transparency    string
	Color           string
	Recurrence      *AgendaRecurrence
	RecurrenceUntil string
	Attendees       []string
	Reminders       []AgendaReminderInput
	CanInvite       bool
	CanModify       bool
	CanSeeOthers    bool
	Attachments     []AgendaAttachment
}

func (s *Store) FindAgendaEvent(userID, id int64) (AgendaEvent, error) {
	events, err := s.agendaEventsByQuery(`SELECT e.*, c.name AS calendar_name, c.color AS calendar_color,
		TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS organizer_name, u.email AS organizer_email
		FROM agenda_events e
		JOIN agenda_calendars c ON c.id = e.agenda_calendar_id
		JOIN users u ON u.id = e.user_id
		WHERE e.id = ?`, id)
	if err != nil {
		return AgendaEvent{}, err
	}
	if len(events) == 0 {
		return AgendaEvent{}, ErrNotFound
	}
	event := events[0]
	calendar, err := s.FindAgendaCalendar(userID, event.CalendarID)
	if err != nil {
		for _, attendee := range event.Attendees {
			if attendee.UserID != nil && *attendee.UserID == userID {
				return event, nil
			}
		}
		return event, ErrForbidden
	}
	_ = calendar
	return event, nil
}

func (s *Store) agendaEventsByQuery(query string, args ...any) ([]AgendaEvent, error) {
	events := []AgendaEvent{}
	if err := s.DB.Select(&events, query, args...); err != nil {
		return nil, err
	}
	if err := s.attachAgendaChildren(events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Store) CreateAgendaEvent(userID int64, input AgendaEventInput) (AgendaEvent, error) {
	calendar, err := s.FindAgendaCalendar(userID, input.CalendarID)
	if err != nil {
		return AgendaEvent{}, err
	}
	if calendar.Role != "owner" && calendar.Role != "writer" {
		return AgendaEvent{}, ErrForbidden
	}
	start, end, err := agendaStartEnd(input)
	if err != nil {
		return AgendaEvent{}, ErrValidation
	}
	recurrenceJSON, recurrenceUntil := agendaRecurrenceJSON(input)
	attachments, _ := json.Marshal(input.Attachments)
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO agenda_events (agenda_calendar_id, user_id, title, description, location, meeting_url,
			start_at, end_at, is_all_day, timezone, status, visibility, transparency, color, recurrence,
			recurrence_until, attendees_can_invite_others, attendees_can_modify, attendees_can_see_other_attendees,
			attachments, sequence, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		input.CalendarID, userID, strings.TrimSpace(input.Title), nullString(input.Description),
		nullString(input.Location), nullString(input.MeetingURL), start, end, input.IsAllDay,
		defaultString(input.Timezone, time.Local.String()), defaultString(input.Status, "confirmed"),
		defaultString(input.Visibility, "default"), defaultString(input.Transparency, "opaque"),
		nullString(input.Color), recurrenceJSON, recurrenceUntil, input.CanInvite, input.CanModify,
		input.CanSeeOthers, string(attachments), timestamp, timestamp)
	if err != nil {
		return AgendaEvent{}, err
	}
	id, _ := result.LastInsertId()
	if err := s.syncAgendaAttendees(id, userID, input.Attendees); err != nil {
		return AgendaEvent{}, err
	}
	if err := s.syncAgendaReminders(id, input.Reminders); err != nil {
		return AgendaEvent{}, err
	}
	return s.FindAgendaEvent(userID, id)
}

func agendaStartEnd(input AgendaEventInput) (time.Time, time.Time, error) {
	start, err := parseDateTime(input.StartAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseDateTime(input.EndAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if input.IsAllDay {
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, ErrValidation
	}
	return start, end, nil
}

func agendaRecurrenceJSON(input AgendaEventInput) (any, any) {
	if input.Recurrence == nil || input.Recurrence.Frequency == "" {
		return nil, nil
	}
	recurrence := *input.Recurrence
	if recurrence.Interval < 1 {
		recurrence.Interval = 1
	}
	encoded, _ := json.Marshal(recurrence)
	until := nullDate(input.RecurrenceUntil)
	if until == nil && recurrence.Until != "" {
		value := recurrence.Until
		until = &value
	}
	return string(encoded), until
}

func (s *Store) syncAgendaAttendees(eventID, organizerID int64, emails []string) error {
	target := map[string]bool{}
	organizerEmail := ""
	organizerName := ""
	if err := s.DB.Get(&organizerEmail, `SELECT email FROM users WHERE id = ?`, organizerID); err == nil {
		target[strings.ToLower(organizerEmail)] = true
	}
	_ = s.DB.Get(&organizerName, `SELECT TRIM(name || ' ' || COALESCE(surname, '')) FROM users WHERE id = ?`, organizerID)
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email != "" {
			target[email] = true
		}
	}
	existing := []AgendaAttendee{}
	_ = s.DB.Select(&existing, `SELECT * FROM agenda_event_attendees WHERE agenda_event_id = ?`, eventID)
	existingByEmail := map[string]AgendaAttendee{}
	for _, attendee := range existing {
		existingByEmail[strings.ToLower(attendee.Email)] = attendee
		if !target[strings.ToLower(attendee.Email)] {
			_, _ = s.DB.Exec(`DELETE FROM agenda_event_attendees WHERE id = ?`, attendee.ID)
		}
	}
	timestamp := now()
	for email := range target {
		if attendee, ok := existingByEmail[email]; ok {
			_, _ = s.DB.Exec(`UPDATE agenda_event_attendees SET is_organizer = ?, updated_at = ? WHERE id = ?`,
				email == strings.ToLower(organizerEmail), timestamp, attendee.ID)
			continue
		}
		var userID *int64
		var name *string
		var foundID int64
		var foundName string
		if err := s.DB.Get(&foundID, `SELECT id FROM users WHERE email = ?`, email); err == nil {
			userID = &foundID
			_ = s.DB.Get(&foundName, `SELECT TRIM(name || ' ' || COALESCE(surname, '')) FROM users WHERE id = ?`, foundID)
			name = &foundName
		}
		status := "needs_action"
		var respondedAt *time.Time
		isOrganizer := email == strings.ToLower(organizerEmail)
		if isOrganizer {
			status = "accepted"
			respondedAt = &timestamp
			if organizerName != "" {
				name = &organizerName
			}
		}
		_, _ = s.DB.Exec(`INSERT INTO agenda_event_attendees (agenda_event_id, user_id, email, name, status,
			is_organizer, responded_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			eventID, userID, email, name, status, isOrganizer, respondedAt, timestamp, timestamp)
	}
	return nil
}

func (s *Store) syncAgendaReminders(eventID int64, reminders []AgendaReminderInput) error {
	_, _ = s.DB.Exec(`DELETE FROM agenda_event_reminders WHERE agenda_event_id = ?`, eventID)
	for _, reminder := range reminders {
		if reminder.Method == "" {
			continue
		}
		_, err := s.DB.Exec(`INSERT INTO agenda_event_reminders (agenda_event_id, method, minutes_before, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, eventID, reminder.Method, reminder.MinutesBefore, now(), now())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateAgendaEvent(userID, id int64, input AgendaEventInput, scope string, recurrenceID *time.Time) (AgendaEvent, error) {
	event, err := s.FindAgendaEvent(userID, id)
	if err != nil {
		return event, err
	}
	calendar, err := s.FindAgendaCalendar(userID, event.CalendarID)
	if err != nil || (calendar.Role != "owner" && calendar.Role != "writer") {
		return event, ErrForbidden
	}
	if input.CalendarID != 0 && input.CalendarID != event.CalendarID {
		target, err := s.FindAgendaCalendar(userID, input.CalendarID)
		if err != nil || (target.Role != "owner" && target.Role != "writer") {
			return event, ErrForbidden
		}
		event.CalendarID = input.CalendarID
	}
	start, end, err := agendaStartEnd(input)
	if err != nil {
		return event, ErrValidation
	}
	if event.RecurrenceData != nil && recurrenceID != nil && scope == "this" {
		overrides := map[string]any{
			"title": input.Title, "description": input.Description, "location": input.Location,
			"start_at": start.Format(time.RFC3339), "end_at": end.Format(time.RFC3339), "is_all_day": input.IsAllDay,
		}
		encoded, _ := json.Marshal(overrides)
		_, err := s.DB.Exec(`INSERT INTO agenda_event_exceptions (agenda_event_id, recurrence_id, is_cancelled, overrides, created_at, updated_at)
			VALUES (?, ?, 0, ?, ?, ?)
			ON CONFLICT(agenda_event_id, recurrence_id) DO UPDATE SET is_cancelled = 0, overrides = excluded.overrides, updated_at = excluded.updated_at`,
			id, *recurrenceID, string(encoded), now(), now())
		if err != nil {
			return event, err
		}
		return s.FindAgendaEvent(userID, id)
	}
	if event.RecurrenceData != nil && recurrenceID != nil && scope == "following" {
		if _, err := s.DB.Exec(`UPDATE agenda_events SET recurrence_until = ?, updated_at = ? WHERE id = ?`,
			recurrenceID.AddDate(0, 0, -1).Format("2006-01-02"), now(), id); err != nil {
			return event, err
		}
		input.Recurrence = event.RecurrenceData
		clone, err := s.CreateAgendaEvent(userID, input)
		if err != nil {
			return event, err
		}
		return clone, nil
	}
	recurrenceJSON, recurrenceUntil := agendaRecurrenceJSON(input)
	attachments, _ := json.Marshal(input.Attachments)
	_, err = s.DB.Exec(`
		UPDATE agenda_events SET agenda_calendar_id = ?, title = ?, description = ?, location = ?, meeting_url = ?,
			start_at = ?, end_at = ?, is_all_day = ?, timezone = ?, status = ?, visibility = ?, transparency = ?,
			color = ?, recurrence = ?, recurrence_until = ?, attendees_can_invite_others = ?, attendees_can_modify = ?,
			attendees_can_see_other_attendees = ?, attachments = ?, sequence = sequence + 1, updated_at = ?
		WHERE id = ?`,
		event.CalendarID, strings.TrimSpace(input.Title), nullString(input.Description), nullString(input.Location),
		nullString(input.MeetingURL), start, end, input.IsAllDay, defaultString(input.Timezone, event.Timezone),
		defaultString(input.Status, "confirmed"), defaultString(input.Visibility, "default"),
		defaultString(input.Transparency, "opaque"), nullString(input.Color), recurrenceJSON, recurrenceUntil,
		input.CanInvite, input.CanModify, input.CanSeeOthers, string(attachments), now(), id)
	if err != nil {
		return event, err
	}
	if err := s.syncAgendaAttendees(id, userID, input.Attendees); err != nil {
		return event, err
	}
	if err := s.syncAgendaReminders(id, input.Reminders); err != nil {
		return event, err
	}
	return s.FindAgendaEvent(userID, id)
}

func (s *Store) DeleteAgendaEvent(userID, id int64, scope string, recurrenceID *time.Time) error {
	event, err := s.FindAgendaEvent(userID, id)
	if err != nil {
		return err
	}
	calendar, err := s.FindAgendaCalendar(userID, event.CalendarID)
	if err != nil || (calendar.Role != "owner" && calendar.Role != "writer") {
		return ErrForbidden
	}
	if event.RecurrenceData != nil && recurrenceID != nil && scope == "this" {
		_, err := s.DB.Exec(`INSERT INTO agenda_event_exceptions (agenda_event_id, recurrence_id, is_cancelled, created_at, updated_at)
			VALUES (?, ?, 1, ?, ?)
			ON CONFLICT(agenda_event_id, recurrence_id) DO UPDATE SET is_cancelled = 1, updated_at = excluded.updated_at`,
			id, *recurrenceID, now(), now())
		return err
	}
	_, err = s.DB.Exec(`DELETE FROM agenda_events WHERE id = ?`, id)
	return err
}

func (s *Store) RespondAgendaEvent(userID, id int64, status, comment string) error {
	calendar, _ := s.agendaRoleForEvent(userID, id)
	_ = calendar
	result, err := s.DB.Exec(`UPDATE agenda_event_attendees SET status = ?, comment = ?, responded_at = ?, updated_at = ?
		WHERE agenda_event_id = ? AND user_id = ?`, status, nullString(comment), now(), now(), id, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrForbidden
	}
	return nil
}

func (s *Store) agendaRoleForEvent(userID, eventID int64) (string, error) {
	var calendarID int64
	if err := s.DB.Get(&calendarID, `SELECT agenda_calendar_id FROM agenda_events WHERE id = ?`, eventID); err != nil {
		return "", err
	}
	calendar, err := s.FindAgendaCalendar(userID, calendarID)
	if err != nil {
		return "", err
	}
	return calendar.Role, nil
}

func (s *Store) DuplicateAgendaEvent(userID, id int64) (AgendaEvent, error) {
	event, err := s.FindAgendaEvent(userID, id)
	if err != nil {
		return event, err
	}
	start := event.StartAt.Add(24 * time.Hour)
	end := event.EndAt.Add(24 * time.Hour)
	input := AgendaEventInput{
		CalendarID: event.CalendarID, Title: event.Title + " (cópia)", Description: deref(event.Description),
		Location: deref(event.Location), MeetingURL: deref(event.MeetingURL), StartAt: start.Format(time.RFC3339),
		EndAt: end.Format(time.RFC3339), IsAllDay: event.IsAllDay, Timezone: event.Timezone,
		Status: event.Status, Visibility: event.Visibility, Transparency: event.Transparency,
		Color: deref(event.Color), Recurrence: event.RecurrenceData, RecurrenceUntil: deref(event.RecurrenceUntil),
		CanInvite: event.AttendeesCanInviteOthers, CanModify: event.AttendeesCanModify,
		CanSeeOthers: event.AttendeesCanSeeOtherAttendees, Attachments: event.AttachmentData,
	}
	for _, attendee := range event.Attendees {
		if !attendee.IsOrganizer {
			input.Attendees = append(input.Attendees, attendee.Email)
		}
	}
	for _, reminder := range event.Reminders {
		input.Reminders = append(input.Reminders, AgendaReminderInput{Method: reminder.Method, MinutesBefore: reminder.MinutesBefore})
	}
	return s.CreateAgendaEvent(userID, input)
}

func (s *Store) MoveAgendaEvent(userID, id int64, startRaw, endRaw string, scope string, recurrenceID *time.Time) error {
	start, err := parseDateTime(startRaw)
	if err != nil {
		return ErrValidation
	}
	end, err := parseDateTime(endRaw)
	if err != nil || end.Before(start) {
		return ErrValidation
	}
	event, err := s.FindAgendaEvent(userID, id)
	if err != nil {
		return err
	}
	calendar, err := s.FindAgendaCalendar(userID, event.CalendarID)
	if err != nil || (calendar.Role != "owner" && calendar.Role != "writer") {
		return ErrForbidden
	}
	if event.RecurrenceData != nil && recurrenceID != nil && scope == "this" {
		overrides := map[string]any{"start_at": start.Format(time.RFC3339), "end_at": end.Format(time.RFC3339)}
		encoded, _ := json.Marshal(overrides)
		_, err := s.DB.Exec(`INSERT INTO agenda_event_exceptions (agenda_event_id, recurrence_id, is_cancelled, overrides, created_at, updated_at)
			VALUES (?, ?, 0, ?, ?, ?)
			ON CONFLICT(agenda_event_id, recurrence_id) DO UPDATE SET overrides = excluded.overrides, updated_at = excluded.updated_at`,
			id, *recurrenceID, string(encoded), now(), now())
		return err
	}
	_, err = s.DB.Exec(`UPDATE agenda_events SET start_at = ?, end_at = ?, sequence = sequence + 1, updated_at = ? WHERE id = ?`,
		start, end, now(), id)
	return err
}

func (s *Store) AgendaEvent(id int64) (AgendaEvent, error) {
	events, err := s.agendaEventsByQuery(`SELECT e.*, c.name AS calendar_name, c.color AS calendar_color,
		TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS organizer_name, u.email AS organizer_email
		FROM agenda_events e JOIN agenda_calendars c ON c.id = e.agenda_calendar_id
		JOIN users u ON u.id = e.user_id WHERE e.id = ?`, id)
	if err != nil {
		return AgendaEvent{}, err
	}
	if len(events) == 0 {
		return AgendaEvent{}, ErrNotFound
	}
	return events[0], nil
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

var _ = sql.ErrNoRows
var _ = errors.Is
