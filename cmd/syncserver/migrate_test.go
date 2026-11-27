package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jmoiron/sqlx"
)

func testDSN(t *testing.T) string {
	t.Helper()
	return "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) + "?_pragma=busy_timeout(5000)"
}

func openTestStore(t *testing.T, dsn string) *Store {
	t.Helper()
	st, err := OpenStore("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.db.Close() })
	return st
}

// latestVersion is the number of the newest shipped sqlite migration file.
func latestVersion(t *testing.T) int {
	t.Helper()
	list, err := loadMigrations("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	return list[len(list)-1].version
}

// withMigrationFiles swaps the migration source for the test's duration.
func withMigrationFiles(t *testing.T, files fstest.MapFS) {
	t.Helper()
	saved := migrationsFS
	t.Cleanup(func() { migrationsFS = saved })
	migrationsFS = files
}

// The shipped files must load cleanly, and every dialect must have the same
// version numbers - otherwise sqlite and postgres deployments drift apart.
func TestShippedMigrationFiles(t *testing.T) {
	lite, err := loadMigrations("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	pg, err := loadMigrations("postgres")
	if err != nil {
		t.Fatal(err)
	}
	if len(lite) != len(pg) {
		t.Fatalf("sqlite has %d migrations, postgres %d", len(lite), len(pg))
	}
	for i := range lite {
		if lite[i].name != pg[i].name {
			t.Errorf("migration %03d: sqlite %q vs postgres %q", i+1, lite[i].name, pg[i].name)
		}
	}
	if lite[0].name != "baseline" {
		t.Errorf("001 must stay %q (production DBs recorded it under that name), got %q", "baseline", lite[0].name)
	}
}

func TestLoadMigrationsRejectsBadNumbering(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"gap": {
			"migrations/sqlite/001_a.sql": {Data: []byte("SELECT 1;")},
			"migrations/sqlite/003_c.sql": {Data: []byte("SELECT 1;")},
		},
		"duplicate": {
			"migrations/sqlite/001_a.sql": {Data: []byte("SELECT 1;")},
			"migrations/sqlite/001_b.sql": {Data: []byte("SELECT 1;")},
		},
		"bad name": {
			"migrations/sqlite/001_a.sql":     {Data: []byte("SELECT 1;")},
			"migrations/sqlite/two-thing.sql": {Data: []byte("SELECT 1;")},
		},
		"empty": {
			"migrations/sqlite": {Mode: fs.ModeDir},
		},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			withMigrationFiles(t, files)
			if _, err := loadMigrations("sqlite"); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestMigrateFreshDB(t *testing.T) {
	st := openTestStore(t, testDSN(t))
	v, err := st.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if v != latestVersion(t) {
		t.Fatalf("version = %d, want %d", v, latestVersion(t))
	}
	// schema actually usable
	uid := mustUser(t, st, "alice")
	if _, err := st.db.Exec(`INSERT INTO gifs (user_id, data, name, tags) VALUES (?, x'00', 'n', 't')`, uid); err != nil {
		t.Fatal(err)
	}
}

// A production DB from before versioned migrations: tables exist (gifs even
// without the later name/tags columns), no schema_migrations table. It must
// be adopted as version 1 without losing rows.
func TestMigrateLegacyDB(t *testing.T) {
	dsn := testDSN(t)
	raw, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL, is_admin INTEGER NOT NULL DEFAULT 0,
			gif_quota INTEGER NOT NULL DEFAULT 3, created_at TEXT NOT NULL DEFAULT (datetime('now')));
		CREATE TABLE gifs (
			id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id),
			data BLOB NOT NULL, is_public INTEGER NOT NULL DEFAULT 0, slug TEXT UNIQUE,
			created_at TEXT NOT NULL DEFAULT (datetime('now')), updated_at TEXT NOT NULL DEFAULT (datetime('now')));
		INSERT INTO users (username, password_hash) VALUES ('old', 'h');
		INSERT INTO gifs (user_id, data) VALUES (1, x'00');
	`)
	raw.Close()
	if err != nil {
		t.Fatal(err)
	}

	st := openTestStore(t, dsn)
	if v, _ := st.SchemaVersion(); v != latestVersion(t) {
		t.Fatalf("version = %d, want %d", v, latestVersion(t))
	}
	var n int
	if err := st.db.Get(&n, `SELECT COUNT(*) FROM gifs WHERE name IS NULL AND tags IS NULL`); err != nil {
		t.Fatal(err) // fails if name/tags weren't added
	}
	if n != 1 {
		t.Fatalf("legacy gif rows = %d, want 1", n)
	}
	if _, err := st.GetUserByUsername("old"); err != nil {
		t.Fatalf("legacy user lost: %v", err)
	}
}

func TestMigrateRerunIsNoop(t *testing.T) {
	dsn := testDSN(t)
	st := openTestStore(t, dsn)
	st.db.Close()
	st2 := openTestStore(t, dsn)
	var rows int
	if err := st2.db.Get(&rows, `SELECT COUNT(*) FROM schema_migrations`); err != nil {
		t.Fatal(err)
	}
	if rows != latestVersion(t) {
		t.Fatalf("schema_migrations rows = %d, want %d (migration ran twice?)", rows, latestVersion(t))
	}
}

func TestMigrateRefusesNewerDB(t *testing.T) {
	dsn := testDSN(t)
	st := openTestStore(t, dsn)
	if _, err := st.db.Exec(`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, 'future', 'x')`, latestVersion(t)+1); err != nil {
		t.Fatal(err)
	}
	st.db.Close()
	if s, err := OpenStore("sqlite", dsn); err == nil {
		s.db.Close()
		t.Fatal("expected error opening a DB with a newer schema version")
	} else if !strings.Contains(err.Error(), "newer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// A new migration file is picked up on top of an existing DB, and a failing
// one rolls back entirely without bumping the version.
func TestMigrateAddsNewFileAndRollsBackFailure(t *testing.T) {
	baseline, err := fs.ReadFile(embeddedMigrations, "migrations/sqlite/001_baseline.sql")
	if err != nil {
		t.Fatal(err)
	}
	dsn := testDSN(t)

	withMigrationFiles(t, fstest.MapFS{
		"migrations/sqlite/001_baseline.sql": {Data: baseline},
		"migrations/sqlite/002_add_note.sql": {Data: []byte("ALTER TABLE users ADD COLUMN note TEXT;")},
		"migrations/sqlite/003_broken.sql":   {Data: []byte("CREATE TABLE half_done (id INTEGER);\nTHIS IS NOT SQL;")},
	})
	if s, err := OpenStore("sqlite", dsn); err == nil {
		s.db.Close()
		t.Fatal("expected 003_broken to fail")
	} else if !strings.Contains(err.Error(), "003_broken") {
		t.Fatalf("error should name the failing file: %v", err)
	}

	// drop the broken file: 001 and 002 are already applied, nothing reruns
	withMigrationFiles(t, fstest.MapFS{
		"migrations/sqlite/001_baseline.sql": {Data: baseline},
		"migrations/sqlite/002_add_note.sql": {Data: []byte("ALTER TABLE users ADD COLUMN note TEXT;")},
	})
	st := openTestStore(t, dsn)
	if v, _ := st.SchemaVersion(); v != 2 {
		t.Fatalf("version = %d, want 2", v)
	}
	var n int
	st.db.Get(&n, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'half_done'`)
	if n != 0 {
		t.Fatal("half_done table survived a failed migration")
	}
	if _, err := st.db.Exec(`UPDATE users SET note = 'x'`); err != nil {
		t.Fatalf("002 not applied: %v", err)
	}
}
