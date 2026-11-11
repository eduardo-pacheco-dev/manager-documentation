package httpapi

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/crypto"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/notifier"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/upload"
)

type Server struct {
	Config   config.Config
	Store    *store.Store
	Cipher   *crypto.Cipher
	Notifier *notifier.Notifier
	Uploads  *upload.Service
	Limiter  *loginLimiter
}

func New(cfg config.Config, s *store.Store, cipher *crypto.Cipher) *Server {
	return &Server{
		Config:   cfg,
		Store:    s,
		Cipher:   cipher,
		Notifier: notifier.New(s, cfg),
		Uploads:  upload.New(s, cfg, cipher),
		Limiter:  newLoginLimiter(5, time.Minute),
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.securityHeaders)
	r.Use(s.requestLogger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.Config.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", s.health)
	r.Get("/up", s.health)
	r.Get("/api/documentation", s.openAPIDocs)
	r.Get("/docs", s.openAPISpec)
	r.Handle("/uploads/*", s.uploadFileServer())

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(middleware.Throttle(180))
		api.Post("/auth/signup", s.signup)
		api.Post("/auth/login", s.login)

		api.Group(func(auth chi.Router) {
			auth.Use(s.requireAuth)
			auth.Use(s.requireCSRF)
			auth.Get("/auth/me", s.me)
			auth.Post("/auth/logout", s.logout)
			auth.Post("/uploads", s.upload)
			auth.Get("/profile", s.showProfile)
			auth.Put("/profile", s.updateProfile)
			auth.Get("/settings", s.showSettings)
			auth.Put("/settings/theme", s.updateTheme)
			auth.Put("/settings/modules", s.updateModules)
			auth.Put("/settings/notifications", s.updateNotificationSettings)
			auth.Put("/settings/password", s.updatePassword)
			auth.Delete("/settings/account", s.deleteAccount)
			auth.Get("/notifications", s.listNotifications)
			auth.Patch("/notifications/read-all", s.markAllNotificationsRead)
			auth.Patch("/notifications/{id}/read", s.markNotificationRead)
			auth.Delete("/notifications/{id}", s.deleteNotification)
			auth.Delete("/notifications", s.clearNotifications)
			auth.Get("/dashboard", s.dashboard)
			auth.Put("/dashboard/widgets/order", s.updateWidgetOrder)
			auth.Put("/dashboard/widgets/{id}", s.updateWidgetVisibility)
			auth.Put("/dashboard/widgets/{id}/layout", s.updateWidgetLayout)

			auth.Group(func(admin chi.Router) {
				admin.Use(s.requireAdmin)
				admin.Get("/users", s.listUsers)
				admin.Post("/users", s.createUser)
				admin.Put("/users/{id}", s.updateUser)
				admin.Patch("/users/{id}/toggle", s.toggleUser)
				admin.Delete("/users/{id}", s.deleteUser)
				admin.Get("/dev-docs", s.listDocs)
				admin.Post("/dev-docs", s.createDoc)
				admin.Get("/dev-docs/{id}", s.showDoc)
				admin.Put("/dev-docs/{id}", s.updateDoc)
				admin.Delete("/dev-docs/{id}", s.deleteDoc)
				admin.Put("/settings/system", s.updateTimezone)
				admin.Put("/settings/storage", s.updateStorage)
			})
		})
	})

	s.mountSPA(r)
	return r
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)
		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.Status(), time.Since(start).Round(time.Millisecond))
		}
	})
}

func (s *Server) uploadFileServer() http.Handler {
	dir, err := filepath.Abs(s.Config.UploadDir)
	if err != nil {
		dir = s.Config.UploadDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("httpapi: could not create upload dir: %v", err)
	}
	files := http.FileServer(http.Dir(dir))
	return http.StripPrefix("/uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2592000")
		files.ServeHTTP(w, r)
	}))
}

func (s *Server) mountSPA(r chi.Router) {
	index := filepath.Join(s.Config.StaticDir, "index.html")
	assets := filepath.Join(s.Config.StaticDir, "assets")
	if _, err := os.Stat(assets); err == nil {
		r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir(assets))))
	}
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/uploads/") {
			http.NotFound(w, r)
			return
		}
		relative := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if relative != "" && relative != "." {
			candidate := filepath.Join(s.Config.StaticDir, filepath.FromSlash(relative))
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				http.ServeFile(w, r, candidate)
				return
			}
		}
		if _, err := os.Stat(index); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Documentation Center API is running. Build the frontend or run the Vite dev server at http://localhost:5173.\n"))
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}
