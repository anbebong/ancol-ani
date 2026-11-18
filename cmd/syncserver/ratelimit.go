package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// In-memory fixed-window rate limiter for the public auth endpoints
// (login / register / password change). State is per process and resets
// on restart, which is fine for slowing down password guessing and mass
// sign-ups on a single small server.
type limiter struct {
	mu   sync.Mutex
	hits map[string]*window
}

type window struct {
	count int
	reset time.Time
}

func newLimiter() *limiter {
	l := &limiter{hits: map[string]*window{}}
	go func() {
		for range time.Tick(time.Minute) {
			now := time.Now()
			l.mu.Lock()
			for k, w := range l.hits {
				if now.After(w.reset) {
					delete(l.hits, k)
				}
			}
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *limiter) current(key string, now time.Time) *window {
	w := l.hits[key]
	if w == nil || now.After(w.reset) {
		return nil
	}
	return w
}

// allow counts one hit for key and reports whether it is still within max
// hits per window; when not, it also returns how long until the window ends.
func (l *limiter) allow(key string, max int, per time.Duration) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.current(key, now)
	if w == nil {
		w = &window{reset: now.Add(per)}
		l.hits[key] = w
	}
	if w.count >= max {
		return false, w.reset.Sub(now)
	}
	w.count++
	return true, 0
}

// blocked reports whether key already reached max in its current window,
// without counting a hit (used for "failed attempts" counters).
func (l *limiter) blocked(key string, max int) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if w := l.current(key, now); w != nil && w.count >= max {
		return true, w.reset.Sub(now)
	}
	return false, 0
}

// fail records one failed attempt for key.
func (l *limiter) fail(key string, per time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.current(key, now)
	if w == nil {
		w = &window{reset: now.Add(per)}
		l.hits[key] = w
	}
	w.count++
}

func (l *limiter) clear(key string) {
	l.mu.Lock()
	delete(l.hits, key)
	l.mu.Unlock()
}

// Limits (per client IP unless noted).
const (
	loginPerIP         = 20
	loginPerIPWindow   = 10 * time.Minute
	loginFailsPerUser  = 5 // wrong passwords per username, then locked
	loginFailWindow    = 15 * time.Minute
	registerPerIP      = 5
	registerPerIPWindw = time.Hour
)

func tooMany(w http.ResponseWriter, retry time.Duration) {
	secs := int(retry.Seconds()) + 1
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	mins := (secs + 59) / 60
	http.Error(w, fmt.Sprintf("thu qua nhieu lan, vui long doi khoang %d phut roi thu lai", mins), http.StatusTooManyRequests)
}

// clientIP is the caller's real IP. In production every request arrives
// through cloudflared on the same phone (a loopback connection), with the
// visitor's address in CF-Connecting-IP. That header is only trusted on
// loopback connections - anyone reaching the port directly (e.g. from the
// LAN) could set it to anything.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
			return cf
		}
	}
	return host
}
