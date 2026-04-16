package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lerko/helm/internal/api/middleware"
)

const testSecret = "test-secret-for-auth-middleware"

func mintToken(t *testing.T, secret string, expiry time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiry),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func callMiddleware(t *testing.T, r *http.Request) int {
	t.Helper()
	rr := httptest.NewRecorder()
	middleware.Auth(testSecret)(okHandler()).ServeHTTP(rr, r)
	return rr.Code
}

func TestMissingToken(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/protected", nil)
	if code := callMiddleware(t, r); code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", code)
	}
}

func TestValidBearerToken(t *testing.T) {
	tok := mintToken(t, testSecret, time.Now().Add(time.Hour))
	r := httptest.NewRequest("GET", "/api/protected", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	if code := callMiddleware(t, r); code != http.StatusOK {
		t.Errorf("got %d, want 200", code)
	}
}

func TestExpiredToken(t *testing.T) {
	tok := mintToken(t, testSecret, time.Now().Add(-time.Second))
	r := httptest.NewRequest("GET", "/api/protected", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	if code := callMiddleware(t, r); code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", code)
	}
}

func TestTamperedToken(t *testing.T) {
	tok := mintToken(t, testSecret, time.Now().Add(time.Hour))
	tok += "tampered"
	r := httptest.NewRequest("GET", "/api/protected", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	if code := callMiddleware(t, r); code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", code)
	}
}

func TestWrongSecretToken(t *testing.T) {
	tok := mintToken(t, "different-secret", time.Now().Add(time.Hour))
	r := httptest.NewRequest("GET", "/api/protected", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	if code := callMiddleware(t, r); code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", code)
	}
}

func TestQueryParamToken(t *testing.T) {
	tok := mintToken(t, testSecret, time.Now().Add(time.Hour))
	r := httptest.NewRequest("GET", "/api/events?token="+tok, nil)
	if code := callMiddleware(t, r); code != http.StatusOK {
		t.Errorf("got %d, want 200", code)
	}
}

func TestMalformedBearerHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/protected", nil)
	r.Header.Set("Authorization", "notbearer")
	if code := callMiddleware(t, r); code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", code)
	}
}
