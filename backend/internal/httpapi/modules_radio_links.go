package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

var radioLinkStatuses = map[string]bool{
	"planejado":  true,
	"ativo":      true,
	"inativo":    true,
	"manutencao": true,
}

type radioLinkPayload struct {
	Name         string   `json:"name"`
	Code         string   `json:"code"`
	StationAID   int64    `json:"station_a_id"`
	StationBID   int64    `json:"station_b_id"`
	Status       string   `json:"status"`
	Technology   string   `json:"technology"`
	FrequencyMHz *float64 `json:"frequency_mhz"`
	CapacityMbps *float64 `json:"capacity_mbps"`
	DistanceKm   *float64 `json:"distance_km"`
	Notes        string   `json:"notes"`
}

func (p radioLinkPayload) input() modules.RadioLinkInput {
	return modules.RadioLinkInput{
		Name:         p.Name,
		Code:         p.Code,
		StationAID:   p.StationAID,
		StationBID:   p.StationBID,
		Status:       p.Status,
		Technology:   p.Technology,
		FrequencyMHz: p.FrequencyMHz,
		CapacityMbps: p.CapacityMbps,
		DistanceKm:   p.DistanceKm,
		Notes:        p.Notes,
	}
}

func (s *Server) radioLinksIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	items, total, stats, statuses, technologies, err := s.Modules.ListRadioLinks(modules.RadioLinkFilter{
		Search:     query.Get("search"),
		Status:     query.Get("status"),
		Technology: query.Get("technology"),
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os rádio links.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":         items,
		"stats":        stats,
		"statuses":     statuses,
		"technologies": technologies,
		"meta":         paginationMeta(page, perPage, total),
	})
}

func (s *Server) radioLinksShow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	link, err := s.Modules.FindRadioLink(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"radio_link": link}})
}

func (s *Server) radioLinksCreate(w http.ResponseWriter, r *http.Request) {
	var payload radioLinkPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	normalizeRadioLinkPayload(&payload)
	fields := s.validateRadioLink(payload)
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	link, err := s.Modules.CreateRadioLink(payload.input())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o rádio link.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Rádio link criado com sucesso.",
		"data":    map[string]any{"radio_link": link},
	})
}

func (s *Server) radioLinksUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	var payload radioLinkPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	normalizeRadioLinkPayload(&payload)
	fields := s.validateRadioLink(payload)
	if !fields.empty() {
		writeValidation(w, fields)
		return
	}
	link, err := s.Modules.UpdateRadioLink(id, payload.input())
	if err != nil {
		if errors.Is(err, modules.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o rádio link.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Rádio link atualizado com sucesso.",
		"data":    map[string]any{"radio_link": link},
	})
}

func (s *Server) radioLinksDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	paths, err := s.Modules.DeleteRadioLink(id)
	if err != nil {
		if errors.Is(err, modules.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o rádio link.", nil)
		return
	}
	for _, path := range paths {
		s.Uploads.DeleteLocal(path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Rádio link removido com sucesso."})
}

func (s *Server) radioLinksUploadAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	if _, err := s.Modules.FindRadioLink(id); err != nil {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	if err := r.ParseMultipartForm(40 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeValidation(w, validationErrors{"files": "Selecione ao menos um arquivo."})
		return
	}
	if len(files) > 10 {
		writeValidation(w, validationErrors{"files": "Envie no máximo 10 arquivos por vez."})
		return
	}
	user := currentUser(r)
	created := 0
	for _, header := range files {
		if header.Size > 20<<20 {
			writeValidation(w, validationErrors{"files": "Cada arquivo deve ter no máximo 20 MB."})
			return
		}
		file, err := header.Open()
		if err != nil {
			continue
		}
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "radio-links")
		file.Close()
		if err != nil {
			writeValidation(w, validationErrors{"files": "Formato de arquivo não permitido."})
			return
		}
		if _, err := s.Modules.CreateRadioLinkAttachment(id, user.ID, header.Filename, result.Path, "local", header.Header.Get("Content-Type"), header.Size); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": fmt.Sprintf("%d anexo(s) enviado(s).", created)})
}

func (s *Server) radioLinksDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	attachment, err := s.Modules.FindRadioLinkAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo não encontrado.", nil)
		return
	}
	http.Redirect(w, r, "/uploads/"+attachment.Path, http.StatusTemporaryRedirect)
}

