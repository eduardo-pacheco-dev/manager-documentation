package seed

import (
	"fmt"
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
	collaborators, err := seedCollaborators(m)
	if err != nil {
		return err
	}
	if err := seedProjects(m, admin, collaborators); err != nil {
		return err
	}
	return seedAgenda(m, admin)
}

// seedCollaborators registers the demo people used across the modules. It
// returns a name lookup so project resources can link to the central registry.
func seedCollaborators(m *modules.Store) (map[string]int64, error) {
	seeds := []modules.CollaboratorInput{
		{
			Name: "Ana Souza", SocialName: "Ana Paula Souza", Email: "ana.souza@example.com", PersonalEmail: "ana.souza.pessoal@example.com",
			Phone: "+55 11 3333-1001", Mobile: "+55 11 98888-1001", Role: "Gerente de projetos", Department: "PMO",
			ManagerName: "Fernanda Alves", CPF: "39053344705", RG: "12.345.678-9", RGIssuer: "SSP/SP",
			CTPS: "1234567", CTPSSeries: "001", PIS: "12034567890", VoterRegistration: "123456789012",
			BirthDate: "1988-03-14", Gender: "feminino", MaritalStatus: "casado", Nationality: "brasileira",
			Birthplace: "São Paulo/SP", MotherName: "Marta Souza", FatherName: "José Souza",
			ZipCode: "01310-100", Street: "Av. Paulista", AddressNumber: "1000", Complement: "Apto 101",
			Neighborhood: "Bela Vista", City: "São Paulo", State: "SP", Country: "Brasil",
			EmergencyContactName: "Carlos Souza", EmergencyContactPhone: "+55 11 97777-1001",
			ContractType: "clt", WorkRegime: "hibrido", WorkLocation: "São Paulo/SP",
			HireDate: "2019-02-04", WeeklyHours: 40, Salary: 14000,
			BankName: "Itaú", BankAgency: "1234", BankAccount: "12345-6", BankAccountType: "corrente",
			PixKey: "ana.souza@example.com", CostHour: 120, MaxUnits: 100, Color: "blue", IsActive: true,
			Notes: "Responsável pelo portfólio de implantação.",
		},
		{
			Name: "Bruno Lima", Email: "bruno.lima@example.com", PersonalEmail: "bruno.lima.pessoal@example.com",
			Phone: "+55 11 3333-1002", Mobile: "+55 11 98888-1002", Role: "Engenheiro de campo", Department: "Engenharia",
			ManagerName: "Ana Souza", CPF: "12345678909", RG: "23.456.789-0", RGIssuer: "SSP/SP",
			CTPS: "2234567", CTPSSeries: "002", PIS: "12034567891", CNH: "01234567890", CNHCategory: "B", CNHExpiry: "2028-06-20",
			BirthDate: "1991-07-22", Gender: "masculino", MaritalStatus: "solteiro", Nationality: "brasileira",
			Birthplace: "Campinas/SP", MotherName: "Helena Lima", FatherName: "Roberto Lima",
			ZipCode: "13010-100", Street: "Rua Barão de Jaguara", AddressNumber: "500", Neighborhood: "Centro",
			City: "Campinas", State: "SP", Country: "Brasil",
			EmergencyContactName: "Helena Lima", EmergencyContactPhone: "+55 11 97777-1002",
			ContractType: "clt", WorkRegime: "presencial", WorkLocation: "Campinas/SP",
			HireDate: "2021-01-11", WeeklyHours: 44, Salary: 9800,
			BankName: "Bradesco", BankAgency: "2345", BankAccount: "23456-7", BankAccountType: "corrente",
			PixKey: "bruno.lima@example.com", CostHour: 90, MaxUnits: 100, Color: "green", IsActive: true,
		},
		{
			Name: "Carla Mendes", Email: "carla.mendes@example.com", PersonalEmail: "carla.mendes.pessoal@example.com",
			Phone: "+55 11 3333-1003", Mobile: "+55 11 98888-1003", Role: "Analista de documentação", Department: "Documentação",
			ManagerName: "Ana Souza", CPF: "98765432100", RG: "34.567.890-1", RGIssuer: "SSP/RJ",
			CTPS: "3234567", CTPSSeries: "003", PIS: "12034567892",
			BirthDate: "1994-11-02", Gender: "feminino", MaritalStatus: "solteiro", Nationality: "brasileira",
			Birthplace: "Rio de Janeiro/RJ", MotherName: "Lúcia Mendes", FatherName: "Paulo Mendes",
			ZipCode: "20040-020", Street: "Av. Rio Branco", AddressNumber: "156", Neighborhood: "Centro",
			City: "Rio de Janeiro", State: "RJ", Country: "Brasil",
			EmergencyContactName: "Lúcia Mendes", EmergencyContactPhone: "+55 21 97777-1003",
			ContractType: "clt", WorkRegime: "remoto", WorkLocation: "Rio de Janeiro/RJ",
			HireDate: "2020-09-15", WeeklyHours: 30, Salary: 6500,
			BankName: "Caixa", BankAgency: "3456", BankAccount: "34567-8", BankAccountType: "poupanca",
			PixKey: "carla.mendes@example.com", CostHour: 75, MaxUnits: 50, Color: "purple", IsActive: true,
		},
		{
			Name: "Diego Rocha", Email: "diego.rocha@example.com", PersonalEmail: "diego.rocha.pessoal@example.com",
			Phone: "+55 11 3333-1004", Mobile: "+55 11 98888-1004", Role: "Técnico de instalação", Department: "Campo",
			ManagerName: "Bruno Lima", CPF: "11144477735", RG: "45.678.901-2", RGIssuer: "SSP/SP",
			CTPS: "4234567", CTPSSeries: "004", PIS: "12034567893", CNH: "11234567890", CNHCategory: "AB", CNHExpiry: "2027-09-10",
			BirthDate: "1986-05-30", Gender: "masculino", MaritalStatus: "casado", Nationality: "brasileira",
			Birthplace: "Santo André/SP", MotherName: "Teresa Rocha", FatherName: "Antônio Rocha",
			ZipCode: "09010-000", Street: "Rua Oliveira Lima", AddressNumber: "88", Neighborhood: "Centro",
			City: "Santo André", State: "SP", Country: "Brasil",
			EmergencyContactName: "Marina Rocha", EmergencyContactPhone: "+55 11 97777-1004",
			ContractType: "clt", WorkRegime: "presencial", WorkLocation: "Santo André/SP",
			HireDate: "2018-04-02", WeeklyHours: 44, Salary: 5200,
			BankName: "Santander", BankAgency: "4567", BankAccount: "45678-9", BankAccountType: "corrente",
			PixKey: "diego.rocha@example.com", CostHour: 60, MaxUnits: 100, Color: "orange", IsActive: true,
		},
		{
			Name: "Marina Costa", Email: "marina.costa@example.com", PersonalEmail: "marina.costa.pessoal@example.com",
			Phone: "+55 11 3333-1005", Mobile: "+55 11 98888-1005", Role: "Arquiteta de redes", Department: "Engenharia",
			ManagerName: "Ana Souza", CPF: "52998224725", RG: "56.789.012-3", RGIssuer: "SSP/SP",
			CTPS: "5234567", CTPSSeries: "005", PIS: "12034567894",
			BirthDate: "1989-12-09", Gender: "feminino", MaritalStatus: "casado", Nationality: "brasileira",
			Birthplace: "São Bernardo do Campo/SP", MotherName: "Sônia Costa", FatherName: "Marcos Costa",
			ZipCode: "09710-000", Street: "Rua Marechal Deodoro", AddressNumber: "320", Neighborhood: "Centro",
			City: "São Bernardo do Campo", State: "SP", Country: "Brasil",
			EmergencyContactName: "Rafael Costa", EmergencyContactPhone: "+55 11 97777-1005",
			ContractType: "clt", WorkRegime: "hibrido", WorkLocation: "São Paulo/SP",
			HireDate: "2017-08-21", WeeklyHours: 40, Salary: 15500,
			BankName: "Nubank", BankAgency: "0001", BankAccount: "56789-0", BankAccountType: "corrente",
			PixKey: "marina.costa@example.com", CostHour: 130, MaxUnits: 100, Color: "teal", IsActive: true,
		},
		{
			Name: "Paulo Reis", Email: "paulo.reis@example.com", PersonalEmail: "paulo.reis.pessoal@example.com",
			Phone: "+55 11 3333-1006", Mobile: "+55 11 98888-1006", Role: "Analista de infraestrutura", Department: "Infraestrutura",
			ManagerName: "Marina Costa", CPF: "16899535009", RG: "67.890.123-4", RGIssuer: "SSP/MG",
			CTPS: "6234567", CTPSSeries: "006", PIS: "12034567895",
			BirthDate: "1993-02-17", Gender: "masculino", MaritalStatus: "solteiro", Nationality: "brasileira",
			Birthplace: "Belo Horizonte/MG", MotherName: "Regina Reis", FatherName: "Fernando Reis",
			ZipCode: "30140-071", Street: "Rua da Bahia", AddressNumber: "1200", Neighborhood: "Centro",
			City: "Belo Horizonte", State: "MG", Country: "Brasil",
			EmergencyContactName: "Regina Reis", EmergencyContactPhone: "+55 31 97777-1006",
			ContractType: "clt", WorkRegime: "remoto", WorkLocation: "Belo Horizonte/MG",
			HireDate: "2022-03-07", WeeklyHours: 40, Salary: 8900,
			BankName: "Inter", BankAgency: "0001", BankAccount: "67890-1", BankAccountType: "corrente",
			PixKey: "paulo.reis@example.com", CostHour: 85, MaxUnits: 100, Color: "gray", IsActive: true,
		},
		{
			Name: "Fernanda Alves", Email: "fernanda.alves@example.com", PersonalEmail: "fernanda.alves.pessoal@example.com",
			Phone: "+55 11 3333-1007", Mobile: "+55 11 98888-1007", Role: "Product Owner", Department: "Produto",
			ManagerName: "Ana Souza", CPF: "86288366757", RG: "78.901.234-5", RGIssuer: "SSP/SP",
			CTPS: "7234567", CTPSSeries: "007", PIS: "12034567896",
			BirthDate: "1985-09-25", Gender: "feminino", MaritalStatus: "divorciado", Nationality: "brasileira",
			Birthplace: "Santos/SP", MotherName: "Beatriz Alves", FatherName: "Ricardo Alves",
			ZipCode: "11010-000", Street: "Av. Ana Costa", AddressNumber: "450", Neighborhood: "Gonzaga",
			City: "Santos", State: "SP", Country: "Brasil",
			EmergencyContactName: "Beatriz Alves", EmergencyContactPhone: "+55 13 97777-1007",
			ContractType: "clt", WorkRegime: "hibrido", WorkLocation: "Santos/SP",
			HireDate: "2016-06-13", WeeklyHours: 40, Salary: 16500,
			BankName: "Itaú", BankAgency: "7890", BankAccount: "78901-2", BankAccountType: "corrente",
			PixKey: "fernanda.alves@example.com", CostHour: 110, MaxUnits: 100, Color: "magenta", IsActive: true,
		},
		{
			Name: "Sofia Martins", Email: "sofia.martins@example.com", PersonalEmail: "sofia.martins.pessoal@example.com",
			Phone: "+55 11 3333-1008", Mobile: "+55 11 98888-1008", Role: "Analista de qualidade", Department: "Qualidade",
			ManagerName: "Fernanda Alves", CPF: "15350946056", RG: "89.012.345-6", RGIssuer: "SSP/PR",
			CTPS: "8234567", CTPSSeries: "008", PIS: "12034567897",
			BirthDate: "1996-01-08", Gender: "feminino", MaritalStatus: "solteiro", Nationality: "brasileira",
			Birthplace: "Curitiba/PR", MotherName: "Cláudia Martins", FatherName: "Sérgio Martins",
			ZipCode: "80010-000", Street: "Rua XV de Novembro", AddressNumber: "700", Neighborhood: "Centro",
			City: "Curitiba", State: "PR", Country: "Brasil",
			EmergencyContactName: "Cláudia Martins", EmergencyContactPhone: "+55 41 97777-1008",
			ContractType: "clt", WorkRegime: "remoto", WorkLocation: "Curitiba/PR",
			HireDate: "2022-05-02", TerminationDate: "2026-08-31", WeeklyHours: 30, Salary: 7200,
			BankName: "Banco do Brasil", BankAgency: "8901", BankAccount: "89012-3", BankAccountType: "corrente",
			PixKey: "sofia.martins@example.com", CostHour: 80, MaxUnits: 50, Color: "cyan", IsActive: false,
			Notes: "Desligada em 31/08/2026. Histórico preservado.",
		},
	}
	ids := make(map[string]int64, len(seeds))
	for _, input := range seeds {
		if existing, err := m.FindCollaboratorByEmail(input.Email); err == nil {
			if existing.CPF == nil || *existing.CPF == "" {
				if updated, updateErr := m.UpdateCollaborator(existing.ID, input); updateErr == nil {
					ids[updated.Name] = updated.ID
					continue
				}
			}
			ids[existing.Name] = existing.ID
			continue
		}
		created, err := m.CreateCollaborator(input)
		if err != nil {
			return nil, err
		}
		ids[created.Name] = created.ID
	}
	return ids, nil
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

const (
	seedProjectExpansionName = "Implantação de rede 5G - Lote SP"
	seedProjectBackboneName  = "Expansão do backbone metropolitano"
)

type projectSeedTask struct {
	Key        string
	Parent     string
	Name       string
	Notes      string
	Type       string
	Start      string
	Duration   int
	Progress   int
	Priority   string
	Effort     float64
	FixedCost  float64
	AssigneeID int64
	Stage      string
}

type projectSeedDependency struct {
	From string
	To   string
	Type string
	Lag  int
}

type projectSeedAssignment struct {
	Task     string
	Resource string
	Units    int
	Hours    float64
}

type projectSeedHoliday struct {
	Date string
	Name string
}

type projectSeed struct {
	Name         string
	Description  string
	Status       string
	StartDate    string
	EndDate      string
	Currency     string
	Budget       float64
	WorkingDays  string
	HoursPerDay  float64
	FlowMode     string
	Tasks        []projectSeedTask
	Dependencies []projectSeedDependency
	Resources    []modules.ProjectResourceInput
	Assignments  []projectSeedAssignment
	Holidays     []projectSeedHoliday
	Stages       []modules.ProjectPipelineStageInput
	Baseline     string
}

func seedProjects(m *modules.Store, admin models.User, collaborators map[string]int64) error {
	existing, err := m.ListProjects(true)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(existing))
	for _, project := range existing {
		known[project.Name] = true
	}

	today := time.Now()
	expansionStart := today.AddDate(0, 0, -7)
	expansionDate := expansionStart.Format("2006-01-02")
	if !known[seedProjectExpansionName] {
		if err := createSeedProject(m, admin, collaborators, projectSeed{
			Name:        seedProjectExpansionName,
			Description: "Implantação e integração de estações 5G no lote SP: engenharia, documentação, aceite do cliente e acompanhamento de custos.",
			Status:      "em_andamento",
			StartDate:   expansionDate,
			EndDate:     expansionStart.AddDate(0, 0, 60).Format("2006-01-02"),
			Currency:    "BRL",
			Budget:      180000,
			WorkingDays: "1,2,3,4,5",
			HoursPerDay: 8,
			FlowMode:    "pipeline",
			Tasks: []projectSeedTask{
				{Key: "planning", Name: "Planejamento", Type: "summary"},
				{Key: "kickoff", Parent: "planning", Name: "Kickoff e alinhamento com o cliente", Notes: "Reunião inicial para alinhar escopo, prazos e responsáveis.", Type: "task", Start: expansionDate, Duration: 2, Progress: 100, Priority: "alta", Effort: 16, AssigneeID: admin.ID, Stage: "Concluído"},
				{Key: "requirements", Parent: "planning", Name: "Levantamento de requisitos", Notes: "Consolidar requisitos técnicos e documentais das estações.", Type: "task", Start: expansionDate, Duration: 3, Progress: 100, Priority: "alta", Effort: 24, AssigneeID: admin.ID, Stage: "Concluído"},
				{Key: "engineering", Name: "Engenharia e documentação", Type: "summary"},
				{Key: "design", Parent: "engineering", Name: "Projeto executivo das estações", Notes: "Plantas, memorial descritivo e lista de materiais.", Type: "task", Start: expansionDate, Duration: 5, Progress: 60, Priority: "alta", Effort: 40, Stage: "Em execução"},
				{Key: "approval", Parent: "engineering", Name: "Aprovação do cliente", Type: "milestone", Start: expansionDate, Priority: "critica", Stage: "Validação"},
				{Key: "rollout", Name: "Implantação", Type: "summary"},
				{Key: "mobilize", Parent: "rollout", Name: "Mobilização de equipes", Notes: "Liberar acesso aos sites, ferramentas e EPIs.", Type: "task", Start: expansionDate, Duration: 2, Priority: "media", Effort: 16, FixedCost: 2000, Stage: "Backlog"},
				{Key: "install", Parent: "rollout", Name: "Instalação de equipamentos", Notes: "Instalar RRUs, cabeamento e validar energia nos sites do lote.", Type: "task", Start: expansionDate, Duration: 10, Priority: "alta", Effort: 80, FixedCost: 12000, Stage: "Backlog"},
				{Key: "integrate", Parent: "rollout", Name: "Integração e testes", Notes: "Integração com a rede core e testes de aceitação.", Type: "task", Start: expansionDate, Duration: 4, Priority: "alta", Effort: 32, FixedCost: 3500, Stage: "Backlog"},
				{Key: "acceptance", Parent: "rollout", Name: "Aceite final", Type: "milestone", Start: expansionDate, Priority: "critica", Stage: "Backlog"},
			},
			Dependencies: []projectSeedDependency{
				{From: "kickoff", To: "requirements", Type: "FS"},
				{From: "requirements", To: "design", Type: "FS"},
				{From: "design", To: "approval", Type: "FS"},
				{From: "approval", To: "mobilize", Type: "FS"},
				{From: "mobilize", To: "install", Type: "FS"},
				{From: "install", To: "integrate", Type: "FS"},
				{From: "integrate", To: "acceptance", Type: "FS"},
			},
			Resources: []modules.ProjectResourceInput{
				{Name: "Ana Souza", Role: "Gerente de projetos", Email: "ana.souza@example.com", CostHour: 120, MaxUnits: 100, Color: "blue"},
				{Name: "Bruno Lima", Role: "Engenheiro de campo", Email: "bruno.lima@example.com", CostHour: 90, MaxUnits: 100, Color: "green"},
				{Name: "Carla Mendes", Role: "Analista de documentação", Email: "carla.mendes@example.com", CostHour: 75, MaxUnits: 50, Color: "purple"},
				{Name: "Diego Rocha", Role: "Técnico de instalação", Email: "diego.rocha@example.com", CostHour: 60, MaxUnits: 100, Color: "orange"},
			},
			Assignments: []projectSeedAssignment{
				{Task: "kickoff", Resource: "Ana Souza", Units: 100, Hours: 16},
				{Task: "requirements", Resource: "Carla Mendes", Units: 50, Hours: 24},
				{Task: "design", Resource: "Bruno Lima", Units: 100, Hours: 40},
				{Task: "approval", Resource: "Ana Souza", Units: 50, Hours: 8},
				{Task: "mobilize", Resource: "Diego Rocha", Units: 100, Hours: 16},
				{Task: "install", Resource: "Diego Rocha", Units: 100, Hours: 80},
				{Task: "install", Resource: "Bruno Lima", Units: 50, Hours: 40},
				{Task: "integrate", Resource: "Bruno Lima", Units: 100, Hours: 32},
			},
			Holidays: []projectSeedHoliday{
				{Date: fmt.Sprintf("%d-11-20", expansionStart.Year()), Name: "Dia da Consciência Negra"},
				{Date: fmt.Sprintf("%d-12-25", expansionStart.Year()), Name: "Natal"},
			},
			Stages: []modules.ProjectPipelineStageInput{
				{Name: "Backlog", Color: "blue"},
				{Name: "Em execução", Color: "purple"},
				{Name: "Validação", Color: "orange"},
				{Name: "Concluído", Color: "green", IsDone: true},
			},
			Baseline: "Linha de base inicial",
		}); err != nil {
			return err
		}
	}

	backboneStart := today.AddDate(0, 0, 14)
	backboneDate := backboneStart.Format("2006-01-02")
	if !known[seedProjectBackboneName] {
		if err := createSeedProject(m, admin, collaborators, projectSeed{
			Name:        seedProjectBackboneName,
			Description: "Estudo e planejamento da expansão do backbone metropolitano, com análise de capacidade e aprovação de orçamento.",
			Status:      "planejamento",
			StartDate:   backboneDate,
			EndDate:     backboneStart.AddDate(0, 0, 75).Format("2006-01-02"),
			Currency:    "BRL",
			Budget:      95000,
			WorkingDays: "1,2,3,4,5",
			HoursPerDay: 8,
			Tasks: []projectSeedTask{
				{Key: "discovery", Name: "Descoberta", Type: "summary"},
				{Key: "mapping", Parent: "discovery", Name: "Mapeamento da rede atual", Notes: "Inventário de enlaces, capacidade e pontos de falha.", Type: "task", Start: backboneDate, Duration: 4, Progress: 25, Priority: "alta", Effort: 32},
				{Key: "workshop", Parent: "discovery", Name: "Workshop de requisitos", Notes: "Validar requisitos com operações e engenharia.", Type: "task", Start: backboneDate, Duration: 2, Priority: "alta", Effort: 16},
				{Key: "planning", Name: "Planejamento", Type: "summary"},
				{Key: "capacity", Parent: "planning", Name: "Estudo de capacidade", Notes: "Simular demanda e definir upgrades necessários.", Type: "task", Start: backboneDate, Duration: 5, Priority: "media", Effort: 40},
				{Key: "budget", Parent: "planning", Name: "Aprovação do orçamento", Type: "milestone", Start: backboneDate, Priority: "critica"},
			},
			Dependencies: []projectSeedDependency{
				{From: "mapping", To: "workshop", Type: "FS"},
				{From: "workshop", To: "capacity", Type: "FS"},
				{From: "capacity", To: "budget", Type: "FS"},
			},
			Resources: []modules.ProjectResourceInput{
				{Name: "Marina Costa", Role: "Arquiteta de redes", Email: "marina.costa@example.com", CostHour: 130, MaxUnits: 100, Color: "teal"},
				{Name: "Paulo Reis", Role: "Analista de infraestrutura", Email: "paulo.reis@example.com", CostHour: 85, MaxUnits: 100, Color: "gray"},
			},
			Assignments: []projectSeedAssignment{
				{Task: "mapping", Resource: "Paulo Reis", Units: 100, Hours: 24},
				{Task: "workshop", Resource: "Marina Costa", Units: 50, Hours: 12},
				{Task: "capacity", Resource: "Marina Costa", Units: 100, Hours: 32},
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func createSeedProject(m *modules.Store, admin models.User, collaborators map[string]int64, data projectSeed) error {
	for index := range data.Resources {
		if collaboratorID, ok := collaborators[data.Resources[index].Name]; ok {
			data.Resources[index].CollaboratorID = collaboratorID
		}
	}
	projectInput := modules.ProjectInput{
		Name:        data.Name,
		Description: data.Description,
		Status:      data.Status,
		StartDate:   data.StartDate,
		EndDate:     data.EndDate,
		Currency:    data.Currency,
		Budget:      data.Budget,
		WorkingDays: data.WorkingDays,
		HoursPerDay: data.HoursPerDay,
		FlowMode:    data.FlowMode,
	}
	// Create as WBS first so custom pipeline stages can be seeded before the
	// default stages are generated. The flow mode is applied after stages exist.
	createInput := projectInput
	if createInput.FlowMode == "pipeline" {
		createInput.FlowMode = "wbs"
	}
	project, err := m.CreateProject(admin.ID, createInput)
	if err != nil {
		return err
	}

	stageIDs := make(map[string]int64, len(data.Stages))
	for _, item := range data.Stages {
		stage, err := m.CreatePipelineStage(project.ID, item)
		if err != nil {
			return fmt.Errorf("projeto %q, etapa %q: %w", data.Name, item.Name, err)
		}
		stageIDs[stage.Name] = stage.ID
	}
	if projectInput.FlowMode == "pipeline" {
		if _, err := m.UpdateProject(project.ID, projectInput); err != nil {
			return fmt.Errorf("projeto %q, modo pipeline: %w", data.Name, err)
		}
	}

	taskIDs := make(map[string]int64, len(data.Tasks))
	for _, item := range data.Tasks {
		task, err := m.CreateProjectTask(project.ID, modules.ProjectTaskInput{
			ParentID:        taskIDs[item.Parent],
			Name:            item.Name,
			Notes:           item.Notes,
			TaskType:        item.Type,
			StartDate:       item.Start,
			DurationDays:    item.Duration,
			Progress:        item.Progress,
			Priority:        item.Priority,
			EffortHours:     item.Effort,
			FixedCost:       item.FixedCost,
			AssigneeID:      item.AssigneeID,
			PipelineStageID: stageIDs[item.Stage],
		})
		if err != nil {
			return fmt.Errorf("projeto %q, tarefa %q: %w", data.Name, item.Name, err)
		}
		taskIDs[item.Key] = task.ID
	}

	resourceIDs := make(map[string]int64, len(data.Resources))
	for _, item := range data.Resources {
		resource, err := m.CreateProjectResource(project.ID, item)
		if err != nil {
			return fmt.Errorf("projeto %q, recurso %q: %w", data.Name, item.Name, err)
		}
		resourceIDs[resource.Name] = resource.ID
	}

	for _, item := range data.Assignments {
		if _, err := m.CreateProjectAssignment(project.ID, modules.ProjectAssignmentInput{
			TaskID:       taskIDs[item.Task],
			ResourceID:   resourceIDs[item.Resource],
			Units:        item.Units,
			PlannedHours: item.Hours,
		}); err != nil {
			return fmt.Errorf("projeto %q, alocação %q/%q: %w", data.Name, item.Resource, item.Task, err)
		}
	}

	for _, item := range data.Holidays {
		if _, err := m.CreateProjectHoliday(project.ID, item.Date, item.Name); err != nil {
			return fmt.Errorf("projeto %q, feriado %q: %w", data.Name, item.Name, err)
		}
	}

	for _, item := range data.Dependencies {
		if _, err := m.CreateProjectDependency(project.ID, taskIDs[item.From], taskIDs[item.To], item.Type, item.Lag); err != nil {
			return fmt.Errorf("projeto %q, dependência %q -> %q: %w", data.Name, item.From, item.To, err)
		}
	}

	if data.Baseline != "" {
		if _, err := m.CreateProjectBaseline(project.ID, data.Baseline); err != nil {
			return fmt.Errorf("projeto %q, linha de base: %w", data.Name, err)
		}
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
