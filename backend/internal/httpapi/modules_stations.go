package httpapi

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type stationPayload struct {
	SiteID           string `json:"site_id"`
	EnderecoID       string `json:"endereco_id"`
	TipoDeElemento   string `json:"tipo_de_elemento"`
	Tecnologia       string `json:"tecnologia"`
	TipoDeConexao    string `json:"tipo_de_conexao"`
	Classificacao    string `json:"classificacao"`
	DetentorArea     string `json:"detentor_area"`
	TipoDeInfra      string `json:"tipo_de_infra"`
	TipoDeEV         string `json:"tipo_de_ev"`
	TipoDeLogradouro string `json:"tipo_de_logradouro"`
	Logradouro       string `json:"logradouro"`
	Numero           string `json:"numero"`
	Complemento      string `json:"complemento"`
	Bairro           string `json:"bairro"`
	Municipio        string `json:"municipio"`
	Estado           string `json:"estado"`
	CEP              string `json:"cep"`
	Regional         string `json:"regional"`
	Latitude         string `json:"latitude"`
	Longitude        string `json:"longitude"`
	Status           string `json:"status"`
	TipoDaTorre      string `json:"tipo_da_torre"`
	StationID        string `json:"station_id"`
}

func (p stationPayload) input() modules.StationInput {
	return modules.StationInput{
		SiteID: p.SiteID, EnderecoID: p.EnderecoID, TipoDeElemento: p.TipoDeElemento, Tecnologia: p.Tecnologia,
		TipoDeConexao: p.TipoDeConexao, Classificacao: p.Classificacao, DetentorArea: p.DetentorArea,
		TipoDeInfra: p.TipoDeInfra, TipoDeEV: p.TipoDeEV, TipoDeLogradouro: p.TipoDeLogradouro,
		Logradouro: p.Logradouro, Numero: p.Numero, Complemento: p.Complemento, Bairro: p.Bairro,
		Municipio: p.Municipio, Estado: p.Estado, CEP: p.CEP, Regional: p.Regional,
		Latitude: p.Latitude, Longitude: p.Longitude, Status: p.Status, TipoDaTorre: p.TipoDaTorre,
		StationID: p.StationID,
	}
}

func (p *stationPayload) fromInput(input modules.StationInput) {
	p.SiteID = input.SiteID
	p.EnderecoID = input.EnderecoID
	p.TipoDeElemento = input.TipoDeElemento
	p.Tecnologia = input.Tecnologia
	p.TipoDeConexao = input.TipoDeConexao
	p.Classificacao = input.Classificacao
	p.DetentorArea = input.DetentorArea
	p.TipoDeInfra = input.TipoDeInfra
	p.TipoDeEV = input.TipoDeEV
	p.TipoDeLogradouro = input.TipoDeLogradouro
	p.Logradouro = input.Logradouro
	p.Numero = input.Numero
	p.Complemento = input.Complemento
	p.Bairro = input.Bairro
	p.Municipio = input.Municipio
	p.Estado = input.Estado
	p.CEP = input.CEP
	p.Regional = input.Regional
	p.Latitude = input.Latitude
	p.Longitude = input.Longitude
	p.Status = input.Status
	p.TipoDaTorre = input.TipoDaTorre
	p.StationID = input.StationID
}

func (s *Server) stationsIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	items, total, stats, statuses, tecnologias, err := s.Modules.ListStations(modules.StationFilter{
		Search: query.Get("search"), Status: query.Get("status"), Tecnologia: query.Get("tecnologia"),
		Page: page, PerPage: 15,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "NÃƒÂ£o foi possÃƒÂ­vel carregar as estaÃƒÂ§ÃƒÂµes.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items, "stats": stats, "statuses": statuses, "tecnologias": tecnologias,
		"meta": paginationMeta(page, 15, total),
	})
}

func (s *Server) stationsShow(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	station, err := s.Modules.FindStation(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "EstaÃƒÂ§ÃƒÂ£o nÃƒÂ£o encontrada.", nil)
		return
	}
	for i := range station.Attachments {
		station.Attachments[i].URL = "/uploads/" + station.Attachments[i].Path
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"station": station}})
}

func (s *Server) stationsCreate(w http.ResponseWriter, r *http.Request) {
	var payload stationPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	normalizeStationPayload(&payload)
	if fields := validateStation(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	station, err := s.Modules.CreateStation(payload.input())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "NÃƒÂ£o foi possÃƒÂ­vel criar a estaÃƒÂ§ÃƒÂ£o.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "EstaÃƒÂ§ÃƒÂ£o cadastrada com sucesso.", "data": map[string]any{"station": station}})
}

func (s *Server) stationsUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload stationPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	normalizeStationPayload(&payload)
	if fields := validateStation(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	station, err := s.Modules.UpdateStation(id, payload.input())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "NÃƒÂ£o foi possÃƒÂ­vel atualizar a estaÃƒÂ§ÃƒÂ£o.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "EstaÃƒÂ§ÃƒÂ£o atualizada com sucesso.", "data": map[string]any{"station": station}})
}

var coordinateNumberPattern = regexp.MustCompile(`-?\d+(?:[.,]\d+)?`)

func normalizeStationPayload(payload *stationPayload) {
	payload.CEP = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, payload.CEP)
	if value, ok := parseStationCoordinate(payload.Latitude); ok {
		payload.Latitude = strconv.FormatFloat(value, 'f', 6, 64)
	}
	if value, ok := parseStationCoordinate(payload.Longitude); ok {
		payload.Longitude = strconv.FormatFloat(value, 'f', 6, 64)
	}
}

