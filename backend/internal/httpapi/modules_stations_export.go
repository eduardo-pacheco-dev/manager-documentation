package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type stationColumn struct {
	Key   string
	Label string
	Width float64
}

var stationColumns = []stationColumn{
	{"site_id", "ID do site", 18},
	{"endereco_id", "ID do endereço", 18},
	{"tipo_de_elemento", "Tipo de elemento", 18},
	{"tecnologia", "Tecnologia", 14},
	{"tipo_de_conexao", "Tipo de conexão", 18},
	{"classificacao", "Classificação", 16},
	{"detentor_area", "Detentor da área", 22},
	{"tipo_de_infra", "Tipo de infra", 18},
	{"tipo_de_ev", "Tipo de EV", 14},
	{"tipo_de_logradouro", "Tipo de logradouro", 18},
	{"logradouro", "Logradouro", 30},
	{"numero", "Número", 11},
	{"complemento", "Complemento", 22},
	{"bairro", "Bairro", 20},
	{"municipio", "Município", 20},
	{"estado", "UF", 8},
	{"cep", "CEP", 13},
	{"regional", "Regional", 14},
	{"latitude", "Latitude", 15},
	{"longitude", "Longitude", 15},
	{"status", "Status", 16},
	{"tipo_da_torre", "Tipo da torre", 18},
	{"station_id", "ID da estação", 18},
}

func stationFieldValue(station modules.Station, key string) string {
	switch key {
	case "site_id":
		return station.SiteID
	case "endereco_id":
		return derefString(station.EnderecoID)
	case "tipo_de_elemento":
		return derefString(station.TipoDeElemento)
	case "tecnologia":
		return derefString(station.Tecnologia)
	case "tipo_de_conexao":
		return derefString(station.TipoDeConexao)
	case "classificacao":
		return derefString(station.Classificacao)
	case "detentor_area":
		return derefString(station.DetentorArea)
	case "tipo_de_infra":
		return derefString(station.TipoDeInfra)
	case "tipo_de_ev":
		return derefString(station.TipoDeEV)
	case "tipo_de_logradouro":
		return derefString(station.TipoDeLogradouro)
	case "logradouro":
		return derefString(station.Logradouro)
	case "numero":
		return derefString(station.Numero)
	case "complemento":
		return derefString(station.Complemento)
	case "bairro":
		return derefString(station.Bairro)
	case "municipio":
		return derefString(station.Municipio)
	case "estado":
		return derefString(station.Estado)
	case "cep":
		return derefString(station.CEP)
	case "regional":
		return derefString(station.Regional)
	case "latitude":
		return derefString(station.Latitude)
	case "longitude":
		return derefString(station.Longitude)
	case "status":
		return derefString(station.Status)
	case "tipo_da_torre":
		return derefString(station.TipoDaTorre)
	case "station_id":
		return derefString(station.StationID)
	default:
		return ""
	}
}

