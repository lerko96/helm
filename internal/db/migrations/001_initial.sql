-- +goose Up

-- ─── Shared Primitives ────────────────────────────────────────────────────────

CREATE TABLE tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 1,
    name       TEXT    NOT NULL,
    color      TEXT    NOT NULL DEFAULT '#6b7280',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

CREATE TABLE attachments (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL DEFAULT 1,
    filename      TEXT    NOT NULL,
    original_name TEXT    NOT NULL,
    mime_type     TEXT    NOT NULL,
    size          INTEGER NOT NULL,
    disk_path     TEXT    NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Polymorphic: one row per (entity_type, entity_id) <-> tag relationship.
-- entity_type values: 'note', 'todo', 'clipboard', 'bookmark', 'memo'
CREATE TABLE entity_tags (
    tag_id      INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    entity_type TEXT    NOT NULL,
    entity_id   INTEGER NOT NULL,
    PRIMARY KEY (tag_id, entity_type, entity_id)
);

CREATE INDEX idx_entity_tags_entity ON entity_tags(entity_type, entity_id);

-- Polymorphic: one row per (entity_type, entity_id) <-> attachment relationship.
CREATE TABLE entity_attachments (
    attachment_id INTEGER NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
    entity_type   TEXT    NOT NULL,
    entity_id     INTEGER NOT NULL,
    PRIMARY KEY (attachment_id, entity_type, entity_id)
);

CREATE INDEX idx_entity_attachments_entity ON entity_attachments(entity_type, entity_id);

CREATE TABLE reminders (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL DEFAULT 1,
    entity_type TEXT    NOT NULL,
    entity_id   INTEGER NOT NULL,
    remind_at   DATETIME NOT NULL,
    is_sent     INTEGER  NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reminders_remind_at ON reminders(remind_at) WHERE is_sent = 0;

-- ─── Notes ────────────────────────────────────────────────────────────────────

CREATE TABLE note_folders (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 1,
    name       TEXT    NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

CREATE TABLE notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 1,
    folder_id  INTEGER REFERENCES note_folders(id) ON DELETE SET NULL,
    title      TEXT    NOT NULL DEFAULT '',
    content    TEXT    NOT NULL DEFAULT '',
    is_pinned  INTEGER NOT NULL DEFAULT 0,
    due_date   DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notes_user ON notes(user_id, updated_at DESC);

-- ─── Todos ────────────────────────────────────────────────────────────────────

CREATE TABLE todo_lists (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 1,
    name       TEXT    NOT NULL,
    color      TEXT    NOT NULL DEFAULT '#6b7280',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

CREATE TABLE todos (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL DEFAULT 1,
    list_id     INTEGER REFERENCES todo_lists(id) ON DELETE SET NULL,
    -- parent_id non-null means this is a subtask (max one level deep, enforced in app)
    parent_id   INTEGER REFERENCES todos(id) ON DELETE CASCADE,
    title       TEXT    NOT NULL,
    description TEXT,
    -- status: not_started | in_progress | done
    status      TEXT    NOT NULL DEFAULT 'not_started',
    -- priority: low | medium | high
    priority    TEXT    NOT NULL DEFAULT 'medium',
    due_date    DATETIME,
    is_pinned   INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_todos_user ON todos(user_id, updated_at DESC);
CREATE INDEX idx_todos_parent ON todos(parent_id);

-- Stores RFC 5545 RRULE strings. Scheduler reads this to spawn new todo copies.
CREATE TABLE todo_recurrences (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    todo_id          INTEGER NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    rrule            TEXT    NOT NULL,
    next_occurrence  DATETIME,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ─── Calendar ─────────────────────────────────────────────────────────────────

CREATE TABLE calendar_sources (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL DEFAULT 1,
    name          TEXT    NOT NULL,
    -- url null when is_local = 1
    url           TEXT,
    username      TEXT,
    -- password stored encrypted (AES-GCM, key derived from auth.secret)
    password_enc  TEXT,
    color         TEXT    NOT NULL DEFAULT '#3b82f6',
    is_local      INTEGER NOT NULL DEFAULT 0,
    last_synced_at DATETIME,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE calendar_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL DEFAULT 1,
    source_id   INTEGER NOT NULL REFERENCES calendar_sources(id) ON DELETE CASCADE,
    -- uid: CalDAV UID for external events; generated UUID for local events
    uid         TEXT    NOT NULL,
    -- etag used to detect changes during CalDAV sync
    etag        TEXT,
    title       TEXT    NOT NULL,
    description TEXT,
    location    TEXT,
    start_at    DATETIME NOT NULL,
    end_at      DATETIME NOT NULL,
    is_all_day  INTEGER  NOT NULL DEFAULT 0,
    -- rrule from CalDAV spec; null for non-recurring
    rrule       TEXT,
    color       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, uid)
);

CREATE INDEX idx_calendar_events_range ON calendar_events(user_id, start_at, end_at);

-- ─── Clipboard ────────────────────────────────────────────────────────────────

CREATE TABLE clipboard_items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 1,
    title      TEXT,
    content    TEXT    NOT NULL,
    -- language hint for syntax highlighting (e.g. 'go', 'python', null = plain text)
    language   TEXT,
    is_pinned  INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_clipboard_user ON clipboard_items(user_id, created_at DESC);

-- ─── Bookmarks ────────────────────────────────────────────────────────────────

CREATE TABLE bookmark_collections (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 1,
    name       TEXT    NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

CREATE TABLE bookmarks (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL DEFAULT 1,
    collection_id INTEGER REFERENCES bookmark_collections(id) ON DELETE SET NULL,
    url           TEXT    NOT NULL,
    title         TEXT    NOT NULL,
    description   TEXT,
    favicon_url   TEXT,
    is_pinned     INTEGER NOT NULL DEFAULT 0,
    is_public     INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bookmarks_user ON bookmarks(user_id, created_at DESC);

-- ─── Memos ────────────────────────────────────────────────────────────────────

CREATE TABLE memos (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL DEFAULT 1,
    content     TEXT    NOT NULL,
    -- visibility: private | public
    visibility  TEXT    NOT NULL DEFAULT 'private',
    -- share_token: set when visibility = public; used in /s/:token route
    share_token TEXT    UNIQUE,
    is_pinned   INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_memos_user ON memos(user_id, created_at DESC);

-- +goose Down

DROP TABLE IF EXISTS memos;
DROP TABLE IF EXISTS bookmarks;
DROP TABLE IF EXISTS bookmark_collections;
DROP TABLE IF EXISTS clipboard_items;
DROP TABLE IF EXISTS calendar_events;
DROP TABLE IF EXISTS calendar_sources;
DROP TABLE IF EXISTS todo_recurrences;
DROP TABLE IF EXISTS todos;
DROP TABLE IF EXISTS todo_lists;
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS note_folders;
DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS entity_attachments;
DROP TABLE IF EXISTS entity_tags;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS tags;
