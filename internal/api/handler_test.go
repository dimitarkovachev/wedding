package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/dimitarkovachev/wedding/internal/store"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewBBoltStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	err = s.Seed(map[string]store.InviteRecord{
		"550e8400-e29b-41d4-a716-446655440000": {
			People:          []string{"Иван Петров", "Мария Петрова"},
			AdditionalCount: 2,
			Accepted:        false,
		},
		"550e8400-e29b-41d4-a716-446655440001": {
			People:          []string{"Георги Димитров"},
			AdditionalCount: 0,
			Accepted:        false,
		},
	})
	if err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	h := NewHandler(s)
	r := gin.New()
	RegisterHandlers(r, h)
	return r
}

func TestHandler_GetHealth(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp.Status)
	}
}

func TestHandler_GetInvite_Found(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invites/550e8400-e29b-41d4-a716-446655440000", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var inv Invite
	if err := json.NewDecoder(w.Body).Decode(&inv); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(inv.People) != 2 {
		t.Fatalf("expected 2 people, got %d", len(inv.People))
	}
	if inv.Accepted {
		t.Fatal("expected accepted=false")
	}
}

func TestHandler_GetInvite_NotFound(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invites/00000000-0000-0000-0000-000000000000", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandler_PutInvite_AcceptNoAdditionals(t *testing.T) {
	r := setupTestRouter(t)

	body, _ := json.Marshal(InviteUpdate{Accepted: true})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var inv Invite
	if err := json.NewDecoder(w.Body).Decode(&inv); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !inv.Accepted {
		t.Fatal("expected accepted=true")
	}
}

func TestHandler_PutInvite_AcceptWithAdditionals(t *testing.T) {
	r := setupTestRouter(t)

	additional := []string{"Николай"}
	body, _ := json.Marshal(InviteUpdate{Accepted: true, Additional: &additional})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var inv Invite
	if err := json.NewDecoder(w.Body).Decode(&inv); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if inv.Additional == nil || len(*inv.Additional) != 1 {
		t.Fatalf("expected 1 additional, got %v", inv.Additional)
	}
}

func TestHandler_PutInvite_AcceptedFalse(t *testing.T) {
	r := setupTestRouter(t)

	body, _ := json.Marshal(InviteUpdate{Accepted: false})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_PutInvite_NotFound(t *testing.T) {
	r := setupTestRouter(t)

	body, _ := json.Marshal(InviteUpdate{Accepted: true})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_PutInvite_TooManyAdditionals(t *testing.T) {
	r := setupTestRouter(t)

	// invite 550e...001 has AdditionalCount=0
	additional := []string{"Иван"}
	body, _ := json.Marshal(InviteUpdate{Accepted: true, Additional: &additional})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
