package modules

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

/* --------------------------------------------------------------------------
 * Agenda module
 * ----------------------------------------------------------------------- */

type AgendaCalendarShare struct {
	ID         int64     `db:"id" json:"id"`
	CalendarID int64     `db:"agenda_calendar_id" json:"-"`
	UserID     int64     `db:"user_id" json:"user_id"`
	Role       string    `db:"role" json:"role"`
	Email      *string   `db:"email" json:"email"`
	Name       *string   `db:"name" json:"name"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

type AgendaCalendar struct {
	ID               int64                 `db:"id" json:"id"`
	UserID           int64                 `db:"user_id" json:"user_id"`
	Name             string                `db:"name" json:"name"`
	Description      *string               `db:"description" json:"description"`
	Color            string                `db:"color" json:"color"`
	Timezone         string                `db:"timezone" json:"timezone"`
	IsDefault        bool                  `db:"is_default" json:"is_default"`
	IsVisible        bool                  `db:"is_visible" json:"is_visible"`
	Position         int                   `db:"position" json:"position"`
	DefaultReminders *string               `db:"default_reminders" json:"-"`
	CreatedAt        time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time             `db:"updated_at" json:"updated_at"`
	Role             string                `db:"-" json:"role"`
	Shares           []AgendaCalendarShare `db:"-" json:"shares"`
	EventCount       int                   `db:"event_count" json:"event_count"`
	OwnerName        *string               `db:"owner_name" json:"owner_name"`
}

type AgendaAttendee struct {
	ID          int64      `db:"id" json:"id"`
	EventID     int64      `db:"agenda_event_id" json:"-"`
	UserID      *int64     `db:"user_id" json:"user_id"`
	Email       string     `db:"email" json:"email"`
	Name        *string    `db:"name" json:"name"`
	Status      string     `db:"status" json:"status"`
	IsOptional  bool       `db:"is_optional" json:"is_optional"`
	IsOrganizer bool       `db:"is_organizer" json:"is_organizer"`
	IsResource  bool       `db:"is_resource" json:"is_resource"`
	Comment     *string    `db:"comment" json:"comment"`
	RespondedAt *time.Time `db:"responded_at" json:"responded_at"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type AgendaReminder struct {
	ID            int64      `db:"id" json:"id"`
	EventID       int64      `db:"agenda_event_id" json:"-"`
	Method        string     `db:"method" json:"method"`
	MinutesBefore *int       `db:"minutes_before" json:"minutes_before"`
	TriggerAt     *time.Time `db:"trigger_at" json:"trigger_at"`
	SentAt        *time.Time `db:"sent_at" json:"sent_at"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

type AgendaException struct {
	ID           int64     `db:"id" json:"id"`
	EventID      int64     `db:"agenda_event_id" json:"-"`
	RecurrenceID time.Time `db:"recurrence_id" json:"recurrence_id"`
	IsCancelled  bool      `db:"is_cancelled" json:"is_cancelled"`
	Overrides    *string   `db:"overrides" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type AgendaRecurrence struct {
	Frequency  string `json:"frequency"`
	Interval   int    `json:"interval"`
	Count      int    `json:"count,omitempty"`
	ByWeekday  []int  `json:"byweekday,omitempty"`
	ByMonthDay []int  `json:"bymonthday,omitempty"`
	Until      string `json:"until,omitempty"`
}

type AgendaEvent struct {
	ID                            int64              `db:"id" json:"id"`
	CalendarID                    int64              `db:"agenda_calendar_id" json:"calendar_id"`
	UserID                        int64              `db:"user_id" json:"user_id"`
	Title                         string             `db:"title" json:"title"`
	Description                   *string            `db:"description" json:"description"`
	Location                      *string            `db:"location" json:"location"`
	MeetingURL                    *string            `db:"meeting_url" json:"meeting_url"`
	StartAt                       time.Time          `db:"start_at" json:"start_at"`
	EndAt                         time.Time          `db:"end_at" json:"end_at"`
	IsAllDay                      bool               `db:"is_all_day" json:"is_all_day"`
	Timezone                      string             `db:"timezone" json:"timezone"`
	Status                        string             `db:"status" json:"status"`
	Visibility                    string             `db:"visibility" json:"visibility"`
	Transparency                  string             `db:"transparency" json:"transparency"`
	Color                         *string            `db:"color" json:"color"`
	Recurrence                    *string            `db:"recurrence" json:"-"`
	RecurrenceUntil               *string            `db:"recurrence_until" json:"recurrence_until"`
	AttendeesCanInviteOthers      bool               `db:"attendees_can_invite_others" json:"attendees_can_invite_others"`
	AttendeesCanModify            bool               `db:"attendees_can_modify" json:"attendees_can_modify"`
	AttendeesCanSeeOtherAttendees bool               `db:"attendees_can_see_other_attendees" json:"attendees_can_see_other_attendees"`
	Attachments                   *string            `db:"attachments" json:"-"`
	Sequence                      int                `db:"sequence" json:"sequence"`
	CreatedAt                     time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt                     time.Time          `db:"updated_at" json:"updated_at"`
	CalendarName                  *string            `db:"calendar_name" json:"calendar_name"`
	CalendarColor                 *string            `db:"calendar_color" json:"calendar_color"`
	OrganizerName                 *string            `db:"organizer_name" json:"organizer_name"`
	OrganizerEmail                *string            `db:"organizer_email" json:"organizer_email"`
	Attendees                     []AgendaAttendee   `db:"-" json:"attendees"`
	Reminders                     []AgendaReminder   `db:"-" json:"reminders"`
	Exceptions                    []AgendaException  `db:"-" json:"-"`
	RecurrenceData                *AgendaRecurrence  `db:"-" json:"recurrence"`
	AttachmentData                []AgendaAttachment `db:"-" json:"attachments"`
}

type AgendaAttachment struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
}

type AgendaOccurrence struct {
	ID             int64            `json:"id"`
	RecurrenceID   *time.Time       `json:"recurrence_id"`
	Title          string           `json:"title"`
	Description    *string          `json:"description"`
	Location       *string          `json:"location"`
	MeetingURL     *string          `json:"meeting_url"`
	Start          time.Time        `json:"start"`
	End            time.Time        `json:"end"`
	IsAllDay       bool             `json:"is_all_day"`
	IsRecurring    bool             `json:"is_recurring"`
	Status         string           `json:"status"`
	Visibility     string           `json:"visibility"`
	Transparency   string           `json:"transparency"`
	Color          string           `json:"color"`
	CalendarID     int64            `json:"calendar_id"`
	CalendarName   string           `json:"calendar_name"`
	CanWrite       bool             `json:"can_write"`
	Attendees      []AgendaAttendee `json:"attendees"`
	Reminders      []AgendaReminder `json:"reminders"`
	OrganizerName  *string          `json:"organizer_name"`
	OrganizerEmail *string          `json:"organizer_email"`
	AttendeeStatus *string          `json:"attendee_status"`
	IsOrganizer    bool             `json:"is_organizer"`
}

var AgendaColors = map[string]string{
	"tomato": "#d50000", "flamingo": "#e67c73", "tangerine": "#f4511e", "banana": "#f6bf26",
	"sage": "#33b679", "basil": "#0b8043", "peacock": "#039be5",
	"blueberry": "#3f51b5", "lavender": "#7986cb", "grape": "#8e24aa", "graphite": "#616161",
}

func (s *Store) AgendaCalendars(userID int64) ([]AgendaCalendar, error) {
	calendars := []AgendaCalendar{}
	err := s.DB.Select(&calendars, `
		SELECT c.*,
			(SELECT COUNT(*) FROM agenda_events e WHERE e.agenda_calendar_id = c.id) AS event_count,
			TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS owner_name
		FROM agenda_calendars c
		JOIN users u ON u.id = c.user_id
		WHERE c.user_id = ? OR c.id IN (SELECT agenda_calendar_id FROM agenda_calendar_shares WHERE user_id = ?)
		ORDER BY c.is_default DESC, c.position ASC, c.name ASC, c.id ASC`, userID, userID)
	if err != nil {
		return nil, err
	}
	for i := range calendars {
		calendars[i].Role = s.agendaRole(userID, calendars[i])
		calendars[i].Shares = []AgendaCalendarShare{}
		_ = s.DB.Select(&calendars[i].Shares, `
			SELECT s.*, u.email AS email, TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS name
			FROM agenda_calendar_shares s JOIN users u ON u.id = s.user_id
			WHERE s.agenda_calendar_id = ? ORDER BY u.name`, calendars[i].ID)
	}
	return calendars, nil
}

func (s *Store) agendaRole(userID int64, calendar AgendaCalendar) string {
	if calendar.UserID == userID {
		return "owner"
	}
	var role string
	if err := s.DB.Get(&role, `SELECT role FROM agenda_calendar_shares WHERE agenda_calendar_id = ? AND user_id = ?`,
		calendar.ID, userID); err == nil {
		return role
	}
	return ""
}

func (s *Store) EnsureAgendaCalendars(userID int64) error {
	calendars, err := s.AgendaCalendars(userID)
	if err != nil {
		return err
	}
	if len(calendars) > 0 {
		return nil
	}
	_, err = s.CreateAgendaCalendar(userID, AgendaCalendarInput{
		Name: "Minha agenda", Color: "blueberry", Timezone: time.Local.String(), IsDefault: true,
	})
	return err
}

func (s *Store) FindAgendaCalendar(userID, id int64) (AgendaCalendar, error) {
	calendars, err := s.AgendaCalendars(userID)
	if err != nil {
		return AgendaCalendar{}, err
	}
	for _, calendar := range calendars {
		if calendar.ID == id {
			return calendar, nil
		}
	}
	return AgendaCalendar{}, ErrNotFound
}

type AgendaCalendarInput struct {
	Name        string
	Description string
	Color       string
	Timezone    string
	IsDefault   bool
	Reminders   []AgendaReminderInput
}

type AgendaReminderInput struct {
	Method        string `json:"method"`
	MinutesBefore *int   `json:"minutes_before"`
}

func (s *Store) CreateAgendaCalendar(userID int64, input AgendaCalendarInput) (AgendaCalendar, error) {
	if input.Color == "" {
		input.Color = "blueberry"
	}
	if input.Timezone == "" {
		input.Timezone = time.Local.String()
	}
	var position int
	_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM agenda_calendars WHERE user_id = ?`, userID)
	var defaultCount int
	_ = s.DB.Get(&defaultCount, `SELECT COUNT(*) FROM agenda_calendars WHERE user_id = ?`, userID)
	isDefault := input.IsDefault || defaultCount == 0
	reminders, _ := json.Marshal(input.Reminders)
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO agenda_calendars (user_id, name, description, color, timezone, is_default, is_visible, position, default_reminders, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
		userID, strings.TrimSpace(input.Name), nullString(input.Description), input.Color, input.Timezone,
		isDefault, position, string(reminders), timestamp, timestamp)
	if err != nil {
		return AgendaCalendar{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindAgendaCalendar(userID, id)
}

func (s *Store) UpdateAgendaCalendar(userID, id int64, input AgendaCalendarInput) error {
	calendar, err := s.FindAgendaCalendar(userID, id)
	if err != nil {
		return err
	}
	if calendar.Role != "owner" && calendar.Role != "writer" {
		return ErrForbidden
	}
	if input.Color == "" {
		input.Color = calendar.Color
	}
	if input.Timezone == "" {
		input.Timezone = calendar.Timezone
	}
	reminders, _ := json.Marshal(input.Reminders)
	_, err = s.DB.Exec(`UPDATE agenda_calendars SET name = ?, description = ?, color = ?, timezone = ?, default_reminders = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.Name), nullString(input.Description), input.Color, input.Timezone, string(reminders), now(), id)
	return err
}

