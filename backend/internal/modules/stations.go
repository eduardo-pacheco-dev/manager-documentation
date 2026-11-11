package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

/* --------------------------------------------------------------------------
 * Stations module
 * ----------------------------------------------------------------------- */

type Station struct {
	ID               int64               `db:"id" json:"id"`
	SiteID           string              `db:"site_id" json:"site_id"`
	EnderecoID       *string             `db:"endereco_id" json:"endereco_id"`
	TipoDeElemento   *string             `db:"tipo_de_elemento" json:"tipo_de_elemento"`
	Tecnologia       *string             `db:"tecnologia" json:"tecnologia"`
	TipoDeConexao    *string             `db:"tipo_de_conexao" json:"tipo_de_conexao"`
	Classificacao    *string             `db:"classificacao" json:"classificacao"`
	DetentorArea     *string             `db:"detentor_area" json:"detentor_area"`
	TipoDeInfra      *string             `db:"tipo_de_infra" json:"tipo_de_infra"`
	TipoDeEV         *string             `db:"tipo_de_ev" json:"tipo_de_ev"`
	TipoDeLogradouro *string             `db:"tipo_de_logradouro" json:"tipo_de_logradouro"`
	Logradouro       *string             `db:"logradouro" json:"logradouro"`
	Numero           *string             `db:"numero" json:"numero"`
	Complemento      *string             `db:"complemento" json:"complemento"`
	Bairro           *string             `db:"bairro" json:"bairro"`
	Municipio        *string             `db:"municipio" json:"municipio"`
	Estado           *string             `db:"estado" json:"estado"`
	CEP              *string             `db:"cep" json:"cep"`
	Regional         *string             `db:"regional" json:"regional"`
	Latitude         *string             `db:"latitude" json:"latitude"`
	Longitude        *string             `db:"longitude" json:"longitude"`
	Status           *string             `db:"status" json:"status"`
	TipoDaTorre      *string             `db:"tipo_da_torre" json:"tipo_da_torre"`
	StationID        *string             `db:"station_id" json:"station_id"`
	NokiaSiteID      *int64              `db:"nokia_site_id" json:"nokia_site_id"`
	CreatedAt        time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time           `db:"updated_at" json:"updated_at"`
	Attachments      []StationAttachment `db:"-" json:"attachments"`
	Comments         []StationComment    `db:"-" json:"comments"`
}

func (s Station) Address() string {
	parts := []string{}
	line := strings.TrimSpace(joinNonEmpty([]string{deref(s.Logradouro), deref(s.Numero), deref(s.Complemento)}, ", "))
	if line != "" {
		parts = append(parts, line)
	}
	if bairro := deref(s.Bairro); bairro != "" {
		parts = append(parts, bairro)
	}
	if municipio := deref(s.Municipio); municipio != "" {
		parts = append(parts, municipio)
	}
	return strings.Join(parts, " — ")
}

type StationAttachment struct {
	ID           int64     `db:"id" json:"id"`
	StationID    int64     `db:"station_id" json:"-"`
	UploadedBy   *int64    `db:"uploaded_by" json:"uploaded_by"`
	Category     string    `db:"category" json:"category"`
	OriginalName string    `db:"original_name" json:"original_name"`
	Path         string    `db:"path" json:"path"`
	Disk         string    `db:"disk" json:"disk"`
	MimeType     *string   `db:"mime_type" json:"mime_type"`
	Size         int64     `db:"size" json:"size"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	UploaderName *string   `db:"uploader_name" json:"uploader_name"`
	URL          string    `db:"-" json:"url"`
}

type StationComment struct {
	ID         int64     `db:"id" json:"id"`
	StationID  int64     `db:"station_id" json:"-"`
	UserID     *int64    `db:"user_id" json:"user_id"`
	Body       string    `db:"body" json:"body"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	AuthorName *string   `db:"author_name" json:"author_name"`
}

type StationFilter struct {
	Search     string
	Status     string
	Tecnologia string
	Page       int
	PerPage    int
}

type StationStats struct {
	Total      int `json:"total"`
	Sites      int `json:"sites"`
	Municipios int `json:"municipios"`
}

