package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type ClipboardItem struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     *string   `json:"title"`
	Content   string    `json:"content"`
	Language  *string   `json:"language"`
	IsPinned  bool      `json:"is_pinned"`
	Tags      []Tag     `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ListClipboardItems(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := `
			SELECT id, user_id, title, content, language, is_pinned, created_at, updated_at
			FROM clipboard_items WHERE user_id = ?`
		args := []any{defaultUserID}

		if v := q.Get("language"); v != "" {
			query += " AND language = ?"
			args = append(args, v)
		}
		if q.Get("pinned") == "true" {
			query += " AND is_pinned = 1"
		}
		if s := q.Get("q"); s != "" {
			query += " AND id IN (SELECT rowid FROM clipboard_fts WHERE clipboard_fts MATCH ?)"
			args = append(args, sanitizeFTSQuery(s))
		}
		query += " ORDER BY is_pinned DESC, created_at DESC"

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		items := []ClipboardItem{}
		for rows.Next() {
			var item ClipboardItem
			if err := rows.Scan(&item.ID, &item.UserID, &item.Title, &item.Content, &item.Language, &item.IsPinned, &item.CreatedAt, &item.UpdatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			items = append(items, item)
		}
		respond(w, http.StatusOK, items)
	}
}

func GetClipboardItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var item ClipboardItem
		err = db.QueryRowContext(r.Context(), `
			SELECT id, user_id, title, content, language, is_pinned, created_at, updated_at
			FROM clipboard_items WHERE id = ? AND user_id = ?`, id, defaultUserID,
		).Scan(&item.ID, &item.UserID, &item.Title, &item.Content, &item.Language, &item.IsPinned, &item.CreatedAt, &item.UpdatedAt)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "item not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}

		item.Tags, _ = GetEntityTags(db, "clipboard", item.ID)
		respond(w, http.StatusOK, item)
	}
}

func CreateClipboardItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title    *string `json:"title"`
			Content  string  `json:"content"`
			Language *string `json:"language"`
			IsPinned bool    `json:"is_pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Content == "" {
			respondError(w, http.StatusBadRequest, "content is required")
			return
		}

		res, err := db.ExecContext(r.Context(),
			`INSERT INTO clipboard_items (user_id, title, content, language, is_pinned) VALUES (?, ?, ?, ?, ?)`,
			defaultUserID, req.Title, req.Content, req.Language, req.IsPinned,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func UpdateClipboardItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req struct {
			Title    *string `json:"title"`
			Content  string  `json:"content"`
			Language *string `json:"language"`
			IsPinned bool    `json:"is_pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		_, err = db.ExecContext(r.Context(), `
			UPDATE clipboard_items SET title=?, content=?, language=?, is_pinned=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND user_id=?`,
			req.Title, req.Content, req.Language, req.IsPinned, id, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "update failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func DeleteClipboardItem(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM clipboard_items WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}
