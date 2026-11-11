package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Cross-module integration
 *
 * Entity links connect work items (tasks, follow-ups and agenda events) to
 * inventory resources (stations and Nokia project sites). The same structure
 * powers the related-work panel on detail pages and the work center widget.
 * ----------------------------------------------------------------------- */

var WorkModules = map[string]bool{"task": true, "follow_up": true, "agenda_event": true}
var WorkResources = map[string]bool{"station": true, "nokia_site": true}

type WorkLink struct {
	ID           int64     `db:"id" json:"id"`
	Module       string    `db:"module" json:"module"`
	EntityID     int64     `db:"entity_id" json:"entity_id"`
	ResourceType string    `db:"resource_type" json:"resource_type"`
	ResourceID   int64     `db:"resource_id" json:"resource_id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type WorkResource struct {
	Type        string  `json:"type"`
	ID          int64   `json:"id"`
	Label       string  `json:"label"`
	Subtitle    string  `json:"subtitle,omitempty"`
	URL         string  `json:"url"`
	StationID   *string `json:"station_id,omitempty"`
	NokiaSiteID *int64  `json:"nokia_site_id,omitempty"`
}

type WorkTaskItem struct {
	ID          int64      `db:"id" json:"id"`
	LinkID      int64      `db:"link_id" json:"link_id,omitempty"`
	Title       string     `db:"title" json:"title"`
	ListName    string     `db:"list_name" json:"list_name"`
	DueAt       *string    `db:"due_at" json:"due_at"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at"`
	IsImportant bool       `db:"is_important" json:"is_important"`
}

type WorkFollowUpItem struct {
	ID           int64   `db:"id" json:"id"`
	LinkID       int64   `db:"link_id" json:"link_id,omitempty"`
	Title        string  `db:"title" json:"title"`
	Status       string  `db:"status" json:"status"`
	Priority     string  `db:"priority" json:"priority"`
	DueDate      *string `db:"due_date" json:"due_date"`
	AssigneeName *string `db:"assignee_name" json:"assignee_name"`
	Overdue      bool    `json:"overdue"`
}

type WorkEventItem struct {
	ID           int64     `db:"id" json:"id"`
	LinkID       int64     `db:"link_id" json:"link_id,omitempty"`
	Title        string    `db:"title" json:"title"`
	StartAt      time.Time `db:"start_at" json:"start_at"`
	EndAt        time.Time `db:"end_at" json:"end_at"`
	Location     *string   `db:"location" json:"location"`
	CalendarName string    `db:"calendar_name" json:"calendar_name"`
	Status       string    `db:"status" json:"status"`
}

type WorkCounts struct {
	TasksOpen        int `json:"tasks_open"`
	TasksOverdue     int `json:"tasks_overdue"`
	FollowUpsOpen    int `json:"follow_ups_open"`
	FollowUpsOverdue int `json:"follow_ups_overdue"`
	EventsToday      int `json:"events_today"`
	EventsUpcoming   int `json:"events_upcoming"`
}

type WorkContext struct {
	Resource  WorkResource       `json:"resource"`
	Linked    *WorkResource      `json:"linked_resource"`
	Tasks     []WorkTaskItem     `json:"tasks"`
	FollowUps []WorkFollowUpItem `json:"follow_ups"`
	Events    []WorkEventItem    `json:"events"`
	Counts    WorkCounts         `json:"counts"`
}

type WorkCenter struct {
	Tasks     []WorkTaskItem     `json:"tasks"`
	FollowUps []WorkFollowUpItem `json:"follow_ups"`
	Events    []WorkEventItem    `json:"events"`
	Counts    WorkCounts         `json:"counts"`
}

// WorkResource exposes a resource summary for other http handlers.
func (s *Store) WorkResource(resourceType string, id int64) (WorkResource, error) {
	return s.workResource(resourceType, id)
}

