-- tables du jeu (section 10 du sujet)

CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS cases (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT NOT NULL UNIQUE,
    language     TEXT NOT NULL,
    difficulty   TEXT NOT NULL,
    content_json TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'local'
);

CREATE TABLE IF NOT EXISTS investigations (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    case_id    INTEGER NOT NULL REFERENCES cases(id),
    status     TEXT NOT NULL DEFAULT 'started'
               CHECK (status IN ('started', 'solved')),
    score      INTEGER CHECK (score >= 0 AND score <= 110),
    started_at TEXT DEFAULT (datetime('now')),
    solved_at  TEXT
);

CREATE TABLE IF NOT EXISTS attempts (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    investigation_id INTEGER NOT NULL REFERENCES investigations(id),
    suspect_id       TEXT NOT NULL,
    is_correct       INTEGER NOT NULL,
    created_at       TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS hints_used (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    investigation_id INTEGER NOT NULL REFERENCES investigations(id),
    hint_level       INTEGER NOT NULL CHECK (hint_level IN (1, 2, 3)),
    created_at       TEXT DEFAULT (datetime('now')),
    -- pas deux fois le meme indice
    UNIQUE (investigation_id, hint_level)
);

CREATE TABLE IF NOT EXISTS reports (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    investigation_id INTEGER NOT NULL UNIQUE REFERENCES investigations(id),
    content_json     TEXT NOT NULL,
    created_at       TEXT DEFAULT (datetime('now'))
);
