# Changelog

All notable changes to Helm are documented here.

## [2026.04.1] — 2026-04-15

Initial public release.

### Features

- **11 widgets** — memos, todos (list + kanban board), notes (editor + folders), calendar (views + sources), bookmarks, clipboard, tags
- **Full CRUD** on every entity with real-time SSE updates pushed to all open tabs
- **Tags** — create, color, attach/detach across notes, todos, memos, bookmarks, clipboard
- **Markdown rendering** — notes (edit/view toggle), memos (expand to render), clipboard code blocks
- **Task board** — Kanban (not started / in progress / done), subtasks, priority cycling, due dates, description
- **Recurring todos** — daily/weekly/monthly schedules; new copies spawn automatically
- **Reminders** — set on any entity; fire as browser notifications via SSE when due
- **File attachments** — upload/download/delete on notes (20 MB max, polymorphic)
- **CalDAV sync** — etag deduplication, auto every 15 min, manual trigger, HTTP Digest auth
- **Global search** — FTS5 full-text search across all widget types from the header bar
- **Public sharing** — memos and bookmarks support public visibility + share tokens
- **YAML layout config** — define pages, columns (small/medium/large), and widget stacks in `config.yml`
- **Single-binary deploy** — Go binary embeds the React SPA; no separate static file server needed

### Security & Reliability (phases 12–13)

- JWT auth with HMAC-SHA256, 30-day expiry
- Per-IP rate limiter on `/api/auth/login` (10 attempts/min)
- AES-256-GCM encryption for CalDAV source passwords (key derived from config secret)
- CGO-free SQLite (`modernc.org/sqlite`); WAL mode; `PRAGMA foreign_keys = ON`
- Strict input validation and SQL injection prevention throughout
- Docker image built from Alpine; no shell, no package manager in runtime layer

### Infrastructure

- Multi-platform Docker image on GHCR (`linux/amd64`, `linux/arm64`)
- GitHub Actions CI: Go build + vet, frontend build, Docker build, GHCR publish on tag
- CalVer versioning (`YYYY.0M.MICRO`)
