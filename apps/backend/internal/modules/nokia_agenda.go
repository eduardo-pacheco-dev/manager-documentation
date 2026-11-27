package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Nokia project dates -> Agenda
 *
 * Every date field of a Nokia site is mirrored into a dedicated agenda
 * calendar named "Projetos Nokia". The mapping table keeps the relationship
 * so edits move the same all-day event and cleared dates remove it.
 * ----------------------------------------------------------------------- */

const NokiaAgendaCalendarName = "Projetos Nokia"

type NokiaDateDefinition struct {
	Field string
	Label string
}

var NokiaDateDefinitions = []NokiaDateDefinition{
	{"mos_plan_at", "MOS - Planejado"},
	{"mos_real_at", "MOS - Realizado"},
	{"instalacao_plan_at", "Instalação - Planejado"},
	{"instalacao_real_at", "Instalação - Realizado"},
	{"integracao_plan_at", "Integração - Planejado"},
	{"integracao_real_at", "Integração - Realizado"},
	{"rfa_plan_at", "RFA - Planejado"},
	{"rfa_real_at", "RFA - Realizado"},
	{"fam_entrega_approved_at", "FAM Entrega - Aprovado"},
	{"fam_instalacao_approved_at", "FAM Instalação - Aprovado"},
	{"fam_panoramica_approved_at", "FAM Panorâmica - Aprovado"},
}

func nokiaDateValue(site NokiaSite, field string) string {
	switch field {
	case "mos_plan_at":
		return deref(site.MOSPlanAt)
	case "mos_real_at":
		return deref(site.MOSRealAt)
	case "instalacao_plan_at":
		return deref(site.InstalacaoPlanAt)
	case "instalacao_real_at":
		return deref(site.InstalacaoRealAt)
	case "integracao_plan_at":
		return deref(site.IntegracaoPlanAt)
	case "integracao_real_at":
		return deref(site.IntegracaoRealAt)
	case "rfa_plan_at":
		return deref(site.RFAPlanAt)
	case "rfa_real_at":
		return deref(site.RFARealAt)
	case "fam_entrega_approved_at":
		return deref(site.FamEntregaApprovedAt)
	case "fam_instalacao_approved_at":
		return deref(site.FamInstalacaoApprovedAt)
	case "fam_panoramica_approved_at":
		return deref(site.FamPanoramicaApprovedAt)
	default:
		return ""
	}
}

func nokiaDateStart(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, ErrValidation
	}
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return parsed, nil
	}
	if parsed, err := parseDateTime(value); err == nil {
		return parsed, nil
	}
	return time.Time{}, ErrValidation
}

// NokiaAgendaCalendar returns (or creates) the dedicated calendar used for
// Nokia project milestones for the given user.
func (s *Store) NokiaAgendaCalendar(userID int64) (AgendaCalendar, error) {
	calendars, err := s.AgendaCalendars(userID)
	if err != nil {
		return AgendaCalendar{}, err
	}
	for _, calendar := range calendars {
		if calendar.Name == NokiaAgendaCalendarName && (calendar.Role == "owner" || calendar.Role == "writer") {
			return calendar, nil
		}
	}
	if len(calendars) == 0 {
		if _, err := s.CreateAgendaCalendar(userID, AgendaCalendarInput{
			Name: "Minha agenda", Color: "blueberry", IsDefault: true,
		}); err != nil {
			return AgendaCalendar{}, err
		}
	} else {
		ownsCalendar := false
		for _, calendar := range calendars {
			if calendar.UserID == userID {
				ownsCalendar = true
				break
			}
		}
		if !ownsCalendar {
			if _, err := s.CreateAgendaCalendar(userID, AgendaCalendarInput{
				Name: "Minha agenda", Color: "blueberry", IsDefault: true,
			}); err != nil {
				return AgendaCalendar{}, err
			}
		}
	}
	return s.CreateAgendaCalendar(userID, AgendaCalendarInput{
		Name:        NokiaAgendaCalendarName,
		Description: "Datas planejadas, realizadas e aprovações dos projetos Nokia.",
		Color:       "peacock",
	})
}

