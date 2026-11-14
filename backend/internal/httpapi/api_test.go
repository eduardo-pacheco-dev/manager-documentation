package httpapi_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/crypto"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/database"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/httpapi"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/models"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/seed"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

type testApp struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client
	store  *store.Store
	csrf   string
	userID int64
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	dir := t.TempDir()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Env = "test"
	cfg.Addr = ":0"
	cfg.DatabaseName = fmt.Sprintf("documentation_center_test_%d", time.Now().UnixNano())
	cfg.UploadDir = filepath.Join(dir, "uploads")
	cfg.BackupDir = filepath.Join(dir, "backups")
	cfg.StaticDir = filepath.Join(dir, "static")
	cfg.PublicURL = "http://example.test"
	cfg.AllowedOrigins = []string{"http://example.test"}
	cfg.SessionTTL = time.Hour
	cfg.UploadMaxMB = 20
	cfg.AppKey = "test-key"
	cfg.SeedAdminName = "Administrador"
	cfg.SeedAdminEmail = "admin@test.local"
	cfg.SeedAdminPass = "password"
	cfg.SeedDemoData = true
	cfg.Timezone = "UTC"

	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repository := store.New(db)
	if err := seed.Run(repository, cfg); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cipher, err := crypto.New(cfg.AppKey)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	handler := httpapi.New(cfg, repository, cipher)
	server := httptest.NewServer(handler.Router())
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		_ = db.Close()
		if err := database.DropDatabase(cfg, cfg.DatabaseName); err != nil {
			t.Logf("drop test database: %v", err)
		}
	})
	jar, _ := cookiejar.New(nil)
	app := &testApp{t: t, server: server, client: &http.Client{Jar: jar}, store: repository}
	app.login("admin@test.local", "password")
	return app
}

func (a *testApp) request(method, path string, body any, csrf string, headers map[string]string) (*http.Response, map[string]any) {
	a.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			a.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, a.server.URL+path, reader)
	if err != nil {
		a.t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := a.client.Do(req)
	if err != nil {
		a.t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	payload := map[string]any{}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 && strings.Contains(res.Header.Get("Content-Type"), "application/json") {
		_ = json.Unmarshal(raw, &payload)
	}
	return res, payload
}

func (a *testApp) login(email, password string) {
	a.t.Helper()
	res, payload := a.request(http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email": email, "password": password,
	}, "", nil)
	if res.StatusCode != http.StatusOK {
		a.t.Fatalf("login status = %d payload = %v", res.StatusCode, payload)
	}
	data := payload["data"].(map[string]any)
	a.csrf = data["csrf_token"].(string)
	a.userID = int64(data["user"].(map[string]any)["id"].(float64))
}

func TestAuthenticationAndCSRF(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodGet, "/api/v1/auth/me", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d", res.StatusCode)
	}
	data := payload["data"].(map[string]any)
	if data["user"].(map[string]any)["email"] != "admin@test.local" {
		t.Fatalf("unexpected user payload: %v", data["user"])
	}
	if len(data["modules"].([]any)) != 12 {
		t.Fatalf("expected 12 modules, got %v", data["modules"])
	}

	res, _ = app.request(http.MethodPost, "/api/v1/users", map[string]any{
		"name": "Sem CSRF", "email": "nocsrf@test.local", "password": "password", "password_confirmation": "password",
	}, "", nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without CSRF, got %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email": "admin@test.local", "password": "wrong-password",
	}, "", nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", res.StatusCode)
	}
}

func TestUserManagement(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/users", map[string]any{
		"name": "Nova", "surname": "Pessoa", "email": "nova@test.local",
		"password": "password123", "password_confirmation": "password123", "is_admin": false,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d payload = %v", res.StatusCode, payload)
	}
	created := payload["data"].(map[string]any)["user"].(map[string]any)
	id := int64(created["id"].(float64))

	res, payload = app.request(http.MethodGet, "/api/v1/users?search=Nova&sort=name&direction=asc", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", res.StatusCode)
	}
	if int(payload["meta"].(map[string]any)["total"].(float64)) != 1 {
		t.Fatalf("expected one result, got %v", payload["meta"])
	}

	res, _ = app.request(http.MethodPut, fmt.Sprintf("/api/v1/users/%d", id), map[string]any{
		"name": "Nova", "surname": "Pessoa", "email": "nova2@test.local",
		"password": "", "password_confirmation": "", "is_admin": true,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/users/%d/toggle", id), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("toggle status = %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/users/%d/toggle", app.userID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("self toggle should fail, got %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", id), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", res.StatusCode)
	}
}

func TestDevDocs(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/dev-docs", map[string]any{
		"title":        "Documento de Teste",
		"category":     "testing",
		"summary":      "Resumo",
		"content":      "# Título\n\n<script>alert('xss')</script>\n\nConteúdo.",
		"is_published": true,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create doc status = %d payload = %v", res.StatusCode, payload)
	}
	doc := payload["data"].(map[string]any)["document"].(map[string]any)
	id := int64(doc["id"].(float64))
	slug := doc["slug"].(string)

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/dev-docs/%d", id), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("show doc status = %d", res.StatusCode)
	}
	html := payload["data"].(map[string]any)["html"].(string)
	if strings.Contains(html, "<script>") {
		t.Fatalf("markdown output was not sanitized: %s", html)
	}

	res, _ = app.request(http.MethodPut, fmt.Sprintf("/api/v1/dev-docs/%d", id), map[string]any{
		"title":        "Documento Renomeado",
		"category":     "api",
		"summary":      "Novo resumo",
		"content":      "Atualizado",
		"is_published": false,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update doc status = %d", res.StatusCode)
	}
	updated, _ := app.store.FindDoc(id)
	if updated.Slug != slug {
		t.Fatalf("slug changed on update: %s != %s", updated.Slug, slug)
	}

	res, payload = app.request(http.MethodGet, "/api/v1/dev-docs?search=Renomeado&category=api", nil, "", nil)
	if res.StatusCode != http.StatusOK || payload["data"].([]any) == nil || len(payload["data"].([]any)) != 1 {
		t.Fatalf("search failed: %v", payload)
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/dev-docs/%d", id), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete doc status = %d", res.StatusCode)
	}
}

