package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
)

func loadTestSpec(t *testing.T) *openapi3.T {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromFile("../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatalf("failed to load openapi spec: %v", err)
	}
	if err := spec.Validate(context.Background()); err != nil {
		t.Fatalf("invalid openapi spec: %v", err)
	}
	return spec
}

func setupValidationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	spec := loadTestSpec(t)

	mw, err := NewOpenAPIValidator(spec)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	r := gin.New()
	r.Use(mw)
	r.PUT("/invites/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/invites/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestValidation_ValidPutRequest(t *testing.T) {
	r := setupValidationRouter(t)

	body, _ := json.Marshal(map[string]any{
		"isAccepted":   true,
		"additional": []string{"Иван Петров"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidation_InvalidLatinNames(t *testing.T) {
	r := setupValidationRouter(t)

	body, _ := json.Marshal(map[string]any{
		"isAccepted":   true,
		"additional": []string{"John Doe"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for Latin names, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidation_TooManyAdditionalItems(t *testing.T) {
	r := setupValidationRouter(t)

	body, _ := json.Marshal(map[string]any{
		"isAccepted":   true,
		"additional": []string{"Иван Петров", "Мария Петрова", "Георги Димитров", "Елена Стоянова", "Петър Стоянов", "Ана Иванова"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for >5 items, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidation_MissingRequiredAccepted(t *testing.T) {
	r := setupValidationRouter(t)

	body, _ := json.Marshal(map[string]any{
		"additional": []string{"Иван"},
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invites/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing accepted, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidation_HealthEndpointPassesThrough(t *testing.T) {
	r := setupValidationRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for health, got %d: %s", w.Code, w.Body.String())
	}
}