func (s *Server) radioLinksDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	path, err := s.Modules.DeleteRadioLinkAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo não encontrado.", nil)
		return
	}
	s.Uploads.DeleteLocal(path)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Anexo removido com sucesso."})
}

func (s *Server) radioLinksCreateComment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Rádio link não encontrado.", nil)
		return
	}
	var payload struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	body := strings.TrimSpace(payload.Body)
	if len([]rune(body)) < 2 {
		writeValidation(w, validationErrors{"body": "O comentário precisa ter pelo menos 2 caracteres."})
		return
	}
	comment, err := s.Modules.CreateRadioLinkComment(id, currentUser(r).ID, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível publicar o comentário.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Comentário publicado.", "data": map[string]any{"comment": comment}})
}

func (s *Server) radioLinksDeleteComment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	commentID, _ := parseID(r, "commentId")
	comment, err := s.Modules.FindRadioLinkComment(id, commentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Comentário não encontrado.", nil)
		return
	}
	auth := authFrom(r)
	if !auth.User.IsAdmin && (comment.UserID == nil || *comment.UserID != auth.User.ID) {
		writeError(w, http.StatusForbidden, "Você não pode remover este comentário.", nil)
		return
	}
	if err := s.Modules.DeleteRadioLinkComment(id, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível remover o comentário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Comentário removido com sucesso."})
}

// radioLinkStations feeds the station pickers used for the A and B endpoints.
func (s *Server) radioLinkStations(w http.ResponseWriter, r *http.Request) {
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 30)
	items, err := s.Modules.SearchDemandStations(r.URL.Query().Get("search"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível buscar as estações.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

// radioLinkOptions feeds the radio link pickers used by other modules.
func (s *Server) radioLinkOptions(w http.ResponseWriter, r *http.Request) {
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 30)
	items, err := s.Modules.SearchRadioLinkOptions(r.URL.Query().Get("search"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível buscar os rádio links.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func normalizeRadioLinkPayload(payload *radioLinkPayload) {
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Code = strings.TrimSpace(payload.Code)
	payload.Technology = strings.TrimSpace(payload.Technology)
	payload.Notes = strings.TrimSpace(payload.Notes)
	payload.Status = strings.ToLower(strings.TrimSpace(payload.Status))
	if payload.Status == "" {
		payload.Status = "planejado"
	}
}

func (s *Server) validateRadioLink(payload radioLinkPayload) validationErrors {
	fields := validationErrors{}
	if payload.Name == "" {
		fields.add("name", "Informe um nome para o rádio link.")
	}
	if payload.StationAID <= 0 {
		fields.add("station_a_id", "Selecione a estação da ponta A.")
	}
	if payload.StationBID <= 0 {
		fields.add("station_b_id", "Selecione a estação da ponta B.")
	}
	if payload.StationAID > 0 && payload.StationAID == payload.StationBID {
		fields.add("station_b_id", "As pontas A e B precisam ser estações diferentes.")
	}
	if payload.StationAID > 0 {
		if _, err := s.Modules.FindStation(payload.StationAID); err != nil {
			fields.add("station_a_id", "Estação da ponta A não encontrada.")
		}
	}
	if payload.StationBID > 0 {
		if _, err := s.Modules.FindStation(payload.StationBID); err != nil {
			fields.add("station_b_id", "Estação da ponta B não encontrada.")
		}
	}
	if !radioLinkStatuses[payload.Status] {
		fields.add("status", "Status inválido.")
	}
	for field, value := range map[string]*float64{
		"frequency_mhz": payload.FrequencyMHz,
		"capacity_mbps": payload.CapacityMbps,
		"distance_km":   payload.DistanceKm,
	} {
		if value != nil && *value < 0 {
			fields.add(field, "Informe um valor maior ou igual a zero.")
		}
	}
	return fields
}
