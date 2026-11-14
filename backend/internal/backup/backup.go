// Package backup creates, lists, restores and rotates backups of the
// application database and local uploads.
package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
)

const (
	filePrefix = "backup-"
	formatV    = 1
)

var (
	namePattern = regexp.MustCompile(`^backup-[0-9]{8}-[0-9]{6}(-[a-z0-9]{4})?\.zip$`)

	// ErrNotFound is returned when a backup file does not exist.
	ErrNotFound = errors.New("backup not found")

	// ErrInvalidArchive is returned when an imported file is not a valid backup.
	ErrInvalidArchive = errors.New("invalid backup archive")
)

// Entry describes a backup file stored on disk.
type Entry struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// Settings is the read-only view of the backup configuration exposed to the UI.
type Settings struct {
	Directory  string     `json:"directory"`
	Retention  int        `json:"retention"`
	Automatic  bool       `json:"automatic"`
	Hour       int        `json:"hour"`
	Minute     int        `json:"minute"`
	Weekdays   []int      `json:"weekdays"`
	NextRun    *time.Time `json:"next_run,omitempty"`
	UploadsDir string     `json:"uploads_dir"`
}

// Schedule controls the automatic backup routine.
type Schedule struct {
	Enabled  bool  `json:"enabled"`
	Hour     int   `json:"hour"`
	Minute   int   `json:"minute"`
	Weekdays []int `json:"weekdays"`
}

type manifest struct {
	App         string    `json:"app"`
	Format      int       `json:"format"`
	CreatedAt   time.Time `json:"created_at"`
	Database    string    `json:"database"`
	HasUploads  bool      `json:"has_uploads"`
	HasRemote   bool      `json:"has_remote_attachments"`
	Description string    `json:"description,omitempty"`
}

// RemoteStore exposes remote objects (S3/R2) to the backup service.
type RemoteStore interface {
	Provider() string
	ExportObjects(ctx context.Context, add func(key string, body io.Reader) error) error
	ImportObject(ctx context.Context, key string, body io.Reader) error
}

// ConfigStore persists the schedule choices changed by administrators.
type ConfigStore interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}

// Service owns the backup directory and the external MySQL client binaries.
type Service struct {
	cfg    config.Config
	now    func() time.Time
	mu     sync.Mutex
	remote RemoteStore
	store  ConfigStore
	wake   chan struct{}
}

// New creates the service and ensures the backup directory exists.
func New(cfg config.Config) *Service {
	service := &Service{cfg: cfg, now: time.Now, wake: make(chan struct{}, 1)}
	if err := os.MkdirAll(cfg.BackupDir, 0o750); err != nil {
		log.Printf("backup: could not create directory %s: %v", cfg.BackupDir, err)
	}
	return service
}

// SetConfigStore enables persistent schedule settings.
func (s *Service) SetConfigStore(store ConfigStore) {
	s.store = store
}

// SetRemote registers the remote object store used for attachments kept in
// S3/R2. It is safe to call once at startup.
func (s *Service) SetRemote(remote RemoteStore) {
	s.remote = remote
}

// Settings returns the active backup configuration.
func (s *Service) Settings() Settings {
	dir, _ := filepath.Abs(s.cfg.BackupDir)
	uploads, _ := filepath.Abs(s.cfg.UploadDir)
	schedule := s.Schedule()
	settings := Settings{
		Directory:  dir,
		Retention:  s.cfg.BackupRetention,
		Automatic:  schedule.Enabled,
		Hour:       schedule.Hour,
		Minute:     schedule.Minute,
		Weekdays:   append([]int{}, schedule.Weekdays...),
		UploadsDir: uploads,
	}
	if next, ok := schedule.Next(s.now()); ok {
		settings.NextRun = &next
	}
	return settings
}

const (
	scheduleEnabledKey  = "backup.enabled"
	scheduleHourKey     = "backup.hour"
	scheduleMinuteKey   = "backup.minute"
	scheduleWeekdaysKey = "backup.weekdays"
)

func defaultSchedule(cfg config.Config) Schedule {
	return Schedule{
		Enabled:  cfg.BackupAuto,
		Hour:     cfg.BackupHour,
		Minute:   0,
		Weekdays: []int{1, 2, 3, 4, 5, 6, 7},
	}
}