func (s *Store) ListStations(filter StationFilter) ([]Station, int, StationStats, []string, []string, error) {
	conditions := []string{"1 = 1"}
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, `(site_id LIKE ? OR COALESCE(station_id, '') LIKE ? OR COALESCE(endereco_id, '') LIKE ? OR COALESCE(tipo_de_elemento, '') LIKE ? OR COALESCE(tecnologia, '') LIKE ? OR COALESCE(logradouro, '') LIKE ? OR COALESCE(bairro, '') LIKE ? OR COALESCE(municipio, '') LIKE ?)`)
		for i := 0; i < 8; i++ {
			args = append(args, like)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Tecnologia != "" {
		conditions = append(conditions, "tecnologia = ?")
		args = append(args, filter.Tecnologia)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 15
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM stations WHERE `+where, args...); err != nil {
		return nil, 0, StationStats{}, nil, nil, err
	}
	query := `SELECT * FROM stations WHERE ` + where + ` ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), filter.PerPage, (filter.Page-1)*filter.PerPage)
	stations := []Station{}
	if err := s.DB.Select(&stations, query, queryArgs...); err != nil {
		return nil, 0, StationStats{}, nil, nil, err
	}
	var stats StationStats
	if err := s.DB.Get(&stats, `SELECT COUNT(*) AS total,
		COUNT(DISTINCT site_id) AS sites,
		COUNT(DISTINCT COALESCE(NULLIF(municipio, ''), NULL)) AS municipios FROM stations`); err != nil {
		return nil, 0, stats, nil, nil, err
	}
	statuses := []string{}
	tecnologias := []string{}
	_ = s.DB.Select(&statuses, `SELECT DISTINCT status FROM stations WHERE status IS NOT NULL AND status <> '' ORDER BY status`)
	_ = s.DB.Select(&tecnologias, `SELECT DISTINCT tecnologia FROM stations WHERE tecnologia IS NOT NULL AND tecnologia <> '' ORDER BY tecnologia`)
	return stations, total, stats, statuses, tecnologias, nil
}

func (s *Store) FindStation(id int64) (Station, error) {
	var station Station
	err := s.DB.Get(&station, `SELECT * FROM stations WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return station, ErrNotFound
	}
	if err != nil {
		return station, err
	}
	station.Attachments = []StationAttachment{}
	station.Comments = []StationComment{}
	_ = s.DB.Select(&station.Attachments, `
		SELECT a.*, TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS uploader_name
		FROM station_attachments a LEFT JOIN users u ON u.id = a.uploaded_by
		WHERE a.station_id = ? ORDER BY a.created_at DESC, a.id DESC`, id)
	_ = s.DB.Select(&station.Comments, `
		SELECT c.*, TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS author_name
		FROM station_comments c LEFT JOIN users u ON u.id = c.user_id
		WHERE c.station_id = ? ORDER BY c.created_at DESC, c.id DESC`, id)
	return station, nil
}

type StationInput struct {
	SiteID, EnderecoID, TipoDeElemento, Tecnologia, TipoDeConexao, Classificacao string
	DetentorArea, TipoDeInfra, TipoDeEV, TipoDeLogradouro, Logradouro, Numero    string
	Complemento, Bairro, Municipio, Estado, CEP, Regional, Latitude, Longitude   string
	Status, TipoDaTorre, StationID                                               string
}

func (s *Store) CreateStation(input StationInput) (Station, error) {
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO stations (site_id, endereco_id, tipo_de_elemento, tecnologia, tipo_de_conexao,
			classificacao, detentor_area, tipo_de_infra, tipo_de_ev, tipo_de_logradouro, logradouro,
			numero, complemento, bairro, municipio, estado, cep, regional, latitude, longitude,
			status, tipo_da_torre, station_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.SiteID), nullString(input.EnderecoID), nullString(input.TipoDeElemento),
		nullString(input.Tecnologia), nullString(input.TipoDeConexao), nullString(input.Classificacao),
		nullString(input.DetentorArea), nullString(input.TipoDeInfra), nullString(input.TipoDeEV),
		nullString(input.TipoDeLogradouro), nullString(input.Logradouro), nullString(input.Numero),
		nullString(input.Complemento), nullString(input.Bairro), nullString(input.Municipio),
		nullString(input.Estado), nullString(input.CEP), nullString(input.Regional), nullString(input.Latitude),
		nullString(input.Longitude), nullString(input.Status), nullString(input.TipoDaTorre),
		nullString(input.StationID), timestamp, timestamp)
	if err != nil {
		return Station{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindStation(id)
}

func (s *Store) UpdateStation(id int64, input StationInput) (Station, error) {
	_, err := s.DB.Exec(`
		UPDATE stations SET site_id = ?, endereco_id = ?, tipo_de_elemento = ?, tecnologia = ?,
			tipo_de_conexao = ?, classificacao = ?, detentor_area = ?, tipo_de_infra = ?, tipo_de_ev = ?,
			tipo_de_logradouro = ?, logradouro = ?, numero = ?, complemento = ?, bairro = ?, municipio = ?,
			estado = ?, cep = ?, regional = ?, latitude = ?, longitude = ?, status = ?, tipo_da_torre = ?,
			station_id = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.SiteID), nullString(input.EnderecoID), nullString(input.TipoDeElemento),
		nullString(input.Tecnologia), nullString(input.TipoDeConexao), nullString(input.Classificacao),
		nullString(input.DetentorArea), nullString(input.TipoDeInfra), nullString(input.TipoDeEV),
		nullString(input.TipoDeLogradouro), nullString(input.Logradouro), nullString(input.Numero),
		nullString(input.Complemento), nullString(input.Bairro), nullString(input.Municipio),
		nullString(input.Estado), nullString(input.CEP), nullString(input.Regional), nullString(input.Latitude),
		nullString(input.Longitude), nullString(input.Status), nullString(input.TipoDaTorre),
		nullString(input.StationID), now(), id)
	if err != nil {
		return Station{}, err
	}
	return s.FindStation(id)
}

