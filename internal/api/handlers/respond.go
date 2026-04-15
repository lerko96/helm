package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lerko/helm/internal/broker"
)

// defaultUserID is used for all queries until multi-user is implemented.
// Every table has a user_id column already, so enabling multi-user later
// only requires auth middleware to inject the real user ID.
const defaultUserID = 1

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

func idParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// publishMutation broadcasts a mutation event to all SSE clients.
// entity_type: note, todo, memo, bookmark, clipboard
// action: create, update, delete
func publishMutation(b *broker.Broker, entityType, action string) {
	if b == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"type":        "mutation",
		"entity_type": entityType,
		"action":      action,
	})
	b.Publish(string(payload))
}
