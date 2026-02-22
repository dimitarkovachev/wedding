package admin

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

func setupAdminRouter(t *testing.T) *gin.Engine {
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
	})
	if err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	h := NewHandler(s)
	r := gin.New()
	RegisterHandlers(r, h)
	return r
}

func TestHandler_GetAdminInvites_Expected(t *testing.T) {
	r := setupAdminRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var invites map[string]store.InviteRecord
	if err := json.NewDecoder(w.Body).Decode(&invites); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("expected 1 invite, got %d", len(invites))
	}
	rec, ok := invites["550e8400-e29b-41d4-a716-446655440000"]
	if !ok {
		t.Fatal("expected invite key not found")
	}
	if len(rec.People) != 2 {
		t.Fatalf("expected 2 people, got %d", len(rec.People))
	}
}

func TestHandler_GetAdminInvites_EmptyBucket(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewBBoltStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	h := NewHandler(s)
	r := gin.New()
	RegisterHandlers(r, h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var invites map[string]store.InviteRecord
	if err := json.NewDecoder(w.Body).Decode(&invites); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(invites) != 0 {
		t.Fatalf("expected 0 invites, got %d", len(invites))
	}
}

func TestHandler_PutAdminInvites_Expected(t *testing.T) {
	r := setupAdminRouter(t)

	newInvites := map[string]store.InviteRecord{
		"bbb-001": {People: []string{"Нов Гост"}, AdditionalCount: 1, Accepted: false},
	}
	body, _ := json.Marshal(newInvites)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/invites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify data was replaced by doing a GET
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	r.ServeHTTP(w, req)

	var invites map[string]store.InviteRecord
	if err := json.NewDecoder(w.Body).Decode(&invites); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("expected 1 invite after replace, got %d", len(invites))
	}
	if _, ok := invites["550e8400-e29b-41d4-a716-446655440000"]; ok {
		t.Fatal("old invite should have been removed")
	}
	if _, ok := invites["bbb-001"]; !ok {
		t.Fatal("new invite bbb-001 not found")
	}
}

func TestHandler_PutAdminInvites_InvalidBody(t *testing.T) {
	r := setupAdminRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/invites", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_PutAdminInvites_EmptyMap(t *testing.T) {
	r := setupAdminRouter(t)

	body, _ := json.Marshal(map[string]store.InviteRecord{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/invites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify bucket is now empty
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/invites", nil)
	r.ServeHTTP(w, req)

	var invites map[string]store.InviteRecord
	if err := json.NewDecoder(w.Body).Decode(&invites); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(invites) != 0 {
		t.Fatalf("expected 0 invites after empty replace, got %d", len(invites))
	}
}