func (s *Server) stationsExport(w http.ResponseWriter, r *http.Request) {
	items, _, _, _, _, err := s.Modules.ListStations(modules.StationFilter{Page: 1, PerPage: 10000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar as estações.", nil)
		return
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "Estações"
	f.SetSheetName("Sheet1", sheet)
	headers := make([]string, 0, len(stationColumns)+2)
	widths := make([]float64, 0, len(stationColumns)+2)
	for _, column := range stationColumns {
		headers = append(headers, column.Label)
		widths = append(widths, column.Width)
	}
	headers = append(headers, "Criado em", "Atualizado em")
	widths = append(widths, 20, 20)
	writeXLSXRow(f, sheet, 1, headers)
	for index, station := range items {
		row := make([]string, 0, len(headers))
		for _, column := range stationColumns {
			row = append(row, stationFieldValue(station, column.Key))
		}
		row = append(row, station.CreatedAt.Format("02/01/2006 15:04"), station.UpdatedAt.Format("02/01/2006 15:04"))
		writeXLSXRow(f, sheet, index+2, row)
	}
	styleNokiaSheet(f, sheet, len(headers), len(items)+1)
	for index, width := range widths {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(sheet, col, col, width)
	}
	writeStationSitesSheet(f, items)
	writeWorkbook(w, f, "estacoes-"+time.Now().Format("2006-01-02")+".xlsx")
}

func writeStationSitesSheet(f *excelize.File, stations []modules.Station) {
	sheet := "Sites"
	bucket := map[string][]modules.Station{}
	for _, station := range stations {
		bucket[station.SiteID] = append(bucket[station.SiteID], station)
	}
	headers := []string{"ID do site", "Município", "UF", "Estações", "Atualizado em"}
	if _, err := f.NewSheet(sheet); err != nil {
		return
	}
	writeXLSXRow(f, sheet, 1, headers)
	row := 2
	for siteID, items := range bucket {
		latest := items[0]
		for _, item := range items[1:] {
			if item.UpdatedAt.After(latest.UpdatedAt) {
				latest = item
			}
		}
		writeXLSXRow(f, sheet, row, []string{
			siteID, derefString(latest.Municipio), derefString(latest.Estado), fmt.Sprintf("%d", len(items)),
			latest.UpdatedAt.Format("02/01/2006 15:04"),
		})
		row++
	}
	styleNokiaSheet(f, sheet, len(headers), row-1)
	for index, width := range []float64{20, 26, 10, 12, 20} {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(sheet, col, col, width)
	}
}

func (s *Server) stationExportOne(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Estação não encontrada.", nil)
		return
	}
	station, err := s.Modules.FindStation(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Estação não encontrada.", nil)
		return
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "Estação"
	f.SetSheetName("Sheet1", sheet)
	_ = f.MergeCell(sheet, "A1", "B1")
	_ = f.SetCellValue(sheet, "A1", "Estação "+station.SiteID)
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 16},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4F46E5"}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	_ = f.SetCellStyle(sheet, "A1", "B1", titleStyle)
	_ = f.SetRowHeight(sheet, 1, 34)
	writeXLSXRow(f, sheet, 2, []string{"Campo", "Valor"})
	styleNokiaSheet(f, sheet, 2, 2)
	row := 4
	groups := []struct {
		Title  string
		Fields []string
	}{
		{"Identificação", []string{"site_id", "endereco_id", "tipo_de_elemento", "tecnologia", "tipo_de_conexao", "classificacao", "status"}},
		{"Infraestrutura", []string{"station_id", "detentor_area", "tipo_de_infra", "tipo_de_ev", "tipo_da_torre", "regional"}},
		{"Endereço", []string{"cep", "tipo_de_logradouro", "logradouro", "numero", "complemento", "bairro", "municipio", "estado"}},
		{"Localização", []string{"latitude", "longitude"}},
	}
	labels := map[string]string{}
	for _, column := range stationColumns {
		labels[column.Key] = column.Label
	}
	for _, group := range groups {
		_ = f.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row))
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), group.Title)
		row++
		for _, field := range group.Fields {
			value := stationFieldValue(station, field)
			if value == "" {
				value = "—"
			}
			writeXLSXRow(f, sheet, row, []string{labels[field], value})
			row++
		}
	}
	writeXLSXRow(f, sheet, row, []string{"Registro", ""})
	row++
	writeXLSXRow(f, sheet, row, []string{"Criado em", station.CreatedAt.Format("02/01/2006 15:04")})
	row++
	writeXLSXRow(f, sheet, row, []string{"Atualizado em", station.UpdatedAt.Format("02/01/2006 15:04")})
	_ = f.SetColWidth(sheet, "A", "A", 30)
	_ = f.SetColWidth(sheet, "B", "B", 80)
	_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 2, TopLeftCell: "A3", ActivePane: "bottomLeft"})

	attachmentsSheet := "Anexos"
	_, _ = f.NewSheet(attachmentsSheet)
	writeXLSXRow(f, attachmentsSheet, 1, []string{"Categoria", "Arquivo", "Tamanho", "Enviado por", "Data"})
	for index, attachment := range station.Attachments {
		category := "Anexo"
		if attachment.Category == "ppi_tssr" {
			category = "PPI/TSSR"
		}
		writeXLSXRow(f, attachmentsSheet, index+2, []string{
			category, attachment.OriginalName, humanFileSize(attachment.Size), derefString(attachment.UploaderName),
			attachment.CreatedAt.Format("02/01/2006 15:04"),
		})
	}
	styleNokiaSheet(f, attachmentsSheet, 5, len(station.Attachments)+1)
	for index, width := range []float64{18, 45, 14, 30, 20} {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(attachmentsSheet, col, col, width)
	}

	commentsSheet := "Comentários"
	_, _ = f.NewSheet(commentsSheet)
	writeXLSXRow(f, commentsSheet, 1, []string{"Autor", "Data", "Comentário"})
	for index, comment := range station.Comments {
		writeXLSXRow(f, commentsSheet, index+2, []string{
			derefString(comment.AuthorName), comment.CreatedAt.Format("02/01/2006 15:04"), comment.Body,
		})
	}
	styleNokiaSheet(f, commentsSheet, 3, len(station.Comments)+1)
	for index, width := range []float64{30, 20, 80} {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(commentsSheet, col, col, width)
	}
	writeWorkbook(w, f, "estacao-"+station.SiteID+"-"+time.Now().Format("2006-01-02")+".xlsx")
}

func humanFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func (s *Server) stationsTemplate(w http.ResponseWriter, r *http.Request) {
	headers := make([]string, 0, len(stationColumns))
	for index, column := range stationColumns {
		label := column.Label
		if index == 0 {
			label += " *"
		}
		headers = append(headers, label)
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "Estações"
	f.SetSheetName("Sheet1", sheet)
	writeXLSXRow(f, sheet, 1, headers)
	writeXLSXRow(f, sheet, 2, []string{
		"SPEXEMPLO01", "SPSPO_0001", "eNodeB", "LTE", "Fibra", "Acesso", "Eletropaulo",
		"Greenfield", "EV", "Rua", "Rua das Palmas", "213", "Bloco B", "Jabaquara",
		"São Paulo", "SP", "04339000", "TSP", "-23.665278", "-46.641111", "Aquisitado",
		"Torre", "SPEXEMPLO01",
	})
	styleNokiaSheet(f, sheet, len(headers), 2)
	for index, column := range stationColumns {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(sheet, col, col, column.Width)
	}
	infoSheet := "Instruções"
	_, _ = f.NewSheet(infoSheet)
	instructions := [][]string{
		{"Instruções de importação"},
		{"Preencha a aba Estações a partir da linha 2. A linha de exemplo pode ser removida."},
		{"ID do site é obrigatório."},
		{"O CEP é normalizado para apenas números."},
		{"Coordenadas aceitam graus decimais ou graus/minutos/segundos."},
		{"A importação é atômica e aceita no máximo 100 estações."},
	}
	for index, row := range instructions {
		writeXLSXRow(f, infoSheet, index+1, row)
	}
	f.SetColWidth(infoSheet, "A", "A", 110)
	writeWorkbook(w, f, "modelo-importacao-estacoes.xlsx")
}

func (s *Server) stationsImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Arquivo inválido.", nil)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeValidation(w, validationErrors{"file": "Selecione uma planilha CSV ou XLSX."})
		return
	}
	defer file.Close()
	matrix, err := readNokiaImportMatrix(file, header.Filename)
	if err != nil || len(matrix) < 2 {
		writeValidation(w, validationErrors{"file": "O arquivo está vazio ou inválido."})
		return
	}
	index := map[string]int{}
	for i, value := range matrix[0] {
		index[normalizeNokiaHeader(value)] = i
	}
	if _, ok := index["id_do_site"]; !ok {
		if _, ok := index["site_id"]; !ok {
			writeValidation(w, validationErrors{"file": "O arquivo precisa conter a coluna ID do site."})
			return
		}
	}
	get := func(row []string, names ...string) string {
		for _, name := range names {
			if column, ok := index[normalizeNokiaHeader(name)]; ok && column < len(row) {
				return strings.TrimSpace(row[column])
			}
		}
		return ""
	}
	payloads := make([]stationPayload, 0, len(matrix)-1)
	errorsList := []string{}
	for line, row := range matrix[1:] {
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		payload := stationPayload{
			SiteID:           get(row, "id_do_site", "site_id", "site"),
			EnderecoID:       get(row, "id_do_endereco", "endereco_id"),
			TipoDeElemento:   get(row, "tipo_de_elemento"),
			Tecnologia:       get(row, "tecnologia"),
			TipoDeConexao:    get(row, "tipo_de_conexao"),
			Classificacao:    get(row, "classificacao"),
			DetentorArea:     get(row, "detentor_da_area", "detentor_area"),
			TipoDeInfra:      get(row, "tipo_de_infra"),
			TipoDeEV:         get(row, "tipo_de_ev", "tipo_de_ev"),
			TipoDeLogradouro: get(row, "tipo_de_logradouro"),
			Logradouro:       get(row, "logradouro"),
			Numero:           get(row, "numero", "número"),
			Complemento:      get(row, "complemento"),
			Bairro:           get(row, "bairro"),
			Municipio:        get(row, "municipio", "município", "cidade"),
			Estado:           get(row, "uf", "estado"),
			CEP:              get(row, "cep"),
			Regional:         get(row, "regional"),
			Latitude:         get(row, "latitude", "lat"),
			Longitude:        get(row, "longitude", "long", "lng", "lon"),
			Status:           get(row, "status"),
			TipoDaTorre:      get(row, "tipo_da_torre"),
			StationID:        get(row, "id_da_estacao", "station_id"),
		}
		normalizeStationPayload(&payload)
		if fields := validateStation(payload); !fields.empty() {
			for _, message := range fields {
				errorsList = append(errorsList, fmt.Sprintf("Linha %d: %s", line+2, message))
			}
			continue
		}
		payloads = append(payloads, payload)
	}
	if len(payloads) == 0 && len(errorsList) == 0 {
		writeValidation(w, validationErrors{"file": "O arquivo não possui nenhuma estação para importar."})
		return
	}
	if len(payloads) > 100 {
		writeValidation(w, validationErrors{"file": "A importação aceita no máximo 100 estações por arquivo."})
		return
	}
	if len(errorsList) > 0 {
		message := strings.Join(firstN(errorsList, 5), " ")
		if len(errorsList) > 5 {
			message += fmt.Sprintf(" E mais %d erro(s).", len(errorsList)-5)
		}
		writeValidation(w, validationErrors{"file": message})
		return
	}
	for _, payload := range payloads {
		if _, err := s.Modules.CreateStation(payload.input()); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Não foi possível importar as estações.", nil)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("%d estação(ões) importada(s).", len(payloads)),
	})
}
