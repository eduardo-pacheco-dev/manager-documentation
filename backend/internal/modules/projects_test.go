package modules

import "testing"

func testProject(start string) Project {
	return Project{ID: 1, Name: "Teste", StartDate: start, WorkingDays: "1,2,3,4,5", HoursPerDay: 8}
}

func testTask(id int64, name string, duration int, start string) ProjectTask {
	task := ProjectTask{ID: id, ProjectID: 1, Name: name, TaskType: "task", DurationDays: duration}
	if start != "" {
		value := start
		task.StartDate = &value
	}
	return task
}

func scheduleByID(scheduled []scheduledTask) map[int64]scheduledTask {
	result := map[int64]scheduledTask{}
	for _, item := range scheduled {
		result[item.ID] = item
	}
	return result
}

func TestProjectScheduleSkipsWeekendsAndHolidays(t *testing.T) {
	tasks := []ProjectTask{testTask(1, "Cinco dias", 5, "2026-01-09")} // sexta-feira
	scheduled, err := scheduleProject(testProject("2026-01-09"), tasks, nil, nil)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	item := scheduleByID(scheduled)[1]
	if got := formatDate(item.End); got != "2026-01-15" {
		t.Fatalf("fim sem feriado = %s, esperado 2026-01-15", got)
	}

	holidays := []ProjectHoliday{{ProjectID: 1, Date: "2026-01-12", Name: "Feriado"}}
	scheduled, err = scheduleProject(testProject("2026-01-09"), tasks, nil, holidays)
	if err != nil {
		t.Fatalf("schedule com feriado: %v", err)
	}
	item = scheduleByID(scheduled)[1]
	if got := formatDate(item.End); got != "2026-01-16" {
		t.Fatalf("fim com feriado = %s, esperado 2026-01-16", got)
	}
}

func TestProjectDependenciesAndCriticalPath(t *testing.T) {
	tasks := []ProjectTask{
		testTask(1, "A", 2, "2026-01-05"),
		testTask(2, "B", 2, ""),
		testTask(3, "Independente", 1, "2026-01-05"),
	}
	dependencies := []ProjectDependency{{ProjectID: 1, PredecessorID: 1, SuccessorID: 2, Type: "FS"}}
	scheduled, err := scheduleProject(testProject("2026-01-05"), tasks, dependencies, nil)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	byID := scheduleByID(scheduled)
	if got := formatDate(byID[2].Start); got != "2026-01-07" {
		t.Fatalf("início de B = %s, esperado 2026-01-07", got)
	}
	if !byID[1].Critical || !byID[2].Critical {
		t.Fatal("A e B devem estar no caminho crítico")
	}
	if byID[3].Critical {
		t.Fatal("tarefa independente não deve ser crítica")
	}
}

func TestProjectStartToStartAndLag(t *testing.T) {
	tasks := []ProjectTask{
		testTask(1, "A", 3, "2026-01-05"),
		testTask(2, "B", 2, ""),
	}
	dependencies := []ProjectDependency{{ProjectID: 1, PredecessorID: 1, SuccessorID: 2, Type: "SS", LagDays: 2}}
	scheduled, err := scheduleProject(testProject("2026-01-05"), tasks, dependencies, nil)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	byID := scheduleByID(scheduled)
	if got := formatDate(byID[2].Start); got != "2026-01-07" {
		t.Fatalf("SS com lag 2 iniciou em %s, esperado 2026-01-07", got)
	}
}

func TestProjectSummaryRollup(t *testing.T) {
	parentID := int64(10)
	tasks := []ProjectTask{
		{ID: 10, ProjectID: 1, Name: "Resumo", TaskType: "summary", DurationDays: 0, Progress: 0},
		{ID: 1, ProjectID: 1, ParentID: &parentID, Name: "A", TaskType: "task", DurationDays: 2, Progress: 100,
			StartDate: strPointer("2026-01-05")},
		{ID: 2, ProjectID: 1, ParentID: &parentID, Name: "B", TaskType: "task", DurationDays: 3, Progress: 0,
			StartDate: strPointer("2026-01-07")},
	}
	scheduled, err := scheduleProject(testProject("2026-01-05"), tasks, nil, nil)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	byID := scheduleByID(scheduled)
	summary := byID[10]
	if got := formatDate(summary.Start); got != "2026-01-05" {
		t.Fatalf("início do resumo = %s", got)
	}
	if got := formatDate(summary.End); got != "2026-01-09" {
		t.Fatalf("fim do resumo = %s, esperado 2026-01-09", got)
	}
	if summary.Duration != 5 {
		t.Fatalf("duração do resumo = %d, esperado 5", summary.Duration)
	}
	if !summary.Critical {
		t.Fatal("resumo com tarefa crítica deve ser crítico")
	}
}

func TestProjectRejectsCircularDependencies(t *testing.T) {
	dependencies := []ProjectDependency{{PredecessorID: 1, SuccessorID: 2}}
	if wouldCreateProjectCycle(dependencies, 2, 1) {
		return
	}
	t.Fatal("esperava detectar o ciclo 1 -> 2 -> 1")
}

func strPointer(value string) *string {
	return &value
}

func TestProjectMetricsDoNotDoubleCountSummaryCosts(t *testing.T) {
	tasks := []ProjectTask{
		{ID: 1, TaskType: "task", TotalCost: 100},
		{ID: 2, TaskType: "summary", TotalCost: 100},
	}
	metrics := projectMetrics(tasks, nil, Project{})
	if metrics.PlannedCost != 100 {
		t.Fatalf("custo planejado = %v, esperado 100", metrics.PlannedCost)
	}
}
