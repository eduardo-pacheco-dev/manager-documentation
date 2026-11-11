package modules

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")
var ErrValidation = errors.New("validation")

type Store struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *Store {
	return &Store{DB: db}
}

func now() time.Time { return time.Now().UTC() }

func nullString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nullDate(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

/* --------------------------------------------------------------------------
 * Tasks module
 * ----------------------------------------------------------------------- */

type TodoList struct {
	ID        int64      `db:"id" json:"id"`
	UserID    int64      `db:"user_id" json:"-"`
	Name      string     `db:"name" json:"name"`
	Color     string     `db:"color" json:"color"`
	Icon      string     `db:"icon" json:"icon"`
	Position  int        `db:"position" json:"position"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
	TaskCount int        `db:"task_count" json:"task_count"`
}

type TodoStep struct {
	ID          int64      `db:"id" json:"id"`
	TodoTaskID  int64      `db:"todo_task_id" json:"-"`
	Title       string     `db:"title" json:"title"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at"`
	Position    int        `db:"position" json:"position"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type TodoTask struct {
	ID          int64      `db:"id" json:"id"`
	UserID      int64      `db:"user_id" json:"-"`
	TodoListID  int64      `db:"todo_list_id" json:"todo_list_id"`
	Title       string     `db:"title" json:"title"`
	Notes       *string    `db:"notes" json:"notes"`
	DueAt       *string    `db:"due_at" json:"due_at"`
	RemindAt    *time.Time `db:"remind_at" json:"remind_at"`
	RepeatType  string     `db:"repeat_type" json:"repeat_type"`
	IsImportant bool       `db:"is_important" json:"is_important"`
	MyDayOn     *string    `db:"my_day_on" json:"my_day_on"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at"`
	Position    int        `db:"position" json:"position"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	Steps       []TodoStep `db:"-" json:"steps"`
	ListName    string     `db:"list_name" json:"list_name"`
	ListColor   string     `db:"list_color" json:"list_color"`
	ListIcon    string     `db:"list_icon" json:"list_icon"`
}

type TaskFilter struct {
	View    string
	ListID  int64
	Search  string
	Sort    string
	Today   string
	WeekEnd string
}

func (s *Store) TodoLists(userID int64) ([]TodoList, error) {
	lists := []TodoList{}
	err := s.DB.Select(&lists, `
		SELECT l.*, (
			SELECT COUNT(*) FROM todo_tasks t
			WHERE t.todo_list_id = l.id AND t.completed_at IS NULL
		) AS task_count
		FROM todo_lists l
		WHERE l.user_id = ? AND l.deleted_at IS NULL
		ORDER BY l.position ASC, l.id ASC`, userID)
	return lists, err
}

func (s *Store) CreateTodoList(userID int64, name, color, icon string) (TodoList, error) {
	if color == "" {
		color = "indigo"
	}
	if icon == "" {
		icon = "check-circle"
	}
	var position int
	_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM todo_lists WHERE user_id = ?`, userID)
	timestamp := now()
	result, err := s.DB.Exec(`
		INSERT INTO todo_lists (user_id, name, color, icon, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, strings.TrimSpace(name), color, icon, position, timestamp, timestamp)
	if err != nil {
		return TodoList{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindTodoList(userID, id)
}

func (s *Store) FindTodoList(userID, id int64) (TodoList, error) {
	var list TodoList
	err := s.DB.Get(&list, `SELECT * FROM todo_lists WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return list, ErrNotFound
	}
	return list, err
}