func (s *Store) workResource(resourceType string, id int64) (WorkResource, error) {
	switch resourceType {
	case "station":
		var row struct {
			ID          int64   `db:"id"`
			SiteID      string  `db:"site_id"`
			StationID   *string `db:"station_id"`
			Municipio   *string `db:"municipio"`
			Estado      *string `db:"estado"`
			Logradouro  *string `db:"logradouro"`
			Numero      *string `db:"numero"`
			NokiaSiteID *int64  `db:"nokia_site_id"`
		}
		err := s.DB.Get(&row, `SELECT id, site_id, station_id, municipio, estado, logradouro, numero, nokia_site_id
			FROM stations WHERE id = ?`, id)
		if errors.Is(err, sql.ErrNoRows) {
			return WorkResource{}, ErrNotFound
		}
		if err != nil {
			return WorkResource{}, err
		}
		subtitle := strings.TrimSpace(joinNonEmpty([]string{
			joinNonEmpty([]string{deref(row.Logradouro), deref(row.Numero)}, ", "),
			joinNonEmpty([]string{deref(row.Municipio), deref(row.Estado)}, "/"),
		}, " — "))
		return WorkResource{
			Type: "station", ID: row.ID, Label: row.SiteID, Subtitle: subtitle,
			URL: fmt.Sprintf("/stations/%d", row.ID), StationID: row.StationID, NokiaSiteID: row.NokiaSiteID,
		}, nil
	case "nokia_site":
		var row struct {
			ID       int64   `db:"id"`
			SiteID   string  `db:"site_id"`
			OC       *string `db:"oc"`
			Regional *string `db:"regional"`
		}
		err := s.DB.Get(&row, `SELECT id, site_id, oc, regional FROM nokia_sites WHERE id = ?`, id)
		if errors.Is(err, sql.ErrNoRows) {
			return WorkResource{}, ErrNotFound
		}
		if err != nil {
			return WorkResource{}, err
		}
		subtitle := strings.TrimSpace(joinNonEmpty([]string{
			prefixIf(deref(row.OC), "OC "),
			prefixIf(deref(row.Regional), "Regional "),
		}, " — "))
		return WorkResource{
			Type: "nokia_site", ID: row.ID, Label: row.SiteID, Subtitle: subtitle,
			URL: fmt.Sprintf("/nokia/%d", row.ID),
		}, nil
	default:
		return WorkResource{}, ErrValidation
	}
}

func prefixIf(value, prefix string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return prefix + value
}

func (s *Store) linkedResource(resource WorkResource) *WorkResource {
	if resource.Type == "station" && resource.NokiaSiteID != nil {
		linked, err := s.workResource("nokia_site", *resource.NokiaSiteID)
		if err == nil {
			return &linked
		}
	}
	if resource.Type == "nokia_site" {
		var stationID int64
		if err := s.DB.Get(&stationID, `SELECT id FROM stations WHERE nokia_site_id = ?`, resource.ID); err == nil {
			linked, findErr := s.workResource("station", stationID)
			if findErr == nil {
				return &linked
			}
		}
	}
	return nil
}

