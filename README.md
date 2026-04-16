# Helm

Self-hosted personal productivity dashboard. Think [Glance](https://github.com/glanceapp/glance), but for managing your day instead of watching feeds.

**Notes · Todos · Calendar · Clipboard · Bookmarks · Memos**

Layout is YAML-defined — multiple pages, multiple columns, widgets stacked per column.

---

## Stack

- **Backend:** Go + chi + SQLite (`modernc.org/sqlite`, no CGO)
- **Frontend:** React + TypeScript + Vite + Tailwind v4
- **Auth:** Single-user JWT, 30-day expiry, rate-limited login
- **Deploy:** Docker or bare metal

---

## Deploy

```bash
cp config.example.yml config.yml
# edit config.yml — set auth.password and auth.secret at minimum
```

```yaml
# docker-compose.yml
services:
  helm:
    image: ghcr.io/lerko96/helm:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yml:/config/config.yml:ro
      - helm-data:/data
    restart: unless-stopped

volumes:
  helm-data:
```

```bash
docker compose up -d
```

Serves on `:8080`. Pin a release by replacing `latest` with a tag e.g. `2026.04.1`.

---

## Dev

```bash
cp config.example.yml config.yml

# backend
go run ./cmd/helm

# frontend (HMR, proxies /api/* to :8080)
cd web && npm run dev
```

---

## Widgets

| Type | What it does |
|------|-------------|
| `memos` | Short-form feed. Public/private, share tokens, markdown, pinning |
| `todos` | Flat task list with priority, due date, status |
| `task-board` | Kanban view of the same tasks. Expandable detail: subtasks, description, tags, due date, priority |
| `task-lists` | Manage todo lists — create with color, delete |
| `calendar` | Month view + local event creation. CalDAV source management via `cal-sources` |
| `cal-sources` | Add/delete CalDAV sources, trigger manual sync |
| `clipboard` | Paste store with optional title and language. Code items render in `<pre>` blocks |
| `bookmarks` | URL store. Collections, public/private, pin, tags |
| `notes-folders` | Folder picker for notes |
| `notes-editor` | Note list + editor with markdown view/edit toggle, tags, pin |
| `tags` | Tag management — create, delete, view all |

All widgets support global search from the header bar. Tags attach/detach from every entity type.

---

## What works

- Full CRUD on every entity: notes, todos, memos, bookmarks, clipboard, calendar events, CalDAV sources
- Tags: create, attach, detach — wired to notes, todos, memos, bookmarks, clipboard
- Markdown rendering: notes (edit/view toggle), memos (expand to render)
- Task board: status cycling (not started → in progress → done), priority cycling, subtasks, description, due date
- Todo lists with color
- Memos: public/private visibility, share tokens, pin
- Bookmarks: collections, edit, pin, public/private
- Calendar: local events, source management UI
- Global search: FTS5 full-text search across all widgets from the header bar
- File attachments: upload/download/delete on notes
- Reminders: set on any entity; due reminders fire as browser notifications via SSE
- Recurring todos: daily/weekly/monthly; new copies spawn on schedule
- CalDAV sync: etag deduplication, auto every 15 min, manual sync from `cal-sources` widget

---

## Config

```yaml
server:
  host: 0.0.0.0
  port: 8080

auth:
  password: changeme
  secret: replace-with-a-long-random-string  # openssl rand -hex 32

storage:
  db_path: ./data/helm.db
  attachments_path: ./data/attachments

pages:
  - name: Home
    columns:
      - size: small
        widgets:
          - type: memos
          - type: todos
      - size: large
        widgets:
          - type: task-board
      - size: small
        widgets:
          - type: notes-editor
          - type: bookmarks
          - type: clipboard
```

Column sizes: `small` (~25%), `medium` (~33%), `large` (~50%).

---

## Upgrading

```bash
docker compose pull && docker compose up -d
```

Migrations run automatically on startup. Data persists in the `helm-data` named volume. No manual steps needed between releases.

---

## Health check

`GET /healthz` returns `{"status":"ok"}` — use this for reverse proxy health probes (Caddy, nginx, Traefik).

```
GET /healthz → 200 {"status":"ok"}
```

---

## Reverse proxy (HTTPS)

**Caddy** (`/etc/caddy/Caddyfile`):
```
helm.example.com {
    reverse_proxy localhost:8080
}
```

**nginx** (`/etc/nginx/sites-available/helm`):
```nginx
server {
    listen 443 ssl;
    server_name helm.example.com;
    # ssl_certificate / ssl_certificate_key ...

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## Bare metal deploy (systemd)

Build the binary, place it and the config, then install the service:

```bash
go build -o /usr/local/bin/helm ./cmd/helm
cp config.yml /etc/helm/config.yml
```

`/etc/systemd/system/helm.service`:
```ini
[Unit]
Description=Helm dashboard
After=network.target

[Service]
ExecStart=/usr/local/bin/helm /etc/helm/config.yml
WorkingDirectory=/var/lib/helm
User=helm
Group=helm
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

```bash
useradd -r -s /sbin/nologin helm
mkdir -p /var/lib/helm/data
chown -R helm:helm /var/lib/helm
systemctl enable --now helm
```

---

## Backup and restore

Helm stores all data in two places:

```bash
# Backup
cp /var/lib/helm/data/helm.db helm.db.bak
cp -r /var/lib/helm/data/attachments attachments.bak

# Restore (stop Helm first)
systemctl stop helm
cp helm.db.bak /var/lib/helm/data/helm.db
cp -r attachments.bak /var/lib/helm/data/attachments
systemctl start helm
```

For Docker, the named volume `helm-data` contains both. Use `docker cp` or mount a backup container.

---

## Troubleshooting

**Helm won't start:**
- `auth.secret must be at least 32 characters` — generate a proper secret: `openssl rand -hex 32`
- Port already in use — change `server.port` in config.yml

**401 Unauthorized after config change:**
- Changing `auth.secret` invalidates all existing sessions. Log in again.
- CalDAV passwords are encrypted with `auth.secret`. After rotating the secret, re-enter CalDAV passwords in the cal-sources widget.

**Logs:** Helm writes to stdout/stderr. With Docker: `docker compose logs -f helm`. With systemd: `journalctl -u helm -f`.

**Database locked:** SQLite uses WAL mode with a single writer. If Helm was killed mid-write, the WAL file recovers automatically on next startup.

---

## Security notes

- Single-user only. There is no CSRF protection — use HTTPS and a reverse proxy in front.
- Rate limiting applies only to `POST /api/auth/login` (10 attempts/min per IP). Put Helm behind a firewall; do not expose port 8080 directly.
- CalDAV source passwords are stored encrypted (AES-256-GCM, key derived from `auth.secret`).