func (s *Store) DeleteAgendaCalendar(userID, id int64) error {
	calendar, err := s.FindAgendaCalendar(userID, id)
	if err != nil {
		return err
	}
	if calendar.Role != "owner" {
		return ErrForbidden
	}
	_, err = s.DB.Exec(`DELETE FROM agenda_calendars WHERE id = ?`, id)
	return err
}

func (s *Store) SetAgendaCalendarVisibility(userID, id int64, visible bool) error {
	calendar, err := s.FindAgendaCalendar(userID, id)
	if err != nil {
		return err
	}
	if calendar.Role != "owner" && calendar.Role != "writer" {
		return ErrForbidden
	}
	_, err = s.DB.Exec(`UPDATE agenda_calendars SET is_visible = ?, updated_at = ? WHERE id = ?`, visible, now(), id)
	return err
}

func (s *Store) MakeAgendaCalendarDefault(userID, id int64) error {
	calendar, err := s.FindAgendaCalendar(userID, id)
	if err != nil {
		return err
	}
	if calendar.Role != "owner" && calendar.Role != "writer" {
		return ErrForbidden
	}
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE agenda_calendars SET is_default = 0 WHERE user_id = ?`, calendar.UserID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE agenda_calendars SET is_default = 1, updated_at = ? WHERE id = ?`, now(), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ShareAgendaCalendar(userID, calendarID, targetUserID int64, role string) error {
	calendar, err := s.FindAgendaCalendar(userID, calendarID)
	if err != nil {
		return err
	}
	if calendar.Role != "owner" {
		return ErrForbidden
	}
	if calendar.UserID == targetUserID {
		return ErrValidation
	}
	_, err = s.DB.Exec(`INSERT INTO agenda_calendar_shares (agenda_calendar_id, user_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(agenda_calendar_id, user_id) DO UPDATE SET role = excluded.role, updated_at = excluded.updated_at`,
		calendarID, targetUserID, role, now(), now())
	return err
}

func (s *Store) UnshareAgendaCalendar(userID, calendarID, shareID int64) error {
	calendar, err := s.FindAgendaCalendar(userID, calendarID)
	if err != nil {
		return err
	}
	if calendar.Role != "owner" {
		return ErrForbidden
	}
	_, err = s.DB.Exec(`DELETE FROM agenda_calendar_shares WHERE id = ? AND agenda_calendar_id = ?`, shareID, calendarID)
	return err
}

func (s *Store) AgendaEventsInRange(userID int64, calendarIDs []int64, from, to time.Time, search string) ([]AgendaOccurrence, error) {
	calendars, err := s.AgendaCalendars(userID)
	if err != nil {
		return nil, err
	}
	allowed := map[int64]AgendaCalendar{}
	for _, calendar := range calendars {
		if len(calendarIDs) == 0 || containsInt64(calendarIDs, calendar.ID) {
			allowed[calendar.ID] = calendar
		}
	}
	if len(allowed) == 0 {
		return []AgendaOccurrence{}, nil
	}
	ids := make([]int64, 0, len(allowed))
	for id := range allowed {
		ids = append(ids, id)
	}
	query, args, err := sqlxIn(`SELECT e.*, c.name AS calendar_name, c.color AS calendar_color,
		TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS organizer_name, u.email AS organizer_email
		FROM agenda_events e
		JOIN agenda_calendars c ON c.id = e.agenda_calendar_id
		JOIN users u ON u.id = e.user_id
		WHERE e.agenda_calendar_id IN (?) ORDER BY e.start_at ASC`, ids)
	if err != nil {
		return nil, err
	}
	events := []AgendaEvent{}
	if err := s.DB.Select(&events, s.DB.Rebind(query), args...); err != nil {
		return nil, err
	}
	if err := s.attachAgendaChildren(events); err != nil {
		return nil, err
	}
	occurrences := []AgendaOccurrence{}
	search = strings.ToLower(strings.TrimSpace(search))
	for _, event := range events {
		for _, occurrence := range expandAgendaEvent(event, allowed[event.CalendarID], from, to) {
			if search != "" {
				text := strings.ToLower(occurrence.Title + " " + deref(occurrence.Location) + " " + deref(occurrence.Description))
				if !strings.Contains(text, search) {
					continue
				}
			}
			occurrences = append(occurrences, occurrence)
		}
	}
	sort.Slice(occurrences, func(i, j int) bool { return occurrences[i].Start.Before(occurrences[j].Start) })
	return occurrences, nil
}

func (s *Store) attachAgendaChildren(events []AgendaEvent) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(events))
	index := map[int64]int{}
	for i := range events {
		ids = append(ids, events[i].ID)
		index[events[i].ID] = i
		events[i].Attendees = []AgendaAttendee{}
		events[i].Reminders = []AgendaReminder{}
		events[i].Exceptions = []AgendaException{}
		events[i].AttachmentData = []AgendaAttachment{}
		if events[i].Recurrence != nil {
			var recurrence AgendaRecurrence
			if json.Unmarshal([]byte(*events[i].Recurrence), &recurrence) == nil {
				events[i].RecurrenceData = &recurrence
			}
		}
		if events[i].Attachments != nil {
			_ = json.Unmarshal([]byte(*events[i].Attachments), &events[i].AttachmentData)
		}
	}
	query, args, err := sqlxIn(`SELECT * FROM agenda_event_attendees WHERE agenda_event_id IN (?) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	attendees := []AgendaAttendee{}
	if err := s.DB.Select(&attendees, s.DB.Rebind(query), args...); err != nil {
		return err
	}
	for _, attendee := range attendees {
		if i, ok := index[attendee.EventID]; ok {
			events[i].Attendees = append(events[i].Attendees, attendee)
		}
	}
	query, args, err = sqlxIn(`SELECT * FROM agenda_event_reminders WHERE agenda_event_id IN (?) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	reminders := []AgendaReminder{}
	if err := s.DB.Select(&reminders, s.DB.Rebind(query), args...); err != nil {
		return err
	}
	for _, reminder := range reminders {
		if i, ok := index[reminder.EventID]; ok {
			events[i].Reminders = append(events[i].Reminders, reminder)
		}
	}
	query, args, err = sqlxIn(`SELECT * FROM agenda_event_exceptions WHERE agenda_event_id IN (?)`, ids)
	if err != nil {
		return err
	}
	exceptions := []AgendaException{}
	if err := s.DB.Select(&exceptions, s.DB.Rebind(query), args...); err != nil {
		return err
	}
	for _, exception := range exceptions {
		if i, ok := index[exception.EventID]; ok {
			events[i].Exceptions = append(events[i].Exceptions, exception)
		}
	}
	return nil
}

