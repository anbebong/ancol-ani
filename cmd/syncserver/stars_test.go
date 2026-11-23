package main

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) + "?_pragma=busy_timeout(5000)"
	st, err := OpenStore("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	// close before TempDir cleanup - Windows can't delete an open file
	t.Cleanup(func() { st.db.Close() })
	return st
}

func mustUser(t *testing.T, st *Store, name string) int64 {
	t.Helper()
	id, err := st.CreateUser(name, "x")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustPublicGif(t *testing.T, st *Store, owner int64, slug, name string) int64 {
	t.Helper()
	id, err := st.CreateGif(owner, []byte("GIF89a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ShareGif(owner, id, slug, name, ""); err != nil {
		t.Fatal(err)
	}
	return id
}

func findEntry(entries []PublicEntry, slug string) *PublicEntry {
	for i := range entries {
		if entries[i].Slug == slug {
			return &entries[i]
		}
	}
	return nil
}

func TestStars(t *testing.T) {
	st := newTestStore(t)
	alice := mustUser(t, st, "alice")
	bob := mustUser(t, st, "bob")
	carol := mustUser(t, st, "carol")
	gifA := mustPublicGif(t, st, alice, "slug-a", "A")
	mustPublicGif(t, st, alice, "slug-b", "B")

	// star twice = one star; two users = two stars
	for _, u := range []int64{bob, bob, carol} {
		if _, _, err := st.SetStar(u, "slug-b", true); err != nil {
			t.Fatal(err)
		}
	}
	n, starred, err := st.SetStar(bob, "slug-a", true)
	if err != nil || n != 1 || !starred {
		t.Fatalf("star A: n=%d starred=%v err=%v", n, starred, err)
	}

	// list as bob: counts + "starred by me", most-starred first with sort=stars
	list, _, err := st.ListPublic(bob, 10, 0, "", "", "stars")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Slug != "slug-b" || list[0].Stars != 2 || !list[0].Starred {
		t.Fatalf("sort=stars as bob: %+v", list)
	}
	// alice starred nothing
	list, _, _ = st.ListPublic(alice, 10, 0, "", "", "")
	if e := findEntry(list, "slug-b"); e == nil || e.Stars != 2 || e.Starred {
		t.Fatalf("as alice: %+v", e)
	}

	// unstar is idempotent
	for i := 0; i < 2; i++ {
		if n, starred, err = st.SetStar(carol, "slug-b", false); err != nil || n != 1 || starred {
			t.Fatalf("unstar #%d: n=%d starred=%v err=%v", i, n, starred, err)
		}
	}

	// unknown / private slug -> not found
	if _, _, err := st.SetStar(bob, "nope", true); err != ErrNotFound {
		t.Fatalf("unknown slug: %v", err)
	}

	// owner's library shows stars received
	gifs, _, _ := st.ListGifs(alice, 10, 0)
	if len(gifs) != 2 || gifs[0].Stars != 1 || gifs[1].Stars != 1 {
		t.Fatalf("library stars: %+v", gifs)
	}

	// deleting a GIF removes its stars; deleting a user removes stars they gave
	if err := st.DeleteGif(alice, gifA); err != nil {
		t.Fatal(err)
	}
	var left int64
	st.db.Get(&left, `SELECT COUNT(*) FROM gif_stars WHERE gif_id = ?`, gifA)
	if left != 0 {
		t.Fatalf("stars left on deleted gif: %d", left)
	}
	if err := st.DeleteUser(bob); err != nil {
		t.Fatal(err)
	}
	list, _, _ = st.ListPublic(alice, 10, 0, "", "", "")
	if e := findEntry(list, "slug-b"); e == nil || e.Stars != 0 {
		t.Fatalf("after deleting bob: %+v", e)
	}
}
