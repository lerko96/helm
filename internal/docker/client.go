// Package docker is a minimal Docker Engine API client — just enough to
// power the docker-status widget. No Docker SDK dep; we speak raw HTTP over
// the unix socket and decode a narrow subset of /containers/json.
//
// Why not the SDK: it pulls ~10 MiB of transitive deps for a feature that
// needs one endpoint. The engine API is stable and the parts we touch
// haven't shifted in years.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrUnavailable is returned when the socket can't be reached. The caller
// distinguishes "socket down" (503 with message) from "request failed"
// (500) — widgets should render an empty state, not crash.
type ErrUnavailable struct{ Cause error }

func (e *ErrUnavailable) Error() string { return "docker unavailable: " + e.Cause.Error() }
func (e *ErrUnavailable) Unwrap() error { return e.Cause }

// Container is the trimmed shape we expose to the widget. Engine returns
// far more — we drop the rest to keep the contract explicit and small.
type Container struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	State   string   `json:"state"`  // running, exited, paused, restarting, ...
	Status  string   `json:"status"` // human-readable, e.g. "Up 2 hours"
	Created int64    `json:"created"`
	Ports   []string `json:"ports,omitempty"`
}

// Client is the minimal docker engine client. Rebuilt on each helm start;
// no long-lived connections (dial happens per request through the transport).
type Client struct {
	socket string
	http   *http.Client
}

// New returns a client that speaks to the docker engine on the given unix
// socket. Pass "/var/run/docker.sock" for the default.
func New(socket string) *Client {
	return &Client{
		socket: socket,
		http: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					d := net.Dialer{Timeout: 2 * time.Second}
					return d.DialContext(ctx, "unix", socket)
				},
			},
		},
	}
}

// engineContainer mirrors the subset of /containers/json we consume. Fields
// we don't use are omitted so adding them is a deliberate change.
type engineContainer struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	State   string   `json:"State"`
	Status  string   `json:"Status"`
	Created int64    `json:"Created"`
	Ports   []struct {
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
}

// ListContainers returns all containers, including stopped ones — the widget
// shows state color-coded, so filtering by state belongs in the UI, not here.
func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	// Host header is ignored by the engine but required by the HTTP/1.1 parser.
	// The scheme/host portion is a placeholder — DialContext ignores it.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json?all=true", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &ErrUnavailable{Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("engine returned %d", resp.StatusCode)
	}

	var raw []engineContainer
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode engine response: %w", err)
	}

	out := make([]Container, len(raw))
	for i, r := range raw {
		name := ""
		if len(r.Names) > 0 {
			// Engine prefixes names with `/`; strip it for display.
			name = r.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		ports := make([]string, 0, len(r.Ports))
		for _, p := range r.Ports {
			if p.PublicPort != 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}
		out[i] = Container{
			ID:      r.ID,
			Name:    name,
			Image:   r.Image,
			State:   r.State,
			Status:  r.Status,
			Created: r.Created,
			Ports:   ports,
		}
	}
	return out, nil
}
