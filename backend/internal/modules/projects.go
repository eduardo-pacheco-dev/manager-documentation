package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

/* --------------------------------------------------------------------------
 * Projects module
 *
 * A Microsoft Project style planner: WBS tasks and summaries, milestones,
 * finish-to-start / start-to-start / finish-to-finish / start-to-finish
 * dependencies with lag, working-day calendars with holidays, resources and
 * assignments, cost roll-ups, baselines and critical path.
 * ----------------------------------------------------------------------- */

type Project struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	OwnerID     int64     `db:"owner_id" json:"owner_id"`
	OwnerName   string    `db:"owner_name" json:"owner_name"`
	Status      string    `db:"status" json:"status"`
	StartDate   string    `db:"start_date" json:"start_date"`
	EndDate     string    `db:"end_date" json:"end_date"`
	Currency    string    `db:"currency" json:"currency"`
	Budget      float64   `db:"budget" json:"budget"`
	WorkingDays string    `db:"working_days" json:"working_days"`
	HoursPerDay float64   `db:"hours_per_day" json:"hours_per_day"`
	Archived    bool      `db:"archived" json:"archived"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	TaskCount   int       `db:"task_count" json:"task_count"`
	DoneCount   int       `db:"done_count" json:"done_count"`
	Progress    int       `db:"progress" json:"progress"`
}

type ProjectTask struct {
	ID            int64     `db:"id" json:"id"`
	ProjectID     int64     `db:"project_id" json:"project_id"`
	ParentID      *int64    `db:"parent_id" json:"parent_id"`
	WBS           string    `db:"wbs" json:"wbs"`
	Name          string    `db:"name" json:"name"`
	Notes         *string   `db:"notes" json:"notes"`
	TaskType      string    `db:"task_type" json:"task_type"`
	StartDate     *string   `db:"start_date" json:"start_date"`
	EndDate       *string   `db:"end_date" json:"end_date"`
	DurationDays  int       `db:"duration_days" json:"duration_days"`
	Progress      int       `db:"progress" json:"progress"`
	Priority      string    `db:"priority" json:"priority"`
	EffortHours   float64   `db:"effort_hours" json:"effort_hours"`
	FixedCost     float64   `db:"fixed_cost" json:"fixed_cost"`
	AssigneeID    *int64    `db:"assignee_id" json:"assignee_id"`
	AssigneeName  string    `db:"assignee_name" json:"assignee_name"`
	Position      int       `db:"position" json:"position"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	Level         int       `json:"level"`
	ScheduledFrom string    `json:"scheduled_start"`
	ScheduledTo   string    `json:"scheduled_end"`
	EffectiveDays int       `json:"effective_duration"`
	IsCritical    bool      `json:"is_critical"`
	AssignedCost  float64   `json:"assigned_cost"`
	TotalCost     float64   `json:"total_cost"`
	Resources     []string  `json:"resources"`
	BaselineStart *string   `json:"baseline_start,omitempty"`
	BaselineEnd   *string   `json:"baseline_end,omitempty"`
	StartVariance int       `json:"start_variance_days"`
	EndVariance   int       `json:"end_variance_days"`
	CostVariance  float64   `json:"cost_variance"`
}

