package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/lerko/helm/internal/broker"
	"github.com/lerko/helm/internal/caldav"
)

type CalendarSource struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Name         string     `json:"name"`
	URL          *string    `json:"url"`
	Username     *string    `json:"username"`
	Color        string     `json:"color"`
	IsLocal      bool       `json:"is_local"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CalendarEvent struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	SourceID    int64     `json:"source_id"`
	UID         string    `json:"uid"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Location    *string   `json:"location"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	IsAllDay    bool      `json:"is_all_day"`
	RRule       *string   `json:"rrule"`
	Color       *string   `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ── Sources ───────────────────────────────────────────────────────────────────

func ListCalendarSources(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, user_id, name, url, username, color, is_local, last_synced_at, created_at
			FROM calendar_sources WHERE user_id = ? ORDER BY name`, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		sources := []CalendarSource{}
		for rows.Next() {
			var s CalendarSource
			if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.URL, &s.Username, &s.Color, &s.IsLocal, &s.LastSyncedAt, &s.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			sources = append(sources, s)
		}
		if err := rows.Err(); err != nil {
			respondError(w, http.StatusInternalServerError, "row iteration failed")
			return
		}
		respond(w, http.StatusOK, sources)
	}
}

func CreateCalendarSource(db *sql.DB, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string  `json:"name"`
			URL      *string `json:"url"`
			Username *string `json:"username"`
			Password *string `json:"password"`
			Color    string  `json:"color"`
			IsLocal  bool    `json:"is_local"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required")
			return
		}
		if !req.IsLocal && req.URL == nil {
			respondError(w, http.StatusBadRequest, "url is required for remote sources")
			return
		}
		if req.Color == "" {
			req.Color = "#3b82f6"
		}

		var passwordEnc *string
		if req.Password != nil && *req.Password != "" {
			enc, err := encryptString(*req.Password, secret)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "failed to encrypt password")
				return
			}
			passwordEnc = &enc
		}

		res, err := db.ExecContext(r.Context(), `
			INSERT INTO calendar_sources (user_id, name, url, username, password_enc, color, is_local)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			defaultUserID, req.Name, req.URL, req.Username, passwordEnc, req.Color, req.IsLocal,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func DeleteCalendarSource(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM calendar_sources WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

// SyncCalendarSource triggers an immediate CalDAV sync for the given source in a goroutine.
func SyncCalendarSource(db *sql.DB, secret string, b *broker.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var src caldav.CalendarSource
		var urlStr, username, passwordEnc sql.NullString
		err = db.QueryRowContext(r.Context(),
			`SELECT id, name, url, username, password_enc FROM calendar_sources WHERE id = ? AND user_id = ? AND is_local = 0`,
			id, defaultUserID,
		).Scan(&src.ID, &src.Name, &urlStr, &username, &passwordEnc)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "remote calendar source not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}

		src.URL = urlStr.String
		src.Username = username.String
		src.PasswordEnc = passwordEnc.String

		go func() {
			syncErr := caldav.SyncSource(db, src, secret)
			if syncErr != nil {
				log.Printf("caldav: manual sync source %d: %v", src.ID, syncErr)
			}
			payload, _ := json.Marshal(map[string]any{
				"type":      "caldav_synced",
				"source_id": src.ID,
				"error":     syncErr != nil,
			})
			b.Publish(string(payload))
		}()

		respond(w, http.StatusAccepted, map[string]string{"status": "sync queued"})
	}
}

// ── Events ────────────────────────────────────────────────────────────────────

func ListCalendarEvents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := `
			SELECT id, user_id, source_id, uid, title, description, location, start_at, end_at, is_all_day, rrule, color, created_at, updated_at
			FROM calendar_events WHERE user_id = ?`
		args := []any{defaultUserID}

		if v := q.Get("source_id"); v != "" {
			query += " AND source_id = ?"
			args = append(args, v)
		}
		if from := q.Get("from"); from != "" {
			query += " AND end_at >= ?"
			args = append(args, from)
		}
		if to := q.Get("to"); to != "" {
			query += " AND start_at <= ?"
			args = append(args, to)
		}
		query += " ORDER BY start_at ASC"

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		events := []CalendarEvent{}
		for rows.Next() {
			var e CalendarEvent
			if err := rows.Scan(&e.ID, &e.UserID, &e.SourceID, &e.UID, &e.Title, &e.Description, &e.Location, &e.StartAt, &e.EndAt, &e.IsAllDay, &e.RRule, &e.Color, &e.CreatedAt, &e.UpdatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			events = append(events, e)
		}
		if err := rows.Err(); err != nil {
			respondError(w, http.StatusInternalServerError, "row iteration failed")
			return
		}
		respond(w, http.StatusOK, events)
	}
}

func CreateCalendarEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SourceID    int64      `json:"source_id"`
			Title       string     `json:"title"`
			Description *string    `json:"description"`
			Location    *string    `json:"location"`
			StartAt     time.Time  `json:"start_at"`
			EndAt       time.Time  `json:"end_at"`
			IsAllDay    bool       `json:"is_all_day"`
			Color       *string    `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Title == "" {
			respondError(w, http.StatusBadRequest, "title is required")
			return
		}
		if req.StartAt.IsZero() || req.EndAt.IsZero() {
			respondError(w, http.StatusBadRequest, "start_at and end_at are required")
			return
		}

		uid := generateUID()
		res, err := db.ExecContext(r.Context(), `
			INSERT INTO calendar_events (user_id, source_id, uid, title, description, location, start_at, end_at, is_all_day, color)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			defaultUserID, req.SourceID, uid, req.Title, req.Description, req.Location, req.StartAt, req.EndAt, req.IsAllDay, req.Color,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func UpdateCalendarEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req struct {
			Title       string     `json:"title"`
			Description *string    `json:"description"`
			Location    *string    `json:"location"`
			StartAt     time.Time  `json:"start_at"`
			EndAt       time.Time  `json:"end_at"`
			IsAllDay    bool       `json:"is_all_day"`
			Color       *string    `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		_, err = db.ExecContext(r.Context(), `
			UPDATE calendar_events SET title=?, description=?, location=?, start_at=?, end_at=?, is_all_day=?, color=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND user_id=?`,
			req.Title, req.Description, req.Location, req.StartAt, req.EndAt, req.IsAllDay, req.Color, id, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "update failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func DeleteCalendarEvent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM calendar_events WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func generateUID() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b) + "@helm"
}
