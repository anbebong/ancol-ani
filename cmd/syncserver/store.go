package main

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("already exists")
	ErrQuota     = errors.New("gif quota exceeded")
)

const defaultGifQuota = 3

type User struct {
	ID           int64  `db:"id"`
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
	IsAdmin      bool   `db:"is_admin"`
	GifQuota     int64  `db:"gif_quota"`
}

type UserSummary struct {
	ID        int64  `db:"id" json:"id"`
	Username  string `db:"username" json:"username"`
	CreatedAt string `db:"created_at" json:"createdAt"`
	IsAdmin   bool   `db:"is_admin" json:"isAdmin"`
	GifQuota  int64  `db:"gif_quota" json:"gifQuota"`
	GifCount  int64  `db:"gif_count" json:"gifCount"`
	HasPublic bool   `db:"has_public" json:"hasPublic"`
}

// GifMeta is one GIF slot without its blob - used everywhere except the
// raw image-serving endpoints.
type GifMeta struct {
	ID        int64  `db:"id" json:"id"`
	IsPublic  bool   `db:"is_public" json:"isPublic"`
	Slug      string `db:"slug" json:"slug"`
	Name      string `db:"name" json:"name"`
	Tags      string `db:"tags" json:"tags"`
	CreatedAt string `db:"created_at" json:"createdAt"`
	UpdatedAt string `db:"updated_at" json:"updatedAt"`
}

type PublicEntry struct {
	Slug      string `db:"slug" json:"slug"`
	Username  string `db:"username" json:"username"`
	Name      string `db:"name" json:"name"`
	Tags      string `db:"tags" json:"tags"`
	UpdatedAt string `db:"updated_at" json:"updatedAt"`
}

// ShareHistoryEntry is one share/unshare cycle for a gif slot. SharedAt
// is always set, UnsharedAt is empty while the share is still active.
type ShareHistoryEntry struct {
	GifID      int64  `db:"gif_id" json:"gifId"`
	Slug       string `db:"slug" json:"slug"`
	SharedAt   string `db:"shared_at" json:"sharedAt"`
	UnsharedAt string `db:"unshared_at" json:"unsharedAt"`
}

// Store wraps the DB connection. All queries use sqlx named-query syntax
// so switching driver (sqlite -> postgres, ...) is a matter of changing
// the connection string/driver name in config, not rewriting queries.
type Store struct {
	db     *sqlx.DB
	driver string
}

func OpenStore(driver, dsn string) (*Store, error) {
	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, driver: driver}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// migrate creates the schema if it doesn't exist yet. The SQL differs
// slightly per dialect (autoincrement syntax, blob type) - that's isolated
// here; everything past this point uses portable named queries via sqlx.
func (s *Store) migrate() error {
	switch s.driver {
	case "sqlite":
		_, err := s.db.Exec(`
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
		`)
		if err != nil {
			return err
		}
		if err := s.addColumnIfMissing("gifs", "name", "TEXT"); err != nil {
			return err
		}
		return s.addColumnIfMissing("gifs", "tags", "TEXT")

	case "postgres":
		// Ready for when a real deployment wants Postgres instead of
		// SQLite - same table shapes, Postgres syntax for autoincrement
		// and binary data. Not covered by local testing yet.
		_, err := s.db.Exec(`
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
			ALTER TABLE gifs ADD COLUMN IF NOT EXISTS name TEXT;
			ALTER TABLE gifs ADD COLUMN IF NOT EXISTS tags TEXT;
		`)
		return err

	default:
		return errors.New("unsupported DB driver: " + s.driver)
	}
}

// addColumnIfMissing covers the case where gifs.name/gifs.tags were added
// to the schema after some databases already had a gifs table - SQLite's
// ALTER TABLE has no "ADD COLUMN IF NOT EXISTS", so check pragma first.
func (s *Store) addColumnIfMissing(table, column, columnDef string) error {
	var cols []struct {
		Name string `db:"name"`
	}
	// PRAGMA table_info returns more columns (cid, type, notnull, ...)
	// than the struct above maps - Unsafe() allows the extra ones instead
	// of sqlx erroring on them.
	if err := s.db.Unsafe().Select(&cols, "PRAGMA table_info("+table+")"); err != nil {
		return err
	}
	for _, c := range cols {
		if c.Name == column {
			return nil
		}
	}
	_, err := s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + columnDef)
	return err
}

