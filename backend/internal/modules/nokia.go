package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Nokia module
 * ----------------------------------------------------------------------- */

type NokiaROM struct {
	ID        int64     `db:"id" json:"id"`
	FirstName string    `db:"first_name" json:"first_name"`
	LastName  string    `db:"last_name" json:"last_name"`
	Email     string    `db:"email" json:"email"`
	Phone     string    `db:"phone" json:"phone"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (r NokiaROM) FullName() string { return strings.TrimSpace(r.FirstName + " " + r.LastName) }

type NokiaTask struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Position  int       `db:"position" json:"position"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type NokiaSiteTask struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Position    int        `json:"position"`
	CompletedAt *time.Time `json:"completed_at"`
}

type NokiaSiteAttachment struct {
	ID           int64     `db:"id" json:"id"`
	NokiaSiteID  int64     `db:"nokia_site_id" json:"-"`
	Category     string    `db:"category" json:"category"`
	OriginalName string    `db:"original_name" json:"original_name"`
	Path         string    `db:"path" json:"path"`
	Size         int64     `db:"size" json:"size"`
	MimeType     *string   `db:"mime_type" json:"mime_type"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	URL          string    `db:"-" json:"url"`
}

type NokiaSiteComment struct {
	ID          int64     `db:"id" json:"id"`
	NokiaSiteID int64     `db:"nokia_site_id" json:"-"`
	UserID      *int64    `db:"user_id" json:"user_id"`
	Body        string    `db:"body" json:"body"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	AuthorName  *string   `db:"author_name" json:"author_name"`
}

