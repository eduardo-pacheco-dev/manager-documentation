package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

type iepmsUserPayload struct {
	ResourceID                      string `json:"resource_id"`
	Name                            string `json:"name"`
	Gender                          string `json:"gender"`
	Nationality                     string `json:"nationality"`
	UsGreenCard                     string `json:"us_green_card"`
	Subcontractor                   string `json:"subcontractor"`
	CertificateType                 string `json:"certificate_type"`
	IDNumber                        string `json:"id_number"`
	BirthYear                       string `json:"birth_year"`
	Tel                             string `json:"tel"`
	PersonalMailbox                 string `json:"personal_mailbox"`
	Remarks                         string `json:"remarks"`
	RestrictedPartyScreeningResults string `json:"restricted_party_screening_results"`
	ServiceField                    string `json:"service_field"`
	OutsourcingType                 string `json:"outsourcing_type"`
	PersonnelType                   string `json:"personnel_type"`
	ServiceDepartmentTwoLevel       string `json:"service_department_two_level"`
	ServiceDepartmentThreeLevel     string `json:"service_department_three_level"`
	ServiceDepartmentFourLevel      string `json:"service_department_four_level"`
	PersonnelStatus                 string `json:"personnel_status"`
	OutsourcingAccount              string `json:"outsourcing_account"`
	AccountStatus                   string `json:"account_status"`
	ZTEMail                         string `json:"zte_mail"`
	AcademicQualifications          string `json:"academic_qualifications"`
	LearningMode                    string `json:"learning_mode"`
	GraduationTime                  string `json:"graduation_time"`
	GraduationSchool                string `json:"graduation_school"`
	Major                           string `json:"major"`
	CreatedBy                       string `json:"created_by"`
	LastUpdatedBy                   string `json:"last_updated_by"`
}

func (p iepmsUserPayload) input() modules.IepmsUserInput {
	return modules.IepmsUserInput{
		ResourceID: p.ResourceID, Name: p.Name, Gender: p.Gender, Nationality: p.Nationality,
		UsGreenCard: p.UsGreenCard, Subcontractor: p.Subcontractor, CertificateType: p.CertificateType,
		IDNumber: p.IDNumber, BirthYear: p.BirthYear, Tel: p.Tel, PersonalMailbox: p.PersonalMailbox,
		Remarks: p.Remarks, RestrictedPartyScreeningResults: p.RestrictedPartyScreeningResults,
		ServiceField: p.ServiceField, OutsourcingType: p.OutsourcingType, PersonnelType: p.PersonnelType,
		ServiceDepartmentTwoLevel: p.ServiceDepartmentTwoLevel, ServiceDepartmentThreeLevel: p.ServiceDepartmentThreeLevel,
		ServiceDepartmentFourLevel: p.ServiceDepartmentFourLevel, PersonnelStatus: p.PersonnelStatus,
		OutsourcingAccount: p.OutsourcingAccount, AccountStatus: p.AccountStatus, ZTEMail: p.ZTEMail,
		AcademicQualifications: p.AcademicQualifications, LearningMode: p.LearningMode,
		GraduationTime: p.GraduationTime, GraduationSchool: p.GraduationSchool, Major: p.Major,
		CreatedBy: p.CreatedBy, LastUpdatedBy: p.LastUpdatedBy,
	}
}

func (p *iepmsUserPayload) fromUser(user modules.IepmsUser) {
	p.ResourceID = user.ResourceID
	p.Name = user.Name
	p.Gender = derefString(user.Gender)
	p.Nationality = derefString(user.Nationality)
	p.UsGreenCard = derefString(user.UsGreenCard)
	p.Subcontractor = derefString(user.Subcontractor)
	p.CertificateType = derefString(user.CertificateType)
	p.IDNumber = derefString(user.IDNumber)
	p.BirthYear = derefString(user.BirthYear)
	p.Tel = derefString(user.Tel)
	p.PersonalMailbox = derefString(user.PersonalMailbox)
	p.Remarks = derefString(user.Remarks)
	p.RestrictedPartyScreeningResults = derefString(user.RestrictedPartyScreeningResults)
	p.ServiceField = derefString(user.ServiceField)
	p.OutsourcingType = derefString(user.OutsourcingType)
	p.PersonnelType = derefString(user.PersonnelType)
	p.ServiceDepartmentTwoLevel = derefString(user.ServiceDepartmentTwoLevel)
	p.ServiceDepartmentThreeLevel = derefString(user.ServiceDepartmentThreeLevel)
	p.ServiceDepartmentFourLevel = derefString(user.ServiceDepartmentFourLevel)
	p.PersonnelStatus = derefString(user.PersonnelStatus)
	p.OutsourcingAccount = derefString(user.OutsourcingAccount)
	p.AccountStatus = derefString(user.AccountStatus)
	p.ZTEMail = derefString(user.ZTEMail)
	p.AcademicQualifications = derefString(user.AcademicQualifications)
	p.LearningMode = derefString(user.LearningMode)
	p.GraduationTime = derefString(user.GraduationTime)
	p.GraduationSchool = derefString(user.GraduationSchool)
	p.Major = derefString(user.Major)
	p.CreatedBy = derefString(user.CreatedBy)
	p.LastUpdatedBy = derefString(user.LastUpdatedBy)
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
	return fields
}

