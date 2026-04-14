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

## Run

```bash
cp config.example.yml config.yml
# set auth.password and auth.secret in config.yml

# backend
go run ./cmd/helm

# frontend (dev)
cd web && npm run dev
```

Docker:

```bash
docker compose up --build
```

Serves on `:8080` by default.

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