func (s *Store) UpdateTodoList(userID, id int64, name, color, icon string) error {
	_, err := s.DB.Exec(`
		UPDATE todo_lists SET name = ?, color = ?, icon = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		strings.TrimSpace(name), color, icon, now(), id, userID)
	return err
}

func (s *Store) DeleteTodoList(userID, id int64) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM todo_tasks WHERE todo_list_id = ? AND user_id = ?`, id, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE todo_lists SET deleted_at = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		now(), now(), id, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) TodoTasks(userID int64, filter TaskFilter) ([]TodoTask, error) {
	conditions := []string{"t.user_id = ?"}
	args := []any{userID}
	if filter.View == "completed" {
		conditions = append(conditions, "t.completed_at IS NOT NULL")
	} else {
		conditions = append(conditions, "t.completed_at IS NULL")
	}
	switch filter.View {
	case "list":
		conditions = append(conditions, "t.todo_list_id = ?")
		args = append(args, filter.ListID)
	case "important":
		conditions = append(conditions, "t.is_important = 1")
	case "planned":
		conditions = append(conditions, "t.due_at IS NOT NULL")
	case "today":
		conditions = append(conditions, "t.due_at = ?")
		args = append(args, filter.Today)
	case "week":
		conditions = append(conditions, "t.due_at IS NOT NULL AND t.due_at BETWEEN ? AND ?")
		args = append(args, filter.Today, filter.WeekEnd)
	case "my_day":
		conditions = append(conditions, "t.my_day_on = ?")
		args = append(args, filter.Today)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, "(t.title LIKE ? OR COALESCE(t.notes, '') LIKE ?)")
		args = append(args, like, like)
	}
	order := "t.position ASC, t.id ASC"
	switch filter.Sort {
	case "due":
		order = "CASE WHEN t.due_at IS NULL THEN 1 ELSE 0 END ASC, t.due_at ASC, t.position ASC, t.id ASC"
	case "important":
		order = "t.is_important DESC, t.position ASC, t.id ASC"
	case "title":
		order = "t.title COLLATE NOCASE ASC, t.id ASC"
	case "created":
		order = "t.created_at DESC, t.id DESC"
	}
	query := `
		SELECT t.*, l.name AS list_name, l.color AS list_color, l.icon AS list_icon
		FROM todo_tasks t
		JOIN todo_lists l ON l.id = t.todo_list_id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY ` + order + `
		LIMIT 150`
	tasks := []TodoTask{}
	if err := s.DB.Select(&tasks, query, args...); err != nil {
		return nil, err
	}
	if err := s.attachTodoSteps(tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *Store) attachTodoSteps(tasks []TodoTask) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(tasks))
	index := map[int64]int{}
	for i, task := range tasks {
		ids = append(ids, task.ID)
		index[task.ID] = i
		tasks[i].Steps = []TodoStep{}
	}
	query, args, err := sqlx.In(`SELECT * FROM todo_steps WHERE todo_task_id IN (?) ORDER BY position ASC, id ASC`, ids)
	if err != nil {
		return err
	}
	query = s.DB.Rebind(query)
	steps := []TodoStep{}
	if err := s.DB.Select(&steps, query, args...); err != nil {
		return err
	}
	for _, step := range steps {
		if i, ok := index[step.TodoTaskID]; ok {
			tasks[i].Steps = append(tasks[i].Steps, step)
		}
	}
	return nil
}

func (s *Store) TaskCounts(userID int64, today, weekEnd string) (map[string]int, error) {
	counts := map[string]int{"my_day": 0, "important": 0, "planned": 0, "today": 0, "week": 0, "all": 0, "overdue": 0, "completed": 0}
	queries := map[string]string{
		"my_day":    "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL AND my_day_on = ?",
		"important": "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL AND is_important = 1",
		"planned":   "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL AND due_at IS NOT NULL",
		"today":     "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL AND due_at = ?",
		"week":      "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL AND due_at IS NOT NULL AND due_at BETWEEN ? AND ?",
		"all":       "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL",
		"overdue":   "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NULL AND due_at IS NOT NULL AND due_at < ?",
		"completed": "SELECT COUNT(*) FROM todo_tasks WHERE user_id = ? AND completed_at IS NOT NULL",
	}
	for key, query := range queries {
		var count int
		var err error
		switch key {
		case "my_day", "today", "overdue":
			err = s.DB.Get(&count, query, userID, today)
		case "week":
			err = s.DB.Get(&count, query, userID, today, weekEnd)
		default:
			err = s.DB.Get(&count, query, userID)
		}
		if err != nil {
			return nil, err
		}
		counts[key] = count
	}
	return counts, nil
}