func (s *Store) nokiaAgendaEventID(siteID int64, field string) (int64, bool, error) {
	var eventID int64
	err := s.DB.Get(&eventID, `SELECT agenda_event_id FROM nokia_site_agenda_events
		WHERE nokia_site_id = ? AND date_field = ?`, siteID, field)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return eventID, true, nil
}

// SyncNokiaDatesToAgenda mirrors the current Nokia site dates into the
// dedicated agenda calendar. It is safe to call after every date update.
func (s *Store) SyncNokiaDatesToAgenda(userID, siteID int64) error {
	site, err := s.FindNokiaSite(siteID)
	if err != nil {
		return err
	}
	hasDates := false
	for _, definition := range NokiaDateDefinitions {
		if strings.TrimSpace(nokiaDateValue(site, definition.Field)) != "" {
			hasDates = true
			break
		}
	}

	var calendarID int64
	if hasDates {
		calendar, err := s.NokiaAgendaCalendar(userID)
		if err != nil {
			return err
		}
		calendarID = calendar.ID
	}

	for _, definition := range NokiaDateDefinitions {
		value := strings.TrimSpace(nokiaDateValue(site, definition.Field))
		eventID, found, err := s.nokiaAgendaEventID(siteID, definition.Field)
		if err != nil {
			return err
		}
		if value == "" {
			if found {
				_, _ = s.DB.Exec(`DELETE FROM agenda_events WHERE id = ?`, eventID)
				_, _ = s.DB.Exec(`DELETE FROM nokia_site_agenda_events WHERE nokia_site_id = ? AND date_field = ?`,
					siteID, definition.Field)
			}
			continue
		}
		start, err := nokiaDateStart(value)
		if err != nil {
			continue
		}
		end := start.AddDate(0, 0, 1)
		title := site.SiteID + " - " + definition.Label
		description := fmt.Sprintf("Data do projeto Nokia.\nSite ID: %s\nOC: %s\nEtapa: %s",
			site.SiteID, deref(site.OC), definition.Label)
		if found {
			_, err = s.DB.Exec(`UPDATE agenda_events SET title = ?, description = ?, location = ?,
				start_at = ?, end_at = ?, is_all_day = 1, updated_at = ? WHERE id = ?`,
				title, description, site.SiteID, start, end, now(), eventID)
			if err != nil {
				return err
			}
			continue
		}
		event, err := s.CreateAgendaEvent(userID, AgendaEventInput{
			CalendarID:   calendarID,
			Title:        title,
			Description:  description,
			Location:     site.SiteID,
			StartAt:      start.Format("2006-01-02"),
			EndAt:        end.Format("2006-01-02"),
			IsAllDay:     true,
			Status:       "confirmed",
			Visibility:   "default",
			Transparency: "transparent",
		})
		if err != nil {
			return err
		}
		_, err = s.DB.Exec(`INSERT INTO nokia_site_agenda_events
			(nokia_site_id, date_field, agenda_event_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE agenda_event_id = VALUES(agenda_event_id), updated_at = VALUES(updated_at)`,
			siteID, definition.Field, event.ID, now(), now())
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteNokiaAgendaEvents removes the agenda events generated for a site.
// It is called when the Nokia site itself is deleted.
func (s *Store) DeleteNokiaAgendaEvents(siteID int64) {
	ids := []int64{}
	_ = s.DB.Select(&ids, `SELECT agenda_event_id FROM nokia_site_agenda_events WHERE nokia_site_id = ?`, siteID)
	for _, eventID := range ids {
		_, _ = s.DB.Exec(`DELETE FROM agenda_events WHERE id = ?`, eventID)
	}
	_, _ = s.DB.Exec(`DELETE FROM nokia_site_agenda_events WHERE nokia_site_id = ?`, siteID)
}
