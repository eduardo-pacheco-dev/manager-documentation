package seed

import (
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
)

// ModuleData seeds small, idempotent examples for the native modules so a
// fresh installation showcases every area of the system.
func ModuleData(m *modules.Store, admin models.User) error {
	if err := seedTasks(m, admin); err != nil {
		return err
	}
	if err := seedFollowUps(m, admin); err != nil {
		return err
	}
	if err := seedStations(m); err != nil {
		return err
	}
	if err := seedDemands(m, admin); err != nil {
		return err
	}
	if err := seedNokia(m); err != nil {
		return err
	}
	return seedAgenda(m, admin)
}

func seedTasks(m *modules.Store, admin models.User) error {
	lists, err := m.TodoLists(admin.ID)
	if err != nil || len(lists) > 0 {
		return err
	}
	list, err := m.CreateTodoList(admin.ID, "Projetos", "blue", "clipboard")
	if err != nil {
		return err
	}
	today := time.Now()
	task, err := m.CreateTodoTask(admin.ID, modules.TaskInput{
		TodoListID: list.ID, Title: "Revisar documentação da sprint",
		Notes: "Conferir os documentos publicados e atualizar a wiki.",
		DueAt: today.Format("2006-01-02"), RepeatType: "weekdays", IsImportant: true, MyDay: true,
	})
	if err != nil {
		return err
	}
	_, _ = m.CreateTodoStep(admin.ID, task.ID, "Ler os documentos alterados")
	_, _ = m.CreateTodoStep(admin.ID, task.ID, "Atualizar páginas desatualizadas")
	_, _ = m.CreateTodoTask(admin.ID, modules.TaskInput{
		TodoListID: list.ID, Title: "Preparar apresentação de resultados",
		DueAt: today.AddDate(0, 0, 3).Format("2006-01-02"), RepeatType: "none",
	})
	return nil
}