func TestDashboardPreferences(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPut, "/api/v1/dashboard/widgets/core.users", map[string]any{"hidden": true}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("hide widget status = %d", res.StatusCode)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/dashboard", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dashboard status = %d", res.StatusCode)
	}
	hidden := payload["data"].(map[string]any)["hidden"].([]any)
	if len(hidden) != 1 || hidden[0] != "core.users" {
		t.Fatalf("unexpected hidden widgets: %v", hidden)
	}

	res, payload = app.request(http.MethodPut, "/api/v1/dashboard/widgets/order", map[string]any{
		"order": []string{"core.users.recent", "core.users"},
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("order status = %d", res.StatusCode)
	}
	order := payload["data"].(map[string]any)["order"].([]any)
	if order[0] != "core.users.recent" {
		t.Fatalf("unexpected order: %v", order)
	}

	res, _ = app.request(http.MethodPut, "/api/v1/dashboard/widgets/core.users/layout", map[string]any{"span": 2}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("layout status = %d", res.StatusCode)
	}
	res, _ = app.request(http.MethodPut, "/api/v1/dashboard/widgets/core.users/layout", map[string]any{"span": 4}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid span should be 422, got %d", res.StatusCode)
	}
}

func TestNotifications(t *testing.T) {
	app := newTestApp(t)
	notification := models.Notification{
		ID:        "00000000-0000-0000-0000-000000000001",
		UserID:    app.userID,
		Title:     "Aviso",
		Body:      "Corpo",
		Level:     "info",
		CreatedAt: time.Now().UTC(),
	}
	if err := app.store.CreateNotification(&notification); err != nil {
		t.Fatalf("create notification: %v", err)
	}

	res, payload := app.request(http.MethodGet, "/api/v1/notifications", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["unread"].(float64)) < 1 {
		t.Fatalf("notifications list: %d %v", res.StatusCode, payload)
	}

	res, _ = app.request(http.MethodPatch, "/api/v1/notifications/"+notification.ID+"/read", nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("mark read status = %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodDelete, "/api/v1/notifications", nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("clear status = %d", res.StatusCode)
	}
}

func TestAPITokenAndUpload(t *testing.T) {
	app := newTestApp(t)
	res, payload := app.request(http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email": "admin@test.local", "password": "password", "device_name": "test-cli",
	}, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("token login status = %d", res.StatusCode)
	}
	token := payload["data"].(map[string]any)["token"].(string)
	if token == "" {
		t.Fatal("expected API token")
	}

	req, _ := http.NewRequest(http.MethodGet, app.server.URL+"/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	authenticated := &http.Client{}
	resp, err := authenticated.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("bearer auth failed: %v %v", err, resp)
	}
	_ = resp.Body.Close()

	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, _ := writer.CreateFormFile("file", "pixel.png")
	_, _ = part.Write(png)
	_ = writer.Close()

	req, _ = http.NewRequest(http.MethodPost, app.server.URL+"/api/v1/uploads", &buffer)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = authenticated.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload failed: %v %d %s", err, resp.StatusCode, raw)
	}
	_ = resp.Body.Close()
}