// Schedule reads the persisted schedule, falling back to the environment
// defaults when nothing has been saved yet.
func (s *Service) Schedule() Schedule {
	schedule := defaultSchedule(s.cfg)
	if s.store == nil {
		return schedule
	}
	if value, err := s.store.GetSetting(scheduleEnabledKey); err == nil {
		schedule.Enabled = value == "1" || strings.EqualFold(value, "true")
	}
	if value, err := s.store.GetSetting(scheduleHourKey); err == nil {
		if hour, parseErr := strconv.Atoi(strings.TrimSpace(value)); parseErr == nil && hour >= 0 && hour <= 23 {
			schedule.Hour = hour
		}
	}
	if value, err := s.store.GetSetting(scheduleMinuteKey); err == nil {
		if minute, parseErr := strconv.Atoi(strings.TrimSpace(value)); parseErr == nil && minute >= 0 && minute <= 59 {
			schedule.Minute = minute
		}
	}
	if value, err := s.store.GetSetting(scheduleWeekdaysKey); err == nil {
		if days, parseErr := parseWeekdays(value); parseErr == nil && len(days) > 0 {
			schedule.Weekdays = days
		}
	}
	return schedule
}

// UpdateSchedule validates and persists a new schedule.
func (s *Service) UpdateSchedule(schedule Schedule) error {
	schedule = schedule.Normalized()
	if err := schedule.Validate(); err != nil {
		return err
	}
	if s.store == nil {
		return errors.New("armazenamento de configurações indisponível")
	}
	values := []struct {
		key   string
		value string
	}{
		{scheduleEnabledKey, "0"},
		{scheduleHourKey, strconv.Itoa(schedule.Hour)},
		{scheduleMinuteKey, strconv.Itoa(schedule.Minute)},
		{scheduleWeekdaysKey, weekdaysString(schedule.Weekdays)},
	}
	if schedule.Enabled {
		values[0].value = "1"
	}
	for _, item := range values {
		if err := s.store.SetSetting(item.key, item.value); err != nil {
			return err
		}
	}
	s.NotifyScheduleChanged()
	return nil
}

// NotifyScheduleChanged wakes the scheduler so it recomputes the next run.
func (s *Service) NotifyScheduleChanged() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Validate checks whether the schedule values are usable.
func (schedule Schedule) Validate() error {
	if schedule.Hour < 0 || schedule.Hour > 23 {
		return errors.New("hora inválida")
	}
	if schedule.Minute < 0 || schedule.Minute > 59 {
		return errors.New("minuto inválido")
	}
	for _, day := range schedule.Weekdays {
		if day < 1 || day > 7 {
			return errors.New("dia da semana inválido")
		}
	}
	if schedule.Enabled && len(schedule.Weekdays) == 0 {
		return errors.New("selecione ao menos um dia da semana")
	}
	return nil
}

