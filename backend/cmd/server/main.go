package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/config"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/crypto"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/database"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/httpapi"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/modules"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/seed"
	"github.com/eduardo-pacheco-dev/manager-documentation/backend/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	database.CleanupSessions(db)

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			database.CleanupSessions(db)
		}
	}()

	repository := store.New(db)

	// Apply the persisted system timezone before seeding/rendering anything.
	timezone := cfg.Timezone
	if stored, err := repository.GetSetting("timezone"); err == nil && stored != "" {
		timezone = stored
	}
	if location, err := time.LoadLocation(timezone); err == nil {
		time.Local = location
	} else {
		log.Printf("timezone: invalid %q, using UTC", timezone)
		time.Local = time.UTC
	}

	if err := seed.Run(repository, cfg); err != nil {
		log.Fatalf("seed: %v", err)
	}
	moduleStore := modules.New(db)
	if cfg.SeedDemoData {
		if admin, findErr := repository.FindUserByEmail(cfg.SeedAdminEmail); findErr == nil {
			if moduleErr := seed.ModuleData(moduleStore, admin); moduleErr != nil {
				log.Printf("seed modules: %v", moduleErr)
			}
		}
	}
	if err := moduleStore.SyncAllNokiaSitesToStations(); err != nil {
		log.Printf("nokia/stations sync: %v", err)
	}

	cipher, err := crypto.New(cfg.AppKey)
	if err != nil {
		log.Fatalf("crypto: %v", err)
	}

	server := httpapi.New(cfg, repository, cipher)
	stopIntegration := server.StartIntegrationWorker(5 * time.Minute)
	defer stopIntegration()
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("documentation-center listening on %s (env=%s)", cfg.Addr, cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-shutdown
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
}
