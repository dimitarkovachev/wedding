package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRateLimitRouter(rps float64, burst int) *gin.Engine {
	r := gin.New()
	r.Use(NewRateLimiter(rate.Limit(rps), burst))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimiter_AllowsWithinBurst(t *testing.T) {
	r := setupRateLimitRouter(1, 5)

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimiter_BlocksOverBurst(t *testing.T) {
	r := setupRateLimitRouter(1, 2)

	// Exhaust the burst
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["message"] == "" {
		t.Fatal("expected error message in response body")
	}
}

func TestRateLimiter_DifferentIPsHaveSeparateLimits(t *testing.T) {
	r := setupRateLimitRouter(1, 1)

	// First IP exhausts its limit
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	r.ServeHTTP(w1, req1)

	w1b := httptest.NewRecorder()
	req1b := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1b.RemoteAddr = "1.1.1.1:1234"
	r.ServeHTTP(w1b, req1b)
	if w1b.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for first IP, got %d", w1b.Code)
	}

	// Second IP should still be allowed
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "2.2.2.2:5678"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for second IP, got %d", w2.Code)
	}
}
