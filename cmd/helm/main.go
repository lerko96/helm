package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/lerko/helm/internal/api"
	"github.com/lerko/helm/internal/broker"
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

	b := broker.New()
	stopReminders := reminder.StartScheduler(database, b)
	defer stopReminders()

	stopRecurrence := recurrence.StartScheduler(database)
	defer stopRecurrence()

	var uiFS fs.FS
	if f, err := ui.FS(); err == nil {
		uiFS = f
	} else {
		log.Printf("ui: no embedded assets (%v) — API-only mode", err)
	}

	router := api.NewRouter(cfg, database, uiFS, b)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("helm listening on http://%s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
