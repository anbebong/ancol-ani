-- 001 baseline: the schema as it was before versioned migrations.
-- Production DBs from before schema_migrations already have these tables, so
-- every statement must stay idempotent (IF NOT EXISTS). gifs.name/tags on very
-- old DBs are added in Go (afterSQL in migrations.go) since SQLite has no
-- "ADD COLUMN IF NOT EXISTS".
-- SHIPPED TO PRODUCTION: don't edit; add 002_<what>.sql to change the schema.

CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	username      TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	is_admin      INTEGER NOT NULL DEFAULT 0,
	gif_quota     INTEGER NOT NULL DEFAULT 3,
	created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS sessions (
	jti         TEXT PRIMARY KEY,
	user_id     INTEGER NOT NULL REFERENCES users(id),
	expires_at  TEXT NOT NULL,
	revoked_at  TEXT
);
CREATE TABLE IF NOT EXISTS gifs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    INTEGER NOT NULL REFERENCES users(id),
	data       BLOB NOT NULL,
	is_public  INTEGER NOT NULL DEFAULT 0,
	slug       TEXT UNIQUE,
	name       TEXT,
	tags       TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS share_history (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id     INTEGER NOT NULL REFERENCES users(id),
	gif_id      INTEGER NOT NULL REFERENCES gifs(id),
	slug        TEXT NOT NULL,
	shared_at   TEXT NOT NULL DEFAULT (datetime('now')),
	unshared_at TEXT
);
-- GitHub-style stars on public GIFs: one row per (user, gif).
CREATE TABLE IF NOT EXISTS gif_stars (
	user_id    INTEGER NOT NULL REFERENCES users(id),
	gif_id     INTEGER NOT NULL REFERENCES gifs(id),
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (user_id, gif_id)
);
CREATE INDEX IF NOT EXISTS idx_gif_stars_gif ON gif_stars(gif_id);