type ProjectDependency struct {
	ID            int64     `db:"id" json:"id"`
	ProjectID     int64     `db:"project_id" json:"project_id"`
	PredecessorID int64     `db:"predecessor_id" json:"predecessor_id"`
	SuccessorID   int64     `db:"successor_id" json:"successor_id"`
	Type          string    `db:"dep_type" json:"type"`
	LagDays       int       `db:"lag_days" json:"lag_days"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type ProjectResource struct {
	ID        int64     `db:"id" json:"id"`
	ProjectID int64     `db:"project_id" json:"project_id"`
	Name      string    `db:"name" json:"name"`
	Role      *string   `db:"role" json:"role"`
	Email     *string   `db:"email" json:"email"`
	CostHour  float64   `db:"cost_hour" json:"cost_hour"`
	MaxUnits  int       `db:"max_units" json:"max_units"`
	Color     string    `db:"color" json:"color"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ProjectAssignment struct {
	ID           int64     `db:"id" json:"id"`
	ProjectID    int64     `db:"project_id" json:"project_id"`
	TaskID       int64     `db:"task_id" json:"task_id"`
	ResourceID   int64     `db:"resource_id" json:"resource_id"`
	Units        int       `db:"units" json:"units"`
	PlannedHours float64   `db:"planned_hours" json:"planned_hours"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	ResourceName string    `db:"resource_name" json:"resource_name"`
	CostHour     float64   `db:"cost_hour" json:"cost_hour"`
	Cost         float64   `json:"cost"`
}

type ProjectHoliday struct {
	ID        int64     `db:"id" json:"id"`
	ProjectID int64     `db:"project_id" json:"project_id"`
	Date      string    `db:"holiday_date" json:"date"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ProjectBaseline struct {
	ID        int64     `db:"id" json:"id"`
	ProjectID int64     `db:"project_id" json:"project_id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	TaskCount int       `db:"task_count" json:"task_count"`
}

type ProjectMetrics struct {
	TaskCount     int     `json:"task_count"`
	Completed     int     `json:"completed"`
	InProgress    int     `json:"in_progress"`
	Overdue       int     `json:"overdue"`
	Milestones    int     `json:"milestones"`
	Critical      int     `json:"critical"`
	Progress      int     `json:"progress"`
	PlannedCost   float64 `json:"planned_cost"`
	Budget        float64 `json:"budget"`
	CostVariance  float64 `json:"cost_variance"`
	ScheduledFrom string  `json:"scheduled_start"`
	ScheduledTo   string  `json:"scheduled_end"`
	DurationDays  int     `json:"duration_days"`
	DaysRemaining int     `json:"days_remaining"`
}

type ProjectBundle struct {
	Project      Project             `json:"project"`
	Tasks        []ProjectTask       `json:"tasks"`
	Dependencies []ProjectDependency `json:"dependencies"`
	Resources    []ProjectResource   `json:"resources"`
	Assignments  []ProjectAssignment `json:"assignments"`
	Holidays     []ProjectHoliday    `json:"holidays"`
	Baselines    []ProjectBaseline   `json:"baselines"`
	Metrics      ProjectMetrics      `json:"metrics"`
}

type ProjectInput struct {
	Name        string
	Description string
	Status      string
	StartDate   string
	EndDate     string
	Currency    string
	Budget      float64
	WorkingDays string
	HoursPerDay float64
}

type ProjectTaskInput struct {
	ParentID     int64
	Name         string
	Notes        string
	TaskType     string
	StartDate    string
	EndDate      string
	DurationDays int
	Progress     int
	Priority     string
	EffortHours  float64
	FixedCost    float64
	AssigneeID   int64
	Position     int
}

type ProjectResourceInput struct {
	Name     string
	Role     string
	Email    string
	CostHour float64
	MaxUnits int
	Color    string
}

type ProjectAssignmentInput struct {
	TaskID       int64
	ResourceID   int64
	Units        int
	PlannedHours float64
}

var projectStatuses = map[string]bool{
	"planejamento": true, "em_andamento": true, "pausado": true, "concluido": true, "cancelado": true,
}

var projectPriorities = map[string]bool{"baixa": true, "media": true, "alta": true, "critica": true}

var dependencyTypes = map[string]bool{"FS": true, "SS": true, "FF": true, "SF": true}

/* --------------------------------------------------------------------------
 * Projects CRUD
 * ----------------------------------------------------------------------- */

const projectSelect = `
	SELECT p.*, COALESCE(u.name, '') AS owner_name,
		(SELECT COUNT(*) FROM project_tasks t WHERE t.project_id = p.id AND t.task_type <> 'summary') AS task_count,
		(SELECT COUNT(*) FROM project_tasks t WHERE t.project_id = p.id AND t.task_type <> 'summary' AND t.progress >= 100) AS done_count,
		COALESCE((SELECT ROUND(AVG(t.progress)) FROM project_tasks t
			WHERE t.project_id = p.id AND t.task_type <> 'summary'), 0) AS progress
	FROM projects p
	LEFT JOIN users u ON u.id = p.owner_id`

func (s *Store) ListProjects(includeArchived bool) ([]Project, error) {
	condition := "p.archived = 0"
	if includeArchived {
		condition = "1 = 1"
	}
	projects := []Project{}
	err := s.DB.Select(&projects, projectSelect+`
		WHERE `+condition+`
		ORDER BY p.archived ASC, p.updated_at DESC, p.id DESC`)
	return projects, err
}

func (s *Store) FindProject(id int64) (Project, error) {
	var project Project
	err := s.DB.Get(&project, projectSelect+` WHERE p.id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return project, ErrNotFound
	}
	return project, err
}

func (s *Store) CreateProject(ownerID int64, input ProjectInput) (Project, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Project{}, fmt.Errorf("%w: nome é obrigatório", ErrValidation)
	}
	start := normalizeDate(input.StartDate)
	if start == "" {
		start = time.Now().Format("2006-01-02")
	}
	end := normalizeDate(input.EndDate)
	if end == "" {
		end = start
	}
	if end < start {
		return Project{}, fmt.Errorf("%w: a data final deve ser posterior à inicial", ErrValidation)
	}
	status := input.Status
	if !projectStatuses[status] {
		status = "planejamento"
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = "BRL"
	}
	workingDays := normalizeWorkingDays(input.WorkingDays)
	hours := input.HoursPerDay
	if hours <= 0 || hours > 24 {
		hours = 8
	}
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO projects (name, description, owner_id, status, start_date, end_date, currency,
			budget, working_days, hours_per_day, archived, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		name, nullString(input.Description), ownerID, status, start, end, currency,
		input.Budget, workingDays, hours, timestamp, timestamp)
	if err != nil {
		return Project{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindProject(id)
}

func (s *Store) UpdateProject(id int64, input ProjectInput) (Project, error) {
	project, err := s.FindProject(id)
	if err != nil {
		return project, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return project, fmt.Errorf("%w: nome é obrigatório", ErrValidation)
	}
	start := normalizeDate(input.StartDate)
	if start == "" {
		start = project.StartDate
	}
	end := normalizeDate(input.EndDate)
	if end == "" {
		end = project.EndDate
	}
	if end < start {
		return project, fmt.Errorf("%w: a data final deve ser posterior à inicial", ErrValidation)
	}
	status := input.Status
	if !projectStatuses[status] {
		status = project.Status
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = project.Currency
	}
	workingDays := normalizeWorkingDays(input.WorkingDays)
	hours := input.HoursPerDay
	if hours <= 0 || hours > 24 {
		hours = project.HoursPerDay
	}
	_, err = s.DB.Exec(`
		UPDATE projects SET name = ?, description = ?, status = ?, start_date = ?, end_date = ?,
			currency = ?, budget = ?, working_days = ?, hours_per_day = ?, updated_at = ?
		WHERE id = ?`,
		name, nullString(input.Description), status, start, end, currency, input.Budget, workingDays, hours, now(), id)
	if err != nil {
		return project, err
	}
	return s.FindProject(id)
}

func (s *Store) DeleteProject(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (s *Store) ToggleProjectArchived(id int64) (Project, error) {
	project, err := s.FindProject(id)
	if err != nil {
		return project, err
	}
	_, err = s.DB.Exec(`UPDATE projects SET archived = ?, updated_at = ? WHERE id = ?`, !project.Archived, now(), id)
	if err != nil {
		return project, err
	}
	return s.FindProject(id)
}

/* --------------------------------------------------------------------------
 * Tasks
 * ----------------------------------------------------------------------- */

func (s *Store) ProjectTasks(projectID int64) ([]ProjectTask, error) {
	tasks := []ProjectTask{}
	err := s.DB.Select(&tasks, `
		SELECT t.*, COALESCE(u.name, '') AS assignee_name
		FROM project_tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id = ?
		ORDER BY t.position ASC, t.id ASC`, projectID)
	return tasks, err
}

func (s *Store) FindProjectTask(projectID, id int64) (ProjectTask, error) {
	var task ProjectTask
	err := s.DB.Get(&task, `
		SELECT t.*, COALESCE(u.name, '') AS assignee_name
		FROM project_tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.id = ? AND t.project_id = ?`, id, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return task, ErrNotFound
	}
	return task, err
}

func (s *Store) CreateProjectTask(projectID int64, input ProjectTaskInput) (ProjectTask, error) {
	project, err := s.FindProject(projectID)
	if err != nil {
		return ProjectTask{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return ProjectTask{}, fmt.Errorf("%w: título é obrigatório", ErrValidation)
	}
	taskType := normalizeTaskType(input.TaskType)
	if input.ParentID > 0 {
		parent, err := s.FindProjectTask(projectID, input.ParentID)
		if err != nil {
			return ProjectTask{}, fmt.Errorf("%w: tarefa pai não encontrada", ErrValidation)
		}
		if parent.TaskType != "summary" {
			if _, err := s.DB.Exec(`UPDATE project_tasks SET task_type = 'summary', duration_days = 0, updated_at = ? WHERE id = ?`, now(), parent.ID); err != nil {
				return ProjectTask{}, err
			}
		}
	}
	duration := input.DurationDays
	if taskType != "task" {
		duration = 0
	}
	if duration < 1 && taskType == "task" {
		duration = 1
	}
	progress := clampInt(input.Progress, 0, 100)
	priority := input.Priority
	if !projectPriorities[priority] {
		priority = "media"
	}
	start := normalizeDate(input.StartDate)
	end := normalizeDate(input.EndDate)
	if start == "" {
		start = project.StartDate
	}
	if taskType == "summary" {
		start, end = "", ""
	}
	position := input.Position
	if position <= 0 {
		_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM project_tasks WHERE project_id = ? AND COALESCE(parent_id, 0) = ?`,
			projectID, input.ParentID)
	}
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO project_tasks (project_id, parent_id, wbs, name, notes, task_type, start_date, end_date,
			duration_days, progress, priority, effort_hours, fixed_cost, assignee_id, position, created_at, updated_at)
		VALUES (?, ?, '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID, nullableID(input.ParentID), name, nullString(input.Notes), taskType, nullDate(start), nullDate(end),
		duration, progress, priority, input.EffortHours, input.FixedCost, nullableID(input.AssigneeID), position, timestamp, timestamp)
	if err != nil {
		return ProjectTask{}, err
	}
	id, _ := result.LastInsertId()
	s.reindexProject(projectID)
	return s.FindProjectTask(projectID, id)
}

func (s *Store) UpdateProjectTask(projectID, id int64, input ProjectTaskInput) (ProjectTask, error) {
	task, err := s.FindProjectTask(projectID, id)
	if err != nil {
		return task, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return task, fmt.Errorf("%w: título é obrigatório", ErrValidation)
	}
	taskType := normalizeTaskType(input.TaskType)
	if input.ParentID == id {
		return task, fmt.Errorf("%w: uma tarefa não pode ser pai de si mesma", ErrValidation)
	}
	if input.ParentID > 0 {
		if _, err := s.FindProjectTask(projectID, input.ParentID); err != nil {
			return task, fmt.Errorf("%w: tarefa pai não encontrada", ErrValidation)
		}
		if s.projectTaskIsDescendant(projectID, input.ParentID, id) {
			return task, fmt.Errorf("%w: hierarquia inválida", ErrValidation)
		}
		if _, err := s.DB.Exec(`UPDATE project_tasks SET task_type = 'summary', duration_days = 0, updated_at = ? WHERE id = ? AND task_type <> 'summary'`, now(), input.ParentID); err != nil {
			return task, err
		}
	}
	duration := input.DurationDays
	if taskType != "task" {
		duration = 0
	}
	if duration < 1 && taskType == "task" {
		duration = 1
	}
	progress := clampInt(input.Progress, 0, 100)
	priority := input.Priority
	if !projectPriorities[priority] {
		priority = "media"
	}
	start := normalizeDate(input.StartDate)
	end := normalizeDate(input.EndDate)
	if taskType == "summary" {
		start, end = "", ""
	}
	position := input.Position
	if position <= 0 {
		position = task.Position
	}
	_, err = s.DB.Exec(`
		UPDATE project_tasks SET parent_id = ?, name = ?, notes = ?, task_type = ?, start_date = ?, end_date = ?,
			duration_days = ?, progress = ?, priority = ?, effort_hours = ?, fixed_cost = ?, assignee_id = ?,
			position = ?, updated_at = ?
		WHERE id = ? AND project_id = ?`,
		nullableID(input.ParentID), name, nullString(input.Notes), taskType, nullDate(start), nullDate(end),
		duration, progress, priority, input.EffortHours, input.FixedCost, nullableID(input.AssigneeID),
		position, now(), id, projectID)
	if err != nil {
		return task, err
	}
	s.reindexProject(projectID)
	return s.FindProjectTask(projectID, id)
}

func (s *Store) DeleteProjectTask(projectID, id int64) error {
	task, err := s.FindProjectTask(projectID, id)
	if err != nil {
		return err
	}
	if _, err := s.DB.Exec(`DELETE FROM project_tasks WHERE id = ? AND project_id = ?`, id, projectID); err != nil {
		return err
	}
	if task.ParentID != nil {
		var siblings int
		_ = s.DB.Get(&siblings, `SELECT COUNT(*) FROM project_tasks WHERE project_id = ? AND parent_id = ?`, projectID, *task.ParentID)
		if siblings == 0 {
			_, _ = s.DB.Exec(`UPDATE project_tasks SET task_type = 'task', duration_days = 1, updated_at = ? WHERE id = ?`, now(), *task.ParentID)
		}
	}
	s.reindexProject(projectID)
	return nil
}

func (s *Store) MoveProjectTask(projectID, id int64, direction string) error {
	tasks, err := s.ProjectTasks(projectID)
	if err != nil {
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
		_, _ = s.DB.Exec(`UPDATE project_tasks SET position = ?, updated_at = ? WHERE id = ?`, i+1, now(), task.ID)
	}
	return nil
}

func (s *Store) projectTaskIsDescendant(projectID, candidateParentID, taskID int64) bool {
	current := candidateParentID
	for depth := 0; depth < 100 && current > 0; depth++ {
		task, err := s.FindProjectTask(projectID, current)
		if err != nil || task.ParentID == nil {
			return false
		}
		if *task.ParentID == taskID {
			return true
		}
		current = *task.ParentID
	}
	return false
}

// reindexProject rebuilds WBS codes and rolls summary progress up.
func (s *Store) reindexProject(projectID int64) {
	tasks, err := s.ProjectTasks(projectID)
	if err != nil {
		return
	}
	byParent := map[int64][]ProjectTask{}
	for _, task := range tasks {
		parent := int64(0)
		if task.ParentID != nil {
			parent = *task.ParentID
		}
		byParent[parent] = append(byParent[parent], task)
	}
	var walk func(parent int64, prefix string)
	walk = func(parent int64, prefix string) {
		children := byParent[parent]
		for index, task := range children {
			code := strconv.Itoa(index + 1)
			if prefix != "" {
				code = prefix + "." + code
			}
			_, _ = s.DB.Exec(`UPDATE project_tasks SET wbs = ? WHERE id = ?`, code, task.ID)
			walk(task.ID, code)
		}
	}
	walk(0, "")

	var rollup func(taskID int64) (int, bool)
	rollup = func(taskID int64) (int, bool) {
		children := byParent[taskID]
		if len(children) == 0 {
			return 0, false
		}
		weighted, weight := 0, 0
		for _, child := range children {
			duration := child.DurationDays
			if duration < 1 {
				duration = 1
			}
			progress := child.Progress
			if child.TaskType == "summary" {
				if value, ok := rollup(child.ID); ok {
					progress = value
				}
			}
			weighted += progress * duration
			weight += duration
		}
		if weight == 0 {
			return 0, false
		}
		return weighted / weight, true
	}
	for _, task := range tasks {
		if task.TaskType != "summary" {
			continue
		}
		if progress, ok := rollup(task.ID); ok {
			_, _ = s.DB.Exec(`UPDATE project_tasks SET progress = ?, updated_at = ? WHERE id = ?`, progress, now(), task.ID)
		}
	}
}

/* --------------------------------------------------------------------------
 * Dependencies
 * ----------------------------------------------------------------------- */

func (s *Store) ProjectDependencies(projectID int64) ([]ProjectDependency, error) {
	dependencies := []ProjectDependency{}
	err := s.DB.Select(&dependencies, `SELECT * FROM project_dependencies WHERE project_id = ? ORDER BY id ASC`, projectID)
	return dependencies, err
}

func (s *Store) CreateProjectDependency(projectID, predecessorID, successorID int64, depType string, lag int) (ProjectDependency, error) {
	if predecessorID == successorID {
		return ProjectDependency{}, fmt.Errorf("%w: uma tarefa não pode depender de si mesma", ErrValidation)
	}
	depType = strings.ToUpper(strings.TrimSpace(depType))
	if !dependencyTypes[depType] {
		depType = "FS"
	}
	if _, err := s.FindProjectTask(projectID, predecessorID); err != nil {
		return ProjectDependency{}, fmt.Errorf("%w: tarefa predecessora não encontrada", ErrValidation)
	}
	if _, err := s.FindProjectTask(projectID, successorID); err != nil {
		return ProjectDependency{}, fmt.Errorf("%w: tarefa sucessora não encontrada", ErrValidation)
	}
	dependencies, err := s.ProjectDependencies(projectID)
	if err != nil {
		return ProjectDependency{}, err
	}
	if wouldCreateProjectCycle(dependencies, predecessorID, successorID) {
		return ProjectDependency{}, fmt.Errorf("%w: a dependência criaria um ciclo", ErrValidation)
	}
	result, err := s.DB.Exec(`
		INSERT INTO project_dependencies (project_id, predecessor_id, successor_id, dep_type, lag_days, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, projectID, predecessorID, successorID, depType, lag, now())
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return ProjectDependency{}, fmt.Errorf("%w: esta dependência já existe", ErrValidation)
		}
		return ProjectDependency{}, err
	}
	id, _ := result.LastInsertId()
	var dependency ProjectDependency
	err = s.DB.Get(&dependency, `SELECT * FROM project_dependencies WHERE id = ?`, id)
	return dependency, err
}

func (s *Store) DeleteProjectDependency(projectID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM project_dependencies WHERE id = ? AND project_id = ?`, id, projectID)
	return err
}

func wouldCreateProjectCycle(dependencies []ProjectDependency, predecessorID, successorID int64) bool {
	graph := map[int64][]int64{}
	for _, dependency := range dependencies {
		graph[dependency.PredecessorID] = append(graph[dependency.PredecessorID], dependency.SuccessorID)
	}
	graph[predecessorID] = append(graph[predecessorID], successorID)
	visited := map[int64]bool{}
	var reaches func(from, target int64) bool
	reaches = func(from, target int64) bool {
		if from == target {
			return true
		}
		if visited[from] {
			return false
		}
		visited[from] = true
		for _, next := range graph[from] {
			if reaches(next, target) {
				return true
			}
		}
		return false
	}
	return reaches(successorID, predecessorID)
}

/* --------------------------------------------------------------------------
 * Resources, assignments, holidays and baselines
 * ----------------------------------------------------------------------- */

func (s *Store) ProjectResources(projectID int64) ([]ProjectResource, error) {
	resources := []ProjectResource{}
	err := s.DB.Select(&resources, `SELECT * FROM project_resources WHERE project_id = ? ORDER BY name ASC`, projectID)
	return resources, err
}

func (s *Store) CreateProjectResource(projectID int64, input ProjectResourceInput) (ProjectResource, error) {
	if strings.TrimSpace(input.Name) == "" {
		return ProjectResource{}, fmt.Errorf("%w: nome do recurso é obrigatório", ErrValidation)
	}
	if input.MaxUnits <= 0 {
		input.MaxUnits = 100
	}
	if input.Color == "" {
		input.Color = "blue"
	}
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO project_resources (project_id, name, role, email, cost_hour, max_units, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		projectID, strings.TrimSpace(input.Name), nullString(input.Role), nullString(input.Email),
		input.CostHour, input.MaxUnits, input.Color, timestamp, timestamp)
	if err != nil {
		return ProjectResource{}, err
	}
	id, _ := result.LastInsertId()
	var resource ProjectResource
	err = s.DB.Get(&resource, `SELECT * FROM project_resources WHERE id = ?`, id)
	return resource, err
}

func (s *Store) UpdateProjectResource(projectID, id int64, input ProjectResourceInput) (ProjectResource, error) {
	if strings.TrimSpace(input.Name) == "" {
		return ProjectResource{}, fmt.Errorf("%w: nome do recurso é obrigatório", ErrValidation)
	}
	if input.MaxUnits <= 0 {
		input.MaxUnits = 100
	}
	if input.Color == "" {
		input.Color = "blue"
	}
	result, err := s.DB.Exec(`
		UPDATE project_resources SET name = ?, role = ?, email = ?, cost_hour = ?, max_units = ?, color = ?, updated_at = ?
		WHERE id = ? AND project_id = ?`,
		strings.TrimSpace(input.Name), nullString(input.Role), nullString(input.Email), input.CostHour,
		input.MaxUnits, input.Color, now(), id, projectID)
	if err != nil {
		return ProjectResource{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ProjectResource{}, ErrNotFound
	}
	var resource ProjectResource
	err = s.DB.Get(&resource, `SELECT * FROM project_resources WHERE id = ?`, id)
	return resource, err
}

func (s *Store) DeleteProjectResource(projectID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM project_resources WHERE id = ? AND project_id = ?`, id, projectID)
	return err
}

func (s *Store) ProjectAssignments(projectID int64) ([]ProjectAssignment, error) {
	assignments := []ProjectAssignment{}
	err := s.DB.Select(&assignments, `
		SELECT a.*, r.name AS resource_name, r.cost_hour
		FROM project_assignments a
		JOIN project_resources r ON r.id = a.resource_id
		WHERE a.project_id = ?
		ORDER BY a.task_id ASC, r.name ASC`, projectID)
	return assignments, err
}

func (s *Store) CreateProjectAssignment(projectID int64, input ProjectAssignmentInput) (ProjectAssignment, error) {
	if input.TaskID <= 0 || input.ResourceID <= 0 {
		return ProjectAssignment{}, fmt.Errorf("%w: tarefa e recurso são obrigatórios", ErrValidation)
	}
	if _, err := s.FindProjectTask(projectID, input.TaskID); err != nil {
		return ProjectAssignment{}, fmt.Errorf("%w: tarefa não encontrada", ErrValidation)
	}
	var resource ProjectResource
	if err := s.DB.Get(&resource, `SELECT * FROM project_resources WHERE id = ? AND project_id = ?`, input.ResourceID, projectID); err != nil {
		return ProjectAssignment{}, fmt.Errorf("%w: recurso não encontrado", ErrValidation)
	}
	if input.Units <= 0 || input.Units > 100 {
		input.Units = 100
	}
	planned := input.PlannedHours
	if planned <= 0 {
		var effort float64
		_ = s.DB.Get(&effort, `SELECT effort_hours FROM project_tasks WHERE id = ?`, input.TaskID)
		if effort > 0 {
			planned = effort * float64(input.Units) / 100
		}
	}
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO project_assignments (project_id, task_id, resource_id, units, planned_hours, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		projectID, input.TaskID, input.ResourceID, input.Units, planned, timestamp, timestamp)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return ProjectAssignment{}, fmt.Errorf("%w: este recurso já está alocado na tarefa", ErrValidation)
		}
		return ProjectAssignment{}, err
	}
	id, _ := result.LastInsertId()
	var assignment ProjectAssignment
	err = s.DB.Get(&assignment, `
		SELECT a.*, r.name AS resource_name, r.cost_hour
		FROM project_assignments a JOIN project_resources r ON r.id = a.resource_id
		WHERE a.id = ?`, id)
	return assignment, err
}

