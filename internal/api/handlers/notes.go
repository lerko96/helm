package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type NoteFolder struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Note struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	FolderID  *int64     `json:"folder_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	IsPinned  bool       `json:"is_pinned"`
	DueDate   *time.Time `json:"due_date"`
	Tags      []Tag      `json:"tags,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ── Folders ───────────────────────────────────────────────────────────────────

func ListNoteFolders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(),
			`SELECT id, user_id, name, created_at FROM note_folders WHERE user_id = ? ORDER BY name`,
			defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		folders := []NoteFolder{}
		for rows.Next() {
			var f NoteFolder
			if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			folders = append(folders, f)
		}
		respond(w, http.StatusOK, folders)
	}
}

func CreateNoteFolder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required")
			return
		}
		res, err := db.ExecContext(r.Context(),
			`INSERT INTO note_folders (user_id, name) VALUES (?, ?)`,
			defaultUserID, req.Name,
		)
		if err != nil {
			respondError(w, http.StatusConflict, "folder already exists or insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func DeleteNoteFolder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM note_folders WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

// ── Notes ─────────────────────────────────────────────────────────────────────

func ListNotes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := `
			SELECT id, user_id, folder_id, title, content, is_pinned, due_date, created_at, updated_at
			FROM notes WHERE user_id = ?`
		args := []any{defaultUserID}

		if v := q.Get("folder_id"); v != "" {
			query += " AND folder_id = ?"
			args = append(args, v)
		}
		if q.Get("pinned") == "true" {
			query += " AND is_pinned = 1"
		}
		if s := q.Get("q"); s != "" {
			query += " AND (title LIKE ? OR content LIKE ?)"
			args = append(args, "%"+s+"%", "%"+s+"%")
		}
		query += " ORDER BY is_pinned DESC, updated_at DESC"

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		notes := []Note{}
		for rows.Next() {
			var n Note
			if err := rows.Scan(&n.ID, &n.UserID, &n.FolderID, &n.Title, &n.Content, &n.IsPinned, &n.DueDate, &n.CreatedAt, &n.UpdatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			notes = append(notes, n)
		}
		respond(w, http.StatusOK, notes)
	}
}

func GetNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var n Note
		err = db.QueryRowContext(r.Context(), `
			SELECT id, user_id, folder_id, title, content, is_pinned, due_date, created_at, updated_at
			FROM notes WHERE id = ? AND user_id = ?`, id, defaultUserID,
		).Scan(&n.ID, &n.UserID, &n.FolderID, &n.Title, &n.Content, &n.IsPinned, &n.DueDate, &n.CreatedAt, &n.UpdatedAt)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "note not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}

		n.Tags, _ = GetEntityTags(db, "note", n.ID)
		respond(w, http.StatusOK, n)
	}
}

func CreateNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FolderID *int64     `json:"folder_id"`
			Title    string     `json:"title"`
			Content  string     `json:"content"`
			IsPinned bool       `json:"is_pinned"`
			DueDate  *time.Time `json:"due_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		res, err := db.ExecContext(r.Context(), `
			INSERT INTO notes (user_id, folder_id, title, content, is_pinned, due_date)
			VALUES (?, ?, ?, ?, ?, ?)`,
			defaultUserID, req.FolderID, req.Title, req.Content, req.IsPinned, req.DueDate,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func UpdateNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req struct {
			FolderID *int64     `json:"folder_id"`
			Title    string     `json:"title"`
			Content  string     `json:"content"`
			IsPinned bool       `json:"is_pinned"`
			DueDate  *time.Time `json:"due_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		_, err = db.ExecContext(r.Context(), `
			UPDATE notes SET folder_id=?, title=?, content=?, is_pinned=?, due_date=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND user_id=?`,
			req.FolderID, req.Title, req.Content, req.IsPinned, req.DueDate, id, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "update failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func DeleteNote(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM notes WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}
