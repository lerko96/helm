package caldav

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/lerko/helm/internal/crypto"
)

var privateRanges = []net.IPNet{
	parseCIDR("127.0.0.0/8"),
	parseCIDR("10.0.0.0/8"),
	parseCIDR("172.16.0.0/12"),
	parseCIDR("192.168.0.0/16"),
	parseCIDR("169.254.0.0/16"), // link-local
	parseCIDR("::1/128"),        // IPv6 loopback
	parseCIDR("fc00::/7"),       // IPv6 ULA
}

func parseCIDR(s string) net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return *n
}

// validateCalDAVURL rejects non-HTTPS schemes and URLs that resolve to
// private/loopback addresses to prevent SSRF attacks.
func validateCalDAVURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("caldav URL must use https (got %q)", u.Scheme)
	}
	host := u.Hostname()
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("could not resolve host %q: %w", host, err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		for _, block := range privateRanges {
			if block.Contains(ip) {
				return fmt.Errorf("caldav URL resolves to private/loopback address (%s)", addr)
			}
		}
	}
	return nil
}

// CalendarSource mirrors the DB row fields needed for sync.
type CalendarSource struct {
	ID          int64
	Name        string
	URL         string
	Username    string
	PasswordEnc string // hex AES-GCM ciphertext
}

// SyncSource fetches events from a remote CalDAV source and upserts them into the DB.
// It skips events whose etag is unchanged, and deletes DB events not present in the remote response.
func SyncSource(db *sql.DB, source CalendarSource, secret string) error {
	if err := validateCalDAVURL(source.URL); err != nil {
		return fmt.Errorf("source %d URL rejected: %w", source.ID, err)
	}

	password := ""
	if source.PasswordEnc != "" {
		p, err := crypto.DecryptString(source.PasswordEnc, secret)
		if err != nil {
			return fmt.Errorf("decrypt password for source %d: %w", source.ID, err)
		}
		password = p
	}

	client := NewClient(source.URL, source.Username, password)

	from := time.Now().AddDate(-1, 0, 0)
	to := time.Now().AddDate(2, 0, 0)

	events, err := client.FetchEvents(from, to)
	if err != nil {
		return fmt.Errorf("fetch events for source %d (%s): %w", source.ID, source.Name, err)
	}

	// Build set of UIDs returned by remote for cleanup
	remoteUIDs := make(map[string]struct{}, len(events))
	for _, ev := range events {
		remoteUIDs[ev.UID] = struct{}{}
	}

	// Upsert each event
	for _, ev := range events {
		if err := upsertEvent(db, source.ID, ev); err != nil {
			log.Printf("caldav: upsert event %s (source %d): %v", ev.UID, source.ID, err)
		}
	}

	// Delete events in DB that are no longer in the remote response
	if err := deleteStaleEvents(db, source.ID, remoteUIDs); err != nil {
		log.Printf("caldav: cleanup stale events for source %d: %v", source.ID, err)
	}

	// Update last_synced_at
	_, _ = db.Exec(
		`UPDATE calendar_sources SET last_synced_at = CURRENT_TIMESTAMP WHERE id = ?`,
		source.ID,
	)

	log.Printf("caldav: synced source %d (%s): %d events", source.ID, source.Name, len(events))
	return nil
}

func upsertEvent(db *sql.DB, sourceID int64, ev Event) error {
	// Check existing etag
	var existingETag sql.NullString
	err := db.QueryRow(
		`SELECT etag FROM calendar_events WHERE source_id = ? AND uid = ?`,
		sourceID, ev.UID,
	).Scan(&existingETag)

	if err == nil && existingETag.Valid && existingETag.String == ev.ETag && ev.ETag != "" {
		return nil // unchanged
	}

	var description, location, rrule *string
	if ev.Description != "" {
		description = &ev.Description
	}
	if ev.Location != "" {
		location = &ev.Location
	}
	if ev.RRule != "" {
		rrule = &ev.RRule
	}

	isAllDay := 0
	if ev.IsAllDay {
		isAllDay = 1
	}

	if err == sql.ErrNoRows {
		// INSERT
		_, err = db.Exec(`
			INSERT INTO calendar_events
				(user_id, source_id, uid, etag, title, description, location, start_at, end_at, is_all_day, rrule)
			VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sourceID, ev.UID, ev.ETag, ev.Title, description, location,
			ev.StartAt.UTC().Format("2006-01-02 15:04:05"),
			ev.EndAt.UTC().Format("2006-01-02 15:04:05"),
			isAllDay, rrule,
		)
		return err
	}

	// UPDATE
	_, err = db.Exec(`
		UPDATE calendar_events
		SET etag=?, title=?, description=?, location=?, start_at=?, end_at=?, is_all_day=?, rrule=?, updated_at=CURRENT_TIMESTAMP
		WHERE source_id=? AND uid=?`,
		ev.ETag, ev.Title, description, location,
		ev.StartAt.UTC().Format("2006-01-02 15:04:05"),
		ev.EndAt.UTC().Format("2006-01-02 15:04:05"),
		isAllDay, rrule,
		sourceID, ev.UID,
	)
	return err
}

func deleteStaleEvents(db *sql.DB, sourceID int64, keepUIDs map[string]struct{}) error {
	rows, err := db.Query(
		`SELECT uid FROM calendar_events WHERE source_id = ?`, sourceID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		if _, ok := keepUIDs[uid]; !ok {
			toDelete = append(toDelete, uid)
		}
	}
	rows.Close()

	for _, uid := range toDelete {
		_, _ = db.Exec(
			`DELETE FROM calendar_events WHERE source_id = ? AND uid = ?`,
			sourceID, uid,
		)
	}
	return nil
}
