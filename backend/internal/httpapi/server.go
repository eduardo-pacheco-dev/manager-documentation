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
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/notifier"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/upload"
)

type Server struct {
	Config   config.Config
	Store    *store.Store
	Modules  *modules.Store
	Cipher   *crypto.Cipher
	Notifier *notifier.Notifier
	Uploads  *upload.Service
	Limiter  *loginLimiter
}

func New(cfg config.Config, s *store.Store, cipher *crypto.Cipher) *Server {
	return &Server{
		Config:   cfg,
		Store:    s,
		Modules:  modules.New(s.DB),
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

			auth.Route("/agenda", func(agenda chi.Router) {
				agenda.Get("/", s.agendaIndex)
				agenda.Get("/events/{id}", s.agendaShowEvent)
				agenda.Post("/events", s.agendaCreateEvent)
				agenda.Put("/events/{id}", s.agendaUpdateEvent)
				agenda.Delete("/events/{id}", s.agendaDeleteEvent)
				agenda.Post("/events/{id}/duplicate", s.agendaDuplicateEvent)
				agenda.Patch("/events/{id}/respond", s.agendaRespondEvent)
				agenda.Patch("/events/{id}/move", s.agendaMoveEvent)
				agenda.Post("/calendars", s.agendaCreateCalendar)
				agenda.Put("/calendars/{id}", s.agendaUpdateCalendar)
				agenda.Delete("/calendars/{id}", s.agendaDeleteCalendar)
				agenda.Patch("/calendars/{id}/visibility", s.agendaCalendarVisibility)
				agenda.Patch("/calendars/{id}/default", s.agendaCalendarDefault)
				agenda.Post("/calendars/{id}/shares", s.agendaShareCalendar)
				agenda.Delete("/calendars/{id}/shares/{shareId}", s.agendaUnshareCalendar)
				agenda.Get("/ics", s.agendaExportICS)
				agenda.Post("/ics/import", s.agendaImportICS)
			})

			auth.Route("/tasks", func(tasks chi.Router) {
				tasks.Get("/", s.tasksIndex)
				tasks.Post("/", s.tasksCreate)
				tasks.Get("/export", s.tasksExport)
				tasks.Get("/{id}", s.tasksShow)
				tasks.Put("/{id}", s.tasksUpdate)
				tasks.Delete("/{id}", s.tasksDelete)
				tasks.Patch("/{id}/toggle", s.tasksToggle)
				tasks.Patch("/{id}/important", s.tasksToggleImportant)
				tasks.Patch("/{id}/my-day", s.tasksToggleMyDay)
				tasks.Patch("/{id}/move", s.tasksMove)
				tasks.Post("/{id}/steps", s.tasksCreateStep)
				tasks.Put("/{id}/steps/{stepId}", s.tasksUpdateStep)
				tasks.Delete("/{id}/steps/{stepId}", s.tasksDeleteStep)
				tasks.Post("/lists", s.tasksCreateList)
				tasks.Put("/lists/{id}", s.tasksUpdateList)
				tasks.Delete("/lists/{id}", s.tasksDeleteList)
			})

			auth.Route("/follow-ups", func(follow chi.Router) {
				follow.Get("/", s.followUpsIndex)
				follow.Get("/export", s.followUpsExport)
				follow.Post("/import", s.followUpsImport)
				follow.Post("/email", s.followUpsEmailTable)
				follow.Get("/{id}", s.followUpsShow)
				follow.Post("/", s.followUpsCreate)
				follow.Put("/{id}", s.followUpsUpdate)
				follow.Delete("/{id}", s.followUpsDelete)
				follow.Post("/{id}/updates", s.followUpsAddUpdate)
				follow.Delete("/{id}/updates/{updateId}", s.followUpsDeleteUpdate)
				follow.Post("/{id}/email", s.followUpsEmailOne)
				follow.Group(func(admin chi.Router) {
					admin.Use(s.requireAdmin)
					admin.Put("/settings", s.followUpsUpdateSettings)
					admin.Post("/settings/assignees", s.followUpsCreateAssignee)
					admin.Delete("/settings/assignees/{id}", s.followUpsDeleteAssignee)
					admin.Post("/settings/clients", s.followUpsCreateClient)
					admin.Put("/settings/clients/{id}", s.followUpsUpdateClient)
					admin.Delete("/settings/clients/{id}", s.followUpsDeleteClient)
					admin.Post("/settings/projects", s.followUpsCreateProject)
					admin.Put("/settings/projects/{id}", s.followUpsUpdateProject)
					admin.Delete("/settings/projects/{id}", s.followUpsDeleteProject)
				})
			})

			auth.Route("/stations", func(stations chi.Router) {
				stations.Get("/", s.stationsIndex)
				stations.Post("/", s.stationsCreate)
				stations.Get("/export", s.stationsExport)
				stations.Get("/template", s.stationsTemplate)
				stations.Post("/import", s.stationsImport)
				stations.Get("/{id}", s.stationsShow)
				stations.Get("/{id}/export", s.stationExportOne)
				stations.Put("/{id}", s.stationsUpdate)
				stations.Delete("/{id}", s.stationsDelete)
				stations.Post("/{id}/attachments", s.stationsUploadAttachment)
				stations.Get("/{id}/attachments/{attachmentId}", s.stationsDownloadAttachment)
				stations.Delete("/{id}/attachments/{attachmentId}", s.stationsDeleteAttachment)
				stations.Post("/{id}/comments", s.stationsCreateComment)
				stations.Delete("/{id}/comments/{commentId}", s.stationsDeleteComment)
			})

			auth.Route("/nokia", func(nokia chi.Router) {
				nokia.Get("/sites", s.nokiaIndex)
				nokia.Get("/roms", s.nokiaROMList)
				nokia.Get("/station-sites", s.nokiaStationSites)
				nokia.Post("/sites", s.nokiaCreate)
				nokia.Get("/sites/export", s.nokiaExport)
				nokia.Get("/sites/export/pending", s.nokiaPendingExport)
				nokia.Get("/sites/template", s.nokiaTemplate)
				nokia.Post("/sites/import", s.nokiaImport)
				nokia.Get("/sites/{id}", s.nokiaShow)
				nokia.Put("/sites/{id}", s.nokiaUpdate)
				nokia.Delete("/sites/{id}", s.nokiaDelete)
				nokia.Put("/sites/{id}/checklist/{field}", s.nokiaUpdateChecklist)
				nokia.Put("/sites/{id}/dates", s.nokiaUpdateDates)
				nokia.Put("/sites/{id}/interruption", s.nokiaUpdateInterruption)
				nokia.Delete("/sites/{id}/interruption", s.nokiaClearInterruption)
				nokia.Post("/sites/{id}/tasks/{taskId}", s.nokiaToggleTask)
				nokia.Post("/sites/{id}/rejections", s.nokiaCreateRejection)
				nokia.Patch("/sites/{id}/rejections/{rejectionId}", s.nokiaResolveRejection)
				nokia.Post("/sites/{id}/attachments", s.nokiaUploadAttachment)
				nokia.Get("/sites/{id}/attachments/{attachmentId}", s.nokiaDownloadAttachment)
				nokia.Delete("/sites/{id}/attachments/{attachmentId}", s.nokiaDeleteAttachment)
				nokia.Post("/sites/{id}/comments", s.nokiaCreateComment)
				nokia.Delete("/sites/{id}/comments/{commentId}", s.nokiaDeleteComment)
				nokia.Group(func(admin chi.Router) {
					admin.Use(s.requireAdmin)
					admin.Get("/settings", s.nokiaSettings)
					admin.Post("/settings/roms", s.nokiaCreateROM)
					admin.Put("/settings/roms/{id}", s.nokiaUpdateROM)
					admin.Delete("/settings/roms/{id}", s.nokiaDeleteROM)
					admin.Post("/settings/tasks", s.nokiaCreateTask)
					admin.Put("/settings/tasks/{id}", s.nokiaUpdateTask)
					admin.Patch("/settings/tasks/{id}/move", s.nokiaMoveTask)
					admin.Delete("/settings/tasks/{id}", s.nokiaDeleteTask)
				})
			})

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