func (s *Store) DeleteProjectAssignment(projectID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM project_assignments WHERE id = ? AND project_id = ?`, id, projectID)
	return err
}

func (s *Store) ProjectHolidays(projectID int64) ([]ProjectHoliday, error) {
	holidays := []ProjectHoliday{}
	err := s.DB.Select(&holidays, `SELECT * FROM project_holidays WHERE project_id = ? ORDER BY holiday_date ASC`, projectID)
	return holidays, err
}

func (s *Store) CreateProjectHoliday(projectID int64, date, name string) (ProjectHoliday, error) {
	date = normalizeDate(date)
	if date == "" {
		return ProjectHoliday{}, fmt.Errorf("%w: informe a data do feriado", ErrValidation)
	}
	if strings.TrimSpace(name) == "" {
		name = "Feriado"
	}
	result, err := s.DB.Exec(`
		INSERT INTO project_holidays (project_id, holiday_date, name, created_at) VALUES (?, ?, ?, ?)`,
		projectID, date, strings.TrimSpace(name), now())
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			return ProjectHoliday{}, fmt.Errorf("%w: este feriado já está cadastrado", ErrValidation)
		}
		return ProjectHoliday{}, err
	}
	id, _ := result.LastInsertId()
	var holiday ProjectHoliday
	err = s.DB.Get(&holiday, `SELECT * FROM project_holidays WHERE id = ?`, id)
	return holiday, err
}

func (s *Store) DeleteProjectHoliday(projectID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM project_holidays WHERE id = ? AND project_id = ?`, id, projectID)
	return err
}

