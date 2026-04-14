-- +goose Up

-- ── FTS5 virtual tables (content tables → base table owns the data) ───────────

CREATE VIRTUAL TABLE notes_fts USING fts5(
    title,
    content,
    content='notes',
    content_rowid='id'
);

CREATE VIRTUAL TABLE todos_fts USING fts5(
    title,
    description,
    content='todos',
    content_rowid='id'
);

CREATE VIRTUAL TABLE memos_fts USING fts5(
    content,
    content='memos',
    content_rowid='id'
);

CREATE VIRTUAL TABLE bookmarks_fts USING fts5(
    title,
    url,
    description,
    content='bookmarks',
    content_rowid='id'
);

CREATE VIRTUAL TABLE clipboard_fts USING fts5(
    title,
    content,
    content='clipboard_items',
    content_rowid='id'
);

-- ── Seed FTS indexes from existing rows ───────────────────────────────────────

INSERT INTO notes_fts(rowid, title, content) SELECT id, title, content FROM notes;
INSERT INTO todos_fts(rowid, title, description) SELECT id, title, description FROM todos;
INSERT INTO memos_fts(rowid, content) SELECT id, content FROM memos;
INSERT INTO bookmarks_fts(rowid, title, url, description) SELECT id, title, url, description FROM bookmarks;
INSERT INTO clipboard_fts(rowid, title, content) SELECT id, title, content FROM clipboard_items;

-- ── notes triggers ────────────────────────────────────────────────────────────

-- +goose StatementBegin
CREATE TRIGGER notes_ai AFTER INSERT ON notes BEGIN
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER notes_au AFTER UPDATE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
    INSERT INTO notes_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER notes_ad AFTER DELETE ON notes BEGIN
    INSERT INTO notes_fts(notes_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
END;
-- +goose StatementEnd

-- ── todos triggers ────────────────────────────────────────────────────────────

-- +goose StatementBegin
CREATE TRIGGER todos_ai AFTER INSERT ON todos BEGIN
    INSERT INTO todos_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER todos_au AFTER UPDATE ON todos BEGIN
    INSERT INTO todos_fts(todos_fts, rowid, title, description) VALUES ('delete', old.id, old.title, old.description);
    INSERT INTO todos_fts(rowid, title, description) VALUES (new.id, new.title, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER todos_ad AFTER DELETE ON todos BEGIN
    INSERT INTO todos_fts(todos_fts, rowid, title, description) VALUES ('delete', old.id, old.title, old.description);
END;
-- +goose StatementEnd

-- ── memos triggers ────────────────────────────────────────────────────────────

-- +goose StatementBegin
CREATE TRIGGER memos_ai AFTER INSERT ON memos BEGIN
    INSERT INTO memos_fts(rowid, content) VALUES (new.id, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER memos_au AFTER UPDATE ON memos BEGIN
    INSERT INTO memos_fts(memos_fts, rowid, content) VALUES ('delete', old.id, old.content);
    INSERT INTO memos_fts(rowid, content) VALUES (new.id, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER memos_ad AFTER DELETE ON memos BEGIN
    INSERT INTO memos_fts(memos_fts, rowid, content) VALUES ('delete', old.id, old.content);
END;
-- +goose StatementEnd

-- ── bookmarks triggers ────────────────────────────────────────────────────────

-- +goose StatementBegin
CREATE TRIGGER bookmarks_ai AFTER INSERT ON bookmarks BEGIN
    INSERT INTO bookmarks_fts(rowid, title, url, description) VALUES (new.id, new.title, new.url, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER bookmarks_au AFTER UPDATE ON bookmarks BEGIN
    INSERT INTO bookmarks_fts(bookmarks_fts, rowid, title, url, description) VALUES ('delete', old.id, old.title, old.url, old.description);
    INSERT INTO bookmarks_fts(rowid, title, url, description) VALUES (new.id, new.title, new.url, new.description);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER bookmarks_ad AFTER DELETE ON bookmarks BEGIN
    INSERT INTO bookmarks_fts(bookmarks_fts, rowid, title, url, description) VALUES ('delete', old.id, old.title, old.url, old.description);
END;
-- +goose StatementEnd

-- ── clipboard_items triggers ──────────────────────────────────────────────────

-- +goose StatementBegin
CREATE TRIGGER clipboard_ai AFTER INSERT ON clipboard_items BEGIN
    INSERT INTO clipboard_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER clipboard_au AFTER UPDATE ON clipboard_items BEGIN
    INSERT INTO clipboard_fts(clipboard_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
    INSERT INTO clipboard_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER clipboard_ad AFTER DELETE ON clipboard_items BEGIN
    INSERT INTO clipboard_fts(clipboard_fts, rowid, title, content) VALUES ('delete', old.id, old.title, old.content);
END;
-- +goose StatementEnd

-- +goose Down

DROP TRIGGER IF EXISTS clipboard_ad;
DROP TRIGGER IF EXISTS clipboard_au;
DROP TRIGGER IF EXISTS clipboard_ai;
DROP TRIGGER IF EXISTS bookmarks_ad;
DROP TRIGGER IF EXISTS bookmarks_au;
DROP TRIGGER IF EXISTS bookmarks_ai;
DROP TRIGGER IF EXISTS memos_ad;
DROP TRIGGER IF EXISTS memos_au;
DROP TRIGGER IF EXISTS memos_ai;
DROP TRIGGER IF EXISTS todos_ad;
DROP TRIGGER IF EXISTS todos_au;
DROP TRIGGER IF EXISTS todos_ai;
DROP TRIGGER IF EXISTS notes_ad;
DROP TRIGGER IF EXISTS notes_au;
DROP TRIGGER IF EXISTS notes_ai;

DROP TABLE IF EXISTS clipboard_fts;
DROP TABLE IF EXISTS bookmarks_fts;
DROP TABLE IF EXISTS memos_fts;
DROP TABLE IF EXISTS todos_fts;
DROP TABLE IF EXISTS notes_fts;
