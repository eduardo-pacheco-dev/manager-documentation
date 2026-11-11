package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar as estações.", nil)
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
		writeError(w, http.StatusNotFound, "Estação não encontrada.", nil)
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
	if fields := validateStation(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	station, err := s.Modules.CreateStation(payload.input())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível criar a estação.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Estação cadastrada com sucesso.", "data": map[string]any{"station": station}})
}

func (s *Server) stationsUpdate(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	var payload stationPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateStation(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	station, err := s.Modules.UpdateStation(id, payload.input())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar a estação.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Estação atualizada com sucesso.", "data": map[string]any{"station": station}})
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
			fields.add(field, "Informe uma coordenada válida.")
		}
	}
	return fields
}

func (s *Server) stationsDelete(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	paths, err := s.Modules.DeleteStation(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir a estação.", nil)
		return
	}
	for _, path := range paths {
		s.Uploads.DeleteLocal(path)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Estação excluída com sucesso."})
}

func (s *Server) stationsUploadAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	if _, err := s.Modules.FindStation(id); err != nil {
		writeError(w, http.StatusNotFound, "Estação não encontrada.", nil)
		return
	}
	if err := r.ParseMultipartForm(40 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	category := r.FormValue("category")
	if category != "ppi_tssr" && category != "attachment" {
		writeValidation(w, validationErrors{"category": "Categoria inválida."})
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
		result, err := s.Uploads.SaveDocument(r.Context(), file, header, "stations")
		file.Close()
		if err != nil {
			writeValidation(w, validationErrors{"files": "Formato de arquivo não permitido."})
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
		writeError(w, http.StatusNotFound, "Anexo não encontrado.", nil)
		return
	}
	http.Redirect(w, r, "/uploads/"+attachment.Path, http.StatusTemporaryRedirect)
}

func (s *Server) stationsDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	attachmentID, _ := parseID(r, "attachmentId")
	path, err := s.Modules.DeleteStationAttachment(id, attachmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Anexo não encontrado.", nil)
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
		writeValidation(w, validationErrors{"body": "O comentário precisa ter pelo menos 2 caracteres."})
		return
	}
	comment, err := s.Modules.CreateStationComment(id, currentUser(r).ID, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível publicar o comentário.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "Comentário publicado.", "data": map[string]any{"comment": comment}})
}

func (s *Server) stationsDeleteComment(w http.ResponseWriter, r *http.Request) {
	id, _ := parseID(r, "id")
	commentID, _ := parseID(r, "commentId")
	comment, err := s.Modules.FindStationComment(id, commentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Comentário não encontrado.", nil)
		return
	}
	auth := authFrom(r)
	if !auth.User.IsAdmin && (comment.UserID == nil || *comment.UserID != auth.User.ID) {
		writeError(w, http.StatusForbidden, "Você não pode remover este comentário.", nil)
		return
	}
	if err := s.Modules.DeleteStationComment(id, commentID); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível remover o comentário.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Comentário removido."})
}

func (s *Server) stationsExport(w http.ResponseWriter, r *http.Request) {
	items, _, _, _, _, err := s.Modules.ListStations(modules.StationFilter{Page: 1, PerPage: 10000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar as estações.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="estacoes-%s.csv"`, time.Now().Format("2006-01-02")))
	writer := csv.NewWriter(w)
	writer.Comma = ';'
	_ = writer.Write([]string{"ID do site", "ID do endereço", "Tipo de elemento", "Tecnologia", "Tipo de conexão", "Classificação", "Detentor da área", "Tipo de infra", "Tipo de EV", "Logradouro", "Número", "Complemento", "Bairro", "Município", "UF", "CEP", "Regional", "Latitude", "Longitude", "Status", "Tipo da torre", "ID da estação"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.SiteID, derefString(item.EnderecoID), derefString(item.TipoDeElemento), derefString(item.Tecnologia),
			derefString(item.TipoDeConexao), derefString(item.Classificacao), derefString(item.DetentorArea),
			derefString(item.TipoDeInfra), derefString(item.TipoDeEV), derefString(item.Logradouro), derefString(item.Numero),
			derefString(item.Complemento), derefString(item.Bairro), derefString(item.Municipio), derefString(item.Estado),
			derefString(item.CEP), derefString(item.Regional), derefString(item.Latitude), derefString(item.Longitude),
			derefString(item.Status), derefString(item.TipoDaTorre), derefString(item.StationID),
		})
	}
	writer.Flush()
}

func (s *Server) stationsTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="modelo-importacao-estacoes.csv"`)
	_, _ = w.Write([]byte("ID do site *;ID do endereço;Tipo de elemento;Tecnologia;Latitude;Longitude;Município;UF\n"))
}

func (s *Server) stationsImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione um arquivo CSV."})
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		writeValidation(w, validationErrors{"file": "O arquivo está vazio ou inválido."})
		return
	}
	headers := map[string]int{}
	for index, value := range rows[0] {
		headers[normalizeHeader(value)] = index
	}
	if _, ok := headers["id_do_site"]; !ok {
		writeValidation(w, validationErrors{"file": "O arquivo precisa conter a coluna ID do site."})
		return
	}
	created := 0
	for _, row := range rows[1:] {
		get := func(names ...string) string {
			for _, name := range names {
				if index, ok := headers[normalizeHeader(name)]; ok && index < len(row) {
					return strings.TrimSpace(row[index])
				}
			}
			return ""
		}
		siteID := get("ID do site", "site_id")
		if siteID == "" {
			continue
		}
		payload := stationPayload{
			SiteID: siteID, EnderecoID: get("ID do endereço"), TipoDeElemento: get("Tipo de elemento"),
			Tecnologia: get("Tecnologia"), TipoDeConexao: get("Tipo de conexão"), Classificacao: get("Classificação"),
			DetentorArea: get("Detentor da área"), TipoDeInfra: get("Tipo de infra"), TipoDeEV: get("Tipo de EV"),
			Logradouro: get("Logradouro"), Numero: get("Número"), Complemento: get("Complemento"),
			Bairro: get("Bairro"), Municipio: get("Município"), Estado: get("UF"), CEP: get("CEP"),
			Regional: get("Regional"), Latitude: get("Latitude"), Longitude: get("Longitude"),
			Status: get("Status"), TipoDaTorre: get("Tipo da torre"), StationID: get("ID da estação"),
		}
		if _, err := s.Modules.CreateStation(payload.input()); err == nil {
			created++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("%d estação(ões) importada(s).", created)})
}