func (s *Server) iepmsUsersIndex(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page := parsePositiveInt(query.Get("page"), 1)
	perPage := parsePositiveInt(query.Get("per_page"), 20)
	if perPage < 5 {
		perPage = 5
	}
	if perPage > 100 {
		perPage = 100
	}
	items, total, stats, statuses, serviceFields, err := s.Modules.ListIepmsUsers(modules.IepmsUserFilter{
		Search:       query.Get("search"),
		Status:       query.Get("status"),
		ServiceField: query.Get("service_field"),
		Page:         page,
		PerPage:      perPage,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível carregar os usuários do iEPMS.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":           items,
		"stats":          stats,
		"statuses":       statuses,
		"service_fields": serviceFields,
		"meta":           paginationMeta(page, perPage, total),
	})
}

func (s *Server) iepmsUsersShow(w http.ResponseWriter, r *http.Request) {
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
	var payload iepmsUserPayload
	payload.fromUser(user)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"user": payload}})
}

func (s *Server) iepmsUsersCreate(w http.ResponseWriter, r *http.Request) {
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
	user, err := s.Modules.CreateIepmsUser(payload.input())
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
	user, err := s.Modules.UpdateIepmsUser(id, payload.input())
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
	{"us_green_card", "Having US green cards or not", 26},
	{"subcontractor", "Subcontractor", 28},
	{"certificate_type", "Certificate Type", 18},
	{"id_number", "ID Number", 20},
	{"birth_year", "Birth year", 12},
	{"tel", "Tel", 24},
	{"personal_mailbox", "Personal mailbox", 30},
	{"remarks", "Remarks", 24},
	{"restricted_party_screening_results", "Restricted party screening results", 30},
	{"service_field", "Service Field", 20},
	{"outsourcing_type", "Outsourcing Type", 22},
	{"personnel_type", "Personnel Type", 22},
	{"service_department_two_level", "Service Department - two level", 28},
	{"service_department_three_level", "Service Department - three level", 30},
	{"service_department_four_level", "Service Department - four level", 30},
	{"personnel_status", "Personnel Status", 18},
	{"outsourcing_account", "Outsourcing account", 22},
	{"account_status", "Account Status", 16},
	{"zte_mail", "ZTE mail", 28},
	{"academic_qualifications", "Academic qualifications", 24},
	{"learning_mode", "Learning Mode", 16},
	{"graduation_time", "Graduation time", 18},
	{"graduation_school", "Graduation school", 30},
	{"major", "Major", 24},
	{"created_by", "Created by", 30},
	{"last_updated_by", "Last updated by", 30},
}

func iepmsUserFieldValue(user modules.IepmsUser, key string) string {
	values := map[string]string{
		"resource_id":                        user.ResourceID,
		"name":                               user.Name,
		"gender":                             derefString(user.Gender),
		"nationality":                        derefString(user.Nationality),
		"us_green_card":                      derefString(user.UsGreenCard),
		"subcontractor":                      derefString(user.Subcontractor),
		"certificate_type":                   derefString(user.CertificateType),
		"id_number":                          derefString(user.IDNumber),
		"birth_year":                         derefString(user.BirthYear),
		"tel":                                derefString(user.Tel),
		"personal_mailbox":                   derefString(user.PersonalMailbox),
		"remarks":                            derefString(user.Remarks),
		"restricted_party_screening_results": derefString(user.RestrictedPartyScreeningResults),
		"service_field":                      derefString(user.ServiceField),
		"outsourcing_type":                   derefString(user.OutsourcingType),
		"personnel_type":                     derefString(user.PersonnelType),
		"service_department_two_level":       derefString(user.ServiceDepartmentTwoLevel),
		"service_department_three_level":     derefString(user.ServiceDepartmentThreeLevel),
		"service_department_four_level":      derefString(user.ServiceDepartmentFourLevel),
		"personnel_status":                   derefString(user.PersonnelStatus),
		"outsourcing_account":                derefString(user.OutsourcingAccount),
		"account_status":                     derefString(user.AccountStatus),
		"zte_mail":                           derefString(user.ZTEMail),
		"academic_qualifications":            derefString(user.AcademicQualifications),
		"learning_mode":                      derefString(user.LearningMode),
		"graduation_time":                    derefString(user.GraduationTime),
		"graduation_school":                  derefString(user.GraduationSchool),
		"major":                              derefString(user.Major),
		"created_by":                         derefString(user.CreatedBy),
		"last_updated_by":                    derefString(user.LastUpdatedBy),
	}
	return values[key]
}

