# Helm

A personal productivity dashboard. Think [Glance](https://github.com/glanceapp/glance), but for managing your day.

**Notes · Todos · Calendar · Clipboard · Bookmarks · Memos**

## Stack

- **Backend:** Go + chi + SQLite (no CGO)
- **Frontend:** React + TypeScript + Vite + Tailwind v4
- **Auth:** Single-user, JWT
- **Sync:** CalDAV for calendar

## Run

```bash
cp config.example.yml config.yml
# set password + secret in config.yml

# backend
go run ./cmd/helm

# frontend (dev)
cd web && npm run dev
```

Or with Docker:

```bash
docker compose up --build
```

## Features

- **Notes** — folders, tags, attachments, full-text search
- **Todos** — lists, subtasks (one level), recurring tasks, due dates, kanban board
- **Calendar** — CalDAV sync, not Google OAuth
- **Clipboard** — code snippets with language hints, copy-to-clipboard
- **Bookmarks** — organized into collections, public/private
- **Memos** — short-form feed, public/private with share tokens

Shared across everything: tags, attachments, pinning, markdown, reminders, full-text search.

## Config

Dashboard layout is YAML-based (Glance-style), supports multiple pages and columns. See `config.example.yml`.

Widget types: `memos`, `todos`, `calendar`, `clipboard`, `bookmarks`, `notes-folders`, `notes-editor`, `task-lists`, `task-board`