func parseStationCoordinate(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	normalized := strings.NewReplacer(
		"Ã‚Âº", " ", "Ã‚Â°", " ", "Ã‹Å¡", " ", "Ã¢â‚¬Â²", " ", "Ã¢â‚¬â„¢", " ", "Ã¢â‚¬Â³", " ", "Ã¢â‚¬Â", " ",
	).Replace(strings.ToUpper(value))
	matches := coordinateNumberPattern.FindAllString(normalized, -1)
	if len(matches) == 0 || len(matches) > 3 {
		return 0, false
	}
	numbers := make([]float64, 0, len(matches))
	for _, match := range matches {
		number, err := strconv.ParseFloat(strings.ReplaceAll(match, ",", "."), 64)
		if err != nil {
			return 0, false
		}
		numbers = append(numbers, number)
	}
	degrees := numbers[0]
	if degrees < 0 {
		degrees = -degrees
	}
	minutes := 0.0
	seconds := 0.0
	if len(numbers) > 1 {
		minutes = numbers[1]
	}
	if len(numbers) > 2 {
		seconds = numbers[2]
	}
	if minutes >= 60 || seconds >= 60 {
		return 0, false
	}
	result := degrees + minutes/60 + seconds/3600
	if strings.HasPrefix(strings.TrimSpace(normalized), "-") ||
		strings.Contains(normalized, "S") || strings.Contains(normalized, "W") {
		result = -result
	}
	return result, true
}

func validateStation(payload stationPayload) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.SiteID) == "" {
		fields.add("site_id", "Informe o ID do site.")
	}
	if len([]rune(payload.Estado)) > 0 && len([]rune(payload.Estado)) != 2 {
		fields.add("estado", "O estado deve ter 2 caracteres.")
	}
	for field, value := range map[string]string{"latitude": payload.Latitude, "longitude": payload.Longitude} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		number, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
		limit := 90.0
		if field == "longitude" {
			limit = 180.0
		}
		if err != nil || number < -limit || number > limit {
			fields.add(field, "Informe uma coordenada vÃƒÂ¡lida.")
		}
	}
	return fields
}

func (s *Server) stationsDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	paths, err := s.Modules.DeleteStation(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "NÃƒÂ£o foi possÃƒÂ­vel excluir a estaÃƒÂ§ÃƒÂ£o.", nil)
		return
	}
	for _, path := range paths {
		s.Uploads.DeleteLocal(path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "EstaÃƒÂ§ÃƒÂ£o excluÃƒÂ­da com sucesso."})
}

func (s *Server) stationsUploadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if _, err := s.Modules.FindStation(id); err != nil {
		writeError(w, http.StatusNotFound, "EstaÃƒÂ§ÃƒÂ£o nÃƒÂ£o encontrada.", nil)
		return
	}
	if err := r.ParseMultipartForm(40 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo invÃƒÂ¡lido.", nil)
		return
	}
	category := r.FormValue("category")
	if category != "ppi_tssr" && category != "attachment" {
		writeValidation(w, validationErrors{"category": "Categoria invÃƒÂ¡lida."})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeValidation(w, validationErrors{"files": "Selecione ao menos um arquivo."})
		return
	}
	if len(files) > 10 {
		writeValidation(w, validationErrors{"files": "Envie no mÃƒÂ¡ximo 10 arquivos por vez."})
		return
	}
	user := currentUser(r)
	created := 0
	for _, header := range files {
		if header.Size > 20<<20 {
			writeValidation(w, validationErrors{"files": "Cada arquivo deve ter no mÃƒÂ¡ximo 20 MB."})
			return
		}
		file, err := header.Open()
		if err != nil {
			continue
		}
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "stations")
		file.Close()
		if err != nil {
			writeValidation(w, validationErrors{"files": "Formato de arquivo nÃƒÂ£o permitido."})
			return
		}
		if _, err := s.Modules.CreateStationAttachment(id, user.ID, category, header.Filename, result.Path, "local", header.Header.Get("Content-Type"), header.Size); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": fmt.Sprintf("%d anexo(s) enviado(s).", created)})
}

func (s *Server) stationsDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	attachment, err := s.Modules.FindStationAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo nÃƒÂ£o encontrado.", nil)
		return
	}
	http.Redirect(w, r, "/uploads/"+attachment.Path, http.StatusTemporaryRedirect)
}

func (s *Server) stationsDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	path, err := s.Modules.DeleteStationAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo nÃƒÂ£o encontrado.", nil)
		return
	}
	s.Uploads.DeleteLocal(path)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Anexo removido com sucesso."})
}

func (s *Server) stationsCreateComment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload struct {
		Body string `json:"body"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	body := strings.TrimSpace(payload.Body)
	if len([]rune(body)) < 2 {
		writeValidation(w, validationErrors{"body": "O comentÃƒÂ¡rio precisa ter pelo menos 2 caracteres."})
		return
	}
	comment, err := s.Modules.CreateStationComment(id, currentUser(r).ID, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "NÃƒÂ£o foi possÃƒÂ­vel publicar o comentÃƒÂ¡rio.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "ComentÃƒÂ¡rio publicado.", "data": map[string]any{"comment": comment}})
}

func (s *Server) stationsDeleteComment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	commentID, _ := parseID(r, "commentId")
	comment, err := s.Modules.FindStationComment(id, commentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "ComentÃƒÂ¡rio nÃƒÂ£o encontrado.", nil)
		return
	}
	auth := authFrom(r)
	if !auth.User.IsAdmin && (comment.UserID == nil || *comment.UserID != auth.User.ID) {
		writeError(w, http.StatusForbidden, "VocÃƒÂª nÃƒÂ£o pode remover este comentÃƒÂ¡rio.", nil)
		return
	}
	if err := s.Modules.DeleteStationComment(id, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "NÃƒÂ£o foi possÃƒÂ­vel remover o comentÃƒÂ¡rio.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ComentÃƒÂ¡rio removido."})
}
