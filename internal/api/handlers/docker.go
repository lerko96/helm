package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lerko/helm/internal/config"
	"github.com/lerko/helm/internal/docker"
)

// ListContainers returns the docker-status endpoint handler.
//
//   - docker.enabled=false → 404 (route behaves as if it doesn't exist;
//     simpler than a distinct "disabled" code, and it prevents enumeration
//     from telling an attacker whether the socket is mounted).
//   - socket unreachable   → 503 with {"error": "docker unavailable: ..."}
//   - engine error         → 502
//   - happy path           → 200 JSON array
func ListContainers(cfg *config.Config) http.HandlerFunc {
	if !cfg.Docker.Enabled {
		return func(w http.ResponseWriter, _ *http.Request) {
			respondError(w, http.StatusNotFound, "not found")
		}
	}
	client := docker.New(cfg.Docker.Socket)
	return func(w http.ResponseWriter, r *http.Request) {
		containers, err := client.ListContainers(r.Context())
		if err != nil {
			var unavail *docker.ErrUnavailable
			if errors.As(err, &unavail) {
				respondError(w, http.StatusServiceUnavailable, unavail.Error())
				return
			}
			respondError(w, http.StatusBadGateway, "docker engine error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(containers)
	}
}