func expandAgendaEvent(event AgendaEvent, calendar AgendaCalendar, from, to time.Time) []AgendaOccurrence {
	duration := event.EndAt.Sub(event.StartAt)
	if duration < time.Minute {
		duration = time.Minute
	}
	presentation := func(start, end time.Time, recurrenceID *time.Time) AgendaOccurrence {
		color := ""
		if event.Color != nil {
			color = AgendaColors[*event.Color]
		}
		if color == "" {
			color = AgendaColors[calendar.Color]
		}
		if color == "" {
			color = "#3f51b5"
		}
		return AgendaOccurrence{
			ID: event.ID, RecurrenceID: recurrenceID, Title: event.Title, Description: event.Description,
			Location: event.Location, MeetingURL: event.MeetingURL, Start: start, End: end,
			IsAllDay: event.IsAllDay, IsRecurring: event.RecurrenceData != nil, Status: event.Status,
			Visibility: event.Visibility, Transparency: event.Transparency, Color: color,
			CalendarID: event.CalendarID, CalendarName: deref(event.CalendarName),
			Attendees: event.Attendees, Reminders: event.Reminders, OrganizerName: event.OrganizerName,
			OrganizerEmail: event.OrganizerEmail, IsOrganizer: false,
		}
	}
	if event.RecurrenceData == nil {
		if event.StartAt.Before(to) && event.EndAt.After(from) {
			return []AgendaOccurrence{presentation(event.StartAt, event.EndAt, nil)}
		}
		return nil
	}
	recurrence := *event.RecurrenceData
	if recurrence.Interval < 1 {
		recurrence.Interval = 1
	}
	var until time.Time
	if event.RecurrenceUntil != nil {
		if parsed, err := time.Parse("2006-01-02", *event.RecurrenceUntil); err == nil {
			until = parsed.Add(24*time.Hour - time.Second)
		}
	} else if recurrence.Until != "" {
		if parsed, err := time.Parse("2006-01-02", recurrence.Until); err == nil {
			until = parsed.Add(24*time.Hour - time.Second)
		}
	}
	candidates := generateAgendaStarts(event.StartAt, recurrence, to.Add(24*time.Hour))
	result := []AgendaOccurrence{}
	generated := 0
	for _, candidate := range candidates {
		generated++
		if !until.IsZero() && candidate.After(until) {
			break
		}
		if recurrence.Count > 0 && generated > recurrence.Count {
			break
		}
		start, end := candidate, candidate.Add(duration)
		cancelled := false
		var overrides map[string]any
		for _, exception := range event.Exceptions {
			if exception.RecurrenceID.Equal(candidate) {
				cancelled = exception.IsCancelled
				if exception.Overrides != nil {
					_ = json.Unmarshal([]byte(*exception.Overrides), &overrides)
				}
				break
			}
		}
		if cancelled {
			continue
		}
		occurrence := presentation(start, end, &candidate)
		if overrides != nil {
			applyAgendaOverrides(&occurrence, overrides)
		}
		if occurrence.Start.Before(to) && occurrence.End.After(from) {
			result = append(result, occurrence)
		}
		if candidate.After(to) {
			break
		}
	}
	return result
}