type NokiaRejection struct {
	ID                  int64      `db:"id" json:"id"`
	NokiaSiteID         int64      `db:"nokia_site_id" json:"-"`
	ChecklistField      string     `db:"checklist_field" json:"checklist_field"`
	Reason              string     `db:"reason" json:"reason"`
	ReturnDays          int        `db:"return_days" json:"return_days"`
	RequiresFieldReturn bool       `db:"requires_field_return" json:"requires_field_return"`
	StartedAt           time.Time  `db:"started_at" json:"started_at"`
	ResolvedAt          *time.Time `db:"resolved_at" json:"resolved_at"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
}

type NokiaSite struct {
	ID                           int64                 `db:"id" json:"id"`
	SiteID                       string                `db:"site_id" json:"site_id"`
	EndID                        *string               `db:"end_id" json:"end_id"`
	OC                           *string               `db:"oc" json:"oc"`
	OSFamEntrega                 *string               `db:"os_fam_entrega" json:"os_fam_entrega"`
	OSFamInstalacao              *string               `db:"os_fam_instalacao" json:"os_fam_instalacao"`
	Regional                     *string               `db:"regional" json:"regional"`
	Subcon                       *string               `db:"subcon" json:"subcon"`
	InterruptionReason           *string               `db:"interruption_reason" json:"interruption_reason"`
	InterruptionReturnDays       *int                  `db:"interruption_return_days" json:"interruption_return_days"`
	InterruptionStartedAt        *time.Time            `db:"interruption_started_at" json:"interruption_started_at"`
	ChecklistFamEntrega          bool                  `db:"checklist_fam_entrega" json:"checklist_fam_entrega"`
	ChecklistFamEntregaStatus    string                `db:"checklist_fam_entrega_status" json:"checklist_fam_entrega_status"`
	ChecklistFamInstalacao       bool                  `db:"checklist_fam_instalacao" json:"checklist_fam_instalacao"`
	ChecklistFamInstalacaoStatus string                `db:"checklist_fam_instalacao_status" json:"checklist_fam_instalacao_status"`
	ChecklistFamPanoramica       bool                  `db:"checklist_fam_panoramica" json:"checklist_fam_panoramica"`
	ChecklistFamPanoramicaStatus string                `db:"checklist_fam_panoramica_status" json:"checklist_fam_panoramica_status"`
	MOSPlanAt                    *string               `db:"mos_plan_at" json:"mos_plan_at"`
	MOSRealAt                    *string               `db:"mos_real_at" json:"mos_real_at"`
	InstalacaoPlanAt             *string               `db:"instalacao_plan_at" json:"instalacao_plan_at"`
	InstalacaoRealAt             *string               `db:"instalacao_real_at" json:"instalacao_real_at"`
	IntegracaoPlanAt             *string               `db:"integracao_plan_at" json:"integracao_plan_at"`
	IntegracaoRealAt             *string               `db:"integracao_real_at" json:"integracao_real_at"`
	RFAPlanAt                    *string               `db:"rfa_plan_at" json:"rfa_plan_at"`
	RFARealAt                    *string               `db:"rfa_real_at" json:"rfa_real_at"`
	FamEntregaApprovedAt         *string               `db:"fam_entrega_approved_at" json:"fam_entrega_approved_at"`
	FamInstalacaoApprovedAt      *string               `db:"fam_instalacao_approved_at" json:"fam_instalacao_approved_at"`
	FamPanoramicaApprovedAt      *string               `db:"fam_panoramica_approved_at" json:"fam_panoramica_approved_at"`
	NokiaROMID                   *int64                `db:"nokia_rom_id" json:"nokia_rom_id"`
	CreatedAt                    time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt                    time.Time             `db:"updated_at" json:"updated_at"`
	ROMName                      *string               `db:"rom_name" json:"rom_name"`
	Tasks                        []NokiaSiteTask       `db:"-" json:"tasks"`
	Attachments                  []NokiaSiteAttachment `db:"-" json:"attachments"`
	Comments                     []NokiaSiteComment    `db:"-" json:"comments"`
	Rejections                   []NokiaRejection      `db:"-" json:"rejections"`
}

var NokiaChecklists = map[string]string{
	"checklist_fam_entrega":    "FAM Entrega",
	"checklist_fam_instalacao": "FAM Instalação",
	"checklist_fam_panoramica": "FAM Panorâmica",
}

var nokiaChecklistOrder = []string{"checklist_fam_entrega", "checklist_fam_instalacao", "checklist_fam_panoramica"}

var nokiaDateFields = []string{
	"mos_plan_at", "mos_real_at", "instalacao_plan_at", "instalacao_real_at",
	"integracao_plan_at", "integracao_real_at", "rfa_plan_at", "rfa_real_at",
	"fam_entrega_approved_at", "fam_instalacao_approved_at", "fam_panoramica_approved_at",
}

func (n NokiaSite) ChecklistProgress() (done int, total int) {
	for _, field := range nokiaChecklistOrder {
		total++
		if n.checklistStatus(field) == "sent" {
			done++
		}
	}
	for _, task := range n.Tasks {
		total++
		if task.CompletedAt != nil {
			done++
		}
	}
	return done, total
}

func (n NokiaSite) checklistStatus(field string) string {
	switch field {
	case "checklist_fam_entrega":
		return n.ChecklistFamEntregaStatus
	case "checklist_fam_instalacao":
		return n.ChecklistFamInstalacaoStatus
	case "checklist_fam_panoramica":
		return n.ChecklistFamPanoramicaStatus
	}
	return "pending"
}

type NokiaSiteFilter struct {
	Search  string
	Page    int
	PerPage int
}

type NokiaStats struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Pending   int `json:"pending"`
}

func (s *Store) ListNokiaSites(filter NokiaSiteFilter) ([]NokiaSite, int, NokiaStats, int, error) {
	where := "1 = 1"
	args := []any{}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		where = `(s.site_id LIKE ? OR COALESCE(s.end_id, '') LIKE ? OR COALESCE(s.oc, '') LIKE ? OR COALESCE(s.os_fam_entrega, '') LIKE ? OR COALESCE(s.os_fam_instalacao, '') LIKE ?)`
		args = append(args, like, like, like, like, like)
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 15
	}
	var total int
	if err := s.DB.Get(&total, `SELECT COUNT(*) FROM nokia_sites s WHERE `+where, args...); err != nil {
		return nil, 0, NokiaStats{}, 0, err
	}
	query := `SELECT s.*, TRIM(r.first_name || ' ' || r.last_name) AS rom_name
		FROM nokia_sites s LEFT JOIN nokia_roms r ON r.id = s.nokia_rom_id
		WHERE ` + where + ` ORDER BY s.updated_at DESC, s.id DESC LIMIT ? OFFSET ?`
	sites := []NokiaSite{}
	if err := s.DB.Select(&sites, query, append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)...); err != nil {
		return nil, 0, NokiaStats{}, 0, err
	}
	var tasksTotal int
	_ = s.DB.Get(&tasksTotal, `SELECT COUNT(*) FROM nokia_tasks`)
	_ = s.attachNokiaTasks(sites)
	var stats NokiaStats
	stats.Total = total
	for _, site := range sites {
		done, all := site.ChecklistProgress()
		if all > 0 && done == all {
			stats.Completed++
		}
	}
	stats.Pending = stats.Total - stats.Completed
	return sites, total, stats, tasksTotal, nil
}

func (s *Store) attachNokiaTasks(sites []NokiaSite) error {
	if len(sites) == 0 {
		return nil
	}
	for i := range sites {
		sites[i].Tasks = []NokiaSiteTask{}
		rows, err := s.DB.Queryx(`
			SELECT t.id, t.name, t.position, p.completed_at
			FROM nokia_tasks t
			LEFT JOIN nokia_site_task p ON p.nokia_task_id = t.id AND p.nokia_site_id = ?
			ORDER BY t.position ASC, t.name ASC`, sites[i].ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var task NokiaSiteTask
			if err := rows.Scan(&task.ID, &task.Name, &task.Position, &task.CompletedAt); err != nil {
				rows.Close()
				return err
			}
			sites[i].Tasks = append(sites[i].Tasks, task)
		}
		rows.Close()
	}
	return nil
}

func (s *Store) FindNokiaSite(id int64) (NokiaSite, error) {
	var site NokiaSite
	err := s.DB.Get(&site, `SELECT s.*, TRIM(r.first_name || ' ' || r.last_name) AS rom_name
		FROM nokia_sites s LEFT JOIN nokia_roms r ON r.id = s.nokia_rom_id WHERE s.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return site, ErrNotFound
	}
	if err != nil {
		return site, err
	}
	sites := []NokiaSite{site}
	_ = s.attachNokiaTasks(sites)
	site = sites[0]
	site.Attachments = []NokiaSiteAttachment{}
	site.Comments = []NokiaSiteComment{}
	site.Rejections = []NokiaRejection{}
	if err := s.DB.Select(&site.Attachments, `SELECT * FROM nokia_site_attachments WHERE nokia_site_id = ? ORDER BY created_at DESC`, id); err != nil {
		return site, err
	}
	if err := s.DB.Select(&site.Comments, `SELECT c.*, TRIM(u.name || ' ' || COALESCE(u.surname, '')) AS author_name
		FROM nokia_site_comments c LEFT JOIN users u ON u.id = c.user_id WHERE c.nokia_site_id = ? ORDER BY c.created_at DESC`, id); err != nil {
		return site, err
	}
	if err := s.DB.Select(&site.Rejections, `SELECT * FROM nokia_site_rejections WHERE nokia_site_id = ? ORDER BY started_at DESC, id DESC`, id); err != nil {
		return site, err
	}
	return site, nil
}

