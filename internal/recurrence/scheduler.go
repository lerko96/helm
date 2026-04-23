package recurrence

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// StartScheduler polls for due recurrences every hour and spawns new todo copies.
// The scheduler stops when parent is cancelled or when the returned stop function is invoked.
// The stop function blocks until the goroutine returns.
func StartScheduler(parent context.Context, db *sql.DB) func() {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				spawnDue(ctx, db)
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func spawnDue(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT tr.id, tr.todo_id, tr.rrule, tr.next_occurrence
		FROM todo_recurrences tr
		WHERE tr.next_occurrence <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		log.Printf("recurrence scheduler: query: %v", err)
		return
	}

	type row struct {
		id             int64
		todoID         int64
		rrule          string
		nextOccurrence time.Time
	}
	var due []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.todoID, &r.rrule, &r.nextOccurrence); err != nil {
			log.Printf("recurrence scheduler: scan: %v", err)
			continue
		}
		due = append(due, r)
	}
	rows.Close()

	for _, r := range due {
		if err := spawnTodo(ctx, db, r.id, r.todoID, r.rrule, r.nextOccurrence); err != nil {
			log.Printf("recurrence scheduler: spawn todo %d: %v", r.todoID, err)
		}
	}
}

func spawnTodo(ctx context.Context, db *sql.DB, recurrenceID, parentID int64, rrule string, nextOcc time.Time) error {
	freq, interval, err := ParseRRule(rrule)
	if err != nil {
		return err
	}

	var (
		listID      sql.NullInt64
		title       string
		description sql.NullString
		priority    string
	)
	err = db.QueryRowContext(ctx,
		`SELECT list_id, title, description, priority FROM todos WHERE id = ?`, parentID,
	).Scan(&listID, &title, &description, &priority)
	if err == sql.ErrNoRows {
		// Parent deleted — clean up the recurrence.
		_, _ = db.ExecContext(ctx, `DELETE FROM todo_recurrences WHERE id = ?`, recurrenceID)
		return nil
	}
	if err != nil {
		return err
	}

	var listIDVal *int64
	if listID.Valid {
		listIDVal = &listID.Int64
	}
	var descVal *string
	if description.Valid {
		descVal = &description.String
	}

	next := Advance(nextOcc, freq, interval)

	// INSERT + UPDATE in a single transaction to prevent duplicate todos on crash/restart.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO todos (user_id, list_id, title, description, status, priority, due_date)
		VALUES (1, ?, ?, ?, 'not_started', ?, ?)`,
		listIDVal, title, descVal, priority, nextOcc,
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE todo_recurrences SET next_occurrence = ? WHERE id = ?`, next, recurrenceID,
	); err != nil {
		return err
	}

	return tx.Commit()
}
