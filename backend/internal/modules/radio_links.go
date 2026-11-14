package modules

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Radio links module (point-to-point links between stations)
 * ----------------------------------------------------------------------- */

// RadioLink is a point-to-point radio connection between two stations.
type RadioLink struct {
	ID            int64     `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Code          *string   `db:"code" json:"code"`
	StationAID    int64     `db:"station_a_id" json:"station_a_id"`
	StationBID    int64     `db:"station_b_id" json:"station_b_id"`
	Status        string    `db:"status" json:"status"`
	Technology    *string   `db:"technology" json:"technology"`
	FrequencyMHz  *float64  `db:"frequency_mhz" json:"frequency_mhz"`
	CapacityMbps  *float64  `db:"capacity_mbps" json:"capacity_mbps"`
	DistanceKm    *float64  `db:"distance_km" json:"distance_km"`
	Notes         *string   `db:"notes" json:"notes"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	StationASite  *string   `db:"station_a_site_id" json:"station_a_site_id"`
	StationACode  *string   `db:"station_a_station_id" json:"station_a_station_id"`
	StationACity  *string   `db:"station_a_municipio" json:"station_a_municipio"`
	StationAState *string   `db:"station_a_estado" json:"station_a_estado"`
	StationAType  *string   `db:"station_a_tipo_de_elemento" json:"station_a_tipo_de_elemento"`
	StationBSite  *string   `db:"station_b_site_id" json:"station_b_site_id"`
	StationBCode  *string   `db:"station_b_station_id" json:"station_b_station_id"`
	StationBCity  *string   `db:"station_b_municipio" json:"station_b_municipio"`
	StationBState *string   `db:"station_b_estado" json:"station_b_estado"`
	StationBType  *string   `db:"station_b_tipo_de_elemento" json:"station_b_tipo_de_elemento"`
}

type RadioLinkStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Planned  int `json:"planned"`
	Inactive int `json:"inactive"`
	Stations int `json:"stations"`
}

type RadioLinkFilter struct {
	Search     string
	Status     string
	Technology string
	Page       int
	PerPage    int
}

const radioLinkSelect = `SELECT r.*,
	a.site_id AS station_a_site_id, a.station_id AS station_a_station_id,
	a.municipio AS station_a_municipio, a.estado AS station_a_estado,
	a.tipo_de_elemento AS station_a_tipo_de_elemento,
	b.site_id AS station_b_site_id, b.station_id AS station_b_station_id,
	b.municipio AS station_b_municipio, b.estado AS station_b_estado,
	b.tipo_de_elemento AS station_b_tipo_de_elemento
	FROM radio_links r
	JOIN stations a ON a.id = r.station_a_id
	JOIN stations b ON b.id = r.station_b_id`

var radioLinkOrder = ` ORDER BY r.updated_at DESC, r.id DESC`

func (s *Store) ListRadioLinks(filter RadioLinkFilter) ([]RadioLink, int, RadioLinkStats, []string, []string, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(r.name LIKE ? OR COALESCE(r.code, '') LIKE ? OR COALESCE(r.technology, '') LIKE ? OR a.site_id LIKE ? OR b.site_id LIKE ? OR COALESCE(a.station_id, '') LIKE ? OR COALESCE(b.station_id, '') LIKE ? OR COALESCE(a.municipio, '') LIKE ? OR COALESCE(b.municipio, '') LIKE ?)`)
		for i := 0; i < 9; i++ {
			args = append(args, like)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "r.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Technology != "" {
		conditions = append(conditions, "r.technology = ?")
		args = append(args, filter.Technology)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 15
	}
	where := strings.Join(conditions, " AND ")

	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM radio_links r
		JOIN stations a ON a.id = r.station_a_id
		JOIN stations b ON b.id = r.station_b_id WHERE `+where, args...); err != nil {
		return nil, 0, RadioLinkStats{}, nil, nil, err
	}

	query := radioLinkSelect + ` WHERE ` + where + radioLinkOrder + ` LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	links := []RadioLink{}
	if err := s.DB.Select(&links, query, queryArgs...); err != nil {
		return nil, 0, RadioLinkStats{}, nil, nil, err
	}

	var stats RadioLinkStats
	if err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COALESCE(SUM(status = 'ativo'), 0) AS active,
		COALESCE(SUM(status = 'planejado'), 0) AS planned,
		COALESCE(SUM(status = 'inativo'), 0) AS inactive,
		COUNT(DISTINCT station_a_id) + COUNT(DISTINCT station_b_id) AS stations
		FROM radio_links`); err != nil {
		return nil, 0, stats, nil, nil, err
	}
	statuses := []string{}
	technologies := []string{}
	_ = s.DB.Select(&statuses, `SELECT DISTINCT status FROM radio_links WHERE status <> '' ORDER BY status`)
	_ = s.DB.Select(&technologies, `SELECT DISTINCT technology FROM radio_links WHERE technology IS NOT NULL AND technology <> '' ORDER BY technology`)
	return links, total, stats, statuses, technologies, nil
}

func (s *Store) FindRadioLink(id int64) (RadioLink, error) {
	var link RadioLink
	err := s.DB.Get(&link, radioLinkSelect+` WHERE r.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return link, ErrNotFound
	}
	return link, err
}

type RadioLinkInput struct {
	Name         string
	Code         string
	StationAID   int64
	StationBID   int64
	Status       string
	Technology   string
	FrequencyMHz *float64
	CapacityMbps *float64
	DistanceKm   *float64
	Notes        string
}

func (s *Store) CreateRadioLink(input RadioLinkInput) (RadioLink, error) {
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO radio_links (name, code, station_a_id, station_b_id, status, technology,
			frequency_mhz, capacity_mbps, distance_km, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.Name), nullString(input.Code), input.StationAID, input.StationBID,
		input.Status, nullString(input.Technology), nullFloat(input.FrequencyMHz),
		nullFloat(input.CapacityMbps), nullFloat(input.DistanceKm), nullString(input.Notes),
		timestamp, timestamp)
	if err != nil {
		return RadioLink{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindRadioLink(id)
}

func (s *Store) UpdateRadioLink(id int64, input RadioLinkInput) (RadioLink, error) {
	result, err := s.DB.Exec(`
		UPDATE radio_links SET name = ?, code = ?, station_a_id = ?, station_b_id = ?, status = ?,
			technology = ?, frequency_mhz = ?, capacity_mbps = ?, distance_km = ?, notes = ?, updated_at = ?
		WHERE id = ?`,
		strings.TrimSpace(input.Name), nullString(input.Code), input.StationAID, input.StationBID,
		input.Status, nullString(input.Technology), nullFloat(input.FrequencyMHz),
		nullFloat(input.CapacityMbps), nullFloat(input.DistanceKm), nullString(input.Notes),
		now(), id)
	if err != nil {
		return RadioLink{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return RadioLink{}, ErrNotFound
	}
	return s.FindRadioLink(id)
}

func (s *Store) DeleteRadioLink(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM radio_links WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func nullFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
