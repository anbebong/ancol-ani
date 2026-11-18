package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const tokenTTL = 1 * time.Hour

var errUnauthorized = errors.New("unauthorized")

type authClaims struct {
	jwt.RegisteredClaims
}

// issueToken creates a session row (so it can be revoked later) and signs
// a JWT carrying its jti. Verifying a request checks both the JWT
// signature/expiry AND that the session hasn't been revoked - a bare JWT
// can't be revoked before it expires, so the session table is what makes
// logout actually work.
func (s *server) issueToken(userID int64) (string, error) {
	jti := randomHex(16)
	expiresAt := time.Now().Add(tokenTTL)

	if err := s.store.CreateSession(jti, userID, expiresAt); err != nil {
		return "", err
	}

	claims := authClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *server) verifyToken(raw string) (userID int64, jti string, err error) {
	claims := &authClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, "", errUnauthorized
	}

	uid, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, "", errUnauthorized
	}

	// JWT itself says "not expired yet", but the session might have been
	// revoked (logout) - that's the part the token's own signature can't
	// tell us, so we always check the DB too.
	validUID, err := s.store.IsSessionValid(claims.ID)
	if err != nil || validUID != uid {
		return 0, "", errUnauthorized
	}
	return uid, claims.ID, nil
}

// requireAuth extracts "Authorization: Bearer <token>", verifies it, and
// passes the authenticated user id to next.
func (s *server) requireAuth(next func(w http.ResponseWriter, r *http.Request, userID int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		raw := strings.TrimPrefix(header, "Bearer ")
		if raw == "" || raw == header {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		userID, _, err := s.verifyToken(raw)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		next(w, r, userID)
	}
}

// requireAdmin builds on requireAuth: the caller must also have is_admin
// set. Checked server-side on every request, not just hidden in the UI.
func (s *server) requireAdmin(next func(w http.ResponseWriter, r *http.Request, userID int64)) http.HandlerFunc {
	return s.requireAuth(func(w http.ResponseWriter, r *http.Request, userID int64) {
		user, err := s.store.GetUserByID(userID)
		if err != nil || !user.IsAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r, userID)
	})
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