func (s *Store) entityAccessible(userID int64, module string, entityID int64) (bool, error) {
	var count int
	var err error
	switch module {
	case "task":
		err = s.DB.Get(&count, `SELECT COUNT(*) FROM todo_tasks WHERE id = ? AND user_id = ?`, entityID, userID)
	case "follow_up":
		err = s.DB.Get(&count, `SELECT COUNT(*) FROM follow_ups WHERE id = ?`, entityID)
	case "agenda_event":
		err = s.DB.Get(&count, `SELECT COUNT(*) FROM agenda_events e
			JOIN agenda_calendars c ON c.id = e.agenda_calendar_id
			WHERE e.id = ? AND (c.user_id = ? OR c.id IN (SELECT agenda_calendar_id FROM agenda_calendar_shares WHERE user_id = ?))`,
			entityID, userID, userID)
	default:
		return false, ErrValidation
	}
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateWorkLink(userID int64, module string, entityID int64, resourceType string, resourceID int64) (WorkLink, error) {
	if !WorkModules[module] || !WorkResources[resourceType] {
		return WorkLink{}, ErrValidation
	}
	accessible, err := s.entityAccessible(userID, module, entityID)
	if err != nil {
		return WorkLink{}, err
	}
	if !accessible {
		return WorkLink{}, ErrNotFound
	}
	if _, err := s.workResource(resourceType, resourceID); err != nil {
		return WorkLink{}, err
	}
	timestamp := now()
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO entity_links
		(module, entity_id, resource_type, resource_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		module, entityID, resourceType, resourceID, timestamp); err != nil {
		return WorkLink{}, err
	}
	var link WorkLink
	err = s.DB.Get(&link, `SELECT * FROM entity_links
		WHERE module = ? AND entity_id = ? AND resource_type = ? AND resource_id = ?`,
		module, entityID, resourceType, resourceID)
	return link, err
}

func (s *Store) DeleteWorkLink(userID, linkID int64) error {
	var link WorkLink
	if err := s.DB.Get(&link, `SELECT * FROM entity_links WHERE id = ?`, linkID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	accessible, err := s.entityAccessible(userID, link.Module, link.EntityID)
	if err != nil {
		return err
	}
	if !accessible {
		return ErrForbidden
	}
	_, err = s.DB.Exec(`DELETE FROM entity_links WHERE id = ?`, linkID)
	return err
}

// DeleteEntityLinks removes every link attached to an entity. It is called
// when the entity itself is deleted so orphan rows never accumulate.
func (s *Store) DeleteEntityLinks(module string, entityID int64) {
	_, _ = s.DB.Exec(`DELETE FROM entity_links WHERE module = ? AND entity_id = ?`, module, entityID)
}

func (s *Store) DeleteResourceLinks(resourceType string, resourceID int64) {
	_, _ = s.DB.Exec(`DELETE FROM entity_links WHERE resource_type = ? AND resource_id = ?`, resourceType, resourceID)
}

func (s *Store) WorkContext(userID int64, resourceType string, resourceID int64) (WorkContext, error) {
	resource, err := s.workResource(resourceType, resourceID)
	if err != nil {
		return WorkContext{}, err
	}
	context := WorkContext{
		Resource:  resource,
		Linked:    s.linkedResource(resource),
		Tasks:     []WorkTaskItem{},
		FollowUps: []WorkFollowUpItem{},
		Events:    []WorkEventItem{},
	}
	if err := s.DB.Select(&context.Tasks, `SELECT t.id, el.id AS link_id, t.title, l.name AS list_name,
			t.due_at, t.completed_at, t.is_important
		FROM entity_links el
		JOIN todo_tasks t ON t.id = el.entity_id
		JOIN todo_lists l ON l.id = t.todo_list_id
		WHERE el.module = 'task' AND el.resource_type = ? AND el.resource_id = ? AND t.user_id = ?
		ORDER BY (t.completed_at IS NOT NULL) ASC, t.position ASC, t.id ASC`, resourceType, resourceID, userID); err != nil {
		return WorkContext{}, err
	}
	if err := s.DB.Select(&context.FollowUps, `SELECT f.id, el.id AS link_id, f.title, f.status, f.priority,
			f.due_date, a.name AS assignee_name
		FROM entity_links el
		JOIN follow_ups f ON f.id = el.entity_id
		LEFT JOIN follow_up_assignees a ON a.id = f.assignee_id
		WHERE el.module = 'follow_up' AND el.resource_type = ? AND el.resource_id = ?
		ORDER BY (f.status IN ('concluido','cancelado')) ASC, f.due_date IS NULL ASC, f.due_date ASC, f.id DESC`,
		resourceType, resourceID); err != nil {
		return WorkContext{}, err
	}
	if err := s.DB.Select(&context.Events, `SELECT e.id, el.id AS link_id, e.title, e.start_at, e.end_at,
			e.location, c.name AS calendar_name, e.status
		FROM entity_links el
		JOIN agenda_events e ON e.id = el.entity_id
		JOIN agenda_calendars c ON c.id = e.agenda_calendar_id
		WHERE el.module = 'agenda_event' AND el.resource_type = ? AND el.resource_id = ?
			AND (c.user_id = ? OR c.id IN (SELECT agenda_calendar_id FROM agenda_calendar_shares WHERE user_id = ?))
		ORDER BY e.start_at DESC LIMIT 20`, resourceType, resourceID, userID, userID); err != nil {
		return WorkContext{}, err
	}
	today := time.Now().Format("2006-01-02")
	for i := range context.FollowUps {
		item := &context.FollowUps[i]
		if item.DueDate != nil && item.Status != "concluido" && item.Status != "cancelado" && *item.DueDate < today {
			item.Overdue = true
		}
		if item.Status != "concluido" && item.Status != "cancelado" {
			context.Counts.FollowUpsOpen++
		}
		if item.Overdue {
			context.Counts.FollowUpsOverdue++
		}
	}
	for i := range context.Tasks {
		if context.Tasks[i].CompletedAt == nil {
			context.Counts.TasksOpen++
		}
	}
	now := time.Now()
	for i := range context.Events {
		if context.Events[i].EndAt.After(now) {
			context.Counts.EventsUpcoming++
		}
		if sameDay(context.Events[i].StartAt, now) {
			context.Counts.EventsToday++
		}
	}
	return context, nil
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Local().Date()
	by, bm, bd := b.Local().Date()
	return ay == by && am == bm && ad == bd
}

func (s *Store) WorkCenter(userID int64, email string) (WorkCenter, error) {
	result := WorkCenter{
		Tasks:     []WorkTaskItem{},
		FollowUps: []WorkFollowUpItem{},
		Events:    []WorkEventItem{},
	}
	today := time.Now().Format("2006-01-02")
	week := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	if err := s.DB.Select(&result.Tasks, `SELECT t.id, t.title, l.name AS list_name, t.due_at,
			t.completed_at, t.is_important
		FROM todo_tasks t JOIN todo_lists l ON l.id = t.todo_list_id
		WHERE t.user_id = ? AND t.completed_at IS NULL AND t.due_at IS NOT NULL AND t.due_at <= ?
		ORDER BY t.due_at ASC, t.position ASC LIMIT 6`, userID, week); err != nil {
		return WorkCenter{}, err
	}
	if err := s.DB.Get(&result.Counts.TasksOverdue, `SELECT COUNT(*) FROM todo_tasks
		WHERE user_id = ? AND completed_at IS NULL AND due_at IS NOT NULL AND due_at < ?`, userID, today); err != nil {
		return WorkCenter{}, err
	}
	if err := s.DB.Get(&result.Counts.TasksOpen, `SELECT COUNT(*) FROM todo_tasks
		WHERE user_id = ? AND completed_at IS NULL`, userID); err != nil {
		return WorkCenter{}, err
	}
	if strings.TrimSpace(email) != "" {
		if err := s.DB.Select(&result.FollowUps, `SELECT f.id, f.title, f.status, f.priority, f.due_date,
				a.name AS assignee_name
			FROM follow_ups f JOIN follow_up_assignees a ON a.id = f.assignee_id
			WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ?
				AND f.due_date IS NOT NULL AND f.due_date <= ?
			ORDER BY f.due_date ASC LIMIT 6`, strings.ToLower(email), week); err != nil {
			return WorkCenter{}, err
		}
		if err := s.DB.Get(&result.Counts.FollowUpsOpen, `SELECT COUNT(*) FROM follow_ups f
			JOIN follow_up_assignees a ON a.id = f.assignee_id
			WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ?`, strings.ToLower(email)); err != nil {
			return WorkCenter{}, err
		}
		if err := s.DB.Get(&result.Counts.FollowUpsOverdue, `SELECT COUNT(*) FROM follow_ups f
			JOIN follow_up_assignees a ON a.id = f.assignee_id
			WHERE f.status NOT IN ('concluido','cancelado') AND a.email = ? AND f.due_date < ?`,
			strings.ToLower(email), today); err != nil {
			return WorkCenter{}, err
		}
	}
	for i := range result.FollowUps {
		if result.FollowUps[i].DueDate != nil && *result.FollowUps[i].DueDate < today {
			result.FollowUps[i].Overdue = true
		}
	}
	from := time.Now()
	occurrences, err := s.AgendaEventsInRange(userID, nil, from, from.AddDate(0, 0, 7), "")
	if err != nil {
		return WorkCenter{}, err
	}
	for _, occurrence := range occurrences {
		if len(result.Events) >= 6 {
			break
		}
		result.Events = append(result.Events, WorkEventItem{
			ID: occurrence.ID, Title: occurrence.Title, StartAt: occurrence.Start,
			EndAt: occurrence.End, Location: occurrence.Location,
			CalendarName: occurrence.CalendarName, Status: occurrence.Status,
		})
	}
	result.Counts.EventsUpcoming = len(occurrences)
	for _, occurrence := range occurrences {
		if sameDay(occurrence.Start, from) {
			result.Counts.EventsToday++
		}
	}
	return result, nil
}