type NokiaSiteInput struct {
	SiteID, EndID, OC, OSFamEntrega, OSFamInstalacao, Regional, Subcon  string
	ChecklistFamEntrega, ChecklistFamInstalacao, ChecklistFamPanoramica bool
	NokiaROMID                                                          *int64
}

func (s *Store) CreateNokiaSite(input NokiaSiteInput) (NokiaSite, error) {
	timestamp := now()
	statusEntrega := "pending"
	statusInstalacao := "pending"
	statusPanoramica := "pending"
	if input.ChecklistFamEntrega {
		statusEntrega = "sent"
	}
	if input.ChecklistFamInstalacao {
		statusInstalacao = "sent"
	}
	if input.ChecklistFamPanoramica {
		statusPanoramica = "sent"
	}
	result, err := s.DB.Exec(`
		INSERT INTO nokia_sites (site_id, end_id, oc, os_fam_entrega, os_fam_instalacao, regional, subcon,
			checklist_fam_entrega, checklist_fam_entrega_status, checklist_fam_instalacao,
			checklist_fam_instalacao_status, checklist_fam_panoramica, checklist_fam_panoramica_status,
			nokia_rom_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		strings.TrimSpace(input.SiteID), nullString(input.EndID), nullString(input.OC),
		nullString(input.OSFamEntrega), nullString(input.OSFamInstalacao), nullString(input.Regional),
		nullString(input.Subcon), input.ChecklistFamEntrega, statusEntrega, input.ChecklistFamInstalacao,
		statusInstalacao, input.ChecklistFamPanoramica, statusPanoramica, input.NokiaROMID, timestamp, timestamp)
	if err != nil {
		return NokiaSite{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindNokiaSite(id)
}

func (s *Store) UpdateNokiaSite(id int64, input NokiaSiteInput) (NokiaSite, error) {
	statusEntrega := "pending"
	statusInstalacao := "pending"
	statusPanoramica := "pending"
	if input.ChecklistFamEntrega {
		statusEntrega = "sent"
	}
	if input.ChecklistFamInstalacao {
		statusInstalacao = "sent"
	}
	if input.ChecklistFamPanoramica {
		statusPanoramica = "sent"
	}
	_, err := s.DB.Exec(`
		UPDATE nokia_sites SET site_id = ?, end_id = ?, oc = ?, os_fam_entrega = ?, os_fam_instalacao = ?,
			regional = ?, subcon = ?, checklist_fam_entrega = ?, checklist_fam_entrega_status = ?,
			checklist_fam_instalacao = ?, checklist_fam_instalacao_status = ?, checklist_fam_panoramica = ?,
			checklist_fam_panoramica_status = ?, nokia_rom_id = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.SiteID), nullString(input.EndID), nullString(input.OC),
		nullString(input.OSFamEntrega), nullString(input.OSFamInstalacao), nullString(input.Regional),
		nullString(input.Subcon), input.ChecklistFamEntrega, statusEntrega, input.ChecklistFamInstalacao,
		statusInstalacao, input.ChecklistFamPanoramica, statusPanoramica, input.NokiaROMID, now(), id)
	if err != nil {
		return NokiaSite{}, err
	}
	return s.FindNokiaSite(id)
}

