package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lerko/helm/internal/config"
)

func TestDockerHandler_DisabledReturns404(t *testing.T) {
	cfg := &config.Config{Docker: config.DockerConfig{Enabled: false}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docker/containers", nil)
	ListContainers(cfg)(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDockerHandler_SocketDownReturns503(t *testing.T) {
	// Point at a socket path that doesn't exist — Dial will fail.
	cfg := &config.Config{Docker: config.DockerConfig{
		Enabled: true,
		Socket:  filepath.Join(t.TempDir(), "nope.sock"),
	}}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docker/containers", nil)
	ListContainers(cfg)(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestDockerHandler_HappyPath(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "docker.sock")

	// Fake docker engine. Listens on a unix socket and serves one endpoint.
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	defer os.Remove(sock)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/containers/json" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`[
				{"Id":"abc","Names":["/web"],"Image":"nginx","State":"running","Status":"Up 5m","Created":1700000000},
				{"Id":"def","Names":["/db"],"Image":"postgres","State":"exited","Status":"Exited (0) 1h ago","Created":1699999000}
			]`))
		}),
	}
	go srv.Serve(ln)
	defer srv.Close()

	cfg := &config.Config{Docker: config.DockerConfig{Enabled: true, Socket: sock}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docker/containers", nil)
	ListContainers(cfg)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var got []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Image string `json:"image"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 containers, got %d", len(got))
	}
	if got[0].Name != "web" { // leading slash stripped
		t.Errorf("Name[0] = %q, want %q", got[0].Name, "web")
	}
	if got[1].State != "exited" {
		t.Errorf("State[1] = %q", got[1].State)
	}
}

func TestDockerHandler_EngineNon200Returns502(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "docker.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	}
	go srv.Serve(ln)
	defer srv.Close()

	cfg := &config.Config{Docker: config.DockerConfig{Enabled: true, Socket: sock}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docker/containers", nil)
	ListContainers(cfg)(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", w.Code)
	}
}
