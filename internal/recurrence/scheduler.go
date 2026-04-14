package recurrence

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// StartScheduler polls for due recurrences every hour and spawns new todo copies.
// Returns a cancel function that stops the scheduler.
func StartScheduler(db *sql.DB) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				spawnDue(db)
			}
		}
	}()
	return cancel
}

func spawnDue(db *sql.DB) {
	rows, err := db.QueryContext(context.Background(), `
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
		if err := spawnTodo(db, r.id, r.todoID, r.rrule, r.nextOccurrence); err != nil {
			log.Printf("recurrence scheduler: spawn todo %d: %v", r.todoID, err)
		}
	}
}

func spawnTodo(db *sql.DB, recurrenceID, parentID int64, rrule string, nextOcc time.Time) error {
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
	err = db.QueryRowContext(context.Background(),
		`SELECT list_id, title, description, priority FROM todos WHERE id = ?`, parentID,
	).Scan(&listID, &title, &description, &priority)
	if err == sql.ErrNoRows {
		// Parent deleted — clean up the recurrence.
		_, _ = db.ExecContext(context.Background(), `DELETE FROM todo_recurrences WHERE id = ?`, recurrenceID)
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

	if _, err = db.ExecContext(context.Background(), `
		INSERT INTO todos (user_id, list_id, title, description, status, priority, due_date)
		VALUES (1, ?, ?, ?, 'not_started', ?, ?)`,
		listIDVal, title, descVal, priority, nextOcc,
	); err != nil {
		return err
	}

	next := Advance(nextOcc, freq, interval)
	_, err = db.ExecContext(context.Background(),
		`UPDATE todo_recurrences SET next_occurrence = ? WHERE id = ?`, next, recurrenceID,
	)
	return err
}
