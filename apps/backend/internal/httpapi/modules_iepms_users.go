package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/modules"
)

type iepmsUserPayload struct {
	ResourceID         string `json:"resource_id"`
	Name               string `json:"name"`
	Gender             string `json:"gender"`
	Nationality        string `json:"nationality"`
	CertificateType    string `json:"certificate_type"`
	IDNumber           string `json:"id_number"`
	BirthYear          string `json:"birth_year"`
	Tel                string `json:"tel"`
	PersonalMailbox    string `json:"personal_mailbox"`
	OutsourcingAccount string `json:"outsourcing_account"`
	LastUpdatedBy      string `json:"last_updated_by"`
	Status             string `json:"status"`
	Password           string `json:"password"`
}

func (p iepmsUserPayload) input() modules.IepmsUserInput {
	return modules.IepmsUserInput{
		ResourceID: p.ResourceID, Name: p.Name, Gender: p.Gender, Nationality: p.Nationality,
		CertificateType: p.CertificateType,
		IDNumber:        p.IDNumber, BirthYear: p.BirthYear, Tel: p.Tel, PersonalMailbox: p.PersonalMailbox,
		OutsourcingAccount: p.OutsourcingAccount,
		LastUpdatedBy:      p.LastUpdatedBy, Status: p.Status,
	}
}

func validateIepmsUser(payload iepmsUserPayload) validationErrors {
	fields := validationErrors{}
	if strings.TrimSpace(payload.ResourceID) == "" {
		fields.add("resource_id", "Informe o Resource ID.")
	}
	if strings.TrimSpace(payload.Name) == "" {
		fields.add("name", "Informe o nome.")
	}
	if len([]rune(payload.ResourceID)) > 64 {
		fields.add("resource_id", "O Resource ID deve ter no máximo 64 caracteres.")
	}
	status := normalizeIepmsStatus(payload.Status)
	if status != "" && status != "ativo" && status != "inativado" {
		fields.add("status", "Selecione Ativo ou Inativado.")
	}
	return fields
}

func normalizeIepmsStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ativo", "active", "1", "sim", "yes":
		return "ativo"
	case "inativado", "inativo", "inactive", "0", "nao", "não":
		return "inativado"
	default:
		return strings.TrimSpace(value)
	}
}

// iepmsUserInput converts the payload into the store input, encrypting the
// password with AES-GCM when a new password was provided. A blank password
// keeps the existing secret on updates.
func (s *Server) iepmsUserInput(payload iepmsUserPayload) (modules.IepmsUserInput, error) {
	input := payload.input()
	input.Status = normalizeIepmsStatus(payload.Status)
	if payload.Password == "" {
		return input, nil
	}
	encrypted, err := s.Cipher.Encrypt(payload.Password)
	if err != nil {
		return input, err
	}
	input.PasswordEncrypted = encrypted
	input.PasswordChanged = true
	return input, nil
}

func (s *Server) iepmsUsersIndex(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 20)
	if perPage < 5 {
		perPage = 5
	}
	if perPage > 100 {
		perPage = 100
	}
	items, total, stats, statuses, err := s.Modules.ListIepmsUsers(modules.IepmsUserFilter{
		Search:  query.Get("search"),
		Status:  query.Get("status"),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os usuários do iEPMS.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     items,
		"stats":    stats,
		"statuses": statuses,
		"meta":     paginationMeta(page, perPage, total),
	})
}

func (s *Server) iepmsUsersShow(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	user, err := s.Modules.FindIepmsUser(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"user": user}})
}

