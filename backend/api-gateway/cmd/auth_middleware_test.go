package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newAuthTestRouter(secret []byte) *gin.Engine {
	r := gin.New()
	r.GET("/protected", authMiddleware(secret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.Request.Header.Get("X-User-ID"),
			"role":    c.Request.Header.Get("X-User-ROLE"),
		})
	})
	return r
}

func signHMAC(t *testing.T, secret []byte, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signed
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	secret := []byte("test-secret")
	r := newAuthTestRouter(secret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MalformedToken(t *testing.T) {
	secret := []byte("test-secret")
	r := newAuthTestRouter(secret)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSigningAlg(t *testing.T) {
	secret := []byte("test-secret")
	r := newAuthTestRouter(secret)

	// alg=none, unsigned — must be rejected even though it "parses".
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "1",
		"role":    "admin",
	})
	tokenString, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build alg=none token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for alg=none token, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingClaims(t *testing.T) {
	secret := []byte("test-secret")
	r := newAuthTestRouter(secret)

	tokenString := signHMAC(t, secret, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for token missing user_id/role, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidTokenSetsHeaders(t *testing.T) {
	secret := []byte("test-secret")
	r := newAuthTestRouter(secret)

	tokenString := signHMAC(t, secret, jwt.MapClaims{
		"user_id": "42",
		"role":    "agent",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"user_id":"42"`) || !strings.Contains(w.Body.String(), `"role":"agent"`) {
		t.Fatalf("expected X-User-ID/X-User-ROLE to be forwarded, got body: %s", w.Body.String())
	}
}
