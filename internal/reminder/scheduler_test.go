package reminder

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/lerko/helm/internal/broker"
	"github.com/lerko/helm/internal/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmp := t.TempDir() + "/test.db"
	d, err := db.Open(tmp)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Migrate(d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestStartScheduler_CancelStopsGoroutine(t *testing.T) {
	d := openTestDB(t)
	b := broker.New()

	ctx, cancel := context.WithCancel(context.Background())
	stop := StartScheduler(ctx, d, b)

	done := make(chan struct{})
	go func() {
		cancel()
		stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop within 2s after ctx cancel")
	}
}

func TestStartScheduler_StopFuncAlone(t *testing.T) {
	d := openTestDB(t)
	b := broker.New()

	stop := StartScheduler(context.Background(), d, b)
	done := make(chan struct{})
	go func() {
		stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stop func did not return within 2s")
	}
}

func TestFire_MarksReminderSentAndPublishes(t *testing.T) {
	d := openTestDB(t)
	b := broker.New()

	ch := b.Subscribe("test-client")
	defer b.Unsubscribe("test-client")

	// Insert a due reminder (5 minutes in the past).
	past := time.Now().UTC().Add(-5 * time.Minute).Format("2006-01-02 15:04:05")
	_, err := d.Exec(
		`INSERT INTO reminders (user_id, entity_type, entity_id, remind_at, is_sent) VALUES (1, 'todo', 42, ?, 0)`,
		past,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	fire(context.Background(), d, b)

	select {
	case msg := <-ch:
		if msg == "" {
			t.Error("received empty message")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("expected publish after fire, got nothing")
	}

	var sent int
	if err := d.QueryRow(`SELECT is_sent FROM reminders WHERE entity_id = 42`).Scan(&sent); err != nil {
		t.Fatalf("query: %v", err)
	}
	if sent != 1 {
		t.Errorf("is_sent = %d, want 1", sent)
	}
}