func (s *Server) iepmsUsersCreate(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	var payload iepmsUserPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateIepmsUser(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if _, err := s.Modules.IepmsUserByResourceID(payload.ResourceID); err == nil {
		writeValidation(w, validationErrors{"resource_id": "Já existe um usuário com este Resource ID."})
		return
	}
	if account := strings.TrimSpace(payload.OutsourcingAccount); account != "" {
		if _, err := s.Modules.IepmsUserByAccount(account); err == nil {
			writeValidation(w, validationErrors{"outsourcing_account": "Já existe um usuário com este número de conta."})
			return
		}
	}
	input, err := s.iepmsUserInput(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível proteger a senha do usuário.", nil)
		return
	}
	user, err := s.Modules.CreateIepmsUser(input)
	if err != nil {
		if isDuplicateKey(err) {
			writeValidation(w, validationErrors{"resource_id": "Já existe um usuário com este Resource ID."})
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível criar o usuário do iEPMS.", nil)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "Usuário do iEPMS cadastrado com sucesso.",
		"data":    map[string]any{"user": user},
	})
}

func (s *Server) iepmsUsersUpdate(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	var payload iepmsUserPayload
	if !decodeJSON(w, r, &payload) {
		return
	}
	if fields := validateIepmsUser(payload); !fields.empty() {
		writeValidation(w, fields)
		return
	}
	if existing, err := s.Modules.IepmsUserByResourceID(payload.ResourceID); err == nil && existing.ID != id {
		writeValidation(w, validationErrors{"resource_id": "Já existe outro usuário com este Resource ID."})
		return
	}
	if account := strings.TrimSpace(payload.OutsourcingAccount); account != "" {
		if existing, err := s.Modules.IepmsUserByAccount(account); err == nil && existing.ID != id {
			writeValidation(w, validationErrors{"outsourcing_account": "Já existe outro usuário com este número de conta."})
			return
		}
	}
	input, err := s.iepmsUserInput(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível proteger a senha do usuário.", nil)
		return
	}
	user, err := s.Modules.UpdateIepmsUser(id, input)
	if err != nil {
		if isDuplicateKey(err) {
			writeValidation(w, validationErrors{"resource_id": "Já existe outro usuário com este Resource ID."})
			return
		}
		writeError(w, http.StatusInternalServerError, "Não foi possível atualizar o usuário do iEPMS.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Usuário do iEPMS atualizado com sucesso.",
		"data":    map[string]any{"user": user},
	})
}

func (s *Server) iepmsUsersDelete(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	if err := s.Modules.DeleteIepmsUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível excluir o usuário do iEPMS.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Usuário do iEPMS excluído com sucesso."})
}

// iepmsUsersRevealPassword decrypts and returns the stored password only after
// the signed-in user confirms their own account password.
func (s *Server) iepmsUsersRevealPassword(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	var payload struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if strings.TrimSpace(payload.Password) == "" {
		writeValidation(w, validationErrors{"password": "Informe a sua senha para visualizar."})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(currentUser(r).PasswordHash), []byte(payload.Password)) != nil {
		writeValidation(w, validationErrors{"password": "A senha informada não confere."})
		return
	}
	encrypted, err := s.Modules.FindIepmsUserPassword(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Usuário do iEPMS não encontrado.", nil)
		return
	}
	if strings.TrimSpace(encrypted) == "" {
		writeValidation(w, validationErrors{"password": "Este usuário ainda não possui senha cadastrada."})
		return
	}
	plain, err := s.Cipher.Decrypt(encrypted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível descriptografar a senha.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"password": plain}})
}