func TestInactiveUserCannotLogin(t *testing.T) {
	app := newTestApp(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{
		Name: "Inativo", Email: "inativo@test.local", IsActive: true, Theme: "system",
		PasswordHash: string(hash), EmailNotifications: true,
	}
	if err := app.store.CreateUser(&user); err != nil {
		t.Fatalf("create inactive user: %v", err)
	}
	user.IsActive = false
	if err := app.store.UpdateUser(&user); err != nil {
		t.Fatalf("update inactive user: %v", err)
	}
	client := &http.Client{}
	raw, _ := json.Marshal(map[string]any{"email": "inativo@test.local", "password": "password"})
	req, _ := http.NewRequest(http.MethodPost, app.server.URL+"/api/v1/auth/login", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("inactive login status = %d", resp.StatusCode)
	}
}

func TestNativeModules(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodGet, "/api/v1/tasks", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("tasks index status = %d", res.StatusCode)
	}
	res, payload = app.request(http.MethodPost, "/api/v1/tasks/lists", map[string]any{
		"name": "Trabalho", "color": "blue", "icon": "check-circle",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create list status = %d payload=%v", res.StatusCode, payload)
	}
	listID := int64(payload["data"].(map[string]any)["list"].(map[string]any)["id"].(float64))
	res, payload = app.request(http.MethodPost, "/api/v1/tasks", map[string]any{
		"title": "Tarefa teste", "todo_list_id": listID, "is_important": true,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create task status = %d payload=%v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodGet, "/api/v1/agenda", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("agenda status = %d", res.StatusCode)
	}
	calendars := payload["data"].(map[string]any)["calendars"].([]any)
	res, payload = app.request(http.MethodPost, "/api/v1/agenda/events", map[string]any{
		"calendar_id": calendars[0].(map[string]any)["id"], "title": "Evento teste",
		"start_at": "2026-11-12T09:00", "end_at": "2026-11-12T10:00",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create event status = %d payload=%v", res.StatusCode, payload)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/agenda", nil, "", nil)
	if res.StatusCode != http.StatusOK || len(payload["data"].(map[string]any)["occurrences"].([]any)) == 0 {
		t.Fatalf("agenda occurrence missing: %v", payload)
	}

	res, payload = app.request(http.MethodPost, "/api/v1/follow-ups", map[string]any{
		"title": "Follow-up teste", "status": "pendente", "priority": "alta",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create follow-up status = %d payload=%v", res.StatusCode, payload)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/follow-ups", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["meta"].(map[string]any)["total"].(float64)) != 1 {
		t.Fatalf("follow-ups list failed: %d %v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodPost, "/api/v1/stations", map[string]any{
		"site_id": "SPTESTE001", "municipio": "São Paulo", "estado": "SP",
		"latitude": "-23.5", "longitude": "-46.6",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create station status = %d payload=%v", res.StatusCode, payload)
	}
	stationID := int64(payload["data"].(map[string]any)["station"].(map[string]any)["id"].(float64))
	for _, path := range []string{"/api/v1/stations/export", "/api/v1/stations/template", fmt.Sprintf("/api/v1/stations/%d/export", stationID)} {
		res, _ := app.request(http.MethodGet, path, nil, "", nil)
		if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "spreadsheetml") {
			t.Fatalf("%s export failed: %d %q", path, res.StatusCode, res.Header.Get("Content-Type"))
		}
	}

	// Cross-module integration: quick create linked work items and read the
	// related work context back.
	workLinkIDs := []int64{}
	for _, item := range []map[string]any{
		{"kind": "follow_up", "title": "Vistoria vinculada", "priority": "alta", "due_date": "2026-11-20"},
		{"kind": "task", "title": "Conferir documentação", "due_date": "2026-11-18"},
		{"kind": "agenda_event", "title": "Visita técnica", "start_at": "2026-11-19T09:00", "duration_minutes": 90},
	} {
		item["resource_type"] = "station"
		item["resource_id"] = stationID
		res, payload = app.request(http.MethodPost, "/api/v1/work-items", item, app.csrf, nil)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("work item %v status = %d payload=%v", item["kind"], res.StatusCode, payload)
		}
		link := payload["data"].(map[string]any)["link"].(map[string]any)
		workLinkIDs = append(workLinkIDs, int64(link["id"].(float64)))
	}
	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/work-context?resource_type=station&resource_id=%d", stationID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("work context status = %d payload=%v", res.StatusCode, payload)
	}
	context := payload["data"].(map[string]any)
	if len(context["follow_ups"].([]any)) != 1 || len(context["tasks"].([]any)) != 1 || len(context["events"].([]any)) != 1 {
		t.Fatalf("work context missing linked items: %v", context)
	}
	res, payload = app.request(http.MethodPost, "/api/v1/follow-ups", map[string]any{
		"title": "Follow-up vinculado manualmente", "status": "pendente", "priority": "media",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create linked follow-up status = %d payload=%v", res.StatusCode, payload)
	}
	followUpID := int64(payload["data"].(map[string]any)["follow_up"].(map[string]any)["id"].(float64))
	res, payload = app.request(http.MethodPost, "/api/v1/work-links", map[string]any{
		"module": "follow_up", "entity_id": followUpID, "resource_type": "station", "resource_id": stationID,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("manual work link status = %d payload=%v", res.StatusCode, payload)
	}
	workLinkIDs = append(workLinkIDs, int64(payload["data"].(map[string]any)["link"].(map[string]any)["id"].(float64)))
	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/work-links/%d", workLinkIDs[len(workLinkIDs)-1]), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete work link status = %d", res.StatusCode)
	}

	res, payload = app.request(http.MethodPost, "/api/v1/nokia/sites", map[string]any{
		"site_id": "NKTESTE01", "oc": "OC-TESTE-01", "regional": "TSP",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create nokia site status = %d payload=%v", res.StatusCode, payload)
	}
	res, optionPayload := app.request(http.MethodGet, "/api/v1/nokia/station-sites", nil, "", nil)
	if res.StatusCode != http.StatusOK || len(optionPayload["data"].([]any)) == 0 {
		t.Fatalf("station site options missing: %d %v", res.StatusCode, optionPayload)
	}
	siteID := int64(payload["data"].(map[string]any)["site"].(map[string]any)["id"].(float64))
	res, payload = app.request(http.MethodGet, "/api/v1/stations?search=NKTESTE01", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["meta"].(map[string]any)["total"].(float64)) != 1 {
		t.Fatalf("nokia site was not synced to stations: %d %v", res.StatusCode, payload)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/follow-ups?search=NKTESTE01&per_page=50", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["meta"].(map[string]any)["total"].(float64)) != 3 {
		t.Fatalf("nokia checklist follow-ups missing: %d %v", res.StatusCode, payload)
	}
	followUpTitles := map[string]bool{}
	for _, item := range payload["data"].([]any) {
		followUpTitles[item.(map[string]any)["title"].(string)] = true
	}
	for _, title := range []string{"NKTESTE01 - FAM Entrega", "NKTESTE01 - FAM Instalação", "NKTESTE01 - FAM Panorâmica"} {
		if !followUpTitles[title] {
			t.Fatalf("missing follow-up %q: %v", title, followUpTitles)
		}
	}
	res, _ = app.request(http.MethodPut, fmt.Sprintf("/api/v1/nokia/sites/%d/checklist/checklist_fam_entrega", siteID), map[string]any{"status": "analysis"}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("checklist status = %d", res.StatusCode)
	}
	res, payload = app.request(http.MethodPut, fmt.Sprintf("/api/v1/nokia/sites/%d/dates", siteID), map[string]any{
		"fam_instalacao_approved_at": "2026-11-01",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("nokia dates status = %d", res.StatusCode)
	}
	site := payload["data"].(map[string]any)["site"].(map[string]any)
	if site["checklist_fam_instalacao_status"] != "sent" || site["checklist_fam_instalacao"] != true {
		t.Fatalf("approval date did not sync checklist: %v", site)
	}
	res, agendaPayload := app.request(http.MethodGet, "/api/v1/agenda?date=2026-11-01&view=month", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("agenda status = %d payload=%v", res.StatusCode, agendaPayload)
	}
	agendaData := agendaPayload["data"].(map[string]any)
	dedicated := false
	for _, calendar := range agendaData["calendars"].([]any) {
		if calendar.(map[string]any)["name"] == "Projetos Nokia" {
			dedicated = true
			break
		}
	}
	if !dedicated {
		t.Fatalf("dedicated Nokia calendar missing: %v", agendaData["calendars"])
	}
	eventFound := false
	for _, occurrence := range agendaData["occurrences"].([]any) {
		if occurrence.(map[string]any)["title"] == "NKTESTE01 - FAM Instalação - Aprovado" {
			eventFound = true
			break
		}
	}
	if !eventFound {
		t.Fatalf("Nokia date event missing from agenda: %v", agendaData["occurrences"])
	}
	for _, path := range []string{"/api/v1/nokia/sites/template", "/api/v1/nokia/sites/export", "/api/v1/nokia/sites/export/pending"} {
		res, _ := app.request(http.MethodGet, path, nil, "", nil)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d", path, res.StatusCode)
		}
		if !strings.Contains(res.Header.Get("Content-Type"), "spreadsheetml") {
			t.Fatalf("%s content-type = %q", path, res.Header.Get("Content-Type"))
		}
	}

	res, payload = app.request(http.MethodGet, "/api/v1/dashboard", nil, "", nil)
	if res.StatusCode != http.StatusOK || len(payload["data"].(map[string]any)["widgets"].([]any)) < 6 {
		t.Fatalf("module widgets missing: %v", payload)
	}
}

func TestShortLinks(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/links", map[string]any{
		"code": "teste-link", "target_url": "https://example.com/docs", "title": "Documentação",
		"is_active": true,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create short link status = %d payload=%v", res.StatusCode, payload)
	}
	link := payload["data"].(map[string]any)["link"].(map[string]any)
	linkID := int64(link["id"].(float64))
	if link["code"] != "teste-link" || link["short_url"] != "http://example.test/s/teste-link" {
		t.Fatalf("unexpected short link: %v", link)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/links", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["meta"].(map[string]any)["total"].(float64)) != 1 {
		t.Fatalf("short links list failed: %d %v", res.StatusCode, payload)
	}

	noRedirect := &http.Client{
		Jar: app.client.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	publicRes, err := noRedirect.Get(app.server.URL + "/s/teste-link")
	if err != nil {
		t.Fatalf("public short link request: %v", err)
	}
	publicRes.Body.Close()
	if publicRes.StatusCode != http.StatusFound || publicRes.Header.Get("Location") != "https://example.com/docs" {
		t.Fatalf("public redirect status = %d location = %q", publicRes.StatusCode, publicRes.Header.Get("Location"))
	}

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/links/%d", linkID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("short link show status = %d payload=%v", res.StatusCode, payload)
	}
	details := payload["data"].(map[string]any)
	link = details["link"].(map[string]any)
	if int64(link["click_count"].(float64)) != 1 || int64(link["unique_clicks"].(float64)) != 1 {
		t.Fatalf("short link click metrics not recorded: %v", link)
	}
	if len(details["clicks"].([]any)) != 1 || len(details["daily"].([]any)) != 1 {
		t.Fatalf("short link analytics missing: %v", details)
	}

	res, _ = app.request(http.MethodGet, fmt.Sprintf("/api/v1/links/%d/qr", linkID), nil, "", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "image/png") {
		t.Fatalf("short link qr status = %d content-type=%q", res.StatusCode, res.Header.Get("Content-Type"))
	}

	var uploadBuffer bytes.Buffer
	uploadWriter := multipart.NewWriter(&uploadBuffer)
	uploadPart, _ := uploadWriter.CreateFormFile("files", "documento.txt")
	_, _ = uploadPart.Write([]byte("conteudo do arquivo"))
	_ = uploadWriter.Close()
	uploadReq, _ := http.NewRequest(http.MethodPost, app.server.URL+fmt.Sprintf("/api/v1/links/%d/files", linkID), &uploadBuffer)
	uploadReq.Header.Set("Content-Type", uploadWriter.FormDataContentType())
	uploadReq.Header.Set("X-CSRF-Token", app.csrf)
	uploadRes, err := app.client.Do(uploadReq)
	if err != nil || uploadRes.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(uploadRes.Body)
		t.Fatalf("short link file upload failed: %v %d %s", err, uploadRes.StatusCode, raw)
	}
	_ = uploadRes.Body.Close()

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/links/%d", linkID), nil, "", nil)
	files := payload["data"].(map[string]any)["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("short link files missing: %v", payload)
	}
	file := files[0].(map[string]any)
	fileID := int64(file["id"].(float64))
	landing, err := noRedirect.Get(app.server.URL + "/s/teste-link")
	if err != nil {
		t.Fatalf("short link landing request: %v", err)
	}
	landingBody, _ := io.ReadAll(landing.Body)
	landing.Body.Close()
	if landing.StatusCode != http.StatusOK || !strings.Contains(string(landingBody), "documento.txt") {
		t.Fatalf("short link landing status = %d body=%s", landing.StatusCode, landingBody)
	}
	download, err := noRedirect.Get(app.server.URL + fmt.Sprintf("/s/teste-link/files/%d", fileID))
	if err != nil {
		t.Fatalf("short link file download: %v", err)
	}
	downloadBody, _ := io.ReadAll(download.Body)
	download.Body.Close()
	if download.StatusCode != http.StatusOK || !strings.Contains(download.Header.Get("Content-Disposition"), "attachment") || string(downloadBody) != "conteudo do arquivo" {
		t.Fatalf("short link file download status = %d disposition=%q body=%q", download.StatusCode, download.Header.Get("Content-Disposition"), downloadBody)
	}

	res, payload = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/links/%d/toggle", linkID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK || payload["data"].(map[string]any)["link"].(map[string]any)["is_active"] != false {
		t.Fatalf("short link toggle failed: %d %v", res.StatusCode, payload)
	}
	publicRes, err = noRedirect.Get(app.server.URL + "/s/teste-link")
	if err != nil {
		t.Fatalf("disabled short link request: %v", err)
	}
	publicRes.Body.Close()
	if publicRes.StatusCode != http.StatusGone {
		t.Fatalf("disabled short link status = %d", publicRes.StatusCode)
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/links/%d", linkID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("short link delete status = %d", res.StatusCode)
	}
}