func (s *Server) iepmsUsersExport(w http.ResponseWriter, r *http.Request) {
	items, _, _, _, _, err := s.Modules.ListIepmsUsers(modules.IepmsUserFilter{Page: 1, PerPage: 100000})
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
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "iEPMS"
	f.SetSheetName("Sheet1", sheet)
	headers := make([]string, 0, len(iepmsUserColumns))
	for _, column := range iepmsUserColumns {
		headers = append(headers, column.Label)
	}
	writeXLSXRow(f, sheet, 1, headers)
	writeXLSXRow(f, sheet, 2, []string{
		"W0000000000", "NOME DO COLABORADOR", "MALE", "Brazil", "", "EMPRESA LTDA", "ID Card",
		"00000000000", "1990", "0055|11999999999", "email@empresa.com", "", "No",
		"Engineering Service", "Project Outsourcing", "Project subcontractor", "", "", "",
		"Normal", "6000000000", "Normal", "", "Others", "", "", "", "",
		"system", "system",
	})
	styleNokiaSheet(f, sheet, len(headers), 2)
	for index, column := range iepmsUserColumns {
		col, _ := excelize.ColumnNumberToName(index + 1)
		f.SetColWidth(sheet, col, col, column.Width)
	}
	infoSheet := "Instruções"
	_, _ = f.NewSheet(infoSheet)
	instructions := [][]string{
		{"Instruções de importação — Usuários iEPMS"},
		{"Preencha a aba iEPMS a partir da linha 2. A linha de exemplo pode ser removida."},
		{"Resource ID e Name são obrigatórios."},
		{"A importação cria novos registros e atualiza os existentes com o mesmo Resource ID."},
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
			ResourceID:                      get(row, "resource_id", "resource id"),
			Name:                            get(row, "name"),
			Gender:                          get(row, "gender"),
			Nationality:                     get(row, "nationality"),
			UsGreenCard:                     get(row, "having_us_green_cards_or_not", "us_green_card"),
			Subcontractor:                   get(row, "subcontractor"),
			CertificateType:                 get(row, "certificate_type"),
			IDNumber:                        get(row, "id_number"),
			BirthYear:                       get(row, "birth_year"),
			Tel:                             get(row, "tel"),
			PersonalMailbox:                 get(row, "personal_mailbox"),
			Remarks:                         get(row, "remarks"),
			RestrictedPartyScreeningResults: get(row, "restricted_party_screening_results"),
			ServiceField:                    get(row, "service_field"),
			OutsourcingType:                 get(row, "outsourcing_type"),
			PersonnelType:                   get(row, "personnel_type"),
			ServiceDepartmentTwoLevel:       get(row, "service_department_two_level", "service_department_-_two_level"),
			ServiceDepartmentThreeLevel:     get(row, "service_department_three_level", "service_department_-_three_level"),
			ServiceDepartmentFourLevel:      get(row, "service_department_four_level", "service_department_-_four_level"),
			PersonnelStatus:                 get(row, "personnel_status"),
			OutsourcingAccount:              get(row, "outsourcing_account"),
			AccountStatus:                   get(row, "account_status"),
			ZTEMail:                         get(row, "zte_mail"),
			AcademicQualifications:          get(row, "academic_qualifications"),
			LearningMode:                    get(row, "learning_mode"),
			GraduationTime:                  get(row, "graduation_time"),
			GraduationSchool:                get(row, "graduation_school"),
			Major:                           get(row, "major"),
			CreatedBy:                       get(row, "created_by"),
			LastUpdatedBy:                   get(row, "last_updated_by"),
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
		isNew, err := s.Modules.UpsertIepmsUser(payload.input())
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
