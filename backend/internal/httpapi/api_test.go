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
	if len(data["modules"].([]any)) != 8 {
		t.Fatalf("expected 8 modules, got %v", data["modules"])
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
