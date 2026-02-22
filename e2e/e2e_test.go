package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

var baseURL = "http://localhost:8080"

// Response types (self-contained, no dependency on main module)

type Invite struct {
	People          []string `json:"people"`
	AdditionalCount int      `json:"additionalCount"`
	Additional      []string `json:"additional"`
	IsAccepted      bool     `json:"isAccepted"`
	IsOpened        bool     `json:"isOpened"`
}

type InviteUpdate struct {
	IsAccepted bool     `json:"isAccepted"`
	Additional []string `json:"additional,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

func TestMain(m *testing.M) {
	if u := os.Getenv("API_URL"); u != "" {
		baseURL = u
	}

	if !waitForHealthy(15 * time.Second) {
		fmt.Fprintf(os.Stderr, "ERROR: API at %s not healthy after timeout\n", baseURL)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func waitForHealthy(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// --- Happy path ---

func TestGetInvite(t *testing.T) {
	resp, err := http.Get(baseURL + "/invites/aaaa0000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var inv Invite
	if err := json.NewDecoder(resp.Body).Decode(&inv); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(inv.People) != 2 {
		t.Fatalf("expected 2 people, got %d", len(inv.People))
	}
	if inv.People[0] != "Иван Петров" {
		t.Fatalf("expected 'Иван Петров', got %q", inv.People[0])
	}
	if inv.AdditionalCount != 2 {
		t.Fatalf("expected additionalCount=2, got %d", inv.AdditionalCount)
	}
	if inv.IsAccepted {
		t.Fatal("expected accepted=false for fresh invite")
	}
}

func TestAcceptWithoutAdditionals(t *testing.T) {
	// Reason: uses invite 004 which no prior test touches, so the first GET is the true first view
	const inviteURL = "/invites/aaaa0000-0000-0000-0000-000000000004"

	// GET before PUT: not accepted, not opened (first view ever)
	preResp, err := http.Get(baseURL + inviteURL)
	if err != nil {
		t.Fatalf("pre-GET failed: %v", err)
	}
	defer preResp.Body.Close()

	var pre Invite
	if err := json.NewDecoder(preResp.Body).Decode(&pre); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if pre.IsAccepted {
		t.Fatal("expected isAccepted=false before PUT")
	}
	if pre.IsOpened {
		t.Fatal("expected isOpened=false on first view")
	}

	// PUT to accept
	body, _ := json.Marshal(InviteUpdate{IsAccepted: true})
	req, _ := http.NewRequest(http.MethodPut, baseURL+inviteURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET after PUT: accepted and opened
	getResp, err := http.Get(baseURL + inviteURL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer getResp.Body.Close()

	var inv Invite
	if err := json.NewDecoder(getResp.Body).Decode(&inv); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !inv.IsAccepted {
		t.Fatal("expected isAccepted=true after PUT")
	}
	if !inv.IsOpened {
		t.Fatal("expected isOpened=true after prior view")
	}
}

func TestAcceptWithAdditionals(t *testing.T) {
	const inviteURL = "/invites/aaaa0000-0000-0000-0000-000000000002"

	// GET before PUT: not accepted, not opened (first view of invite 002)
	preResp, err := http.Get(baseURL + inviteURL)
	if err != nil {
		t.Fatalf("pre-GET failed: %v", err)
	}
	defer preResp.Body.Close()

	var pre Invite
	if err := json.NewDecoder(preResp.Body).Decode(&pre); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if pre.IsAccepted {
		t.Fatal("expected isAccepted=false before PUT")
	}
	if pre.IsOpened {
		t.Fatal("expected isOpened=false (first view of this invite)")
	}

	// PUT to accept with additionals
	body, _ := json.Marshal(InviteUpdate{
		IsAccepted: true,
		Additional: []string{"Николай Георгиев", "Анна Георгиева"},
	})
	req, _ := http.NewRequest(http.MethodPut, baseURL+inviteURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// GET after PUT: accepted, opened, with additionals
	getResp, err := http.Get(baseURL + inviteURL)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer getResp.Body.Close()

	var inv Invite
	if err := json.NewDecoder(getResp.Body).Decode(&inv); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !inv.IsAccepted {
		t.Fatal("expected isAccepted=true")
	}
	if !inv.IsOpened {
		t.Fatal("expected isOpened=true after previous views")
	}
	if len(inv.Additional) != 2 {
		t.Fatalf("expected 2 additional, got %d", len(inv.Additional))
	}
	if inv.Additional[0] != "Николай Георгиев" {
		t.Fatalf("expected 'Николай Георгиев', got %q", inv.Additional[0])
	}
}

// --- Fault cases ---

func TestGetNonExistentInvite(t *testing.T) {
	resp, err := http.Get(baseURL + "/invites/00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPutInvalidNamesLatin(t *testing.T) {
	body, _ := json.Marshal(InviteUpdate{
		IsAccepted: true,
		Additional: []string{"John Doe"},
	})

	req, _ := http.NewRequest(http.MethodPut,
		baseURL+"/invites/aaaa0000-0000-0000-0000-000000000003",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for Latin names, got %d", resp.StatusCode)
	}
}

func TestPutInvalidNamesNumbers(t *testing.T) {
	body, _ := json.Marshal(InviteUpdate{
		IsAccepted: true,
		Additional: []string{"Иван123"},
	})

	req, _ := http.NewRequest(http.MethodPut,
		baseURL+"/invites/aaaa0000-0000-0000-0000-000000000003",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for names with numbers, got %d", resp.StatusCode)
	}
}

func TestPutTooManyAdditionals(t *testing.T) {
	// Invite 003 has additional_count=1, sending 2 should fail
	body, _ := json.Marshal(InviteUpdate{
		IsAccepted: true,
		Additional: []string{"Иван Петров", "Мария Петрова"},
	})

	req, _ := http.NewRequest(http.MethodPut,
		baseURL+"/invites/aaaa0000-0000-0000-0000-000000000003",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for too many additionals, got %d", resp.StatusCode)
	}
}

func TestPutAcceptedFalse(t *testing.T) {
	body, _ := json.Marshal(InviteUpdate{IsAccepted: false})

	req, _ := http.NewRequest(http.MethodPut,
		baseURL+"/invites/aaaa0000-0000-0000-0000-000000000003",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for accepted=false, got %d", resp.StatusCode)
	}
}
