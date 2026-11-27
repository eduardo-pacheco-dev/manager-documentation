package httpapi

import (
	"context"
	"net/http"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/backup"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/database"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/modules"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/notifier"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/store"
	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/upload"
)

type tenantContextKeyType string

const tenantContextKey tenantContextKeyType = "tenant"

// tenantFor returns the per-company view of the server. It reuses the base
// config/limiter/cipher but swaps every DB-backed service for a pool whose
// session variable and SQL rewriter pin queries to a single company.
func (s *Server) tenantFor(companyID int64) (*Server, error) {
	if cached, ok := s.tenants.Load(companyID); ok {
		return cached.(*Server), nil
	}
	db, err := database.Tenant(s.Config, companyID)
	if err != nil {
		return nil, err
	}
	tenantStore := store.New(db)
	uploads := upload.New(tenantStore, s.Config, s.Cipher)
	backups := backup.New(s.Config)
	backups.SetRemote(uploads)
	backups.SetConfigStore(tenantStore)
	tenantServer := &Server{
		Config:   s.Config,
		Store:    tenantStore,
		Modules:  modules.New(db),
		Cipher:   s.Cipher,
		Notifier: notifier.New(tenantStore, s.Config),
		Uploads:  uploads,
		Backups:  backups,
		Limiter:  s.Limiter,
		tenants:  s.tenants,
	}
	actual, _ := s.tenants.LoadOrStore(companyID, tenantServer)
	return actual.(*Server), nil
}

func withTenant(r *http.Request, tenant *Server) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), tenantContextKey, tenant))
}

func tenantFrom(r *http.Request) (*Server, bool) {
	tenant, ok := r.Context().Value(tenantContextKey).(*Server)
	return tenant, ok
}

// useTenant is inserted at the start of handlers that touch tenant-owned data.
func (s *Server) useTenant(w http.ResponseWriter, r *http.Request) (*Server, bool) {
	tenant, ok := tenantFrom(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Empresa não identificada.", nil)
		return nil, false
	}
	return tenant, true
}

// usePublicTenant resolves the tenant from a public code/token before any
// module query runs. Public links do not carry a session.
func (s *Server) usePublicTenant(w http.ResponseWriter, r *http.Request, table, column, value string) (*Server, *http.Request, bool) {
	companyID, err := s.Store.PublicCompanyID(table, column, value)
	if err != nil {
		writeError(w, http.StatusGone, "Link indisponível.", nil)
		return nil, r, false
	}
	tenant, err := s.tenantFor(companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Não foi possível acessar os dados da empresa.", nil)
		return nil, r, false
	}
	return tenant, withTenant(r, tenant), true
}
