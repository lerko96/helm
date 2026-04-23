package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lerko/helm/internal/api"
	"github.com/lerko/helm/internal/broker"
	"github.com/lerko/helm/internal/caldav"
	"github.com/lerko/helm/internal/config"
	"github.com/lerko/helm/internal/db"
	"github.com/lerko/helm/internal/recurrence"
	"github.com/lerko/helm/internal/reminder"
	"github.com/lerko/helm/ui"
)

func main() {
	cfgPath := "config.yml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	log.Printf("timezone: %s (set TZ env var to override)", time.Local.String())

	database, err := db.Open(cfg.Storage.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	if err := os.MkdirAll(cfg.Storage.AttachmentsPath, 0o755); err != nil {
		log.Fatalf("attachments dir: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	b := broker.New()
	stopReminders := reminder.StartScheduler(ctx, database, b)
	defer stopReminders()

	stopRecurrence := recurrence.StartScheduler(ctx, database)
	defer stopRecurrence()

	stopCalDAV := startCalDAVScheduler(ctx, database, cfg.Auth.Secret)
	defer stopCalDAV()

	var uiFS fs.FS
	if f, err := ui.FS(); err == nil {
		uiFS = f
	} else {
		log.Printf("ui: no embedded assets (%v) — API-only mode", err)
	}

	router := api.NewRouter(cfg, database, uiFS, b)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: router}

	go func() {
		log.Printf("helm listening on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("helm shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
}

// startCalDAVScheduler syncs all non-local calendar sources every 15 minutes.
// The scheduler stops when parent ctx is cancelled or when the returned stop function is
// invoked. Stop blocks until the ticker goroutine and any in-flight sync goroutines return.
func startCalDAVScheduler(parent context.Context, database *sql.DB, secret string) func() {
	ctx, cancel := context.WithCancel(parent)
	var wg sync.WaitGroup

	syncAll := func() {
		rows, err := database.QueryContext(ctx,
			`SELECT id, name, url, username, password_enc FROM calendar_sources WHERE is_local = 0`,
		)
		if err != nil {
			log.Printf("caldav scheduler: query sources: %v", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var src caldav.CalendarSource
			var urlStr, username, passwordEnc sql.NullString
			if err := rows.Scan(&src.ID, &src.Name, &urlStr, &username, &passwordEnc); err != nil {
				continue
			}
			src.URL = urlStr.String
			src.Username = username.String
			src.PasswordEnc = passwordEnc.String

			wg.Add(1)
			go func(s caldav.CalendarSource) {
				defer wg.Done()
				if err := caldav.SyncSource(database, s, secret); err != nil {
					log.Printf("caldav scheduler: source %d: %v", s.ID, err)
				}
			}(src)
		}
		if err := rows.Err(); err != nil {
			log.Printf("caldav scheduler: iterate sources: %v", err)
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				syncAll()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func() {
		cancel()
		wg.Wait()
	}
}