func (s *Store) FindTodoTask(userID, id int64) (TodoTask, error) {
	var task TodoTask
	err := s.DB.Get(&task, `
		SELECT t.*, l.name AS list_name, l.color AS list_color, l.icon AS list_icon
		FROM todo_tasks t JOIN todo_lists l ON l.id = t.todo_list_id
		WHERE t.id = ? AND t.user_id = ?`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return task, ErrNotFound
	}
	if err != nil {
		return task, err
	}
	tasks := []TodoTask{task}
	if err := s.attachTodoSteps(tasks); err != nil {
		return task, err
	}
	return tasks[0], nil
}

type TaskInput struct {
	TodoListID  int64
	Title       string
	Notes       string
	DueAt       string
	RemindAt    string
	RepeatType  string
	IsImportant bool
	MyDay       bool
}

func (s *Store) CreateTodoTask(userID int64, input TaskInput) (TodoTask, error) {
	if _, err := s.FindTodoList(userID, input.TodoListID); err != nil {
		return TodoTask{}, err
	}
	if input.RepeatType == "" {
		input.RepeatType = "none"
	}
	var position int
	_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM todo_tasks WHERE todo_list_id = ?`, input.TodoListID)
	timestamp := now()
	var myDay *string
	if input.MyDay {
		value := time.Now().Format("2006-01-02")
		myDay = &value
	}
	result, err := s.DB.Exec(`
		INSERT INTO todo_tasks (user_id, todo_list_id, title, notes, due_at, remind_at, repeat_type,
			is_important, my_day_on, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, input.TodoListID, strings.TrimSpace(input.Title), nullString(input.Notes),
		nullDate(input.DueAt), nullableTime(input.RemindAt), input.RepeatType, input.IsImportant,
		myDay, position, timestamp, timestamp)
	if err != nil {
		return TodoTask{}, err
	}
	id, _ := result.LastInsertId()
	return s.FindTodoTask(userID, id)
}

func nullableTime(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := parseDateTime(value)
	if err != nil {
		return nil
	}
	return parsed
}

func (s *Store) UpdateTodoTask(userID, id int64, input TaskInput) (TodoTask, error) {
	task, err := s.FindTodoTask(userID, id)
	if err != nil {
		return task, err
	}
	if _, err := s.FindTodoList(userID, input.TodoListID); err != nil {
		return task, err
	}
	if input.RepeatType == "" {
		input.RepeatType = "none"
	}
	myDay := task.MyDayOn
	if input.MyDay {
		if myDay == nil {
			value := time.Now().Format("2006-01-02")
			myDay = &value
		}
	} else {
		myDay = nil
	}
	position := task.Position
	if task.TodoListID != input.TodoListID {
		_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM todo_tasks WHERE todo_list_id = ?`, input.TodoListID)
	}
	_, err = s.DB.Exec(`
		UPDATE todo_tasks SET todo_list_id = ?, title = ?, notes = ?, due_at = ?, remind_at = ?,
			repeat_type = ?, is_important = ?, my_day_on = ?, position = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		input.TodoListID, strings.TrimSpace(input.Title), nullString(input.Notes), nullDate(input.DueAt),
		nullableTime(input.RemindAt), input.RepeatType, input.IsImportant, myDay, position, now(), id, userID)
	return s.FindTodoTask(userID, id)
}