func seedFollowUps(m *modules.Store, admin models.User) error {
	clients, _ := m.FollowUpClients()
	if len(clients) > 0 {
		return nil
	}
	afl, err := m.CreateFollowUpClient("AFL Engenharia")
	if err != nil {
		return err
	}
	nokia, err := m.CreateFollowUpClient("Nokia")
	if err != nil {
		return err
	}
	_, _ = m.CreateFollowUpProject(afl.ID, "Expansão 5G")
	_, _ = m.CreateFollowUpProject(afl.ID, "Manutenção preventiva")
	_, _ = m.CreateFollowUpProject(nokia.ID, "Implantação de sites")
	assignee, _ := m.CreateFollowUpAssignee(admin.FullName(), admin.Email)
	today := time.Now().Format("2006-01-02")
	items := []modules.FollowUpInput{
		{Title: "Enviar proposta comercial", Code: "FU-001", Status: "pendente", Priority: "alta", DueDate: today, ClientID: &afl.ID, AssigneeID: &assignee.ID},
		{Title: "Homologar documentação técnica", Code: "FU-002", Status: "em_andamento", Priority: "urgente", DueDate: time.Now().AddDate(0, 0, 2).Format("2006-01-02"), ClientID: &nokia.ID, AssigneeID: &assignee.ID},
		{Title: "Revisar cronograma de implantação", Code: "FU-003", Status: "pendente", Priority: "media", DueDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"), ClientID: &afl.ID, AssigneeID: &assignee.ID},
		{Title: "Fechar relatório mensal", Code: "FU-004", Status: "concluido", Priority: "baixa", DueDate: today, ClientID: &nokia.ID, AssigneeID: &assignee.ID},
	}
	for _, input := range items {
		if _, err := m.CreateFollowUp(admin.ID, input); err != nil {
			return err
		}
	}
	return nil
}

func seedStations(m *modules.Store) error {
	items, _, _, _, _, err := m.ListStations(modules.StationFilter{Page: 1, PerPage: 1})
	if err != nil || len(items) > 0 {
		return err
	}
	stations := []modules.StationInput{
		{SiteID: "SPJAB051M", EnderecoID: "SPSPO_2360", TipoDeElemento: "eNodeB", Tecnologia: "LTE", TipoDeConexao: "Fibra", Classificacao: "Acesso", DetentorArea: "Eletropaulo", TipoDeInfra: "Greenfield", Logradouro: "Rua das Palmas", Numero: "213", Bairro: "Jabaquara", Municipio: "São Paulo", Estado: "SP", CEP: "04339000", Regional: "TSP", Latitude: "-23.665278", Longitude: "-46.641111", Status: "Aquisitado", StationID: "SPJAB051M"},
		{SiteID: "SPFORA01J", TipoDeElemento: "MSAN", Tecnologia: "IP", Logradouro: "Av. Paulista", Numero: "1000", Bairro: "Bela Vista", Municipio: "São Paulo", Estado: "SP", Regional: "TSP", Latitude: "-23.561389", Longitude: "-46.656389", Status: "Ativo", StationID: "SPFORA01J"},
		{SiteID: "RJCAM001", TipoDeElemento: "ERB", Tecnologia: "5G", Municipio: "Rio de Janeiro", Estado: "RJ", Regional: "TRJ", Status: "Em análise"},
	}
	for _, input := range stations {
		if _, err := m.CreateStation(input); err != nil {
			return err
		}
	}
	return nil
}

func seedNokia(m *modules.Store) error {
	roms, _ := m.NokiaROMs()
	if len(roms) == 0 {
		_, _ = m.CreateNokiaROM("Rafael", "Almeida", "rafael.almeida@example.com", "+55 11 99999-0001")
		_, _ = m.CreateNokiaROM("Camila", "Ferreira", "camila.ferreira@example.com", "+55 11 99999-0002")
	}
	tasks, _ := m.NokiaTasks()
	if len(tasks) == 0 {
		for _, name := range []string{"TSSR aprovado", "DOC-D enviado", "DWG enviado", "Integração concluída"} {
			_, _ = m.CreateNokiaTask(name)
		}
	}
	sites, _, _, _, err := m.ListNokiaSites(modules.NokiaSiteFilter{Page: 1, PerPage: 1})
	if err != nil || len(sites) > 0 {
		return err
	}
	roms, _ = m.NokiaROMs()
	tasks, _ = m.NokiaTasks()
	first, _ := m.CreateNokiaSite(modules.NokiaSiteInput{
		SiteID: "SPJAB051M", EndID: "END-0001", OC: "OC-0001", OSFamEntrega: "OS-E-0001",
		OSFamInstalacao: "OS-I-0001", Regional: "TSP", Subcon: "AFL", NokiaROMID: &roms[0].ID,
		ChecklistFamEntrega: true,
	})
	second, _ := m.CreateNokiaSite(modules.NokiaSiteInput{
		SiteID: "SPFORA01J", EndID: "END-0002", OC: "OC-0002", OSFamEntrega: "OS-E-0002",
		OSFamInstalacao: "OS-I-0002", Regional: "TSP", Subcon: "AFL", NokiaROMID: &roms[1].ID,
	})
	if first.ID > 0 && len(tasks) > 0 {
		_ = m.ToggleNokiaTask(first.ID, tasks[0].ID)
		_ = m.ToggleNokiaTask(first.ID, tasks[1].ID)
		_, _ = m.UpdateNokiaDates(first.ID, map[string]string{
			"mos_plan_at": "2026-10-01", "mos_real_at": "2026-10-03",
			"instalacao_plan_at": "2026-10-05", "instalacao_real_at": "2026-10-08",
			"integracao_plan_at": "2026-10-10", "integracao_real_at": "2026-10-12",
			"fam_instalacao_approved_at": "2026-10-15",
		})
	}
	if second.ID > 0 && len(tasks) > 1 {
		_ = m.ToggleNokiaTask(second.ID, tasks[1].ID)
	}
	return nil
}

func seedDemands(m *modules.Store, admin models.User) error {
	var count int
	if err := m.DB.Get(&count, `SELECT COUNT(*) FROM demands WHERE title = ?`, "Instalação de equipamento 5G"); err != nil || count > 0 {
		return err
	}
	collaborators, _ := m.DemandCollaborators()
	assigneeID := admin.ID
	for _, collaborator := range collaborators {
		if collaborator.ID != admin.ID {
			assigneeID = collaborator.ID
			break
		}
	}
	stations, _ := m.SearchDemandStations("", 2)
	var firstStationID, secondStationID *int64
	if len(stations) > 0 {
		firstStationID = &stations[0].ID
	}
	if len(stations) > 1 {
		secondStationID = &stations[1].ID
	}

	today := time.Now()
	inputs := []modules.DemandInput{
		{
			AssigneeID: assigneeID, Title: "Instalação de equipamento 5G",
			Description: "Instalar e integrar os equipamentos do site, registrar evidências fotográficas e atualizar o as-built.",
			ClientName:  "TIM Brasil", ClientEmail: "cliente.tim@example.com",
			StationID: firstStationID, Operadora: "TIM", DemandType: "Instalação",
			OCTIM: "OC-TIM-2026-001", ClientOS: "OS-CLIENTE-7788", KeyNumber: "CHV-SP-001",
			ActivityScope: "Instalação de RRU, validação de energia, integração e documentação fotográfica.",
			Priority:      "alta", DueAt: today.AddDate(0, 0, 3).Format("2006-01-02"), Status: "em_andamento",
		},
		{
			AssigneeID: admin.ID, Title: "Vistoria técnica de site",
			Description: "Realizar vistoria preventiva e registrar as condições da infraestrutura.",
			ClientName:  "Vivo", ClientEmail: "cliente.vivo@example.com",
			StationID: secondStationID, Operadora: "Vivo", DemandType: "Vistoria",
			OCTIM: "OC-TIM-2026-002", ClientOS: "OS-CLIENTE-9911", KeyNumber: "CHV-SP-002",
			ActivityScope: "Inspeção de torre, abrigo, energia e aterramento.",
			Priority:      "media", DueAt: today.AddDate(0, 0, 7).Format("2006-01-02"), Status: "aberta",
		},
		{
			AssigneeID: assigneeID, Title: "Enviar evidências de conclusão",
			Description: "Consolidar fotos, relatórios e medições para aceite do cliente.",
			ClientName:  "TIM Brasil", ClientEmail: "cliente.tim@example.com",
			Operadora: "TIM", DemandType: "Documentação",
			OCTIM: "OC-TIM-2026-003", ClientOS: "OS-CLIENTE-3344", KeyNumber: "CHV-SP-003",
			ActivityScope: "Organizar evidências e enviar o pacote final de documentação.",
			Priority:      "baixa", DueAt: today.AddDate(0, 0, 10).Format("2006-01-02"), Status: "aguardando",
		},
		{
			AssigneeID: admin.ID, Title: "Revisão de documentação as-built",
			Description: "Revisar a documentação as-built entregue pela equipe de campo.",
			ClientName:  "TIM Brasil", ClientEmail: "cliente.tim@example.com",
			Operadora: "TIM", DemandType: "Revisão",
			OCTIM: "OC-TIM-2026-004", ClientOS: "OS-CLIENTE-5566", KeyNumber: "CHV-SP-004",
			ActivityScope: "Validar plantas, fotos e relatórios antes da conclusão.",
			Priority:      "media", DueAt: today.AddDate(0, 0, -2).Format("2006-01-02"), Status: "concluida",
		},
	}
	var firstDemand modules.Demand
	for index, input := range inputs {
		demand, err := m.CreateDemand(admin.ID, input)
		if err != nil {
			return err
		}
		if index == 0 {
			firstDemand = demand
		}
	}
	if firstDemand.ID > 0 {
		_, _ = m.AddDemandUpdate(firstDemand.ID, admin.ID, "Demanda recebida e equipe de campo acionada.")
		_, _ = m.AddDemandUpdate(firstDemand.ID, assigneeID, "Materiais conferidos e instalação iniciada no site.")
	}
	return nil
}

func seedAgenda(m *modules.Store, admin models.User) error {
	if err := m.EnsureAgendaCalendars(admin.ID); err != nil {
		return err
	}
	calendars, err := m.AgendaCalendars(admin.ID)
	if err != nil || len(calendars) == 0 {
		return err
	}
	from := time.Now()
	to := time.Now().AddDate(0, 0, 60)
	occurrences, _ := m.AgendaEventsInRange(admin.ID, nil, from, to, "")
	if len(occurrences) > 0 {
		return nil
	}
	calendar := calendars[0]
	next := time.Now().AddDate(0, 0, 1)
	start := time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, time.Local)
	recurrence := &modules.AgendaRecurrence{Frequency: "weekly", Interval: 1, Count: 8}
	_, err = m.CreateAgendaEvent(admin.ID, modules.AgendaEventInput{
		CalendarID: calendar.ID, Title: "Alinhamento semanal", Description: "Reunião de acompanhamento do time.",
		StartAt: start.Format(time.RFC3339), EndAt: start.Add(time.Hour).Format(time.RFC3339),
		Status: "confirmed", Recurrence: recurrence, CanInvite: true, CanSeeOthers: true,
		Reminders: []modules.AgendaReminderInput{{Method: "popup", MinutesBefore: intPtr(10)}},
	})
	if err != nil {
		return err
	}
	planning := start.AddDate(0, 0, 2)
	_, err = m.CreateAgendaEvent(admin.ID, modules.AgendaEventInput{
		CalendarID: calendar.ID, Title: "Planejamento de sprint", Location: "Sala 3",
		StartAt: planning.Format(time.RFC3339), EndAt: planning.Add(90 * time.Minute).Format(time.RFC3339),
		Status: "confirmed", CanInvite: true, CanSeeOthers: true,
	})
	return err
}

func intPtr(value int) *int { return &value }
