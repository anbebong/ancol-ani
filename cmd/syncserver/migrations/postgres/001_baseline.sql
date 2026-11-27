-- 001 baseline (Postgres): same table shapes as sqlite/001_baseline.sql, with
-- Postgres syntax for autoincrement and binary data. Not covered by tests yet.
-- Keep version numbers in step with migrations/sqlite/.

CREATE TABLE IF NOT EXISTS users (
	id            BIGSERIAL PRIMARY KEY,
	username      TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	is_admin      BOOLEAN NOT NULL DEFAULT false,
	gif_quota     BIGINT NOT NULL DEFAULT 3,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sessions (
	jti         TEXT PRIMARY KEY,
	user_id     BIGINT NOT NULL REFERENCES users(id),
	expires_at  TIMESTAMPTZ NOT NULL,
	revoked_at  TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS gifs (
	id         BIGSERIAL PRIMARY KEY,
	user_id    BIGINT NOT NULL REFERENCES users(id),
	data       BYTEA NOT NULL,
	is_public  BOOLEAN NOT NULL DEFAULT false,
	slug       TEXT UNIQUE,
	name       TEXT,
	tags       TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS share_history (
	id          BIGSERIAL PRIMARY KEY,
	user_id     BIGINT NOT NULL REFERENCES users(id),
	gif_id      BIGINT NOT NULL REFERENCES gifs(id),
	slug        TEXT NOT NULL,
	shared_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
	unshared_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS gif_stars (
	user_id    BIGINT NOT NULL REFERENCES users(id),
	gif_id     BIGINT NOT NULL REFERENCES gifs(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (user_id, gif_id)
);
CREATE INDEX IF NOT EXISTS idx_gif_stars_gif ON gif_stars(gif_id);
ALTER TABLE gifs ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE gifs ADD COLUMN IF NOT EXISTS tags TEXT;
