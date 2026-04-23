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
// The scheduler stops when parent is cancelled or when the returned stop function is invoked.
// The stop function blocks until the goroutine returns.
func StartScheduler(parent context.Context, db *sql.DB, b *broker.Broker) func() {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fire(ctx, db, b)
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

func fire(ctx context.Context, db *sql.DB, b *broker.Broker) {
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
