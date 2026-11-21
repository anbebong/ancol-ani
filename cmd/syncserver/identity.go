package main

import (
	"encoding/json"
	"net/http"
)

// syncServiceID identifies this syncserver to clients that only have a URL
// a person typed in (the account-link "Server" field). A reachable HTTP
// server at that address proves nothing by itself - it could be someone
// else's app, a captive portal, or a typo that happens to hit something
// else - so the client checks this exact string before treating the
// address as a real syncserver and following a link to it.
const syncServiceID = "ancol-ani-syncserver"

// handleIdentity: GET /identity - public, no auth. Returns a small, stable
// marker clients can check before they treat an address as a real
// syncserver and follow a link to it (e.g. "Mở portal", "Liên kết").
func handleIdentity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"service": syncServiceID})
}