func (s *Store) UpdateNokiaSiteBase(id int64, input NokiaSiteInput) (NokiaSite, error) {
	_, err := s.DB.Exec(`
		UPDATE nokia_sites SET site_id = ?, end_id = ?, oc = ?, os_fam_entrega = ?, os_fam_instalacao = ?,
			regional = ?, subcon = ?, nokia_rom_id = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.SiteID), nullString(input.EndID), nullString(input.OC),
		nullString(input.OSFamEntrega), nullString(input.OSFamInstalacao), nullString(input.Regional),
		nullString(input.Subcon), input.NokiaROMID, now(), id)
	if err != nil {
		return NokiaSite{}, err
	}
	return s.FindNokiaSite(id)
}

func (s *Store) DeleteNokiaSite(id int64) ([]string, error) {
	paths := []string{}
	_ = s.DB.Select(&paths, `SELECT path FROM nokia_site_attachments WHERE nokia_site_id = ?`, id)
	s.DeleteResourceLinks("nokia_site", id)
	_, err := s.DB.Exec(`DELETE FROM nokia_sites WHERE id = ?`, id)
	return paths, err
}

func (s *Store) UpdateNokiaChecklist(id int64, field, status string) (NokiaSite, error) {
	if _, ok := NokiaChecklists[field]; !ok {
		return NokiaSite{}, ErrNotFound
	}
	boolean := status == "sent"
	_, err := s.DB.Exec(`UPDATE nokia_sites SET `+field+` = ?, `+field+`_status = ?, updated_at = ? WHERE id = ?`,
		boolean, status, now(), id)
	if err != nil {
		return NokiaSite{}, err
	}
	return s.FindNokiaSite(id)
}

func (s *Store) UpdateNokiaDates(id int64, values map[string]string) (NokiaSite, error) {
	sets := []string{}
	args := []any{}
	approvalFields := map[string]string{
		"fam_entrega_approved_at":    "checklist_fam_entrega",
		"fam_instalacao_approved_at": "checklist_fam_instalacao",
		"fam_panoramica_approved_at": "checklist_fam_panoramica",
	}
	for _, field := range nokiaDateFields {
		if value, ok := values[field]; ok {
			sets = append(sets, field+" = ?")
			args = append(args, nullDate(value))
			if checklist, isApproval := approvalFields[field]; isApproval && strings.TrimSpace(value) != "" {
				sets = append(sets, checklist+" = 1", checklist+"_status = 'sent'")
			}
		}
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at = ?")
		args = append(args, now(), id)
		_, err := s.DB.Exec(`UPDATE nokia_sites SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
		if err != nil {
			return NokiaSite{}, err
		}
	}
	return s.FindNokiaSite(id)
}

func (s *Store) FindNokiaSiteByOC(oc string) (NokiaSite, error) {
	var id int64
	err := s.DB.Get(&id, `SELECT id FROM nokia_sites WHERE oc = ?`, strings.TrimSpace(oc))
	if errors.Is(err, sql.ErrNoRows) {
		return NokiaSite{}, ErrNotFound
	}
	if err != nil {
		return NokiaSite{}, err
	}
	return s.FindNokiaSite(id)
}

func (s *Store) SetNokiaTaskState(siteID, taskID int64, completed bool) error {
	timestamp := now()
	var completedAt any
	if completed {
		completedAt = timestamp
	}
	result, err := s.DB.Exec(`UPDATE nokia_site_task SET completed_at = ?, updated_at = ?
		WHERE nokia_site_id = ? AND nokia_task_id = ?`, completedAt, timestamp, siteID, taskID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		return nil
	}
	_, err = s.DB.Exec(`INSERT INTO nokia_site_task (nokia_site_id, nokia_task_id, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, siteID, taskID, completedAt, timestamp, timestamp)
	return err
}

func (s *Store) SetNokiaInterruption(id int64, reason string, days int) error {
	_, err := s.DB.Exec(`UPDATE nokia_sites SET interruption_reason = ?, interruption_return_days = ?,
		interruption_started_at = ?, updated_at = ? WHERE id = ?`, strings.TrimSpace(reason), days, now(), now(), id)
	return err
}

func (s *Store) ClearNokiaInterruption(id int64) error {
	_, err := s.DB.Exec(`UPDATE nokia_sites SET interruption_reason = NULL, interruption_return_days = NULL,
		interruption_started_at = NULL, updated_at = ? WHERE id = ?`, now(), id)
	return err
}

func (s *Store) ToggleNokiaTask(siteID, taskID int64) error {
	var completedAt *time.Time
	err := s.DB.Get(&completedAt, `SELECT completed_at FROM nokia_site_task WHERE nokia_site_id = ? AND nokia_task_id = ?`, siteID, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.DB.Exec(`INSERT INTO nokia_site_task (nokia_site_id, nokia_task_id, completed_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, siteID, taskID, now(), now(), now())
		return err
	}
	if err != nil {
		return err
	}
	var value any
	if completedAt == nil {
		value = now()
	}
	_, err = s.DB.Exec(`UPDATE nokia_site_task SET completed_at = ?, updated_at = ? WHERE nokia_site_id = ? AND nokia_task_id = ?`,
		value, now(), siteID, taskID)
	return err
}

