# Helm

A personal productivity dashboard. Think [Glance](https://github.com/glanceapp/glance), but for managing your day.

**Notes · Todos · Calendar · Snippets · Bookmarks · Memos**

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

- **Notes** — with folders, tags, attachments, full-text search
- **Todos** — lists, subtasks (one level), recurring tasks, due dates, reminders
- **Calendar** — CalDAV sync, not Google OAuth
- **Snippets** — clipboard manager / code snippets
- **Bookmarks** — organized into collections
- **Memos** — short-form feed, public/private with share tokens

Shared across everything: tags, attachments, pinning, markdown, reminders, full-text search.

## Config

Dashboard layout is YAML-based (Glance-style). See `config.example.yml`.