// Normalized removes duplicate weekdays and sorts them.
func (schedule Schedule) Normalized() Schedule {
	seen := map[int]bool{}
	days := make([]int, 0, len(schedule.Weekdays))
	for _, day := range schedule.Weekdays {
		if !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	sort.Ints(days)
	schedule.Weekdays = days
	return schedule
}

// Next returns the next scheduled run after the given time.
func (schedule Schedule) Next(after time.Time) (time.Time, bool) {
	if !schedule.Enabled || len(schedule.Weekdays) == 0 {
		return time.Time{}, false
	}
	allowed := make(map[int]bool, len(schedule.Weekdays))
	for _, day := range schedule.Weekdays {
		allowed[day] = true
	}
	for offset := 0; offset <= 7; offset++ {
		day := after.AddDate(0, 0, offset)
		candidate := time.Date(day.Year(), day.Month(), day.Day(), schedule.Hour, schedule.Minute, 0, 0, after.Location())
		if !allowed[isoWeekday(candidate.Weekday())] {
			continue
		}
		if candidate.After(after) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func isoWeekday(day time.Weekday) int {
	if day == time.Sunday {
		return 7
	}
	return int(day)
}

func parseWeekdays(value string) ([]int, error) {
	days := make([]int, 0, 7)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		day, err := strconv.Atoi(part)
		if err != nil || day < 1 || day > 7 {
			return nil, fmt.Errorf("dia inválido %q", part)
		}
		days = append(days, day)
	}
	return Schedule{Weekdays: days}.Normalized().Weekdays, nil
}

func weekdaysString(days []int) string {
	parts := make([]string, 0, len(days))
	for _, day := range days {
		parts = append(parts, strconv.Itoa(day))
	}
	return strings.Join(parts, ",")
}

// List returns the backups ordered from newest to oldest.
func (s *Service) List() ([]Entry, error) {
	entries, err := os.ReadDir(s.cfg.BackupDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !namePattern.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, Entry{
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].Name > result[j].Name
		}
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

// Create writes a zip archive with the database dump and the local uploads,
// then applies the retention policy.
func (s *Service) Create(ctx context.Context) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.cfg.BackupDir, 0o750); err != nil {
		return Entry{}, fmt.Errorf("create backup directory: %w", err)
	}
	dumpBin, err := resolveBinary(s.cfg.MySQLDumpBin, "mysqldump", "MYSQLDUMP_BIN")
	if err != nil {
		return Entry{}, err
	}

	name := s.uniqueName()
	finalPath := filepath.Join(s.cfg.BackupDir, name)
	tempPath := finalPath + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return Entry{}, fmt.Errorf("create backup file: %w", err)
	}
	success := false
	defer func() {
		_ = file.Close()
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	archive := zip.NewWriter(file)
	uploadsIncluded := s.uploadsAvailable()
	remoteIncluded := s.remote != nil && s.remote.Provider() != "local"

	meta := manifest{
		App:         "documentation-center",
		Format:      formatV,
		CreatedAt:   s.now().UTC(),
		Database:    s.cfg.DatabaseName,
		HasUploads:  uploadsIncluded,
		HasRemote:   remoteIncluded,
		Description: "Backup do banco de dados, uploads locais e anexos remotos.",
	}
	metaWriter, err := archive.Create("backup.json")
	if err != nil {
		return Entry{}, fmt.Errorf("create manifest: %w", err)
	}
	encoder := json.NewEncoder(metaWriter)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		return Entry{}, fmt.Errorf("write manifest: %w", err)
	}

	dumpWriter, err := archive.Create("database.sql")
	if err != nil {
		return Entry{}, fmt.Errorf("create database entry: %w", err)
	}
	if err := s.dumpDatabase(ctx, dumpBin, dumpWriter); err != nil {
		return Entry{}, err
	}

	if uploadsIncluded {
		if err := s.addUploads(archive); err != nil {
			return Entry{}, fmt.Errorf("add uploads: %w", err)
		}
	}
	if remoteIncluded {
		if err := s.addRemoteObjects(ctx, archive); err != nil {
			return Entry{}, fmt.Errorf("add remote attachments: %w", err)
		}
	}

	if err := archive.Close(); err != nil {
		return Entry{}, fmt.Errorf("finalize archive: %w", err)
	}
	if err := file.Close(); err != nil {
		return Entry{}, fmt.Errorf("close backup file: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return Entry{}, fmt.Errorf("publish backup file: %w", err)
	}
	success = true

	info, err := os.Stat(finalPath)
	if err != nil {
		return Entry{}, fmt.Errorf("stat backup file: %w", err)
	}
	entry := Entry{Name: name, Size: info.Size(), CreatedAt: info.ModTime()}
	s.prune()
	return entry, nil
}

// Import validates and stores an external backup archive in the backup
// directory. The archive must contain at least a non-empty database.sql.
func (s *Service) Import(_ context.Context, source io.Reader, filename string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.cfg.BackupDir, 0o750); err != nil {
		return Entry{}, fmt.Errorf("create backup directory: %w", err)
	}
	temp, err := os.CreateTemp(s.cfg.BackupDir, "import-*.zip.tmp")
	if err != nil {
		return Entry{}, fmt.Errorf("create temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(temp, source); err != nil {
		return Entry{}, fmt.Errorf("receive backup file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return Entry{}, fmt.Errorf("close temporary file: %w", err)
	}
	if err := validateArchive(tempPath); err != nil {
		return Entry{}, err
	}

	name := s.importName(filename)
	finalPath := filepath.Join(s.cfg.BackupDir, name)
	if err := os.Rename(tempPath, finalPath); err != nil {
		return Entry{}, fmt.Errorf("store backup file: %w", err)
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		return Entry{}, fmt.Errorf("stat backup file: %w", err)
	}
	entry := Entry{Name: name, Size: info.Size(), CreatedAt: info.ModTime()}
	s.prune()
	return entry, nil
}

func (s *Service) importName(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if namePattern.MatchString(base) {
		if _, err := os.Stat(filepath.Join(s.cfg.BackupDir, base)); errors.Is(err, fs.ErrNotExist) {
			return base
		}
	}
	return s.uniqueName()
}

func validateArchive(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidArchive, err)
	}
	defer reader.Close()

	hasDatabase := false
	for _, file := range reader.File {
		if !safeArchivePath(file.Name) {
			return fmt.Errorf("%w: caminho inválido %q", ErrInvalidArchive, file.Name)
		}
		if file.Name == "database.sql" && !file.FileInfo().IsDir() {
			if file.UncompressedSize64 == 0 {
				return fmt.Errorf("%w: database.sql vazio", ErrInvalidArchive)
			}
			hasDatabase = true
		}
	}
	if !hasDatabase {
		return fmt.Errorf("%w: database.sql não encontrado", ErrInvalidArchive)
	}
	return nil
}