func (s *Store) ToggleTodoTask(userID, id int64) (TodoTask, string, error) {
	task, err := s.FindTodoTask(userID, id)
	if err != nil {
		return task, "", err
	}
	if task.CompletedAt != nil {
		_, err = s.DB.Exec(`UPDATE todo_tasks SET completed_at = NULL, updated_at = ? WHERE id = ?`, now(), id)
		if err == nil {
			updated, findErr := s.FindTodoTask(userID, id)
			return updated, "", findErr
		}
		return task, "", err
	}
	timestamp := now()
	_, err = s.DB.Exec(`UPDATE todo_tasks SET completed_at = ?, updated_at = ? WHERE id = ?`, timestamp, timestamp, id)
	if err != nil {
		return task, "", err
	}
	if task.RepeatType != "none" {
		_ = s.createNextTodoOccurrence(task)
	}
	updated, err := s.FindTodoTask(userID, id)
	return updated, "", err
}

func (s *Store) createNextTodoOccurrence(task TodoTask) error {
	next := nextRepeatDate(task.DueAt, task.RepeatType, time.Now())
	var position int
	_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM todo_tasks WHERE todo_list_id = ?`, task.TodoListID)
	timestamp := now()
	var remind any
	if task.RemindAt != nil {
		nextDate, _ := time.Parse("2006-01-02", next)
		remind = time.Date(nextDate.Year(), nextDate.Month(), nextDate.Day(), task.RemindAt.Hour(), task.RemindAt.Minute(), 0, 0, time.UTC)
	}
	result, err := s.DB.Exec(`
		INSERT INTO todo_tasks (user_id, todo_list_id, title, notes, due_at, remind_at, repeat_type,
			is_important, my_day_on, completed_at, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, ?)`,
		task.UserID, task.TodoListID, task.Title, task.Notes, next, remind, task.RepeatType,
		task.IsImportant, position, timestamp, timestamp)
	if err != nil {
		return err
	}
	newID, _ := result.LastInsertId()
	for _, step := range task.Steps {
		_, _ = s.DB.Exec(`INSERT INTO todo_steps (todo_task_id, title, position, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, newID, step.Title, step.Position, timestamp, timestamp)
	}
	return nil
}

func nextRepeatDate(due *string, repeat string, base time.Time) string {
	reference := base
	if due != nil {
		if parsed, err := time.Parse("2006-01-02", *due); err == nil {
			reference = parsed
		}
	}
	switch repeat {
	case "daily":
		reference = reference.AddDate(0, 0, 1)
	case "weekdays":
		reference = reference.AddDate(0, 0, 1)
		for reference.Weekday() == time.Saturday || reference.Weekday() == time.Sunday {
			reference = reference.AddDate(0, 0, 1)
		}
	case "weekly":
		reference = reference.AddDate(0, 0, 7)
	case "monthly":
		reference = reference.AddDate(0, 1, 0)
	case "yearly":
		reference = reference.AddDate(1, 0, 0)
	}
	return reference.Format("2006-01-02")
}

func (s *Store) ToggleTodoImportant(userID, id int64) (TodoTask, error) {
	task, err := s.FindTodoTask(userID, id)
	if err != nil {
		return task, err
	}
	_, err = s.DB.Exec(`UPDATE todo_tasks SET is_important = ?, updated_at = ? WHERE id = ?`, !task.IsImportant, now(), id)
	return s.FindTodoTask(userID, id)
}

func (s *Store) ToggleTodoMyDay(userID, id int64) (TodoTask, error) {
	task, err := s.FindTodoTask(userID, id)
	if err != nil {
		return task, err
	}
	today := time.Now().Format("2006-01-02")
	active := task.MyDayOn != nil && *task.MyDayOn == today
	if active {
		_, err = s.DB.Exec(`UPDATE todo_tasks SET my_day_on = NULL, updated_at = ? WHERE id = ?`, now(), id)
	} else {
		_, err = s.DB.Exec(`UPDATE todo_tasks SET my_day_on = ?, updated_at = ? WHERE id = ?`, today, now(), id)
	}
	return s.FindTodoTask(userID, id)
}

