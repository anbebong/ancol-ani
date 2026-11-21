package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleIdentity_ReturnsServiceID: the client pings this endpoint before
// following a link to the server (e.g. to open the portal) so it can tell
// "this answered" apart from "this is actually our syncserver" - a reverse
// proxy, captive portal, or unrelated service on that host/port could also
// answer with HTTP 200.
func TestHandleIdentity_ReturnsServiceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/identity", nil)
	rec := httptest.NewRecorder()

	handleIdentity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Service string `json:"service"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Service != syncServiceID {
		t.Fatalf("service = %q, want %q", body.Service, syncServiceID)
	}
}
