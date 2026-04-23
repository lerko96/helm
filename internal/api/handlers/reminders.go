package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Reminder struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	EntityType string    `json:"entity_type"`
	EntityID   int64     `json:"entity_id"`
	RemindAt   time.Time `json:"remind_at"`
	IsSent     bool      `json:"is_sent"`
	CreatedAt  time.Time `json:"created_at"`
}

func ListReminders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT id, user_id, entity_type, entity_id, remind_at, is_sent, created_at
			FROM reminders
			WHERE user_id = ?
			ORDER BY remind_at ASC
		`, defaultUserID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		reminders := []Reminder{}
		for rows.Next() {
			var rem Reminder
			if err := rows.Scan(&rem.ID, &rem.UserID, &rem.EntityType, &rem.EntityID, &rem.RemindAt, &rem.IsSent, &rem.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			reminders = append(reminders, rem)
		}
		if err := rows.Err(); err != nil {
			respondError(w, http.StatusInternalServerError, "row iteration failed")
			return
		}
		respond(w, http.StatusOK, reminders)
	}
}

func CreateReminder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntityType string    `json:"entity_type"`
			EntityID   int64     `json:"entity_id"`
			RemindAt   time.Time `json:"remind_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.EntityType == "" || req.EntityID == 0 {
			respondError(w, http.StatusBadRequest, "entity_type and entity_id are required")
			return
		}
		if !validEntityType(req.EntityType) {
			respondError(w, http.StatusBadRequest, "invalid entity_type")
			return
		}

		res, err := db.ExecContext(r.Context(),
			`INSERT INTO reminders (user_id, entity_type, entity_id, remind_at) VALUES (?, ?, ?, ?)`,
			defaultUserID, req.EntityType, req.EntityID, req.RemindAt,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func DeleteReminder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM reminders WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}