func TestDemands(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/stations", map[string]any{
		"site_id": "DEMANDA001", "municipio": "São Paulo", "estado": "SP",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create demand station status = %d payload=%v", res.StatusCode, payload)
	}
	stationID := int64(payload["data"].(map[string]any)["station"].(map[string]any)["id"].(float64))
	res, payload = app.request(http.MethodGet, "/api/v1/demands/stations?search=DEMANDA001", nil, "", nil)
	if res.StatusCode != http.StatusOK || len(payload["data"].([]any)) != 1 {
		t.Fatalf("demand station search failed: %d %v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodPost, "/api/v1/demands", map[string]any{
		"assignee_id": app.userID, "title": "Atualizar documentação do site", "description": "Revisar o procedimento.",
		"priority": "alta", "due_at": "2026-11-20", "status": "aberta",
		"client_name": "Cliente Teste", "client_email": "cliente@test.local",
		"station_id": stationID, "operadora": "TIM", "demand_type": "Instalação",
		"oc_tim": "OC-123", "client_os": "OS-456", "key_number": "CHAVE-789",
		"activity_scope": "Instalar equipamentos e validar integração.",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create demand status = %d payload=%v", res.StatusCode, payload)
	}
	demand := payload["data"].(map[string]any)["demand"].(map[string]any)
	demandID := int64(demand["id"].(float64))
	if demand["status"] != "aberta" || demand["assignee_id"].(float64) != float64(app.userID) ||
		demand["client_email"] != "cliente@test.local" || demand["oc_tim"] != "OC-123" ||
		demand["station_site_id"] != "DEMANDA001" {
		t.Fatalf("unexpected demand: %v", demand)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/links?search=Upload", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["meta"].(map[string]any)["total"].(float64)) < 1 {
		t.Fatalf("demand short links missing from module: %d %v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodGet, "/api/v1/demands?scope=mine", nil, "", nil)
	if res.StatusCode != http.StatusOK || int(payload["meta"].(map[string]any)["total"].(float64)) != 1 {
		t.Fatalf("demands list failed: %d %v", res.StatusCode, payload)
	}
	stats := payload["stats"].(map[string]any)
	if int(stats["open"].(float64)) != 1 || int(stats["mine"].(float64)) != 1 {
		t.Fatalf("demand stats incorrect: %v", stats)
	}
	if len(payload["collaborators"].([]any)) == 0 {
		t.Fatalf("demand collaborators missing: %v", payload)
	}

	res, payload = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/demands/%d/status", demandID), map[string]any{"status": "em_andamento"}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("demand status update = %d payload=%v", res.StatusCode, payload)
	}
	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/demands/%d/updates", demandID), map[string]any{"body": "Iniciando a revisão."}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("demand comment status = %d payload=%v", res.StatusCode, payload)
	}

	var uploadBuffer bytes.Buffer
	uploadWriter := multipart.NewWriter(&uploadBuffer)
	uploadPart, _ := uploadWriter.CreateFormFile("files", "evidencia.txt")
	_, _ = uploadPart.Write([]byte("evidencia da demanda"))
	_ = uploadWriter.Close()
	uploadReq, _ := http.NewRequest(http.MethodPost, app.server.URL+fmt.Sprintf("/api/v1/demands/%d/attachments", demandID), &uploadBuffer)
	uploadReq.Header.Set("Content-Type", uploadWriter.FormDataContentType())
	uploadReq.Header.Set("X-CSRF-Token", app.csrf)
	uploadRes, err := app.client.Do(uploadReq)
	if err != nil || uploadRes.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(uploadRes.Body)
		t.Fatalf("demand attachment upload failed: %v %d %s", err, uploadRes.StatusCode, raw)
	}
	_ = uploadRes.Body.Close()

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/demands/%d", demandID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("demand show status = %d payload=%v", res.StatusCode, payload)
	}
	detail := payload["data"].(map[string]any)["demand"].(map[string]any)
	if len(detail["updates"].([]any)) < 2 {
		t.Fatalf("demand history missing: %v", detail["updates"])
	}
	attachments := detail["attachments"].([]any)
	if len(attachments) != 1 {
		t.Fatalf("demand attachment missing: %v", attachments)
	}
	attachmentID := int64(attachments[0].(map[string]any)["id"].(float64))
	download, err := app.client.Get(app.server.URL + fmt.Sprintf("/api/v1/demands/%d/attachments/%d", demandID, attachmentID))
	if err != nil {
		t.Fatalf("demand attachment download: %v", err)
	}
	downloadBody, _ := io.ReadAll(download.Body)
	download.Body.Close()
	if download.StatusCode != http.StatusOK || string(downloadBody) != "evidencia da demanda" {
		t.Fatalf("demand attachment download status = %d body=%q", download.StatusCode, downloadBody)
	}

	publicClient := &http.Client{
		Jar: app.client.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	uploadPath := strings.TrimPrefix(demand["upload_url"].(string), "http://example.test")
	downloadPath := strings.TrimPrefix(demand["download_url"].(string), "http://example.test")
	uploadPage, err := app.client.Get(app.server.URL + uploadPath)
	if err != nil {
		t.Fatalf("demand public upload page: %v", err)
	}
	finalUploadPath := uploadPage.Request.URL.Path
	uploadPageBody, _ := io.ReadAll(uploadPage.Body)
	uploadPage.Body.Close()
	if uploadPage.StatusCode != http.StatusOK || !strings.Contains(string(uploadPageBody), "Enviar arquivos") {
		t.Fatalf("demand public upload page status = %d body=%s", uploadPage.StatusCode, uploadPageBody)
	}

	var publicUpload bytes.Buffer
	publicWriter := multipart.NewWriter(&publicUpload)
	_ = publicWriter.WriteField("name", "Colaborador Externo")
	publicPart, _ := publicWriter.CreateFormFile("files", "evidencia-publica.txt")
	_, _ = publicPart.Write([]byte("arquivo enviado pelo link publico"))
	_ = publicWriter.Close()
	publicReq, _ := http.NewRequest(http.MethodPost, app.server.URL+finalUploadPath, &publicUpload)
	publicReq.Header.Set("Content-Type", publicWriter.FormDataContentType())
	publicRes, err := publicClient.Do(publicReq)
	if err != nil || publicRes.StatusCode != http.StatusSeeOther {
		t.Fatalf("demand public upload failed: %v %d", err, publicRes.StatusCode)
	}
	_ = publicRes.Body.Close()

	downloadPage, err := app.client.Get(app.server.URL + downloadPath)
	if err != nil {
		t.Fatalf("demand public download page: %v", err)
	}
	finalDownloadPath := downloadPage.Request.URL.Path
	downloadPageBody, _ := io.ReadAll(downloadPage.Body)
	downloadPage.Body.Close()
	if downloadPage.StatusCode != http.StatusOK || !strings.Contains(string(downloadPageBody), "evidencia-publica.txt") {
		t.Fatalf("demand public download page status = %d body=%s", downloadPage.StatusCode, downloadPageBody)
	}

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/demands/%d", demandID), nil, "", nil)
	attachments = payload["data"].(map[string]any)["demand"].(map[string]any)["attachments"].([]any)
	var publicFileID int64
	for _, item := range attachments {
		attachment := item.(map[string]any)
		if attachment["original_name"] == "evidencia-publica.txt" {
			publicFileID = int64(attachment["id"].(float64))
			if attachment["uploaded_by_name"] != "Colaborador Externo" {
				t.Fatalf("public uploader name missing: %v", attachment)
			}
		}
	}
	if publicFileID == 0 {
		t.Fatalf("public attachment missing: %v", attachments)
	}
	publicDownload, err := app.client.Get(app.server.URL + fmt.Sprintf("%s/files/%d", finalDownloadPath, publicFileID))
	if err != nil {
		t.Fatalf("demand public file download: %v", err)
	}
	publicDownloadBody, _ := io.ReadAll(publicDownload.Body)
	publicDownload.Body.Close()
	if publicDownload.StatusCode != http.StatusOK || string(publicDownloadBody) != "arquivo enviado pelo link publico" {
		t.Fatalf("demand public file download status = %d body=%q", publicDownload.StatusCode, publicDownloadBody)
	}

	res, _ = app.request(http.MethodGet, "/api/v1/demands/export", nil, "", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(res.Header.Get("Content-Type"), "text/csv") {
		t.Fatalf("demand export status = %d content-type=%q", res.StatusCode, res.Header.Get("Content-Type"))
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/demands/%d", demandID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("demand delete status = %d", res.StatusCode)
	}
}

func TestSettingsAndStorage(t *testing.T) {
	app := newTestApp(t)

	res, _ := app.request(http.MethodPut, "/api/v1/settings/theme", map[string]any{"theme": "dark"}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("theme status = %d", res.StatusCode)
	}
	res, _ = app.request(http.MethodPut, "/api/v1/settings/theme", map[string]any{"theme": "neon"}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid theme should be 422, got %d", res.StatusCode)
	}

	res, payload := app.request(http.MethodGet, "/api/v1/settings", nil, "", nil)
	if res.StatusCode != http.StatusOK || payload["data"].(map[string]any)["storage"] == nil {
		t.Fatalf("settings payload: %v", payload)
	}

	res, _ = app.request(http.MethodPut, "/api/v1/settings/storage", map[string]any{
		"provider": "local", "max_upload_mb": 30,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("storage status = %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodPut, "/api/v1/settings/modules", map[string]any{
		"modules": []string{"agenda"},
		"order":   []string{"dashboard", "agenda", "users", "dev-docs"},
		"sections": map[string]string{
			"dashboard": "sidebar", "agenda": "sidebar", "users": "sidebar", "dev-docs": "sidebar",
		},
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("modules status = %d", res.StatusCode)
	}
}

func TestCollaboratorsAndProjectResources(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/collaborators", map[string]any{
		"name": "Pessoa Teste", "social_name": "Pessoa de Teste", "email": "pessoa.teste@test.local",
		"personal_email": "pessoa.pessoal@test.local", "phone": "+55 11 90000-0000",
		"role": "QA", "department": "Qualidade", "cost_hour": 90, "max_units": 80,
		"color": "green", "is_active": true, "cpf": "999.888.777-66", "birth_date": "1990-01-01",
		"contract_type": "clt", "work_regime": "hibrido", "hire_date": "2024-02-01",
		"weekly_hours": 40, "salary": 8000, "bank_name": "Banco Teste", "pix_key": "pessoa.teste@test.local",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar colaborador status = %d payload = %v", res.StatusCode, payload)
	}
	collaborator := payload["data"].(map[string]any)["collaborator"].(map[string]any)
	collaboratorID := int64(collaborator["id"].(float64))
	if collaborator["cpf"] != "99988877766" || collaborator["weekly_hours"].(float64) != 40 || collaborator["salary"].(float64) != 8000 {
		t.Fatalf("dados de RH não foram persistidos: %v", collaborator)
	}

	res, _ = app.request(http.MethodPost, "/api/v1/collaborators", map[string]any{
		"name": "Outra Pessoa", "email": "pessoa.teste@test.local",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("e-mail duplicado deveria retornar 422, veio %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodPost, "/api/v1/collaborators", map[string]any{
		"name": "CPF Duplicado", "email": "cpf.duplicado@test.local", "cpf": "99988877766",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("CPF duplicado deveria retornar 422, veio %d", res.StatusCode)
	}

	res, payload = app.request(http.MethodGet, "/api/v1/collaborators?search=Pessoa", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("listar colaboradores status = %d", res.StatusCode)
	}
	if len(payload["data"].([]any)) != 1 {
		t.Fatalf("esperava 1 colaborador na busca, veio %v", payload["data"])
	}

	res, payload = app.request(http.MethodPost, "/api/v1/projects", map[string]any{
		"name": "Projeto com colaborador", "start_date": "2026-01-05", "end_date": "2026-02-27",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar projeto status = %d payload = %v", res.StatusCode, payload)
	}
	projectID := int64(payload["data"].(map[string]any)["project"].(map[string]any)["id"].(float64))

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/resources", projectID), map[string]any{
		"collaborator_id": collaboratorID,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("vincular colaborador status = %d payload = %v", res.StatusCode, payload)
	}
	resource := payload["data"].(map[string]any)["resource"].(map[string]any)
	if resource["name"] != "Pessoa Teste" || resource["cost_hour"].(float64) != 90 || int(resource["max_units"].(float64)) != 80 {
		t.Fatalf("recurso não herdou os dados do colaborador: %v", resource)
	}

	res, _ = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/resources", projectID), map[string]any{
		"collaborator_id": collaboratorID,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("vínculo duplicado deveria retornar 422, veio %d", res.StatusCode)
	}

	res, payload = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/collaborators/%d/toggle", collaboratorID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("alternar status status = %d payload = %v", res.StatusCode, payload)
	}
	if payload["data"].(map[string]any)["collaborator"].(map[string]any)["is_active"].(bool) {
		t.Fatal("colaborador deveria estar inativo após o toggle")
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/collaborators/%d", collaboratorID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("excluir colaborador status = %d", res.StatusCode)
	}

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("buscar projeto status = %d", res.StatusCode)
	}
	resources := payload["data"].(map[string]any)["resources"].([]any)
	if len(resources) != 1 {
		t.Fatalf("esperava 1 recurso preservado, veio %d", len(resources))
	}
	preserved := resources[0].(map[string]any)
	if preserved["collaborator_id"] != nil {
		t.Fatalf("vínculo deveria ser removido, veio %v", preserved["collaborator_id"])
	}
	if preserved["name"] != "Pessoa Teste" {
		t.Fatalf("recurso preservado perdeu o nome: %v", preserved)
	}
}

func TestProjectPipeline(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/projects", map[string]any{
		"name": "Projeto pipeline", "start_date": "2026-01-05", "end_date": "2026-02-27",
		"flow_mode": "pipeline",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar projeto pipeline status = %d payload = %v", res.StatusCode, payload)
	}
	project := payload["data"].(map[string]any)["project"].(map[string]any)
	projectID := int64(project["id"].(float64))
	if project["flow_mode"] != "pipeline" {
		t.Fatalf("flow_mode = %v", project["flow_mode"])
	}

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("buscar projeto status = %d", res.StatusCode)
	}
	bundle := payload["data"].(map[string]any)
	stages := bundle["stages"].([]any)
	if len(stages) != 4 {
		t.Fatalf("esperava 4 etapas padrão, veio %d", len(stages))
	}
	firstStage := int64(stages[0].(map[string]any)["id"].(float64))
	doneStage := int64(stages[3].(map[string]any)["id"].(float64))

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/tasks", projectID), map[string]any{
		"name": "Card no pipeline", "task_type": "task", "duration_days": 3,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar card status = %d payload = %v", res.StatusCode, payload)
	}
	task := payload["data"].(map[string]any)["task"].(map[string]any)
	taskID := int64(task["id"].(float64))
	if task["pipeline_stage_id"] == nil || int64(task["pipeline_stage_id"].(float64)) != firstStage {
		t.Fatalf("card deveria iniciar na primeira etapa: %v", task["pipeline_stage_id"])
	}

	res, payload = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/projects/%d/tasks/%d/stage", projectID, taskID), map[string]any{
		"stage_id": doneStage,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("mover card status = %d payload = %v", res.StatusCode, payload)
	}
	moved := payload["data"].(map[string]any)["task"].(map[string]any)
	if int(moved["progress"].(float64)) != 100 {
		t.Fatalf("card na etapa de conclusão deveria ficar 100%%, veio %v", moved["progress"])
	}

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/stages", projectID), map[string]any{
		"name": "Homologação", "color": "orange", "wip_limit": 5,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar etapa status = %d payload = %v", res.StatusCode, payload)
	}
	stage := payload["data"].(map[string]any)["stage"].(map[string]any)
	stageID := int64(stage["id"].(float64))

	res, _ = app.request(http.MethodPatch, fmt.Sprintf("/api/v1/projects/%d/stages/%d/move", projectID, stageID), map[string]any{
		"direction": "up",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("mover etapa status = %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%d/stages/%d", projectID, stageID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("excluir etapa status = %d", res.StatusCode)
	}

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("buscar projeto final status = %d", res.StatusCode)
	}
	if got := len(payload["data"].(map[string]any)["stages"].([]any)); got != 4 {
		t.Fatalf("esperava 4 etapas após exclusão, veio %d", got)
	}
}