func (s *Store) CreateNokiaRejection(siteID int64, fields []string, reason string, days int, requiresFieldReturn bool) error {
	timestamp := now()
	for _, field := range fields {
		if _, ok := NokiaChecklists[field]; !ok {
			continue
		}
		if _, err := s.DB.Exec(`INSERT INTO nokia_site_rejections (nokia_site_id, checklist_field, reason,
			return_days, requires_field_return, started_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, siteID, field, strings.TrimSpace(reason), days, requiresFieldReturn, timestamp, timestamp, timestamp); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ResolveNokiaRejection(siteID, rejectionID int64) error {
	result, err := s.DB.Exec(`UPDATE nokia_site_rejections SET resolved_at = ?, updated_at = ?
		WHERE id = ? AND nokia_site_id = ?`, now(), now(), rejectionID, siteID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateNokiaAttachment(siteID int64, category, name, path, mime string, size int64) error {
	_, err := s.DB.Exec(`INSERT INTO nokia_site_attachments (nokia_site_id, category, original_name, path, size, mime_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, siteID, category, name, path, size, mime, now(), now())
	return err
}

func (s *Store) FindNokiaAttachment(siteID, attachmentID int64) (NokiaSiteAttachment, error) {
	var attachment NokiaSiteAttachment
	err := s.DB.Get(&attachment, `SELECT * FROM nokia_site_attachments WHERE id = ? AND nokia_site_id = ?`, attachmentID, siteID)
	if errors.Is(err, sql.ErrNoRows) {
		return attachment, ErrNotFound
	}
	return attachment, err
}

func (s *Store) DeleteNokiaAttachment(siteID, attachmentID int64) (string, error) {
	attachment, err := s.FindNokiaAttachment(siteID, attachmentID)
	if err != nil {
		return "", err
	}
	_, err = s.DB.Exec(`DELETE FROM nokia_site_attachments WHERE id = ?`, attachmentID)
	return attachment.Path, err
}

func (s *Store) CreateNokiaComment(siteID, userID int64, body string) error {
	_, err := s.DB.Exec(`INSERT INTO nokia_site_comments (nokia_site_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, siteID, userID, strings.TrimSpace(body), now(), now())
	return err
}

func (s *Store) FindNokiaComment(siteID, commentID int64) (NokiaSiteComment, error) {
	var comment NokiaSiteComment
	err := s.DB.Get(&comment, `SELECT * FROM nokia_site_comments WHERE id = ? AND nokia_site_id = ?`, commentID, siteID)
	if errors.Is(err, sql.ErrNoRows) {
		return comment, ErrNotFound
	}
	return comment, err
}

func (s *Store) DeleteNokiaComment(siteID, commentID int64) error {
	_, err := s.DB.Exec(`DELETE FROM nokia_site_comments WHERE id = ? AND nokia_site_id = ?`, commentID, siteID)
	return err
}

func (s *Store) NokiaROMs() ([]NokiaROM, error) {
	roms := []NokiaROM{}
	err := s.DB.Select(&roms, `SELECT * FROM nokia_roms ORDER BY first_name, last_name`)
	return roms, err
}

func (s *Store) CreateNokiaROM(first, last, email, phone string) (NokiaROM, error) {
	result, err := s.DB.Exec(`INSERT INTO nokia_roms (first_name, last_name, email, phone, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, strings.TrimSpace(first), strings.TrimSpace(last), strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(phone), now(), now())
	if err != nil {
		return NokiaROM{}, err
	}
	id, _ := result.LastInsertId()
	var rom NokiaROM
	err = s.DB.Get(&rom, `SELECT * FROM nokia_roms WHERE id = ?`, id)
	return rom, err
}

func (s *Store) UpdateNokiaROM(id int64, first, last, email, phone string) error {
	_, err := s.DB.Exec(`UPDATE nokia_roms SET first_name = ?, last_name = ?, email = ?, phone = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(first), strings.TrimSpace(last), strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(phone), now(), id)
	return err
}

func (s *Store) DeleteNokiaROM(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM nokia_roms WHERE id = ?`, id)
	return err
}

func (s *Store) NokiaTasks() ([]NokiaTask, error) {
	tasks := []NokiaTask{}
	err := s.DB.Select(&tasks, `SELECT * FROM nokia_tasks ORDER BY position ASC, name ASC`)
	return tasks, err
}

func (s *Store) CreateNokiaTask(name string) (NokiaTask, error) {
	var position int
	_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM nokia_tasks`)
	result, err := s.DB.Exec(`INSERT INTO nokia_tasks (name, position, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		strings.TrimSpace(name), position, now(), now())
	if err != nil {
		return NokiaTask{}, err
	}
	id, _ := result.LastInsertId()
	var task NokiaTask
	err = s.DB.Get(&task, `SELECT * FROM nokia_tasks WHERE id = ?`, id)
	return task, err
}

func (s *Store) UpdateNokiaTask(id int64, name string) error {
	_, err := s.DB.Exec(`UPDATE nokia_tasks SET name = ?, updated_at = ? WHERE id = ?`, strings.TrimSpace(name), now(), id)
	return err
}

func (s *Store) MoveNokiaTask(id int64, direction string) error {
	tasks := []NokiaTask{}
	if err := s.DB.Select(&tasks, `SELECT * FROM nokia_tasks ORDER BY position ASC, id ASC`); err != nil {
		return err
	}
	index := -1
	for i := range tasks {
		if tasks[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return ErrNotFound
	}
	target := index - 1
	if direction == "down" {
		target = index + 1
	}
	if target < 0 || target >= len(tasks) {
		return nil
	}
	tasks[index], tasks[target] = tasks[target], tasks[index]
	for i, task := range tasks {
		_, _ = s.DB.Exec(`UPDATE nokia_tasks SET position = ?, updated_at = ? WHERE id = ?`, i+1, now(), task.ID)
	}
	return nil
}

func (s *Store) DeleteNokiaTask(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM nokia_tasks WHERE id = ?`, id)
	return err
}

type NokiaMetrics struct {
	Total        int              `json:"total"`
	Integrated   int              `json:"integrated"`
	Accepted     int              `json:"accepted"`
	Rejected     int              `json:"rejected"`
	Analysis     int              `json:"analysis"`
	FirstSend    int              `json:"first_send"`
	NoOS         int              `json:"no_os"`
	Pending      int              `json:"pending"`
	Acceptance   float64          `json:"acceptance"`
	Projected    float64          `json:"projected"`
	Statuses     []map[string]any `json:"statuses"`
	Aging        []map[string]any `json:"aging"`
	Regional     []map[string]any `json:"regional"`
	Subcons      []map[string]any `json:"subcons"`
	ROMS         []map[string]any `json:"roms"`
	Sites        []map[string]any `json:"sites"`
	AllSites     []map[string]any `json:"all_sites"`
	PendingItems []map[string]any `json:"pending_items"`
	Insights     []map[string]any `json:"insights"`
}

var nokiaStatusDefinitions = []struct {
	Key   string
	Label string
	Color string
}{
	{"accepted", "Aceitos", "#10b981"},
	{"analysis", "Análise TIM", "#f59e0b"},
	{"first_send", "Primeiro Envio", "#f97316"},
	{"rejected", "Rejeitado", "#ef4444"},
	{"no_os", "Sem OS MOS", "#6b7280"},
}

func nokiaSiteStatus(site NokiaSite) string {
	switch {
	case site.FamInstalacaoApprovedAt != nil:
		return "accepted"
	case site.ChecklistFamInstalacaoStatus == "rejected":
		return "rejected"
	case site.ChecklistFamInstalacaoStatus == "analysis":
		return "analysis"
	case site.OSFamEntrega == nil || strings.TrimSpace(*site.OSFamEntrega) == "":
		return "no_os"
	default:
		return "first_send"
	}
}

func nokiaStatusLabel(key string) string {
	for _, definition := range nokiaStatusDefinitions {
		if definition.Key == key {
			return definition.Label
		}
	}
	return key
}

func nokiaStatusColor(key string) string {
	for _, definition := range nokiaStatusDefinitions {
		if definition.Key == key {
			return definition.Color
		}
	}
	return "#6b7280"
}

func nokiaAgingDays(site NokiaSite) int {
	base := site.CreatedAt
	if site.MOSRealAt != nil {
		if parsed, err := time.Parse("2006-01-02", *site.MOSRealAt); err == nil {
			base = parsed
		}
	}
	days := int(time.Since(base).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (s *Store) NokiaMetrics() (NokiaMetrics, error) {
	metrics := NokiaMetrics{
		Statuses: []map[string]any{}, Aging: []map[string]any{}, Regional: []map[string]any{},
		Subcons: []map[string]any{}, ROMS: []map[string]any{}, Sites: []map[string]any{},
		AllSites: []map[string]any{}, PendingItems: []map[string]any{}, Insights: []map[string]any{},
	}
	sites, _, _, _, err := s.ListNokiaSites(NokiaSiteFilter{Page: 1, PerPage: 1000})
	if err != nil {
		return metrics, err
	}
	metrics.Total = len(sites)
	statusCounts := map[string]int{}
	agingCounts := map[string]int{"0-10": 0, "11-20": 0, "21-60": 0, "61-90": 0, "90+": 0}
	regional := map[string]*struct{ total, accepted int }{}
	subcons := map[string]*struct {
		total, accepted int
		agingTotal      float64
		agingCount      int
	}{}
	roms := map[string]*struct{ total, accepted int }{}
	rejectedOld := 0
	noOS := 0
	withoutROM := 0

	for _, site := range sites {
		status := nokiaSiteStatus(site)
		statusCounts[status]++
		aging := nokiaAgingDays(site)
		integrated := site.IntegracaoRealAt != nil
		if integrated {
			metrics.Integrated++
			switch {
			case aging <= 10:
				agingCounts["0-10"]++
			case aging <= 20:
				agingCounts["11-20"]++
			case aging <= 60:
				agingCounts["21-60"]++
			case aging <= 90:
				agingCounts["61-90"]++
			default:
				agingCounts["90+"]++
			}
		}
		if status == "rejected" && aging > 90 {
			rejectedOld++
		}
		if status == "no_os" {
			noOS++
		}
		if site.ROMName == nil || strings.TrimSpace(*site.ROMName) == "" {
			withoutROM++
		}
		if integrated {
			switch status {
			case "accepted":
				metrics.Accepted++
			case "rejected":
				metrics.Rejected++
			case "analysis":
				metrics.Analysis++
			case "first_send":
				metrics.FirstSend++
			case "no_os":
				metrics.NoOS++
			}
			regionalKey := "Sem regional"
			if site.Regional != nil && strings.TrimSpace(*site.Regional) != "" {
				regionalKey = *site.Regional
			}
			if regional[regionalKey] == nil {
				regional[regionalKey] = &struct{ total, accepted int }{}
			}
			regional[regionalKey].total++
			if status == "accepted" {
				regional[regionalKey].accepted++
			}
			subconKey := "Sem subcon"
			if site.Subcon != nil && strings.TrimSpace(*site.Subcon) != "" {
				subconKey = *site.Subcon
			}
			if subcons[subconKey] == nil {
				subcons[subconKey] = &struct {
					total, accepted int
					agingTotal      float64
					agingCount      int
				}{}
			}
			subcons[subconKey].total++
			if status == "accepted" {
				subcons[subconKey].accepted++
				if site.MOSRealAt != nil && site.FamInstalacaoApprovedAt != nil {
					mos, err1 := time.Parse("2006-01-02", *site.MOSRealAt)
					approved, err2 := time.Parse("2006-01-02", *site.FamInstalacaoApprovedAt)
					if err1 == nil && err2 == nil {
						subcons[subconKey].agingTotal += approved.Sub(mos).Hours() / 24
						subcons[subconKey].agingCount++
					}
				}
			}
			romKey := "Sem ROM"
			if site.ROMName != nil && strings.TrimSpace(*site.ROMName) != "" {
				romKey = *site.ROMName
			}
			if roms[romKey] == nil {
				roms[romKey] = &struct{ total, accepted int }{}
			}
			roms[romKey].total++
			if status == "accepted" {
				roms[romKey].accepted++
			}
		}
		metrics.AllSites = append(metrics.AllSites, map[string]any{
			"site_id": site.SiteID, "oc": deref(site.OC), "regional": deref(site.Regional),
			"subcon": deref(site.Subcon), "rom": deref(site.ROMName), "status": status,
			"status_label": nokiaStatusLabel(status), "aging": aging, "integrated": integrated,
			"mos_real_at": deref(site.MOSRealAt), "instalacao_real_at": deref(site.InstalacaoRealAt),
			"integracao_real_at": deref(site.IntegracaoRealAt), "rfa_real_at": deref(site.RFARealAt),
		})
		for _, field := range nokiaChecklistOrder {
			status := site.checklistStatus(field)
			if status != "sent" {
				metrics.Pending++
				metrics.PendingItems = append(metrics.PendingItems, map[string]any{
					"site_id": site.SiteID, "oc": deref(site.OC), "regional": deref(site.Regional),
					"subcon": deref(site.Subcon), "rom": deref(site.ROMName), "kind": "Atividade",
					"kind_label": "Atividade", "description": NokiaChecklists[field],
					"detail": nokiaChecklistStatusLabel(status), "site_status": nokiaStatusLabel(nokiaSiteStatus(site)),
					"aging": aging,
				})
			}
		}
		for _, task := range site.Tasks {
			if task.CompletedAt == nil {
				metrics.Pending++
				metrics.PendingItems = append(metrics.PendingItems, map[string]any{
					"site_id": site.SiteID, "oc": deref(site.OC), "regional": deref(site.Regional),
					"subcon": deref(site.Subcon), "rom": deref(site.ROMName), "kind": "Tarefa",
					"kind_label": "Tarefa", "description": task.Name, "detail": "",
					"site_status": nokiaStatusLabel(nokiaSiteStatus(site)), "aging": aging,
				})
			}
		}
	}
	if metrics.Integrated > 0 {
		metrics.Acceptance = float64(metrics.Accepted) / float64(metrics.Integrated) * 100
		metrics.Projected = float64(metrics.Accepted+metrics.Analysis+metrics.FirstSend) / float64(metrics.Integrated) * 100
	}
	for _, definition := range nokiaStatusDefinitions {
		count := statusCounts[definition.Key]
		percent := 0.0
		if metrics.Integrated > 0 {
			percent = float64(count) / float64(metrics.Integrated) * 100
		}
		metrics.Statuses = append(metrics.Statuses, map[string]any{
			"key": definition.Key, "label": definition.Label, "color": definition.Color,
			"count": count, "percent": percent,
		})
	}
	for _, key := range []string{"0-10", "11-20", "21-60", "61-90", "90+"} {
		metrics.Aging = append(metrics.Aging, map[string]any{"label": key, "count": agingCounts[key]})
	}
	for key, value := range regional {
		percent := 0.0
		if value.total > 0 {
			percent = float64(value.accepted) / float64(value.total) * 100
		}
		metrics.Regional = append(metrics.Regional, map[string]any{
			"name": key, "total": value.total, "accepted": value.accepted, "percent": percent,
		})
	}
	for key, value := range subcons {
		percent := 0.0
		var aging any
		if value.total > 0 {
			percent = float64(value.accepted) / float64(value.total) * 100
		}
		if value.agingCount > 0 {
			aging = value.agingTotal / float64(value.agingCount)
		}
		metrics.Subcons = append(metrics.Subcons, map[string]any{
			"name": key, "total": value.total, "accepted": value.accepted,
			"gap": value.total - value.accepted, "aging": aging, "percent": percent,
		})
	}
	for key, value := range roms {
		percent := 0.0
		if value.total > 0 {
			percent = float64(value.accepted) / float64(value.total) * 100
		}
		metrics.ROMS = append(metrics.ROMS, map[string]any{
			"name": key, "total": value.total, "accepted": value.accepted, "percent": percent,
		})
	}
	metrics.Insights = append(metrics.Insights, map[string]any{
		"kind": "info", "text": "Aceitação atual de " + formatPercent(metrics.Acceptance) + " (meta: 80%).",
	})
	if rejectedOld > 0 {
		metrics.Insights = append(metrics.Insights, map[string]any{
			"kind": "danger", "text": fmt.Sprintf("%d site(s) rejeitado(s) há mais de 90 dias.", rejectedOld),
		})
	}
	if noOS > 0 {
		metrics.Insights = append(metrics.Insights, map[string]any{
			"kind": "warning", "text": fmt.Sprintf("%d site(s) sem OS de FAM de Entrega.", noOS),
		})
	}
	if withoutROM > 0 {
		metrics.Insights = append(metrics.Insights, map[string]any{
			"kind": "info", "text": fmt.Sprintf("%d site(s) sem ROM responsável.", withoutROM),
		})
	}
	sort.Slice(metrics.Regional, func(i, j int) bool {
		return metrics.Regional[i]["name"].(string) < metrics.Regional[j]["name"].(string)
	})
	sort.Slice(metrics.ROMS, func(i, j int) bool {
		return metrics.ROMS[i]["percent"].(float64) > metrics.ROMS[j]["percent"].(float64)
	})
	return metrics, nil
}

func formatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}

func nokiaChecklistStatusLabel(status string) string {
	labels := map[string]string{"pending": "Pendente", "analysis": "Análise TIM", "rejected": "Rejeitado", "sent": "Aprovado"}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}
