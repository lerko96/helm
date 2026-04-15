package reminder

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/lerko/helm/internal/broker"
)

type payload struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   int64  `json:"entity_id"`
	RemindAt   string `json:"remind_at"`
}

// StartScheduler polls for due reminders every 30s and publishes them to the broker.
// Returns a cancel function that stops the scheduler.
func StartScheduler(db *sql.DB, b *broker.Broker) func() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fire(db, b)
			}
		}
	}()

	return cancel
}

func fire(db *sql.DB, b *broker.Broker) {
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("reminder scheduler: begin tx: %v", err)
		return
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryContext(ctx, `
		SELECT id, entity_type, entity_id, remind_at
		FROM reminders
		WHERE is_sent = 0 AND remind_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		log.Printf("reminder scheduler: query: %v", err)
		return
	}

	type row struct {
		p  payload
		at time.Time
	}
	var due []row

	for rows.Next() {
		var r row
		if err := rows.Scan(&r.p.ID, &r.p.EntityType, &r.p.EntityID, &r.at); err != nil {
			log.Printf("reminder scheduler: scan: %v", err)
			continue
		}
		due = append(due, r)
	}
	rows.Close()

	for _, r := range due {
		if _, err := tx.ExecContext(ctx, `UPDATE reminders SET is_sent = 1 WHERE id = ?`, r.p.ID); err != nil {
			log.Printf("reminder scheduler: mark sent %d: %v", r.p.ID, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("reminder scheduler: commit: %v", err)
		return
	}

	for _, r := range due {
		r.p.RemindAt = r.at.Format(time.RFC3339)
		data, _ := json.Marshal(r.p)
		b.Publish(string(data))
	}
}
