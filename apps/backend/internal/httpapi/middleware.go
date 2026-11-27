package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eduardo-pacheco-dev/manager-documentation/apps/backend/internal/models"
)

type contextKey string

const authContextKey contextKey = "auth"

type AuthInfo struct {
	User            models.User
	TenantCompanyID int64
	CSRFToken       string
	Method          string // "session" or "token"
	RawToken        string
	SessionID       string
}

var (
	errUnauthorized = errors.New("unauthorized")
	errInactive     = errors.New("inactive")
)

// requestedCompanyID lets a super-admin work inside another company without
// changing the account that owns the session. Regular users always stay in
// their own company.
func (s *Server) requestedCompanyID(r *http.Request, user models.User) (int64, error) {
	identifier := strings.TrimSpace(r.Header.Get("X-Company-ID"))
	if !user.IsSuperAdmin || identifier == "" {
		return user.CompanyID, nil
	}
	company, err := s.Store.FindCompanyByIdentifier(identifier)
	if err != nil || !company.IsActive {
		return 0, errUnauthorized
	}
	return company.ID, nil
}

func (s *Server) resolveAuth(r *http.Request) (*AuthInfo, error) {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		raw := strings.TrimSpace(header[7:])
		token, err := s.Store.FindAPIToken(raw)
		if err != nil {
			return nil, errUnauthorized
		}
		user, err := s.Store.FindUserByID(token.UserID)
		if err != nil || token.CompanyID != user.CompanyID {
			return nil, errUnauthorized
		}
		if !user.IsActive {
			return nil, errInactive
		}
		companyID, err := s.requestedCompanyID(r, user)
		if err != nil {
			return nil, err
		}
		return &AuthInfo{User: user, TenantCompanyID: companyID, Method: "token", RawToken: raw}, nil
	}

	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return nil, errUnauthorized
	}
	session, err := s.Store.FindSession(cookie.Value)
	if err != nil {
		return nil, errUnauthorized
	}
	user, err := s.Store.FindUserByID(session.UserID)
	if err != nil || session.CompanyID != user.CompanyID {
		return nil, errUnauthorized
	}
	if !user.IsActive {
		_ = s.Store.DeleteSession(cookie.Value)
		return nil, errInactive
	}
	companyID, err := s.requestedCompanyID(r, user)
	if err != nil {
		return nil, err
	}
	return &AuthInfo{
		User:            user,
		TenantCompanyID: companyID,
		CSRFToken:       session.CSRFToken,
		Method:          "session",
		RawToken:        cookie.Value,
		SessionID:       session.ID,
	}, nil
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, err := s.resolveAuth(r)
		if err != nil {
			if errors.Is(err, errInactive) {
				clearSessionCookie(w, s.Config, r)
				writeError(w, http.StatusForbidden, "Sua conta está desativada.", nil)
				return
			}
			writeError(w, http.StatusUnauthorized, "Não autenticado.", nil)
			return
		}
		tenant, err := s.tenantFor(auth.TenantCompanyID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Não foi possível acessar os dados da empresa.", nil)
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey, auth)
		next.ServeHTTP(w, withTenant(r.WithContext(ctx), tenant))
	})
}

func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		auth := authFrom(r)
		if auth == nil || auth.Method != "session" {
			next.ServeHTTP(w, r)
			return
		}
		header := r.Header.Get("X-CSRF-Token")
		if header == "" || header != auth.CSRFToken {
			writeError(w, http.StatusForbidden, "Token CSRF inválido. Recarregue a página e tente novamente.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := authFrom(r)
		if auth == nil || (!auth.User.IsAdmin && !auth.User.IsSuperAdmin) {
			writeError(w, http.StatusForbidden, "Acesso restrito a administradores.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := authFrom(r)
		if auth == nil || !auth.User.IsSuperAdmin {
			writeError(w, http.StatusForbidden, "Acesso restrito ao super-administrador.", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authFrom(r *http.Request) *AuthInfo {
	value := r.Context().Value(authContextKey)
	auth, _ := value.(*AuthInfo)
	return auth
}

func currentUser(r *http.Request) models.User {
	if auth := authFrom(r); auth != nil {
		return auth.User
	}
	return models.User{}
}

/* --------------------------------------------------------------------------
 * Login rate limiting
 * ----------------------------------------------------------------------- */

type loginLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	attempts map[string][]time.Time
}

func newLoginLimiter(limit int, window time.Duration) *loginLimiter {
	return &loginLimiter{limit: limit, window: window, attempts: map[string][]time.Time{}}
}

func (l *loginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	existing := l.attempts[key]
	kept := existing[:0]
	for _, attempt := range existing {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	allowed := len(kept) < l.limit
	if allowed {
		kept = append(kept, now)
	}
	l.attempts[key] = kept
	return allowed
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
