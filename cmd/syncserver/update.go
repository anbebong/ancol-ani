package main

import (
	"net/http"
	"path/filepath"
)

// Publishing a new release is a manual step (same spirit as deploy/phone/
// deploy.sh already being a manual script, not a CI pipeline): after
// building a new ancol-ani.exe locally, copy it to releasesDir()/ancol-
// ani.exe on the server and write/replace releasesDir()/latest.json with
// that build's version + sha256 (and optional notes). No upload endpoint
// here - exactly two public, read-only routes for the desktop app's
// "Kiểm tra cập nhật" button to call.
func releasesDir() string {
	return "data/releases"
}

// GET /latest-version: whatever's in latest.json, verbatim. The desktop
// app compares the "version" field against its own build and decides
// whether to offer an update - this endpoint doesn't know or care which
// client version is asking.
func (s *server) handleLatestVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	http.ServeFile(w, r, filepath.Join(releasesDir(), "latest.json"))
}

// GET /download/ancol-ani.exe: always the latest build - there's no
// per-version history kept on disk, matching latest.json only ever
// describing "the current latest", not a list of releases.
func (s *server) handleDownloadLatest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", `attachment; filename="ancol-ani.exe"`)
	http.ServeFile(w, r, filepath.Join(releasesDir(), "ancol-ani.exe"))
}