func isDuplicateKey(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

/* --------------------------------------------------------------------------
 * Import / export
 * ----------------------------------------------------------------------- */

type iepmsUserColumn struct {
	Key   string
	Label string
	Width float64
}

// iepmsUserColumns follows the exact header order of the iEPMS Excel export.
var iepmsUserColumns = []iepmsUserColumn{
	{"resource_id", "Resource ID", 18},
	{"name", "Name", 32},
	{"gender", "Gender", 12},
	{"nationality", "Nationality", 16},
	{"certificate_type", "Certificate Type", 18},
	{"id_number", "ID Number", 20},
	{"birth_year", "Birth year", 12},
	{"tel", "Tel", 24},
	{"personal_mailbox", "Personal mailbox", 30},
	{"outsourcing_account", "Outsourcing account", 22},
	{"last_updated_by", "Last updated by", 30},
}

func iepmsUserFieldValue(user modules.IepmsUser, key string) string {
	values := map[string]string{
		"resource_id":         user.ResourceID,
		"name":                user.Name,
		"gender":              derefString(user.Gender),
		"nationality":         derefString(user.Nationality),
		"certificate_type":    derefString(user.CertificateType),
		"id_number":           derefString(user.IDNumber),
		"birth_year":          derefString(user.BirthYear),
		"tel":                 derefString(user.Tel),
		"personal_mailbox":    derefString(user.PersonalMailbox),
		"outsourcing_account": derefString(user.OutsourcingAccount),
		"last_updated_by":     derefString(user.LastUpdatedBy),
	}
	return values[key]
}

func (s *Server) iepmsUsersExport(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	items, _, _, _, err := s.Modules.ListIepmsUsers(modules.IepmsUserFilter{Page: 1, PerPage: 100000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível exportar os usuários do iEPMS.", nil)
		return
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "iEPMS"
	f.SetSheetName("Sheet1", sheet)
	headers := make([]string, 0, len(iepmsUserColumns))
	for _, column := range iepmsUserColumns {
		headers = append(headers, column.Label)
	}
	writeXLSXRow(f, sheet, 1, headers)
	for index, user := range items {
		row := make([]string, 0, len(headers))
		for _, column := range iepmsUserColumns {
			row = append(row, iepmsUserFieldValue(user, column.Key))
		}
		writeXLSXRow(f, sheet, index+2, row)
	}
	styleNokiaSheet(f, sheet, len(headers), len(items)+1)
	for index, column := range iepmsUserColumns {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(sheet, col, col, column.Width)
	}
	writeWorkbook(w, f, "iepms-usuarios-"+time.Now().Format("2006-01-02")+".xlsx")
}

func (s *Server) iepmsUsersTemplate(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "iEPMS"
	f.SetSheetName("Sheet1", sheet)
	headers := make([]string, 0, len(iepmsUserColumns)+1)
	for _, column := range iepmsUserColumns {
		headers = append(headers, column.Label)
	}
	headers = append(headers, "Password (opcional)", "Status")
	writeXLSXRow(f, sheet, 1, headers)
	writeXLSXRow(f, sheet, 2, []string{
		"W0000000000", "NOME DO COLABORADOR", "MALE", "Brazil", "ID Card",
		"00000000000", "1990", "0055|11999999999", "email@empresa.com",
		"system", "", "Ativo",
	})
	styleNokiaSheet(f, sheet, len(headers), 2)
	for index, column := range iepmsUserColumns {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(sheet, col, col, column.Width)
	}
	f.SetColWidth(sheet, "K", "L", 24)
	infoSheet := "Instruções"
	_, _ = f.NewSheet(infoSheet)
	instructions := [][]string{
		{"Instruções de importação — Usuários iEPMS"},
		{"Preencha a aba iEPMS a partir da linha 2. A linha de exemplo pode ser removida."},
		{"Resource ID e Name são obrigatórios."},
		{"A importação cria novos registros e atualiza os existentes pelo número da conta (Outsourcing account) ou, se não houver, pelo Resource ID."},
		{"O número da conta é único: se já existir, o cadastro correspondente é atualizado com os dados da planilha."},
		{"A coluna Password (opcional) define a senha do usuário. Ela é armazenada criptografada e nunca é exportada."},
		{"Ao atualizar um registro, deixe a coluna Password em branco para manter a senha atual."},
		{"A coluna Status (opcional) aceita Ativo ou Inativado. Em branco mantém o status atual ou cria como Ativo."},
		{"Mantenha as colunas ID Number, Birth year, Tel e Outsourcing account formatadas como texto para preservar zeros à esquerda."},
		{"A importação aceita arquivos XLSX ou CSV e até 2.000 linhas por arquivo."},
	}
	for index, row := range instructions {
		writeXLSXRow(f, infoSheet, index+1, row)
	}
	f.SetColWidth(infoSheet, "A", "A", 120)
	writeWorkbook(w, f, "modelo-importacao-usuarios-iepms.xlsx")
}

func (s *Server) iepmsUsersImport(w http.ResponseWriter, r *http.Request) {
	s, ok := s.useTenant(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil {
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
	if _, ok := index["resource_id"]; !ok {
		writeValidation(w, validationErrors{"file": "O arquivo precisa conter a coluna Resource ID."})
		return
	}
	get := func(row []string, names ...string) string {
		for _, name := range names {
			if column, ok := index[normalizeNokiaHeader(name)]; ok && column < len(row) {
				return strings.TrimSpace(row[column])
			}
		}
		return ""
	}
	payloads := make([]iepmsUserPayload, 0, len(matrix)-1)
	errorsList := []string{}
	seen := map[string]int{}
	for line, row := range matrix[1:] {
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		payload := iepmsUserPayload{
			ResourceID:         get(row, "resource_id", "resource id"),
			Name:               get(row, "name"),
			Gender:             get(row, "gender"),
			Nationality:        get(row, "nationality"),
			CertificateType:    get(row, "certificate_type"),
			IDNumber:           get(row, "id_number"),
			BirthYear:          get(row, "birth_year"),
			Tel:                get(row, "tel"),
			PersonalMailbox:    get(row, "personal_mailbox"),
			OutsourcingAccount: get(row, "outsourcing_account"),
			LastUpdatedBy:      get(row, "last_updated_by"),
			Status:             get(row, "status", "situacao", "situação"),
			Password:           get(row, "password", "password_opcional"),
		}
		if fields := validateIepmsUser(payload); !fields.empty() {
			for _, message := range fields {
				errorsList = append(errorsList, fmt.Sprintf("Linha %d: %s", line+2, message))
			}
			continue
		}
		key := strings.ToLower(strings.TrimSpace(payload.ResourceID))
		if first, exists := seen[key]; exists {
			errorsList = append(errorsList, fmt.Sprintf("Linha %d: Resource ID duplicado no arquivo (já usado na linha %d).", line+2, first))
			continue
		}
		seen[key] = line + 2
		payloads = append(payloads, payload)
	}
	if len(payloads) == 0 && len(errorsList) == 0 {
		writeValidation(w, validationErrors{"file": "O arquivo não possui nenhum usuário para importar."})
		return
	}
	if len(payloads) > 2000 {
		writeValidation(w, validationErrors{"file": "A importação aceita no máximo 2.000 usuários por arquivo."})
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
	created, updated := 0, 0
	for _, payload := range payloads {
		input, err := s.iepmsUserInput(payload)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível proteger a senha durante a importação.", nil)
			return
		}
		isNew, err := s.Modules.UpsertIepmsUser(input)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Não foi possível importar os usuários do iEPMS.", nil)
			return
		}
		if isNew {
			created++
		} else {
			updated++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("%d usuário(s) criado(s) e %d atualizado(s) na importação.", created, updated),
		"data":    map[string]any{"created": created, "updated": updated, "total": len(payloads)},
	})
}