func TestProjects(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPost, "/api/v1/projects", map[string]any{
		"name": "Projeto de teste", "description": "Cronograma", "status": "planejamento",
		"start_date": "2026-01-05", "end_date": "2026-02-27", "currency": "BRL",
		"budget": 50000, "working_days": "1,2,3,4,5", "hours_per_day": 8,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar projeto status = %d payload = %v", res.StatusCode, payload)
	}
	project := payload["data"].(map[string]any)["project"].(map[string]any)
	projectID := int64(project["id"].(float64))

	createTask := func(body map[string]any) map[string]any {
		res, payload := app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/tasks", projectID), body, app.csrf, nil)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("criar tarefa status = %d payload = %v", res.StatusCode, payload)
		}
		return payload["data"].(map[string]any)["task"].(map[string]any)
	}
	taskA := createTask(map[string]any{"name": "Planejamento", "task_type": "task", "start_date": "2026-01-05", "duration_days": 5, "priority": "alta", "effort_hours": 20})
	taskB := createTask(map[string]any{"name": "Execucao", "task_type": "task", "duration_days": 3})
	taskID := func(task map[string]any) int64 { return int64(task["id"].(float64)) }

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/dependencies", projectID), map[string]any{
		"predecessor_id": taskID(taskA), "successor_id": taskID(taskB), "type": "FS", "lag_days": 0,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar dependencia status = %d payload = %v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/resources", projectID), map[string]any{
		"name": "Ana", "role": "Engenheira", "cost_hour": 100, "max_units": 100, "color": "blue",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar recurso status = %d payload = %v", res.StatusCode, payload)
	}
	resourceID := int64(payload["data"].(map[string]any)["resource"].(map[string]any)["id"].(float64))

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/assignments", projectID), map[string]any{
		"task_id": taskID(taskA), "resource_id": resourceID, "units": 100, "planned_hours": 20,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar alocacao status = %d payload = %v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/holidays", projectID), map[string]any{
		"date": "2026-01-12", "name": "Feriado",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar feriado status = %d payload = %v", res.StatusCode, payload)
	}

	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d", projectID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("buscar projeto status = %d payload = %v", res.StatusCode, payload)
	}
	bundle := payload["data"].(map[string]any)
	tasks := bundle["tasks"].([]any)
	if len(tasks) != 2 {
		t.Fatalf("esperava 2 tarefas, veio %d", len(tasks))
	}
	var scheduledB string
	var critical int
	for _, raw := range tasks {
		task := raw.(map[string]any)
		if int64(task["id"].(float64)) == taskID(taskB) {
			scheduledB = task["scheduled_start"].(string)
		}
		if task["is_critical"].(bool) {
			critical++
		}
	}
	// Planejamento comeca em 05/01 e dura 5 dias uteis (termina 09/01). Com o
	// feriado de 12/01, a sucessora comeca em 13/01.
	if scheduledB != "2026-01-13" {
		t.Fatalf("inicio da sucessora = %q, esperado 2026-01-13", scheduledB)
	}
	if critical != 2 {
		t.Fatalf("esperava 2 tarefas criticas, veio %d", critical)
	}
	metrics := bundle["metrics"].(map[string]any)
	if metrics["planned_cost"].(float64) != 2000 {
		t.Fatalf("custo planejado = %v, esperado 2000", metrics["planned_cost"])
	}

	res, payload = app.request(http.MethodPost, fmt.Sprintf("/api/v1/projects/%d/baselines", projectID), map[string]any{"name": "Aprovacao"}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("criar linha de base status = %d payload = %v", res.StatusCode, payload)
	}

	res, _ = app.request(http.MethodGet, fmt.Sprintf("/api/v1/projects/%d/export", projectID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("exportar status = %d", res.StatusCode)
	}

	res, _ = app.request(http.MethodDelete, fmt.Sprintf("/api/v1/projects/%d", projectID), nil, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("excluir projeto status = %d", res.StatusCode)
	}
}

