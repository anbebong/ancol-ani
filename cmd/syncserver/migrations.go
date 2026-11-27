package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

// Schema migrations live as plain SQL files, one folder per dialect:
//
//	migrations/sqlite/001_baseline.sql
//	migrations/sqlite/002_<what>.sql
//	migrations/postgres/001_baseline.sql
//	...
//
// Files are embedded in the binary, so deploying is still one file. The DB
// records each version it ran in schema_migrations; on startup the server
// runs the missing ones in order.
//
// Rules: to change the schema, ADD a new file with the next number (same
// number for every dialect). Never edit, rename or renumber a file that has
// shipped to production - the DB already marked that version as done and
// won't rerun it, so the edit would silently never reach existing DBs.
//
//go:embed migrations
var embeddedMigrations embed.FS

// migrationsFS is the source of migration files; tests swap it out.
var migrationsFS fs.FS = embeddedMigrations

type migration struct {
	version int
	name    string
	sql     string
}

// afterSQL holds the few steps plain SQL can't express, run in the same
// transaction right after that version's file.
var afterSQL = map[string]map[int]func(tx *sqlx.Tx) error{
	"sqlite": {
		// Very old DBs created gifs before it had name/tags; CREATE TABLE IF
		// NOT EXISTS doesn't touch an existing table and SQLite has no
		// "ADD COLUMN IF NOT EXISTS". (Postgres does it in its 001 file.)
		1: func(tx *sqlx.Tx) error {
			if err := addColumnIfMissing(tx, "gifs", "name", "TEXT"); err != nil {
				return err
			}
			return addColumnIfMissing(tx, "gifs", "tags", "TEXT")
		},
	},
}

var migrationFileRe = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.sql$`)

// loadMigrations reads migrations/<driver>/*.sql sorted by version and
// checks the numbering is exactly 1, 2, 3, ... (no gap, no duplicate), so a
// mistake fails at startup instead of skipping a migration.
func loadMigrations(driver string) ([]migration, error) {
	dir := path.Join("migrations", driver)
	entries, err := fs.ReadDir(migrationsFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var list []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := migrationFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("%s/%s: migration file must be named NNN_name.sql", dir, e.Name())
		}
		v, _ := strconv.Atoi(m[1])
		body, err := fs.ReadFile(migrationsFS, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		list = append(list, migration{version: v, name: m[2], sql: string(body)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].version < list[j].version })
	for i, m := range list {
		if m.version != i+1 {
			return nil, fmt.Errorf("%s: expected migration %03d, found %03d_%s.sql (gap or duplicate number)", dir, i+1, m.version, m.name)
		}
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s: no migration files", dir)
	}
	return list, nil
}

// migrate brings the DB up to the latest schema version. Each pending
// migration runs in its own transaction together with the row recording
// it, so a failing migration rolls back completely and leaves the DB at
// the previous version (SQLite and Postgres both have transactional DDL).
func (s *Store) migrate() error {
	if s.driver != "sqlite" && s.driver != "postgres" {
		return errors.New("unsupported DB driver: " + s.driver)
	}
	list, err := loadMigrations(s.driver)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return err
	}
	current, err := s.SchemaVersion()
	if err != nil {
		return err
	}
	// A DB migrated by a newer server (e.g. rolling back to an older
	// binary) would be missing what this code expects - refuse to run on
	// it rather than misread data.
	if current > len(list) {
		return fmt.Errorf("DB schema version %d is newer than this server supports (%d)", current, len(list))
	}
	for _, m := range list[current:] {
		if err := s.applyMigration(m); err != nil {
			return fmt.Errorf("migration %03d_%s: %w", m.version, m.name, err)
		}
	}
	return nil
}

func (s *Store) applyMigration(m migration) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after Commit
	if _, err := tx.Exec(m.sql); err != nil {
		return err
	}
	if fn := afterSQL[s.driver][m.version]; fn != nil {
		if err := fn(tx); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(tx.Rebind(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`),
		m.version, m.name, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

// SchemaVersion is the highest migration applied to this DB (0 = none).
func (s *Store) SchemaVersion() (int, error) {
	var v int
	err := s.db.Get(&v, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`)
	return v, err
}

// addColumnIfMissing adds a column unless the table already has it (SQLite
// only - it lacks ADD COLUMN IF NOT EXISTS).
func addColumnIfMissing(tx *sqlx.Tx, table, column, columnDef string) error {
	var cols []struct {
		Name string `db:"name"`
	}
	// PRAGMA table_info returns more columns (cid, type, notnull, ...)
	// than the struct above maps - Unsafe() allows the extra ones instead
	// of sqlx erroring on them.
	if err := tx.Unsafe().Select(&cols, "PRAGMA table_info("+table+")"); err != nil {
		return err
	}
	for _, c := range cols {
		if c.Name == column {
			return nil
		}
	}
	_, err := tx.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + columnDef)
	return err
}