func (s *Store) MoveTodoTask(userID, id int64, listID int64, direction string) (string, error) {
	task, err := s.FindTodoTask(userID, id)
	if err != nil {
		return "", err
	}
	if listID > 0 && listID != task.TodoListID {
		list, err := s.FindTodoList(userID, listID)
		if err != nil {
			return "", err
		}
		var position int
		_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM todo_tasks WHERE todo_list_id = ?`, listID)
		_, err = s.DB.Exec(`UPDATE todo_tasks SET todo_list_id = ?, position = ?, updated_at = ? WHERE id = ?`,
			listID, position, now(), id)
		return "Tarefa movida para \"" + list.Name + "\".", err
	}
	tasks := []TodoTask{}
	if err := s.DB.Select(&tasks, `SELECT * FROM todo_tasks WHERE user_id = ? AND todo_list_id = ? AND completed_at IS NULL ORDER BY position ASC, id ASC`, userID, task.TodoListID); err != nil {
		return "", err
	}
	index := -1
	for i := range tasks {
		if tasks[i].ID == id {
			index = i
			break
		}
	}
	target := index - 1
	if direction == "down" {
		target = index + 1
	}
	if index < 0 || target < 0 || target >= len(tasks) {
		return "", nil
	}
	tasks[index], tasks[target] = tasks[target], tasks[index]
	for i, item := range tasks {
		_, _ = s.DB.Exec(`UPDATE todo_tasks SET position = ?, updated_at = ? WHERE id = ?`, i+1, now(), item.ID)
	}
	return "", nil
}

func (s *Store) DeleteTodoTask(userID, id int64) error {
	s.DeleteEntityLinks("task", id)
	_, err := s.DB.Exec(`DELETE FROM todo_tasks WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) CreateTodoStep(userID, taskID int64, title string) (TodoStep, error) {
	if _, err := s.FindTodoTask(userID, taskID); err != nil {
		return TodoStep{}, err
	}
	var position int
	_ = s.DB.Get(&position, `SELECT COALESCE(MAX(position), 0) + 1 FROM todo_steps WHERE todo_task_id = ?`, taskID)
	timestamp := now()
	result, err := s.DB.Exec(`INSERT INTO todo_steps (todo_task_id, title, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, taskID, strings.TrimSpace(title), position, timestamp, timestamp)
	if err != nil {
		return TodoStep{}, err
	}
	id, _ := result.LastInsertId()
	var step TodoStep
	err = s.DB.Get(&step, `SELECT * FROM todo_steps WHERE id = ?`, id)
	return step, err
}

func (s *Store) UpdateTodoStep(userID, taskID, stepID int64, title string, completed *bool) error {
	if _, err := s.FindTodoTask(userID, taskID); err != nil {
		return err
	}
	step, err := s.findTodoStep(taskID, stepID)
	if err != nil {
		return err
	}
	if title != "" {
		step.Title = strings.TrimSpace(title)
	}
	if completed != nil {
		if *completed {
			timestamp := now()
			step.CompletedAt = &timestamp
		} else {
			step.CompletedAt = nil
		}
	}
	_, err = s.DB.Exec(`UPDATE todo_steps SET title = ?, completed_at = ?, updated_at = ? WHERE id = ?`,
		step.Title, step.CompletedAt, now(), step.ID)
	return err
}

func (s *Store) findTodoStep(taskID, stepID int64) (TodoStep, error) {
	var step TodoStep
	err := s.DB.Get(&step, `SELECT * FROM todo_steps WHERE id = ? AND todo_task_id = ?`, stepID, taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return step, ErrNotFound
	}
	return step, err
}

func (s *Store) DeleteTodoStep(userID, taskID, stepID int64) error {
	if _, err := s.FindTodoTask(userID, taskID); err != nil {
		return err
	}
	_, err := s.DB.Exec(`DELETE FROM todo_steps WHERE id = ? AND todo_task_id = ?`, stepID, taskID)
	return err
}

func parseDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime %q", value)
}
