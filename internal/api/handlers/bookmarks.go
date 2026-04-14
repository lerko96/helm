package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type BookmarkCollection struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Bookmark struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	CollectionID *int64     `json:"collection_id"`
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	Description  *string    `json:"description"`
	FaviconURL   *string    `json:"favicon_url"`
	IsPinned     bool       `json:"is_pinned"`
	IsPublic     bool       `json:"is_public"`
	Tags         []Tag      `json:"tags,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ── Collections ───────────────────────────────────────────────────────────────

func ListBookmarkCollections(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(),
			`SELECT id, user_id, name, created_at FROM bookmark_collections WHERE user_id = ? ORDER BY name`,
			defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		collections := []BookmarkCollection{}
		for rows.Next() {
			var c BookmarkCollection
			if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			collections = append(collections, c)
		}
		respond(w, http.StatusOK, collections)
	}
}

func CreateBookmarkCollection(db *sql.DB) http.HandlerFunc {
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
			`INSERT INTO bookmark_collections (user_id, name) VALUES (?, ?)`,
			defaultUserID, req.Name,
		)
		if err != nil {
			respondError(w, http.StatusConflict, "collection already exists or insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func DeleteBookmarkCollection(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM bookmark_collections WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

// ── Bookmarks ─────────────────────────────────────────────────────────────────

func ListBookmarks(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := `
			SELECT id, user_id, collection_id, url, title, description, favicon_url, is_pinned, is_public, created_at, updated_at
			FROM bookmarks WHERE user_id = ?`
		args := []any{defaultUserID}

		if v := q.Get("collection_id"); v != "" {
			query += " AND collection_id = ?"
			args = append(args, v)
		}
		if q.Get("pinned") == "true" {
			query += " AND is_pinned = 1"
		}
		if s := q.Get("q"); s != "" {
			query += " AND id IN (SELECT rowid FROM bookmarks_fts WHERE bookmarks_fts MATCH ?)"
			args = append(args, s+"*")
		}
		query += " ORDER BY is_pinned DESC, created_at DESC"

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		bookmarks := []Bookmark{}
		for rows.Next() {
			var b Bookmark
			if err := rows.Scan(&b.ID, &b.UserID, &b.CollectionID, &b.URL, &b.Title, &b.Description, &b.FaviconURL, &b.IsPinned, &b.IsPublic, &b.CreatedAt, &b.UpdatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			bookmarks = append(bookmarks, b)
		}
		respond(w, http.StatusOK, bookmarks)
	}
}

func CreateBookmark(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CollectionID *int64  `json:"collection_id"`
			URL          string  `json:"url"`
			Title        string  `json:"title"`
			Description  *string `json:"description"`
			FaviconURL   *string `json:"favicon_url"`
			IsPinned     bool    `json:"is_pinned"`
			IsPublic     bool    `json:"is_public"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.URL == "" {
			respondError(w, http.StatusBadRequest, "url is required")
			return
		}
		if req.Title == "" {
			req.Title = req.URL
		}

		res, err := db.ExecContext(r.Context(), `
			INSERT INTO bookmarks (user_id, collection_id, url, title, description, favicon_url, is_pinned, is_public)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			defaultUserID, req.CollectionID, req.URL, req.Title, req.Description, req.FaviconURL, req.IsPinned, req.IsPublic,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func UpdateBookmark(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req struct {
			CollectionID *int64  `json:"collection_id"`
			URL          string  `json:"url"`
			Title        string  `json:"title"`
			Description  *string `json:"description"`
			FaviconURL   *string `json:"favicon_url"`
			IsPinned     bool    `json:"is_pinned"`
			IsPublic     bool    `json:"is_public"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		_, err = db.ExecContext(r.Context(), `
			UPDATE bookmarks SET collection_id=?, url=?, title=?, description=?, favicon_url=?, is_pinned=?, is_public=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND user_id=?`,
			req.CollectionID, req.URL, req.Title, req.Description, req.FaviconURL, req.IsPinned, req.IsPublic, id, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "update failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func DeleteBookmark(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM bookmarks WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}