func (s *Store) ProjectBaselines(projectID int64) ([]ProjectBaseline, error) {
	baselines := []ProjectBaseline{}
	err := s.DB.Select(&baselines, `
		SELECT b.*, (SELECT COUNT(*) FROM project_baseline_tasks t WHERE t.baseline_id = b.id) AS task_count
		FROM project_baselines b WHERE b.project_id = ?
		ORDER BY b.created_at DESC, b.id DESC`, projectID)
	return baselines, err
}

func (s *Store) CreateProjectBaseline(projectID int64, name string) (ProjectBaseline, error) {
	bundle, err := s.ProjectBundle(projectID)
	if err != nil {
		return ProjectBaseline{}, err
	}
	if strings.TrimSpace(name) == "" {
		name = "Linha de base " + time.Now().Format("02/01/2006 15:04")
	}
	tx, err := s.DB.Beginx()
	if err != nil {
		return ProjectBaseline{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.Exec(`INSERT INTO project_baselines (project_id, name, created_at) VALUES (?, ?, ?)`,
		projectID, strings.TrimSpace(name), now())
	if err != nil {
		return ProjectBaseline{}, err
	}
	baselineID, _ := result.LastInsertId()
	for _, task := range bundle.Tasks {
		if task.TaskType == "summary" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO project_baseline_tasks (baseline_id, task_id, start_date, end_date, duration_days, cost)
			VALUES (?, ?, ?, ?, ?, ?)`,
			baselineID, task.ID, nullDate(task.ScheduledFrom), nullDate(task.ScheduledTo), task.EffectiveDays, task.TotalCost); err != nil {
			return ProjectBaseline{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ProjectBaseline{}, err
	}
	var baseline ProjectBaseline
	err = s.DB.Get(&baseline, `
		SELECT b.*, (SELECT COUNT(*) FROM project_baseline_tasks t WHERE t.baseline_id = b.id) AS task_count
		FROM project_baselines b WHERE b.id = ?`, baselineID)
	return baseline, err
}

func (s *Store) DeleteProjectBaseline(projectID, id int64) error {
	_, err := s.DB.Exec(`DELETE FROM project_baselines WHERE id = ? AND project_id = ?`, id, projectID)
	return err
}

/* --------------------------------------------------------------------------
 * Full project bundle with scheduling
 * ----------------------------------------------------------------------- */

func (s *Store) ProjectBundle(projectID int64) (ProjectBundle, error) {
	bundle := ProjectBundle{}
	project, err := s.FindProject(projectID)
	if err != nil {
		return bundle, err
	}
	bundle.Project = project
	tasks, err := s.ProjectTasks(projectID)
	if err != nil {
		return bundle, err
	}
	dependencies, err := s.ProjectDependencies(projectID)
	if err != nil {
		return bundle, err
	}
	resources, err := s.ProjectResources(projectID)
	if err != nil {
		return bundle, err
	}
	assignments, err := s.ProjectAssignments(projectID)
	if err != nil {
		return bundle, err
	}
	holidays, err := s.ProjectHolidays(projectID)
	if err != nil {
		return bundle, err
	}
	baselines, err := s.ProjectBaselines(projectID)
	if err != nil {
		return bundle, err
	}

	scheduled, err := scheduleProject(project, tasks, dependencies, holidays)
	if err != nil {
		return bundle, err
	}
	tasks = decorateProjectTasks(tasks, scheduled, assignments, resources)

	baselineStart, baselineEnd, baselineCost, err := s.latestBaseline(projectID)
	if err == nil {
		for i := range tasks {
			if value, ok := baselineStart[tasks[i].ID]; ok {
				tasks[i].BaselineStart = value
			}
			if value, ok := baselineEnd[tasks[i].ID]; ok {
				tasks[i].BaselineEnd = value
			}
			if value, ok := baselineCost[tasks[i].ID]; ok {
				tasks[i].CostVariance = round2(tasks[i].TotalCost - value)
			}
			if tasks[i].BaselineStart != nil && tasks[i].ScheduledFrom != "" {
				tasks[i].StartVariance = workingDaysBetween(parseDate(*tasks[i].BaselineStart), parseDate(tasks[i].ScheduledFrom))
			}
			if tasks[i].BaselineEnd != nil && tasks[i].ScheduledTo != "" {
				tasks[i].EndVariance = workingDaysBetween(parseDate(*tasks[i].BaselineEnd), parseDate(tasks[i].ScheduledTo))
			}
		}
	}

	bundle.Tasks = tasks
	bundle.Dependencies = dependencies
	bundle.Resources = resources
	bundle.Assignments = assignments
	bundle.Holidays = holidays
	bundle.Baselines = baselines
	bundle.Metrics = projectMetrics(tasks, scheduled, project)
	return bundle, nil
}

func (s *Store) latestBaseline(projectID int64) (map[int64]*string, map[int64]*string, map[int64]float64, error) {
	starts := map[int64]*string{}
	ends := map[int64]*string{}
	costs := map[int64]float64{}
	var baselineID int64
	err := s.DB.Get(&baselineID, `SELECT id FROM project_baselines WHERE project_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`, projectID)
	if err != nil {
		return starts, ends, costs, err
	}
	rows := []struct {
		TaskID    int64   `db:"task_id"`
		StartDate *string `db:"start_date"`
		EndDate   *string `db:"end_date"`
		Cost      float64 `db:"cost"`
	}{}
	if err := s.DB.Select(&rows, `SELECT * FROM project_baseline_tasks WHERE baseline_id = ?`, baselineID); err != nil {
		return starts, ends, costs, err
	}
	for _, row := range rows {
		starts[row.TaskID] = row.StartDate
		ends[row.TaskID] = row.EndDate
		costs[row.TaskID] = row.Cost
	}
	return starts, ends, costs, nil
}

func decorateProjectTasks(tasks []ProjectTask, scheduled []scheduledTask, assignments []ProjectAssignment, resources []ProjectResource) []ProjectTask {
	byID := map[int64]scheduledTask{}
	for _, item := range scheduled {
		byID[item.ID] = item
	}
	assignmentsByTask := map[int64][]ProjectAssignment{}
	resourceNames := map[int64]string{}
	resourceCosts := map[int64]float64{}
	for _, resource := range resources {
		resourceNames[resource.ID] = resource.Name
		resourceCosts[resource.ID] = resource.CostHour
	}
	for _, assignment := range assignments {
		assignment.Cost = round2(assignment.PlannedHours * resourceCosts[assignment.ResourceID])
		assignmentsByTask[assignment.TaskID] = append(assignmentsByTask[assignment.TaskID], assignment)
	}
	tasksByID := map[int64]int{}
	for i := range tasks {
		tasksByID[tasks[i].ID] = i
		tasks[i].Resources = []string{}
	}
	for i := range tasks {
		task := &tasks[i]
		item, ok := byID[task.ID]
		if !ok {
			continue
		}
		task.Level = item.Level
		task.ScheduledFrom = formatDate(item.Start)
		task.ScheduledTo = formatDate(item.End)
		task.EffectiveDays = item.Duration
		task.IsCritical = item.Critical
		for _, assignment := range assignmentsByTask[task.ID] {
			task.AssignedCost += assignment.Cost
			if name, ok := resourceNames[assignment.ResourceID]; ok {
				task.Resources = append(task.Resources, name)
			}
		}
		task.AssignedCost = round2(task.AssignedCost)
		task.TotalCost = round2(task.FixedCost + task.AssignedCost)
	}
	for _, item := range scheduled {
		if !item.Summary {
			continue
		}
		index, ok := tasksByID[item.ID]
		if !ok {
			continue
		}
		for _, childID := range item.Children {
			if childIndex, ok := tasksByID[childID]; ok {
				tasks[index].TotalCost += tasks[childIndex].TotalCost
				tasks[index].AssignedCost += tasks[childIndex].AssignedCost
				if tasks[childIndex].IsCritical {
					tasks[index].IsCritical = true
				}
			}
		}
		tasks[index].TotalCost = round2(tasks[index].TotalCost)
		tasks[index].AssignedCost = round2(tasks[index].AssignedCost)
	}
	return tasks
}

func projectMetrics(tasks []ProjectTask, scheduled []scheduledTask, project Project) ProjectMetrics {
	metrics := ProjectMetrics{Budget: project.Budget}
	byID := map[int64]scheduledTask{}
	for _, item := range scheduled {
		byID[item.ID] = item
	}
	today := parseDate(time.Now().Format("2006-01-02"))
	var finish time.Time
	weighted, weight := 0, 0
	for _, task := range tasks {
		item := byID[task.ID]
		if task.TaskType != "summary" {
			metrics.TaskCount++
			if task.Progress >= 100 {
				metrics.Completed++
			} else if task.Progress > 0 {
				metrics.InProgress++
			}
			if task.TaskType == "milestone" {
				metrics.Milestones++
			}
			if !item.End.IsZero() && item.End.Before(today) && task.Progress < 100 {
				metrics.Overdue++
			}
			duration := item.Duration
			if duration < 1 {
				duration = 1
			}
			weighted += task.Progress * duration
			weight += duration
		}
		if task.IsCritical && task.TaskType != "summary" {
			metrics.Critical++
		}
		if task.TaskType != "summary" {
			metrics.PlannedCost += task.TotalCost
		}
		if !item.End.IsZero() && item.End.After(finish) {
			finish = item.End
		}
	}
	if weight > 0 {
		metrics.Progress = weighted / weight
	}
	metrics.PlannedCost = round2(metrics.PlannedCost)
	metrics.CostVariance = round2(project.Budget - metrics.PlannedCost)
	start := parseDate(project.StartDate)
	if !finish.IsZero() {
		metrics.ScheduledFrom = project.StartDate
		metrics.ScheduledTo = formatDate(finish)
		metrics.DurationDays = workingDaysBetween(start, finish)
		if finish.After(today) {
			metrics.DaysRemaining = workingDaysBetween(today, finish)
		}
	}
	return metrics
}

/* --------------------------------------------------------------------------
 * Scheduling engine
 * ----------------------------------------------------------------------- */

type projectCalendar struct {
	working map[int]bool
	holiday map[string]bool
}

type scheduledTask struct {
	ID       int64
	Level    int
	Start    time.Time
	End      time.Time
	Duration int
	Summary  bool
	Children []int64
	Critical bool
}

// scheduleProject builds the timeline and marks the critical path. A leaf task
// is critical when extending it by one working day pushes the project finish.
func scheduleProject(project Project, tasks []ProjectTask, dependencies []ProjectDependency, holidays []ProjectHoliday) ([]scheduledTask, error) {
	scheduled, err := buildProjectTimeline(project, tasks, dependencies, holidays)
	if err != nil {
		return nil, err
	}
	baseFinish := scheduleFinish(scheduled)
	if baseFinish.IsZero() {
		return scheduled, nil
	}
	children := map[int64]int{}
	for _, task := range tasks {
		if task.ParentID != nil {
			children[*task.ParentID]++
		}
	}
	critical := map[int64]bool{}
	for _, task := range tasks {
		if task.TaskType == "summary" || children[task.ID] > 0 {
			continue
		}
		extended := make([]ProjectTask, len(tasks))
		copy(extended, tasks)
		for j := range extended {
			if extended[j].ID == task.ID {
				if extended[j].DurationDays < 1 {
					extended[j].DurationDays = 1
				} else {
					extended[j].DurationDays++
				}
				break
			}
		}
		probe, err := buildProjectTimeline(project, extended, dependencies, holidays)
		if err != nil {
			continue
		}
		if scheduleFinish(probe).After(baseFinish) {
			critical[task.ID] = true
		}
	}
	for i := range scheduled {
		if critical[scheduled[i].ID] {
			scheduled[i].Critical = true
		}
	}
	for changed := true; changed; {
		changed = false
		for i := range scheduled {
			if !scheduled[i].Summary || scheduled[i].Critical {
				continue
			}
			for _, childID := range scheduled[i].Children {
				for _, child := range scheduled {
					if child.ID == childID && child.Critical {
						scheduled[i].Critical = true
						changed = true
					}
				}
			}
		}
	}
	return scheduled, nil
}

func buildProjectTimeline(project Project, tasks []ProjectTask, dependencies []ProjectDependency, holidays []ProjectHoliday) ([]scheduledTask, error) {
	if len(tasks) > 500 {
		return nil, fmt.Errorf("%w: limite de 500 tarefas por projeto", ErrValidation)
	}
	calendar := buildProjectCalendar(project, holidays)
	projectStart := parseDate(project.StartDate)
	if projectStart.IsZero() {
		projectStart = parseDate(time.Now().Format("2006-01-02"))
	}
	tasksByID := map[int64]ProjectTask{}
	for _, task := range tasks {
		tasksByID[task.ID] = task
	}
	children := map[int64][]int64{}
	for _, task := range tasks {
		if task.ParentID != nil {
			children[*task.ParentID] = append(children[*task.ParentID], task.ID)
		}
	}
	edges := map[int64][]int64{}
	indegree := map[int64]int{}
	for _, task := range tasks {
		indegree[task.ID] = 0
	}
	addEdge := func(from, to int64) {
		edges[from] = append(edges[from], to)
		indegree[to]++
	}
	for _, dependency := range dependencies {
		addEdge(dependency.PredecessorID, dependency.SuccessorID)
	}
	for parentID := range children {
		for _, childID := range children[parentID] {
			addEdge(childID, parentID)
		}
	}
	order := make([]int64, 0, len(tasks))
	queue := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		if indegree[task.ID] == 0 {
			queue = append(queue, task.ID)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, next := range edges[id] {
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(order) != len(tasks) {
		return nil, fmt.Errorf("%w: existem dependências circulares no projeto", ErrValidation)
	}

	result := map[int64]*scheduledTask{}
	for _, id := range order {
		task := tasksByID[id]
		item := &scheduledTask{ID: task.ID, Summary: task.TaskType == "summary"}
		if len(children[task.ID]) > 0 {
			item.Summary = true
		}
		item.Children = children[task.ID]
		if item.Summary {
			start, end, duration := summaryWindow(item.Children, result)
			if start.IsZero() {
				start, end = projectStart, projectStart
			}
			item.Start, item.End, item.Duration = start, end, duration
			result[id] = item
			continue
		}
		duration := task.DurationDays
		if task.TaskType == "milestone" {
			duration = 0
		}
		if duration < 0 {
			duration = 0
		}
		start := projectStart
		if task.StartDate != nil {
			if parsed := parseDate(*task.StartDate); !parsed.IsZero() {
				start = parsed
			}
		}
		for _, dependency := range dependencies {
			if dependency.SuccessorID != id {
				continue
			}
			predecessor, ok := result[dependency.PredecessorID]
			if !ok || predecessor.Start.IsZero() {
				continue
			}
			candidate := dependencyStart(*predecessor, duration, dependency.Type, dependency.LagDays, calendar)
			if candidate.After(start) {
				start = candidate
			}
		}
		start = nextWorkingDay(start, calendar)
		end := start
		if duration > 0 {
			end = addWorkingDays(start, duration-1, calendar)
		}
		item.Start, item.End, item.Duration = start, end, duration
		result[id] = item
	}

	levelOf := func(task ProjectTask) int {
		level := 0
		current := task.ParentID
		for depth := 0; depth < 100 && current != nil; depth++ {
			level++
			parent, ok := tasksByID[*current]
			if !ok || parent.ParentID == nil {
				break
			}
			current = parent.ParentID
		}
		return level
	}
	scheduled := make([]scheduledTask, 0, len(tasks))
	for _, task := range tasks {
		item, ok := result[task.ID]
		if !ok {
			continue
		}
		item.Level = levelOf(task)
		scheduled = append(scheduled, *item)
	}
	return scheduled, nil
}

func summaryWindow(children []int64, scheduled map[int64]*scheduledTask) (time.Time, time.Time, int) {
	var start, end time.Time
	for _, childID := range children {
		child, ok := scheduled[childID]
		if !ok || child.Start.IsZero() {
			continue
		}
		if start.IsZero() || child.Start.Before(start) {
			start = child.Start
		}
		if end.IsZero() || child.End.After(end) {
			end = child.End
		}
	}
	duration := 0
	if !start.IsZero() && !end.IsZero() {
		duration = workingDaysBetween(start, end)
	}
	return start, end, duration
}

func scheduleFinish(scheduled []scheduledTask) time.Time {
	var finish time.Time
	for _, item := range scheduled {
		if item.End.After(finish) {
			finish = item.End
		}
	}
	return finish
}

func dependencyStart(predecessor scheduledTask, duration int, depType string, lag int, calendar projectCalendar) time.Time {
	switch strings.ToUpper(depType) {
	case "SS":
		return shiftWorkingDays(predecessor.Start, lag, calendar)
	case "FF":
		finish := shiftWorkingDays(predecessor.End, lag, calendar)
		return shiftWorkingDays(finish, -(duration - 1), calendar)
	case "SF":
		finish := shiftWorkingDays(predecessor.Start, lag, calendar)
		return shiftWorkingDays(finish, -(duration - 1), calendar)
	default:
		return shiftWorkingDays(nextWorkingDay(predecessor.End.AddDate(0, 0, 1), calendar), lag, calendar)
	}
}

func buildProjectCalendar(project Project, holidays []ProjectHoliday) projectCalendar {
	working := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	if strings.TrimSpace(project.WorkingDays) != "" {
		working = map[int]bool{}
		for _, piece := range strings.Split(project.WorkingDays, ",") {
			if day, err := strconv.Atoi(strings.TrimSpace(piece)); err == nil && day >= 0 && day <= 6 {
				working[day] = true
			}
		}
		if len(working) == 0 {
			working = map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
		}
	}
	holidaySet := map[string]bool{}
	for _, holiday := range holidays {
		holidaySet[holiday.Date] = true
	}
	return projectCalendar{working: working, holiday: holidaySet}
}

func (c projectCalendar) isWorkingDay(day time.Time) bool {
	return c.working[int(day.Weekday())] && !c.holiday[day.Format("2006-01-02")]
}

func nextWorkingDay(day time.Time, calendar projectCalendar) time.Time {
	for i := 0; i < 3700; i++ {
		if calendar.isWorkingDay(day) {
			return day
		}
		day = day.AddDate(0, 0, 1)
	}
	return day
}

func addWorkingDays(day time.Time, days int, calendar projectCalendar) time.Time {
	current := nextWorkingDay(day, calendar)
	for days > 0 {
		current = nextWorkingDay(current.AddDate(0, 0, 1), calendar)
		days--
	}
	return current
}

func shiftWorkingDays(day time.Time, days int, calendar projectCalendar) time.Time {
	if days == 0 {
		return nextWorkingDay(day, calendar)
	}
	if days > 0 {
		return addWorkingDays(day, days, calendar)
	}
	return subtractWorkingDays(day, -days, calendar)
}

func subtractWorkingDays(day time.Time, days int, calendar projectCalendar) time.Time {
	current := day
	if !calendar.isWorkingDay(current) {
		current = previousWorkingDay(current, calendar)
	}
	for days > 0 {
		current = previousWorkingDay(current.AddDate(0, 0, -1), calendar)
		days--
	}
	return current
}

func previousWorkingDay(day time.Time, calendar projectCalendar) time.Time {
	for i := 0; i < 3700; i++ {
		if calendar.isWorkingDay(day) {
			return day
		}
		day = day.AddDate(0, 0, -1)
	}
	return day
}

func workingDaysBetween(start, end time.Time) int {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	if end.Before(start) {
		start, end = end, start
	}
	days := 0
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		if day.Weekday() != time.Saturday && day.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}

/* --------------------------------------------------------------------------
 * Helpers
 * ----------------------------------------------------------------------- */

func normalizeTaskType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "summary", "resumo":
		return "summary"
	case "milestone", "marco":
		return "milestone"
	default:
		return "task"
	}
}

func normalizeWorkingDays(value string) string {
	values := []string{}
	seen := map[int]bool{}
	for _, piece := range strings.Split(value, ",") {
		day, err := strconv.Atoi(strings.TrimSpace(piece))
		if err == nil && day >= 0 && day <= 6 && !seen[day] {
			seen[day] = true
			values = append(values, strconv.Itoa(day))
		}
	}
	if len(values) == 0 {
		return "1,2,3,4,5"
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func normalizeDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		value = value[:10]
	}
	if parsed := parseDate(value); !parsed.IsZero() {
		return parsed.Format("2006-01-02")
	}
	return ""
}

func parseDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if len(value) >= 10 {
		value = value[:10]
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func nullableID(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