func safeArchivePath(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}

func (s *Service) dumpDatabase(ctx context.Context, binary string, target io.Writer) error {
	args := []string{
		"--host=" + s.cfg.DatabaseHost,
		fmt.Sprintf("--port=%d", s.cfg.DatabasePort),
		"--user=" + s.cfg.DatabaseUser,
		"--protocol=tcp",
		"--single-transaction",
		"--quick",
		"--skip-lock-tables",
		"--no-tablespaces",
		"--default-character-set=utf8mb4",
		"--routines",
		"--triggers",
		s.cfg.DatabaseName,
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Env = s.commandEnv()
	var stderr bytes.Buffer
	command.Stderr = &stderr
	command.Stdout = target
	if err := command.Run(); err != nil {
		return fmt.Errorf("mysqldump failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *Service) addUploads(archive *zip.Writer) error {
	root := s.cfg.UploadDir
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		writer, err := archive.Create("uploads/" + filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, source)
		closeErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func (s *Service) addRemoteObjects(ctx context.Context, archive *zip.Writer) error {
	return s.remote.ExportObjects(ctx, func(key string, body io.Reader) error {
		key = strings.TrimPrefix(strings.TrimSpace(key), "/")
		if key == "" || strings.Contains(key, "..") {
			return fmt.Errorf("invalid remote object key %q", key)
		}
		writer, err := archive.Create("objects/" + key)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, body)
		return err
	})
}

// Path validates a backup name and returns its absolute path.
func (s *Service) Path(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name != filepath.Base(name) || !namePattern.MatchString(name) {
		return "", ErrNotFound
	}
	path := filepath.Join(s.cfg.BackupDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", ErrNotFound
	}
	return path, nil
}

// Delete removes a backup file from disk.
func (s *Service) Delete(name string) error {
	path, err := s.Path(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Restore replaces the current database contents and local uploads with the
// contents of the given backup. It requires the mysql client binary.
func (s *Service) Restore(ctx context.Context, name string) error {
	path, err := s.Path(name)
	if err != nil {
		return err
	}
	binary, err := resolveBinary(s.cfg.MySQLBin, "mysql", "MYSQL_BIN")
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	defer reader.Close()

	sqlEntry := findZipFile(reader.File, "database.sql")
	if sqlEntry == nil {
		return fmt.Errorf("backup does not contain database.sql")
	}
	temp, err := os.CreateTemp("", "documentation-center-restore-*.sql")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}()

	source, err := sqlEntry.Open()
	if err != nil {
		return fmt.Errorf("read database dump: %w", err)
	}
	_, copyErr := io.Copy(temp, source)
	closeErr := source.Close()
	if copyErr != nil {
		return fmt.Errorf("copy database dump: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close database dump: %w", closeErr)
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind database dump: %w", err)
	}

	command := exec.CommandContext(ctx, binary,
		"--host="+s.cfg.DatabaseHost,
		fmt.Sprintf("--port=%d", s.cfg.DatabasePort),
		"--user="+s.cfg.DatabaseUser,
		"--protocol=tcp",
		"--default-character-set=utf8mb4",
		"--database="+s.cfg.DatabaseName,
	)
	command.Env = s.commandEnv()
	var stderr bytes.Buffer
	command.Stderr = &stderr
	command.Stdin = temp
	if err := command.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	if err := s.restoreUploads(reader.File); err != nil {
		return fmt.Errorf("restore uploads: %w", err)
	}
	if err := s.restoreObjects(ctx, reader.File); err != nil {
		return fmt.Errorf("restore attachments: %w", err)
	}
	return nil
}

func (s *Service) restoreObjects(ctx context.Context, files []*zip.File) error {
	for _, file := range files {
		if !strings.HasPrefix(file.Name, "objects/") || file.FileInfo().IsDir() {
			continue
		}
		key := strings.TrimPrefix(file.Name, "objects/")
		if key == "" || strings.Contains(key, "..") {
			return fmt.Errorf("invalid attachment path %q", file.Name)
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		if s.remote != nil {
			err = s.remote.ImportObject(ctx, key, source)
		} else {
			err = writeLocalFile(s.cfg.UploadDir, strings.TrimPrefix(key, "uploads/"), source)
		}
		_ = source.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func writeLocalFile(root, relative string, body io.Reader) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(relative)))
	if err != nil {
		return err
	}
	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) {
		return fmt.Errorf("invalid attachment path %q", relative)
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return err
	}
	destination, err := os.Create(absTarget)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, body)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (s *Service) restoreUploads(files []*zip.File) error {
	root, err := filepath.Abs(s.cfg.UploadDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Name, "uploads/") || file.FileInfo().IsDir() {
			continue
		}
		relative := strings.TrimPrefix(file.Name, "uploads/")
		target := filepath.Join(root, filepath.FromSlash(relative))
		cleanTarget, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if cleanTarget != root && !strings.HasPrefix(cleanTarget, root+string(os.PathSeparator)) {
			return fmt.Errorf("invalid upload path %q", file.Name)
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		destination, err := os.Create(cleanTarget)
		if err != nil {
			_ = source.Close()
			return err
		}
		_, copyErr := io.Copy(destination, source)
		closeErr := destination.Close()
		_ = source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

// RunScheduler creates backups according to the persisted schedule until the
// context is cancelled. Schedule changes wake the loop immediately.
func (s *Service) RunScheduler(ctx context.Context) {
	for {
		schedule := s.Schedule()
		next, ok := schedule.Next(s.now())
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-s.wake:
				continue
			}
		}
		log.Printf("backup: automatic backups enabled, next run at %s", next.Format(time.RFC3339))
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.wake:
			timer.Stop()
			continue
		case <-timer.C:
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
			entry, err := s.Create(runCtx)
			cancel()
			if err != nil {
				log.Printf("backup: automatic backup failed: %v", err)
			} else {
				log.Printf("backup: automatic backup created %s (%d bytes)", entry.Name, entry.Size)
			}
		}
	}
}

// NextRun returns the next scheduled automatic backup time.
func (s *Service) NextRun() time.Time {
	if next, ok := s.Schedule().Next(s.now()); ok {
		return next
	}
	return time.Time{}
}

func (s *Service) prune() {
	entries, err := s.List()
	if err != nil {
		log.Printf("backup: retention listing failed: %v", err)
		return
	}
	for index, entry := range entries {
		if index < s.cfg.BackupRetention {
			continue
		}
		if err := os.Remove(filepath.Join(s.cfg.BackupDir, entry.Name)); err != nil {
			log.Printf("backup: could not remove %s: %v", entry.Name, err)
		}
	}
}

func (s *Service) uniqueName() string {
	base := filePrefix + s.now().Format("20060102-150405")
	name := base + ".zip"
	if _, err := os.Stat(filepath.Join(s.cfg.BackupDir, name)); errors.Is(err, fs.ErrNotExist) {
		return name
	}
	for attempt := 0; attempt < 10; attempt++ {
		suffix := make([]byte, 2)
		if _, err := rand.Read(suffix); err != nil {
			break
		}
		name = base + "-" + hex.EncodeToString(suffix) + ".zip"
		if _, err := os.Stat(filepath.Join(s.cfg.BackupDir, name)); errors.Is(err, fs.ErrNotExist) {
			return name
		}
	}
	return base + "-" + fmt.Sprintf("%d", s.now().UnixNano()%10000) + ".zip"
}

func (s *Service) uploadsAvailable() bool {
	info, err := os.Stat(s.cfg.UploadDir)
	return err == nil && info.IsDir()
}

func (s *Service) commandEnv() []string {
	env := os.Environ()
	if s.cfg.DatabasePassword == "" {
		return env
	}
	return append(env, "MYSQL_PWD="+s.cfg.DatabasePassword)
}

func findZipFile(files []*zip.File, name string) *zip.File {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func resolveBinary(configured, name, envName string) (string, error) {
	if configured = strings.TrimSpace(configured); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured, nil
		}
		return "", fmt.Errorf("%s não encontrado em %q (verifique %s)", name, configured, envName)
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	candidates := []string{
		filepath.Join("/usr/bin", name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/local/mysql/bin", name),
	}
	if runtime.GOOS == "windows" {
		patterns := []string{
			`C:\Program Files\MySQL\MySQL Server *\bin\` + name + `.exe`,
			`C:\Program Files\MariaDB *\bin\` + name + `.exe`,
			`C:\xampp\mysql\bin\` + name + `.exe`,
		}
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(pattern)
			candidates = append(candidates, matches...)
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%s não encontrado no PATH (verifique %s)", name, envName)
}