func (s *Store) DeleteStation(id int64) ([]string, error) {
	paths := []string{}
	_ = s.DB.Select(&paths, `SELECT path FROM station_attachments WHERE station_id = ?`, id)
	s.DeleteResourceLinks("station", id)
	_, err := s.DB.Exec(`DELETE FROM stations WHERE id = ?`, id)
	return paths, err
}

// SyncNokiaSiteToStation keeps the Stations inventory in sync with the Nokia
// project sites. It links by nokia_site_id, falling back to an existing station
// with the same site_id, and only updates the shared identity/location fields.
func (s *Store) SyncNokiaSiteToStation(nokiaSiteID int64) error {
	site, err := s.FindNokiaSite(nokiaSiteID)
	if err != nil {
		return err
	}
	var stationID int64
	err = s.DB.Get(&stationID, `SELECT id FROM stations WHERE nokia_site_id = ?`, nokiaSiteID)
	if errors.Is(err, sql.ErrNoRows) {
		err = s.DB.Get(&stationID, `SELECT id FROM stations WHERE site_id = ? ORDER BY id ASC LIMIT 1`, site.SiteID)
	}
	timestamp := now()
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.DB.Exec(`
			INSERT INTO stations (site_id, endereco_id, tipo_de_elemento, tecnologia, regional, status,
				station_id, nokia_site_id, created_at, updated_at)
			VALUES (?, ?, 'Site Nokia', 'Nokia', ?, 'Projeto Nokia', ?, ?, ?, ?)`,
			site.SiteID, nullString(deref(site.EndID)), nullString(deref(site.Regional)),
			nullString(site.SiteID), nokiaSiteID, timestamp, timestamp)
		return err
	}
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`
		UPDATE stations SET site_id = ?, endereco_id = ?, regional = ?, nokia_site_id = ?, updated_at = ?
		WHERE id = ?`,
		site.SiteID, nullString(deref(site.EndID)), nullString(deref(site.Regional)), nokiaSiteID, timestamp, stationID)
	return err
}