func (s *Store) CreateUser(username, passwordHash string) (int64, error) {
	res, err := s.db.NamedExec(
		`INSERT INTO users (username, password_hash, gif_quota) VALUES (:username, :password_hash, :quota)`,
		map[string]any{"username": username, "password_hash": passwordHash, "quota": defaultGifQuota},
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	var u User
	err := s.db.Get(&u, s.db.Rebind(`SELECT id, username, password_hash, is_admin, gif_quota FROM users WHERE username = ?`), username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	var u User
	err := s.db.Get(&u, s.db.Rebind(`SELECT id, username, password_hash, is_admin, gif_quota FROM users WHERE id = ?`), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (s *Store) ListUsers() ([]UserSummary, error) {
	users := []UserSummary{}
	err := s.db.Select(&users, `
		SELECT u.id, u.username, u.created_at, u.is_admin, u.gif_quota,
		       (SELECT COUNT(*) FROM gifs g WHERE g.user_id = u.id) AS gif_count,
		       EXISTS(SELECT 1 FROM gifs g WHERE g.user_id = u.id AND g.is_public = 1) AS has_public
		FROM users u
		ORDER BY u.created_at DESC
	`)
	return users, err
}

func (s *Store) DeleteUser(id int64) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(tx.Rebind(`DELETE FROM share_history WHERE user_id = ?`), id); err != nil {
		return err
	}
	if _, err := tx.Exec(tx.Rebind(`DELETE FROM gifs WHERE user_id = ?`), id); err != nil {
		return err
	}
	if _, err := tx.Exec(tx.Rebind(`DELETE FROM sessions WHERE user_id = ?`), id); err != nil {
		return err
	}
	res, err := tx.Exec(tx.Rebind(`DELETE FROM users WHERE id = ?`), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

func (s *Store) SetAdmin(username string, isAdmin bool) error {
	res, err := s.db.Exec(s.db.Rebind(`UPDATE users SET is_admin = ? WHERE username = ?`), isAdmin, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetGifQuota is an admin-only action: the per-user GIF slot limit is
// itself a piece of user information admins can view/edit.
func (s *Store) SetGifQuota(userID int64, quota int64) error {
	res, err := s.db.Exec(s.db.Rebind(`UPDATE users SET gif_quota = ? WHERE id = ?`), quota, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AdminUnpublish removes public sharing from whichever gif owns slug -
// unlike UnshareGif (which also checks ownership), this lets an admin
// take down any user's public GIF.
func (s *Store) AdminUnpublish(slug string) error {
	res, err := s.db.Exec(s.db.Rebind(`UPDATE gifs SET is_public = 0, slug = NULL WHERE slug = ?`), slug)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateSession(jti string, userID int64, expiresAt time.Time) error {
	_, err := s.db.NamedExec(
		`INSERT INTO sessions (jti, user_id, expires_at) VALUES (:jti, :user_id, :expires_at)`,
		map[string]any{"jti": jti, "user_id": userID, "expires_at": expiresAt.UTC().Format(time.RFC3339)},
	)
	return err
}

// IsSessionValid returns the user id for a non-revoked, non-expired
// session, or ErrNotFound otherwise.
func (s *Store) IsSessionValid(jti string) (int64, error) {
	var row struct {
		UserID    int64   `db:"user_id"`
		ExpiresAt string  `db:"expires_at"`
		RevokedAt *string `db:"revoked_at"`
	}
	err := s.db.Get(&row, s.db.Rebind(`SELECT user_id, expires_at, revoked_at FROM sessions WHERE jti = ?`), jti)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if row.RevokedAt != nil {
		return 0, ErrNotFound
	}
	expiresAt, err := time.Parse(time.RFC3339, row.ExpiresAt)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return 0, ErrNotFound
	}
	return row.UserID, nil
}

func (s *Store) RevokeSession(jti string) error {
	_, err := s.db.Exec(s.db.Rebind(`UPDATE sessions SET revoked_at = ? WHERE jti = ?`),
		time.Now().UTC().Format(time.RFC3339), jti)
	return err
}

// CreateGif adds a new GIF slot for userID, enforcing the user's quota.
// Returns the new gif's id.
func (s *Store) CreateGif(userID int64, data []byte) (int64, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return 0, err
	}

	var count int64
	if err := s.db.Get(&count, s.db.Rebind(`SELECT COUNT(*) FROM gifs WHERE user_id = ?`), userID); err != nil {
		return 0, err
	}
	if count >= user.GifQuota {
		return 0, ErrQuota
	}

	res, err := s.db.NamedExec(`
		INSERT INTO gifs (user_id, data) VALUES (:user_id, :data)
	`, map[string]any{"user_id": userID, "data": data})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListGifs(userID int64) ([]GifMeta, error) {
	gifs := []GifMeta{}
	err := s.db.Select(&gifs, s.db.Rebind(`
		SELECT id, is_public, COALESCE(slug, '') AS slug,
		       COALESCE(name, '') AS name, COALESCE(tags, '') AS tags,
		       created_at, updated_at
		FROM gifs WHERE user_id = ? ORDER BY created_at ASC
	`), userID)
	return gifs, err
}

// DeleteGif removes a gif slot the caller owns.
func (s *Store) DeleteGif(userID, gifID int64) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(tx.Rebind(`DELETE FROM share_history WHERE gif_id = ?`), gifID); err != nil {
		return err
	}
	res, err := tx.Exec(tx.Rebind(`DELETE FROM gifs WHERE id = ? AND user_id = ?`), gifID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}

	return tx.Commit()
}

// GetGifData returns one gif's blob, scoped to its owner - this is what
// "sync" means: the client picks one slot from the user's library and
// pulls it explicitly, there's no server-side "current" gif anymore.
func (s *Store) GetGifData(userID, gifID int64) ([]byte, error) {
	var data []byte
	err := s.db.Get(&data, s.db.Rebind(`SELECT data FROM gifs WHERE id = ? AND user_id = ?`), gifID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}

// ShareGif publishes one gif slot under a random slug, with a required
// display name and optional comma-separated tags so other users can find
// it (by name/tag text search, or by browsing a given publisher).
// Ownership is enforced by the user_id match in the WHERE clause.
func (s *Store) ShareGif(userID, gifID int64, slug, name, tags string) error {
	res, err := s.db.Exec(s.db.Rebind(`
		UPDATE gifs SET is_public = 1, slug = ?, name = ?, tags = ?, updated_at = `+now(s.driver)+`
		WHERE id = ? AND user_id = ?
	`), slug, name, tags, gifID, userID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return s.logShare(userID, gifID, slug)
}

func (s *Store) UnshareGif(userID, gifID int64) error {
	res, err := s.db.Exec(s.db.Rebind(`
		UPDATE gifs SET is_public = 0, slug = NULL, updated_at = `+now(s.driver)+`
		WHERE id = ? AND user_id = ?
	`), gifID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return s.logUnshare(gifID)
}

func (s *Store) logShare(userID, gifID int64, slug string) error {
	_, err := s.db.NamedExec(
		`INSERT INTO share_history (user_id, gif_id, slug) VALUES (:user_id, :gif_id, :slug)`,
		map[string]any{"user_id": userID, "gif_id": gifID, "slug": slug},
	)
	return err
}

// logUnshare closes out the most recent still-active share period for
// this specific gif slot.
func (s *Store) logUnshare(gifID int64) error {
	_, err := s.db.NamedExec(`
		UPDATE share_history SET unshared_at = `+now(s.driver)+`
		WHERE id = (
			SELECT id FROM share_history
			WHERE gif_id = :gif_id AND unshared_at IS NULL
			ORDER BY id DESC LIMIT 1
		)
	`, map[string]any{"gif_id": gifID})
	return err
}

func (s *Store) ListShareHistory(userID int64, limit int) ([]ShareHistoryEntry, error) {
	entries := []ShareHistoryEntry{}
	err := s.db.Select(&entries, s.db.Rebind(`
		SELECT gif_id, slug, shared_at, COALESCE(unshared_at, '') AS unshared_at
		FROM share_history
		WHERE user_id = ?
		ORDER BY id DESC
		LIMIT ?
	`), userID, limit)
	return entries, err
}

func (s *Store) GetPublicBySlug(slug string) ([]byte, error) {
	var data []byte
	err := s.db.Get(&data, s.db.Rebind(`SELECT data FROM gifs WHERE slug = ? AND is_public = 1`), slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}

// ListPublic lists public gifs, newest first, optionally filtered by a
// free-text search over name+tags (q) and/or an exact publisher username
// (user). Both filters are optional and combine with AND.
func (s *Store) ListPublic(limit int, q, user string) ([]PublicEntry, error) {
	entries := []PublicEntry{}
	query := `
		SELECT g.slug, u.username, COALESCE(g.name, '') AS name, COALESCE(g.tags, '') AS tags, g.updated_at
		FROM gifs g
		JOIN users u ON u.id = g.user_id
		WHERE g.is_public = 1
	`
	args := []any{}
	if q != "" {
		query += ` AND (g.name LIKE ? OR g.tags LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if user != "" {
		query += ` AND u.username = ?`
		args = append(args, user)
	}
	query += ` ORDER BY g.updated_at DESC LIMIT ?`
	args = append(args, limit)

	err := s.db.Select(&entries, s.db.Rebind(query), args...)
	return entries, err
}

func now(driver string) string {
	if driver == "sqlite" {
		return "datetime('now')"
	}
	return "now()"
}

func isUniqueViolation(err error) bool {
	// Covers SQLite's error text; a Postgres driver would need its own
	// check (pq/pgx expose a structured error code instead of a string).
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed") || contains(msg, "unique constraint")
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
