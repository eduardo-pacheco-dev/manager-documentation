package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	dir := t.TempDir()
	return config.Config{
		BackupDir:       filepath.Join(dir, "backups"),
		UploadDir:       filepath.Join(dir, "uploads"),
		BackupRetention: 3,
		BackupAuto:      true,
		BackupHour:      3,
	}
}

func TestListPathAndDelete(t *testing.T) {
	cfg := testConfig(t)
	service := New(cfg)

	valid := "backup-20260101-030000.zip"
	if err := os.WriteFile(filepath.Join(cfg.BackupDir, valid), []byte("zip"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.BackupDir, "notes.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	entries, err := service.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != valid {
		t.Fatalf("unexpected entries: %+v", entries)
	}

	if _, err := service.Path(valid); err != nil {
		t.Fatalf("path valid: %v", err)
	}
	for _, name := range []string{"", "../" + valid, "backup-2026.zip", "notes.txt"} {
		if _, err := service.Path(name); err == nil {
			t.Fatalf("expected invalid name %q", name)
		}
	}

	if err := service.Delete(valid); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := service.Delete(valid); err == nil {
		t.Fatal("expected error deleting missing backup")
	}
}

func TestNextRun(t *testing.T) {
	cfg := testConfig(t)
	service := New(cfg)

	service.now = func() time.Time {
		return time.Date(2026, 11, 14, 2, 30, 0, 0, time.Local)
	}
	if next := service.NextRun(); !next.Equal(time.Date(2026, 11, 14, 3, 0, 0, 0, time.Local)) {
		t.Fatalf("expected same-day run, got %s", next)
	}

	service.now = func() time.Time {
		return time.Date(2026, 11, 14, 4, 0, 0, 0, time.Local)
	}
	if next := service.NextRun(); !next.Equal(time.Date(2026, 11, 15, 3, 0, 0, 0, time.Local)) {
		t.Fatalf("expected next-day run, got %s", next)
	}
}

func TestSettingsMarksNextRun(t *testing.T) {
	cfg := testConfig(t)
	cfg.BackupAuto = false
	service := New(cfg)
	if settings := service.Settings(); settings.NextRun != nil {
		t.Fatalf("expected no next run when automatic is disabled, got %s", settings.NextRun)
	}
	cfg.BackupAuto = true
	service = New(cfg)
	if settings := service.Settings(); settings.NextRun == nil {
		t.Fatal("expected a next run when automatic is enabled")
	}
}

func zipFile(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

func TestImport(t *testing.T) {
	cfg := testConfig(t)
	service := New(cfg)

	invalid := zipFile(t, map[string]string{"notes.txt": "sem dump"})
	if _, err := service.Import(context.Background(), bytes.NewReader(invalid), "backup-20260101-030000.zip"); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected invalid archive error, got %v", err)
	}

	unsafe := zipFile(t, map[string]string{"database.sql": "SELECT 1;", "../evil.txt": "x"})
	if _, err := service.Import(context.Background(), bytes.NewReader(unsafe), "backup-20260101-030001.zip"); !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("expected unsafe path error, got %v", err)
	}

	valid := zipFile(t, map[string]string{
		"database.sql":  "CREATE TABLE exemplo (id INT);",
		"backup.json":   `{"app":"documentation-center"}`,
		"uploads/a.txt": "arquivo",
	})
	entry, err := service.Import(context.Background(), bytes.NewReader(valid), "backup-20260101-030002.zip")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if entry.Name != "backup-20260101-030002.zip" {
		t.Fatalf("expected original name, got %q", entry.Name)
	}
	if _, err := service.Path(entry.Name); err != nil {
		t.Fatalf("imported backup not listed: %v", err)
	}

	duplicate, err := service.Import(context.Background(), bytes.NewReader(valid), entry.Name)
	if err != nil {
		t.Fatalf("import duplicate: %v", err)
	}
	if duplicate.Name == entry.Name {
		t.Fatal("expected a new name when importing a duplicate")
	}
}

type fakeRemote struct {
	provider string
	objects  map[string]string
	imported map[string]string
}

func (f *fakeRemote) Provider() string { return f.provider }

func (f *fakeRemote) ExportObjects(_ context.Context, add func(key string, body io.Reader) error) error {
	for key, content := range f.objects {
		if err := add(key, strings.NewReader(content)); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeRemote) ImportObject(_ context.Context, key string, body io.Reader) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.imported[key] = string(raw)
	return nil
}

