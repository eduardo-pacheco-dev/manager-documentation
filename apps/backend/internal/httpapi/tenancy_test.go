package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func rawRequest(t *testing.T, baseURL, method, path string, body any, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	payload := map[string]any{}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) > 0 && res.Header.Get("Content-Type") != "" {
		_ = json.Unmarshal(raw, &payload)
	}
	return res, payload
}

func loginToken(t *testing.T, baseURL, company, email, password string) string {
	t.Helper()
	res, payload := rawRequest(t, baseURL, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"company": company, "email": email, "password": password, "device_name": "tenancy-test",
	}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login %s/%s status = %d payload = %v", company, email, res.StatusCode, payload)
	}
	token, _ := payload["data"].(map[string]any)["token"].(string)
	if token == "" {
		t.Fatalf("login did not return a token: %v", payload)
	}
	return token
}

func bearer(token, company string) map[string]string {
	headers := map[string]string{"Authorization": "Bearer " + token}
	if company != "" {
		headers["X-Company-ID"] = company
	}
	return headers
}

func TestTenantIsolation(t *testing.T) {
	app := newTestApp(t)
	base := app.server.URL

	// Create the second company together with an admin that uses the same
	// e-mail as a user we are about to create in the default company.
	res, payload := app.request(http.MethodPost, "/api/v1/companies", map[string]any{
		"name":           "Empresa B",
		"slug":           "empresa-b",
		"is_active":      true,
		"admin_name":     "Admin B",
		"admin_email":    "shared@example.com",
		"admin_password": "password-b",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create company status = %d payload = %v", res.StatusCode, payload)
	}
	companyBID := int64(payload["data"].(map[string]any)["id"].(float64))

	res, payload = app.request(http.MethodPost, "/api/v1/users", map[string]any{
		"name":                  "Usuário A",
		"email":                 "shared@example.com",
		"password":              "password-a",
		"is_admin":              true,
		"password_confirmation": "password-a",
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create user A status = %d payload = %v", res.StatusCode, payload)
	}

	res, payload = rawRequest(t, base, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"company": "padrao", "email": "shared@example.com", "password": "password-b",
	}, nil)
	if res.StatusCode == http.StatusOK {
		t.Fatalf("password from company B should not authenticate in company A: %v", payload)
	}
	res, payload = rawRequest(t, base, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"company": "empresa-b", "email": "shared@example.com", "password": "password-a",
	}, nil)
	if res.StatusCode == http.StatusOK {
		t.Fatalf("password from company A should not authenticate in company B: %v", payload)
	}
	tokenB := loginToken(t, base, "empresa-b", "shared@example.com", "password-b")
	tokenA := loginToken(t, base, "padrao", "shared@example.com", "password-a")

	// A company admin cannot manage companies.
	res, _ = rawRequest(t, base, http.MethodGet, "/api/v1/companies", nil, bearer(tokenB, ""))
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("company admin companies status = %d, want 403", res.StatusCode)
	}

	// Each admin only sees users from their own company.
	res, payload = rawRequest(t, base, http.MethodGet, "/api/v1/users?search=shared", nil, bearer(tokenA, ""))
	meta, _ := payload["meta"].(map[string]any)
	if res.StatusCode != http.StatusOK || meta["total"] != float64(1) {
		t.Fatalf("company A users = %d payload = %v", res.StatusCode, payload)
	}
	res, payload = rawRequest(t, base, http.MethodGet, "/api/v1/users?search=shared", nil, bearer(tokenB, ""))
	meta, _ = payload["meta"].(map[string]any)
	if res.StatusCode != http.StatusOK || meta["total"] != float64(1) {
		t.Fatalf("company B users = %d payload = %v", res.StatusCode, payload)
	}

	// A company admin cannot promote anyone to super-admin.
	res, payload = rawRequest(t, base, http.MethodPost, "/api/v1/users", map[string]any{
		"name": "Promovido", "email": "promovido@example.com", "password": "password-x",
		"password_confirmation": "password-x", "is_admin": true, "is_super_admin": true,
	}, bearer(tokenB, ""))
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("company admin promotion status = %d payload = %v", res.StatusCode, payload)
	}

	// The super-admin can create a user directly in another company.
	res, payload = app.request(http.MethodPost, "/api/v1/users", map[string]any{
		"name": "Usuário B2", "email": "b2@example.com", "password": "password-b2",
		"password_confirmation": "password-b2", "is_admin": true,
		"company_id": companyBID, "is_super_admin": false,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("super-admin create in company B status = %d payload = %v", res.StatusCode, payload)
	}
	loginToken(t, base, "empresa-b", "b2@example.com", "password-b2")

	// The default company cannot be disabled.
	res, payload = app.request(http.MethodPut, "/api/v1/companies/1", map[string]any{
		"name": "Padrão", "slug": "padrao", "is_active": false,
	}, app.csrf, nil)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("disable default company status = %d payload = %v", res.StatusCode, payload)
	}

	// A document created in company B is invisible to company A, including
	// the super-admin session while it is scoped to the default company.
	res, payload = rawRequest(t, base, http.MethodPost, "/api/v1/dev-docs", map[string]any{
		"title": "Documento B", "category": "api", "summary": "Privado",
		"content": "# B", "is_published": true,
	}, bearer(tokenB, ""))
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create doc B status = %d payload = %v", res.StatusCode, payload)
	}
	docID := int64(payload["data"].(map[string]any)["document"].(map[string]any)["id"].(float64))

	res, _ = rawRequest(t, base, http.MethodGet, fmt.Sprintf("/api/v1/dev-docs/%d", docID), nil, bearer(tokenA, ""))
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-company doc status = %d, want 404", res.StatusCode)
	}

	// The super-admin session can switch to the other company explicitly.
	res, payload = app.request(http.MethodGet, fmt.Sprintf("/api/v1/dev-docs/%d", docID), nil, "", map[string]string{"X-Company-ID": "empresa-b"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("super-admin switched doc status = %d payload = %v", res.StatusCode, payload)
	}
	res, payload = app.request(http.MethodGet, "/api/v1/auth/me", nil, "", map[string]string{"X-Company-ID": "empresa-b"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("super-admin me status = %d payload = %v", res.StatusCode, payload)
	}
	company := payload["data"].(map[string]any)["company"].(map[string]any)
	if company["slug"] != "empresa-b" {
		t.Fatalf("active company = %v, want empresa-b", company)
	}
}