func TestBackupsAdmin(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodGet, "/api/v1/backups", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("backups index status = %d payload = %v", res.StatusCode, payload)
	}
	if _, ok := payload["settings"].(map[string]any); !ok {
		t.Fatalf("missing backup settings: %v", payload)
	}
	if entries, ok := payload["data"].([]any); !ok || len(entries) != 0 {
		t.Fatalf("expected empty backup list, got %v", payload["data"])
	}

	res, _ = app.request(http.MethodGet, "/api/v1/backups/backup-20260101-030000.zip/download", nil, "", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing backup download status = %d", res.StatusCode)
	}

	res, payload = app.request(http.MethodPost, "/api/v1/backups/backup-20260101-030000.zip/restore", map[string]any{
		"confirm": "outro.zip",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("restore confirmation status = %d payload = %v", res.StatusCode, payload)
	}
}

func TestBackupSchedule(t *testing.T) {
	app := newTestApp(t)

	res, payload := app.request(http.MethodPut, "/api/v1/backups/schedule", map[string]any{
		"enabled": true, "hour": 22, "minute": 30, "weekdays": []int{1, 3, 5},
	}, app.csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update schedule status = %d payload = %v", res.StatusCode, payload)
	}
	settings := payload["settings"].(map[string]any)
	if settings["hour"].(float64) != 22 || settings["minute"].(float64) != 30 {
		t.Fatalf("unexpected schedule settings: %v", settings)
	}

	res, _ = app.request(http.MethodPut, "/api/v1/backups/schedule", map[string]any{
		"enabled": true, "hour": 3, "minute": 0, "weekdays": []int{},
	}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation error without weekdays, got %d", res.StatusCode)
	}
}
