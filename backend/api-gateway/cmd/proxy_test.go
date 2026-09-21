package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// gin's reverse-proxy path needs a ResponseWriter that implements
// http.CloseNotifier, which httptest.ResponseRecorder doesn't — so these
// tests drive real HTTP requests against an httptest.Server wrapping the
// gin engine, instead of calling ServeHTTP directly with a recorder.

func TestProxyTo_ForwardsPathAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"path":"` + r.URL.Path + `"}`))
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	r := gin.New()
	r.Any("/tickets/*path", proxyTo(target, newUpstreamBreaker("test-ticket")))
	gateway := httptest.NewServer(r)
	defer gateway.Close()

	resp, err := http.Get(gateway.URL + "/tickets/42")
	if err != nil {
		t.Fatalf("request to gateway failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), `"path":"/tickets/42"`) {
		t.Fatalf("expected proxyTo to preserve the original path, got body: %s", body)
	}
}

func TestProxyTrim_StripsPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"path":"` + r.URL.Path + `"}`))
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}

	r := gin.New()
	r.Any("/auth/*path", proxyTrim("/auth", target, newUpstreamBreaker("test-auth")))
	gateway := httptest.NewServer(r)
	defer gateway.Close()

	resp, err := http.Post(gateway.URL+"/auth/login", "application/json", nil)
	if err != nil {
		t.Fatalf("request to gateway failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), `"path":"/login"`) {
		t.Fatalf("expected proxyTrim to strip the /auth prefix, got body: %s", body)
	}
}

func TestProxyTo_ReturnsJSON502WhenUpstreamUnreachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("failed to parse upstream URL: %v", err)
	}
	upstream.Close() // now genuinely unreachable

	r := gin.New()
	r.Any("/tickets/*path", proxyTo(target, newUpstreamBreaker("test-unreachable")))
	gateway := httptest.NewServer(r)
	defer gateway.Close()

	resp, err := http.Get(gateway.URL + "/tickets/42")
	if err != nil {
		t.Fatalf("request to gateway failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502 when upstream is unreachable, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), `"BAD_GATEWAY"`) {
		t.Fatalf("expected JSON BAD_GATEWAY body, got: %s", body)
	}
}
