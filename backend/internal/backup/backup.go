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
	NextRun    *time.Time `json:"next_run,omitempty"`
	UploadsDir string     `json:"uploads_dir"`
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

// Service owns the backup directory and the external MySQL client binaries.
type Service struct {
	cfg    config.Config
	now    func() time.Time
	mu     sync.Mutex
	remote RemoteStore
}

// New creates the service and ensures the backup directory exists.
func New(cfg config.Config) *Service {
	service := &Service{cfg: cfg, now: time.Now}
	if err := os.MkdirAll(cfg.BackupDir, 0o750); err != nil {
		log.Printf("backup: could not create directory %s: %v", cfg.BackupDir, err)
	}
	return service
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
	settings := Settings{
		Directory:  dir,
		Retention:  s.cfg.BackupRetention,
		Automatic:  s.cfg.BackupAuto,
		Hour:       s.cfg.BackupHour,
		UploadsDir: uploads,
	}
	if s.cfg.BackupAuto {
		next := s.NextRun()
		settings.NextRun = &next
	}
	return settings
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

// RunScheduler creates one backup per day at the configured local hour until
// the context is cancelled.
func (s *Service) RunScheduler(ctx context.Context) {
	if !s.cfg.BackupAuto {
		log.Printf("backup: automatic backups disabled")
		return
	}
	log.Printf("backup: automatic backups enabled, next run at %s", s.NextRun().Format(time.RFC3339))
	for {
		timer := time.NewTimer(time.Until(s.NextRun()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
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
	now := s.now()
	next := time.Date(now.Year(), now.Month(), now.Day(), s.cfg.BackupHour, 0, 0, 0, time.Local)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
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
