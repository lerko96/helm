package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Tag struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

func ListTags(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		entityType := q.Get("entity_type")
		entityIDStr := q.Get("entity_id")

		var (
			rows *sql.Rows
			err  error
		)

		if entityType != "" && entityIDStr != "" {
			entityID, convErr := strconv.ParseInt(entityIDStr, 10, 64)
			if convErr != nil {
				respondError(w, http.StatusBadRequest, "invalid entity_id")
				return
			}
			rows, err = db.QueryContext(r.Context(), `
				SELECT t.id, t.user_id, t.name, t.color, t.created_at
				FROM tags t
				JOIN entity_tags et ON et.tag_id = t.id
				WHERE t.user_id = ? AND et.entity_type = ? AND et.entity_id = ?
				ORDER BY t.name`,
				defaultUserID, entityType, entityID,
			)
		} else {
			rows, err = db.QueryContext(r.Context(),
				`SELECT id, user_id, name, color, created_at FROM tags WHERE user_id = ? ORDER BY name`,
				defaultUserID,
			)
		}

		if err != nil {
			respondError(w, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()

		tags := []Tag{}
		for rows.Next() {
			var t Tag
			if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
				respondError(w, http.StatusInternalServerError, "scan failed")
				return
			}
			tags = append(tags, t)
		}
		if err := rows.Err(); err != nil {
			respondError(w, http.StatusInternalServerError, "row iteration failed")
			return
		}
		respond(w, http.StatusOK, tags)
	}
}

func CreateTag(db *sql.DB) http.HandlerFunc {
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
			`INSERT INTO tags (user_id, name, color) VALUES (?, ?, ?)`,
			defaultUserID, req.Name, req.Color,
		)
		if err != nil {
			respondError(w, http.StatusConflict, "tag already exists or insert failed")
			return
		}
		id, _ := res.LastInsertId()
		respond(w, http.StatusCreated, map[string]int64{"id": id})
	}
}

func DeleteTag(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM tags WHERE id = ? AND user_id = ?`, id, defaultUserID,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

// ── Entity tag HTTP handlers ──────────────────────────────────────────────────

func AttachTagHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagID, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid tag id")
			return
		}
		var req struct {
			EntityType string `json:"entity_type"`
			EntityID   int64  `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if !validEntityType(req.EntityType) {
			respondError(w, http.StatusBadRequest, "invalid entity_type")
			return
		}
		if err := AttachTag(db, req.EntityType, req.EntityID, tagID); err != nil {
			respondError(w, http.StatusInternalServerError, "attach failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

func DetachTagHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagID, err := idParam(r)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid tag id")
			return
		}
		var req struct {
			EntityType string `json:"entity_type"`
			EntityID   int64  `json:"entity_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if !validEntityType(req.EntityType) {
			respondError(w, http.StatusBadRequest, "invalid entity_type")
			return
		}
		if err := DetachTag(db, req.EntityType, req.EntityID, tagID); err != nil {
			respondError(w, http.StatusInternalServerError, "detach failed")
			return
		}
		respond(w, http.StatusNoContent, nil)
	}
}

// ── Entity tag helpers ────────────────────────────────────────────────────────

func AttachTag(db *sql.DB, entityType string, entityID, tagID int64) error {
	_, err := db.Exec(
		`INSERT OR IGNORE INTO entity_tags (tag_id, entity_type, entity_id) VALUES (?, ?, ?)`,
		tagID, entityType, entityID,
	)
	return err
}

func DetachTag(db *sql.DB, entityType string, entityID, tagID int64) error {
	_, err := db.Exec(
		`DELETE FROM entity_tags WHERE tag_id = ? AND entity_type = ? AND entity_id = ?`,
		tagID, entityType, entityID,
	)
	return err
}

// batchGetEntityTags fetches tags for multiple entity IDs in a single query.
// Returns a map of entityID → []Tag.
func batchGetEntityTags(db *sql.DB, entityType string, ids []int64) map[int64][]Tag {
	result := make(map[int64][]Tag, len(ids))
	if len(ids) == 0 {
		return result
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, entityType)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		SELECT et.entity_id, t.id, t.user_id, t.name, t.color, t.created_at
		FROM tags t
		JOIN entity_tags et ON et.tag_id = t.id
		WHERE et.entity_type = ? AND et.entity_id IN (%s)
		ORDER BY t.name`, strings.Join(placeholders, ","))
	rows, err := db.Query(q, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var entityID int64
		var t Tag
		if err := rows.Scan(&entityID, &t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			continue
		}
		result[entityID] = append(result[entityID], t)
	}
	return result
}

func GetEntityTags(db *sql.DB, entityType string, entityID int64) ([]Tag, error) {
	rows, err := db.Query(`
		SELECT t.id, t.user_id, t.name, t.color, t.created_at
		FROM tags t
		JOIN entity_tags et ON et.tag_id = t.id
		WHERE et.entity_type = ? AND et.entity_id = ?
		ORDER BY t.name
	`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}