func openArchive(t *testing.T, files map[string]string) *zip.ReadCloser {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.zip")
	if err := os.WriteFile(path, zipFile(t, files), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return reader
}

func TestRestoreRemoteObjects(t *testing.T) {
	cfg := testConfig(t)
	service := New(cfg)
	remote := &fakeRemote{provider: "aws", imported: map[string]string{}}
	service.SetRemote(remote)

	reader := openArchive(t, map[string]string{
		"database.sql":                    "SELECT 1;",
		"objects/demands/2026/01/a.txt":   "anexo",
		"objects/uploads/2026/01/img.png": "imagem",
	})
	if err := service.restoreObjects(context.Background(), reader.File); err != nil {
		t.Fatalf("restore objects: %v", err)
	}
	if remote.imported["demands/2026/01/a.txt"] != "anexo" {
		t.Fatalf("missing demand attachment: %v", remote.imported)
	}
	if remote.imported["uploads/2026/01/img.png"] != "imagem" {
		t.Fatalf("missing image: %v", remote.imported)
	}
}

func TestRestoreObjectsFallsBackToLocalDir(t *testing.T) {
	cfg := testConfig(t)
	service := New(cfg)

	reader := openArchive(t, map[string]string{
		"database.sql":                    "SELECT 1;",
		"objects/demands/2026/01/a.txt":   "anexo",
		"objects/uploads/2026/01/img.png": "imagem",
	})
	if err := service.restoreObjects(context.Background(), reader.File); err != nil {
		t.Fatalf("restore objects: %v", err)
	}
	for path, expected := range map[string]string{
		filepath.Join(cfg.UploadDir, "demands", "2026", "01", "a.txt"): "anexo",
		filepath.Join(cfg.UploadDir, "2026", "01", "img.png"):          "imagem",
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(raw) != expected {
			t.Fatalf("unexpected content in %s: %q", path, raw)
		}
	}
}

type fakeConfigStore struct {
	values map[string]string
}

func (f *fakeConfigStore) GetSetting(key string) (string, error) {
	if value, ok := f.values[key]; ok {
		return value, nil
	}
	return "", errors.New("not found")
}

func (f *fakeConfigStore) SetSetting(key, value string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = value
	return nil
}

func TestScheduleNextUsesSelectedWeekdays(t *testing.T) {
	location := time.FixedZone("test", -3*60*60)
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, location) // sabado
	schedule := Schedule{Enabled: true, Hour: 3, Minute: 30, Weekdays: []int{1, 3}}
	next, ok := schedule.Next(now)
	if !ok {
		t.Fatal("expected a next run")
	}
	expected := time.Date(2026, 11, 16, 3, 30, 0, 0, location) // segunda
	if !next.Equal(expected) {
		t.Fatalf("next = %s, esperado %s", next, expected)
	}
	if _, ok := (Schedule{Enabled: false, Weekdays: []int{1}}).Next(now); ok {
		t.Fatal("expected no run when disabled")
	}
}

func TestUpdateSchedulePersists(t *testing.T) {
	cfg := testConfig(t)
	service := New(cfg)
	store := &fakeConfigStore{values: map[string]string{}}
	service.SetConfigStore(store)

	schedule := Schedule{Enabled: true, Hour: 22, Minute: 15, Weekdays: []int{5, 1, 3, 1}}
	if err := service.UpdateSchedule(schedule); err != nil {
		t.Fatalf("update schedule: %v", err)
	}
	if store.values[scheduleHourKey] != "22" || store.values[scheduleMinuteKey] != "15" {
		t.Fatalf("unexpected persisted time: %v", store.values)
	}
	if store.values[scheduleWeekdaysKey] != "1,3,5" {
		t.Fatalf("weekdays not normalized: %q", store.values[scheduleWeekdaysKey])
	}
	loaded := service.Schedule()
	if !loaded.Enabled || loaded.Hour != 22 || loaded.Minute != 15 || len(loaded.Weekdays) != 3 {
		t.Fatalf("unexpected loaded schedule: %+v", loaded)
	}
	if err := (Schedule{Enabled: true, Hour: 3, Minute: 0}).Validate(); err == nil {
		t.Fatal("expected validation error without weekdays")
	}
}
