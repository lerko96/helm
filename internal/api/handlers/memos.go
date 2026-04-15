package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lerko/helm/internal/broker"
)

type Memo struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Content    string    `json:"content"`
	Visibility string    `json:"visibility"`
	ShareToken *string   `json:"share_token,omitempty"`
	IsPinned   bool      `json:"is_pinned"`
	Tags       []Tag     `json:"tags,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ListMemos(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		query := `
			SELECT id, user_id, content, visibility, share_token, is_pinned, created_at, updated_at
			FROM memos WHERE user_id = ?`
		args := []any{defaultUserID}

		if v := q.Get("visibility"); v != "" {
			query += " AND visibility = ?"
			args = append(args, v)
		}
		if q.Get("pinned") == "true" {
			query += " AND is_pinned = 1"
		}
		if s := q.Get("q"); s != "" {
			query += " AND id IN (SELECT rowid FROM memos_fts WHERE memos_fts MATCH ?)"
			args = append(args, sanitizeFTSQuery(s))
		}
		query += " ORDER BY is_pinned DESC, created_at DESC"

		rows, err := db.QueryContext(r.Context(), query, args...)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		memos := []Memo{}
		ids := []int64{}
		for rows.Next() {
			var m Memo
			if err := rows.Scan(&m.ID, &m.UserID, &m.Content, &m.Visibility, &m.ShareToken, &m.IsPinned, &m.CreatedAt, &m.UpdatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			memos = append(memos, m)
			ids = append(ids, m.ID)
		}
		tagMap := batchGetEntityTags(db, "memo", ids)
		for i := range memos {
			if tags, ok := tagMap[memos[i].ID]; ok {
				memos[i].Tags = tags
			}
		}
		respond(w, http.StatusOK, memos)
	}
}

func CreateMemo(db *sql.DB, b *broker.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Content    string `json:"content"`
			Visibility string `json:"visibility"`
			IsPinned   bool   `json:"is_pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Content == "" {
			respondError(w, http.StatusBadRequest, "content is required")
			return
		}
		if req.Visibility == "" {
			req.Visibility = "private"
		}

		var shareToken *string
		if req.Visibility == "public" {
			t, err := generateToken()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "failed to generate share token")
				return
			}
			shareToken = &t
		}

		res, err := db.ExecContext(r.Context(),
			`INSERT INTO memos (user_id, content, visibility, share_token, is_pinned) VALUES (?, ?, ?, ?, ?)`,
			defaultUserID, req.Content, req.Visibility, shareToken, req.IsPinned,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "insert failed")
			return
		}
		id, _ := res.LastInsertId()
		publishMutation(b, "memo", "create")
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func UpdateMemo(db *sql.DB, b *broker.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req struct {
			Content    string `json:"content"`
			Visibility string `json:"visibility"`
			IsPinned   bool   `json:"is_pinned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Fetch current visibility to manage share_token transitions.
		var currentVisibility string
		var currentToken *string
		db.QueryRowContext(r.Context(), //nolint:errcheck
			`SELECT visibility, share_token FROM memos WHERE id = ? AND user_id = ?`, id, defaultUserID,
		).Scan(&currentVisibility, &currentToken)

		var shareToken *string
		switch {
		case req.Visibility == "public" && currentVisibility != "public":
			// Transitioning to public: generate a new token.
			t, err := generateToken()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "failed to generate share token")
				return
			}
			shareToken = &t
		case req.Visibility == "public":
			// Already public: keep existing token.
			shareToken = currentToken
		default:
			// Private: clear the token.
			shareToken = nil
		}

		_, err = db.ExecContext(r.Context(), `
			UPDATE memos SET content=?, visibility=?, share_token=?, is_pinned=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND user_id=?`,
			req.Content, req.Visibility, shareToken, req.IsPinned, id, defaultUserID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "update failed")
			return
		}
		publishMutation(b, "memo", "update")
		respond(w, http.StatusNoContent, nil)
	}
}

func DeleteMemo(db *sql.DB, b *broker.Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM memos WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		publishMutation(b, "memo", "delete")
		respond(w, http.StatusNoContent, nil)
	}
}

// GetSharedMemo serves a public memo by its share token — no auth required.
func GetSharedMemo(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		if token == "" {
			respondError(w, http.StatusBadRequest, "missing token")
			return
		}

		var m Memo
		err := db.QueryRowContext(r.Context(), `
			SELECT id, user_id, content, visibility, share_token, is_pinned, created_at, updated_at
			FROM memos WHERE share_token = ? AND visibility = 'public'`, token,
		).Scan(&m.ID, &m.UserID, &m.Content, &m.Visibility, &m.ShareToken, &m.IsPinned, &m.CreatedAt, &m.UpdatedAt)
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "memo not found")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		respond(w, http.StatusOK, m)
	}
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
