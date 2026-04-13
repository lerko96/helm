package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// parseDueDate accepts "YYYY-MM-DD" or RFC3339; returns nil for empty string.
func parseDueDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	if t, err := time.Parse("2006-01-02", *s); err == nil {
		return &t, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type TodoList struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type Todo struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	ListID      *int64     `json:"list_id"`
	ParentID    *int64     `json:"parent_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	IsPinned    bool       `json:"is_pinned"`
	Tags        []Tag      `json:"tags,omitempty"`
	Subtasks    []Todo     `json:"subtasks,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ── Lists ─────────────────────────────────────────────────────────────────────

func ListTodoLists(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(),
			`SELECT id, user_id, name, color, created_at FROM todo_lists WHERE user_id = ? ORDER BY name`,
			defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		lists := []TodoList{}
		for rows.Next() {
			var l TodoList
			if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			lists = append(lists, l)
		}
		respond(w, http.StatusOK, lists)
	}
}

func CreateTodoList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required")
			return
		}
		if req.Color == "" {
			req.Color = "#6b7280"
		}
		res, err := db.ExecContext(r.Context(),
			`INSERT INTO todo_lists (user_id, name, color) VALUES (?, ?, ?)`,
			defaultUserID, req.Name, req.Color,
		)
		if err != nil {
			respondError(w, http.StatusConflict, "list already exists or insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func DeleteTodoList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM todo_lists WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

// ── Todos ─────────────────────────────────────────────────────────────────────

func ListTodos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// By default only return top-level todos; subtasks are fetched via GetTodo.
		query := `
			SELECT id, user_id, list_id, parent_id, title, description, status, priority, due_date, is_pinned, created_at, updated_at
			FROM todos WHERE user_id = ? AND parent_id IS NULL`
		args := []any{defaultUserID}

		if v := q.Get("list_id"); v != "" {
			query += " AND list_id = ?"
			args = append(args, v)
		}
		if v := q.Get("status"); v != "" {
			query += " AND status = ?"
			args = append(args, v)
		}
		if v := q.Get("priority"); v != "" {
			query += " AND priority = ?"
			args = append(args, v)
		}
		if q.Get("pinned") == "true" {
			query += " AND is_pinned = 1"
		}
		if s := q.Get("q"); s != "" {
			query += " AND (title LIKE ? OR description LIKE ?)"
			args = append(args, "%"+s+"%", "%"+s+"%")
		}
		query += " ORDER BY is_pinned DESC, due_date ASC NULLS LAST, updated_at DESC"

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		todos := []Todo{}
		for rows.Next() {
			var t Todo
			if err := rows.Scan(&t.ID, &t.UserID, &t.ListID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.IsPinned, &t.CreatedAt, &t.UpdatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			todos = append(todos, t)
		}
		respond(w, http.StatusOK, todos)
	}
}

func GetTodo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var t Todo
		err = db.QueryRowContext(r.Context(), `
			SELECT id, user_id, list_id, parent_id, title, description, status, priority, due_date, is_pinned, created_at, updated_at
			FROM todos WHERE id = ? AND user_id = ?`, id, defaultUserID,
		).Scan(&t.ID, &t.UserID, &t.ListID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.IsPinned, &t.CreatedAt, &t.UpdatedAt)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "todo not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}

		t.Tags, _ = GetEntityTags(db, "todo", t.ID)
		t.Subtasks = fetchSubtasks(db, t.ID)
		respond(w, http.StatusOK, t)
	}
}

func fetchSubtasks(db *sql.DB, parentID int64) []Todo {
	rows, err := db.Query(`
		SELECT id, user_id, list_id, parent_id, title, description, status, priority, due_date, is_pinned, created_at, updated_at
		FROM todos WHERE parent_id = ? ORDER BY created_at ASC`, parentID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	subtasks := []Todo{}
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.UserID, &t.ListID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.IsPinned, &t.CreatedAt, &t.UpdatedAt); err != nil {
			continue
		}
		subtasks = append(subtasks, t)
	}
	return subtasks
}

func CreateTodo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ListID      *int64  `json:"list_id"`
			ParentID    *int64  `json:"parent_id"`
			Title       string  `json:"title"`
			Description *string `json:"description"`
			Status      string  `json:"status"`
			Priority    string  `json:"priority"`
			DueDate     *string `json:"due_date"`
			IsPinned    bool    `json:"is_pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Title == "" {
			respondError(w, http.StatusBadRequest, "title is required")
			return
		}
		if req.Status == "" {
			req.Status = "not_started"
		}
		if req.Priority == "" {
			req.Priority = "medium"
		}
		dueDate, err := parseDueDate(req.DueDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid due_date format")
			return
		}

		// Enforce one-level subtask depth.
		if req.ParentID != nil {
			var existingParent *int64
			db.QueryRow(`SELECT parent_id FROM todos WHERE id = ?`, req.ParentID).Scan(&existingParent) //nolint:errcheck
			if existingParent != nil {
				respondError(w, http.StatusBadRequest, "subtasks cannot have subtasks (max one level deep)")
				return
			}
		}

		res, err := db.ExecContext(r.Context(), `
			INSERT INTO todos (user_id, list_id, parent_id, title, description, status, priority, due_date, is_pinned)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			defaultUserID, req.ListID, req.ParentID, req.Title, req.Description, req.Status, req.Priority, dueDate, req.IsPinned,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func UpdateTodo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req struct {
			ListID      *int64  `json:"list_id"`
			Title       string  `json:"title"`
			Description *string `json:"description"`
			Status      string  `json:"status"`
			Priority    string  `json:"priority"`
			DueDate     *string `json:"due_date"`
			IsPinned    bool    `json:"is_pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		dueDate, err := parseDueDate(req.DueDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid due_date format")
			return
		}

		_, err = db.ExecContext(r.Context(), `
			UPDATE todos SET list_id=?, title=?, description=?, status=?, priority=?, due_date=?, is_pinned=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND user_id=?`,
			req.ListID, req.Title, req.Description, req.Status, req.Priority, dueDate, req.IsPinned, id, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "update failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func DeleteTodo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM todos WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}