func generateAgendaStarts(anchor time.Time, recurrence AgendaRecurrence, limit time.Time) []time.Time {
	result := []time.Time{}
	interval := recurrence.Interval
	if interval < 1 {
		interval = 1
	}
	switch recurrence.Frequency {
	case "daily":
		for i := 0; i < 1000; i++ {
			candidate := anchor.AddDate(0, 0, i*interval)
			if candidate.After(limit) {
				break
			}
			result = append(result, candidate)
		}
	case "weekly":
		if len(recurrence.ByWeekday) == 0 {
			for i := 0; i < 1000; i++ {
				candidate := anchor.AddDate(0, 0, 7*i*interval)
				if candidate.After(limit) {
					break
				}
				result = append(result, candidate)
			}
		} else {
			weekStart := anchor.AddDate(0, 0, -((int(anchor.Weekday()) + 6) % 7))
			for week := 0; week < 500; week++ {
				base := weekStart.AddDate(0, 0, 7*week*interval)
				if base.After(limit) {
					break
				}
				for _, weekday := range recurrence.ByWeekday {
					offset := weekday - 1
					candidate := time.Date(base.Year(), base.Month(), base.Day()+offset, anchor.Hour(), anchor.Minute(), 0, 0, anchor.Location())
					if candidate.Before(anchor) || candidate.After(limit) {
						continue
					}
					result = append(result, candidate)
				}
			}
		}
	case "monthly":
		for i := 0; i < 500; i++ {
			base := time.Date(anchor.Year(), anchor.Month()+time.Month(i*interval), 1, anchor.Hour(), anchor.Minute(), 0, 0, anchor.Location())
			if base.After(limit) {
				break
			}
			if len(recurrence.ByMonthDay) == 0 {
				candidate := time.Date(base.Year(), base.Month(), anchor.Day(), anchor.Hour(), anchor.Minute(), 0, 0, anchor.Location())
				if candidate.Month() != base.Month() {
					candidate = time.Date(base.Year(), base.Month()+1, 0, anchor.Hour(), anchor.Minute(), 0, 0, anchor.Location())
				}
				if !candidate.Before(anchor) && !candidate.After(limit) {
					result = append(result, candidate)
				}
			} else {
				for _, day := range recurrence.ByMonthDay {
					last := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, base.Location()).Day()
					if day > last {
						day = last
					}
					candidate := time.Date(base.Year(), base.Month(), day, anchor.Hour(), anchor.Minute(), 0, 0, anchor.Location())
					if !candidate.Before(anchor) && !candidate.After(limit) {
						result = append(result, candidate)
					}
				}
			}
		}
	case "yearly":
		for i := 0; i < 100; i++ {
			candidate := anchor.AddDate(i*interval, 0, 0)
			if candidate.After(limit) {
				break
			}
			result = append(result, candidate)
		}
	default:
		result = append(result, anchor)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Before(result[j]) })
	return result
}

func applyAgendaOverrides(occurrence *AgendaOccurrence, overrides map[string]any) {
	if value, ok := overrides["title"].(string); ok && value != "" {
		occurrence.Title = value
	}
	if value, ok := overrides["description"].(string); ok {
		occurrence.Description = &value
	}
	if value, ok := overrides["location"].(string); ok {
		occurrence.Location = &value
	}
	if value, ok := overrides["status"].(string); ok && value != "" {
		occurrence.Status = value
	}
	if value, ok := overrides["start_at"].(string); ok {
		if parsed, err := parseDateTime(value); err == nil {
			occurrence.Start = parsed
		}
	}
	if value, ok := overrides["end_at"].(string); ok {
		if parsed, err := parseDateTime(value); err == nil {
			occurrence.End = parsed
		}
	}
	if value, ok := overrides["is_all_day"].(bool); ok {
		occurrence.IsAllDay = value
	}
}

func containsInt64(values []int64, wanted int64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func sqlxIn(query string, args ...any) (string, []any, error) {
	return sqlx.In(query, args...)
}
