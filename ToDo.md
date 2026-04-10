 To run it locally right now:
  # 1. Create your config
  cp config.example.yml config.yml
  # Edit config.yml — set password and secret

  # 2. Start the backend
  go run ./cmd/helm

  # 3. In another terminal, start the frontend
  cd web && npm run dev

  What's in place:
  - Full SQLite schema with all tables, indexes, and comments (001_initial.sql)
  - Working REST API — all 6 features with full CRUD + shared tag/reminder primitives
  - JWT auth (single-user, password from config)
  - Public memo share links via /s/:token
  - YAML config-driven layout (Glance-style)
  - Tailwind v4 + React + TanStack Query wired up
  - Multi-stage Dockerfile + docker-compose ready

  What comes next (your call on order):
  1. Style guide / design system — drop those reference images whenever ready
  2. Frontend feature components (one at a time, starting with whichever feature matters
  most to you)
  3. CalDAV sync implementation (internal/caldav/)
  4. Attachment upload handler
  5. Recurring task scheduler