func (s *Store) SyncAllNokiaSitesToStations() error {
	sites, _, _, _, err := s.ListNokiaSites(NokiaSiteFilter{Page: 1, PerPage: 10000})
	if err != nil {
		return err
	}
	for _, site := range sites {
		if err := s.SyncNokiaSiteToStation(site.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateStationAttachment(stationID int64, userID int64, category, originalName, path, disk, mimeType string, size int64) (StationAttachment, error) {
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO station_attachments (station_id, uploaded_by, category, original_name, path, disk, mime_type, size, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, stationID, userID, category, originalName, path, disk, nullString(mimeType), size, timestamp, timestamp)
	if err != nil {
		return StationAttachment{}, err
	}
	id, _ := result.LastInsertId()
	var attachment StationAttachment
	err = s.DB.Get(&attachment, `SELECT * FROM station_attachments WHERE id = ?`, id)
	return attachment, err
}

func (s *Store) FindStationAttachment(stationID, attachmentID int64) (StationAttachment, error) {
	var attachment StationAttachment
	err := s.DB.Get(&attachment, `SELECT * FROM station_attachments WHERE id = ? AND station_id = ?`, attachmentID, stationID)
	if errors.Is(err, sql.ErrNoRows) {
		return attachment, ErrNotFound
	}
	return attachment, err
}

func (s *Store) DeleteStationAttachment(stationID, attachmentID int64) (string, error) {
	attachment, err := s.FindStationAttachment(stationID, attachmentID)
	if err != nil {
		return "", err
	}
	_, err = s.DB.Exec(`DELETE FROM station_attachments WHERE id = ?`, attachmentID)
	return attachment.Path, err
}

func (s *Store) CreateStationComment(stationID, userID int64, body string) (StationComment, error) {
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO station_comments (station_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, stationID, userID, strings.TrimSpace(body), timestamp, timestamp)
	if err != nil {
		return StationComment{}, err
	}
	id, _ := result.LastInsertId()
	var comment StationComment
	err = s.DB.Get(&comment, `
		SELECT c.*, TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS author_name
		FROM station_comments c LEFT JOIN users u ON u.id = c.user_id WHERE c.id = ?`, id)
	return comment, err
}

func (s *Store) FindStationComment(stationID, commentID int64) (StationComment, error) {
	var comment StationComment
	err := s.DB.Get(&comment, `SELECT * FROM station_comments WHERE id = ? AND station_id = ?`, commentID, stationID)
	if errors.Is(err, sql.ErrNoRows) {
		return comment, ErrNotFound
	}
	return comment, err
}

func (s *Store) DeleteStationComment(stationID, commentID int64) error {
	_, err := s.DB.Exec(`DELETE FROM station_comments WHERE id = ? AND station_id = ?`, commentID, stationID)
	return err
}

type StationSiteOption struct {
	SiteID     string  `db:"site_id" json:"site_id"`
	EnderecoID *string `db:"endereco_id" json:"endereco_id"`
	Regional   *string `db:"regional" json:"regional"`
	Municipio  *string `db:"municipio" json:"municipio"`
	Estado     *string `db:"estado" json:"estado"`
}

// StationSiteOptions returns distinct sites registered in the Stations module.
// When excludeLinked is true, sites already linked to a Nokia project are omitted.
func (s *Store) StationSiteOptions(search string, excludeLinked bool) ([]StationSiteOption, error) {
	conditions := []string{"site_id <> ''"}
	args := []any{}
	if search = strings.TrimSpace(search); search != "" {
		conditions = append(conditions, "site_id LIKE ?")
		args = append(args, "%"+search+"%")
	}
	if excludeLinked {
		conditions = append(conditions, "nokia_site_id IS NULL")
	}
	query := `SELECT site_id,
		MIN(NULLIF(endereco_id, '')) AS endereco_id,
		MIN(NULLIF(regional, '')) AS regional,
		MIN(NULLIF(municipio, '')) AS municipio,
		MIN(NULLIF(estado, '')) AS estado
		FROM stations WHERE ` + strings.Join(conditions, " AND ") + `
		GROUP BY site_id ORDER BY site_id COLLATE NOCASE ASC LIMIT 1000`
	options := []StationSiteOption{}
	err := s.DB.Select(&options, query, args...)
	return options, err
}

func (s *Store) StationSites(search string, limit int) ([]map[string]any, int, error) {
	args := []any{}
	where := ""
	if search = strings.TrimSpace(search); search != "" {
		where = "WHERE site_id LIKE ?"
		args = append(args, "%"+search+"%")
	}
	if limit <= 0 {
		limit = 8
	}
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(DISTINCT site_id) FROM stations `+where, args...); err != nil {
		return nil, 0, err
	}
	query := `SELECT site_id, COUNT(*) AS stations_count, MIN(municipio) AS municipio,
		MIN(estado) AS estado, MAX(updated_at) AS last_updated_at
		FROM stations ` + where + ` GROUP BY site_id ORDER BY last_updated_at DESC, site_id ASC LIMIT ?`
	rows, err := s.DB.Queryx(query, append(args, limit)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []map[string]any{}
	for rows.Next() {
		row := map[string]any{}
		if err := rows.MapScan(row); err != nil {
			return nil, 0, err
		}
		result = append(result, row)
	}
	return result, total, rows.Err()
}

func joinNonEmpty(values []string, separator string) string {
	result := []string{}
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, separator)
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ = fmt.Sprintf
var _ = sqlx.In
