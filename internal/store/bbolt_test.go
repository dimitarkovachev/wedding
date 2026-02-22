package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.db")
}

func seedTestStore(t *testing.T) *BBoltStore {
	t.Helper()
	s, err := NewBBoltStore(tempDBPath(t))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	err = s.Seed(map[string]InviteRecord{
		"aaa-001": {
			People:          []string{"Иван Петров", "Мария Петрова"},
			AdditionalCount: 2,
			Accepted:        false,
		},
	})
	if err != nil {
		t.Fatalf("failed to seed: %v", err)
	}
	return s
}

func TestGetInvite_Found(t *testing.T) {
	s := seedTestStore(t)

	rec, err := s.GetInvite(context.Background(), "aaa-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec == nil {
		t.Fatal("expected invite, got nil")
	}
	if len(rec.People) != 2 {
		t.Fatalf("expected 2 people, got %d", len(rec.People))
	}
	if rec.Accepted {
		t.Fatal("expected accepted=false")
	}
	if len(rec.ViewedAt) != 1 {
		t.Fatalf("expected 1 viewed_at entry, got %d", len(rec.ViewedAt))
	}
}

func TestGetInvite_NotFound(t *testing.T) {
	s := seedTestStore(t)

	rec, err := s.GetInvite(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec != nil {
		t.Fatal("expected nil for nonexistent invite")
	}
}

func TestGetInvite_AppendsViewedAt(t *testing.T) {
	s := seedTestStore(t)

	for i := 0; i < 3; i++ {
		_, _ = s.GetInvite(context.Background(), "aaa-001")
	}

	rec, err := s.GetInvite(context.Background(), "aaa-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 3 previous GETs + this one = 4
	if len(rec.ViewedAt) != 4 {
		t.Fatalf("expected 4 viewed_at entries, got %d", len(rec.ViewedAt))
	}
}

func TestUpdateInvite_AcceptWithAdditionals(t *testing.T) {
	s := seedTestStore(t)

	rec, err := s.UpdateInvite(context.Background(), "aaa-001", true, []string{"Георги"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.Accepted {
		t.Fatal("expected accepted=true")
	}
	if len(rec.Additional) != 1 || rec.Additional[0] != "Георги" {
		t.Fatalf("unexpected additional: %v", rec.Additional)
	}
	if rec.AcceptedAt == nil {
		t.Fatal("expected accepted_at to be set")
	}
}

func TestUpdateInvite_NotFound(t *testing.T) {
	s := seedTestStore(t)

	rec, err := s.UpdateInvite(context.Background(), "nonexistent", true, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec != nil {
		t.Fatal("expected nil for nonexistent invite")
	}
}

func TestUpdateInvite_AcceptedFalse(t *testing.T) {
	s := seedTestStore(t)

	_, err := s.UpdateInvite(context.Background(), "aaa-001", false, nil)
	if err == nil {
		t.Fatal("expected error for accepted=false")
	}
}

func TestUpdateInvite_TooManyAdditionals(t *testing.T) {
	s := seedTestStore(t)

	_, err := s.UpdateInvite(context.Background(), "aaa-001", true, []string{"А", "Б", "В"})
	if err == nil {
		t.Fatal("expected error for too many additionals")
	}
}

func TestNewBBoltStore_InvalidPath(t *testing.T) {
	_, err := NewBBoltStore(filepath.Join(os.DevNull, "impossible", "path.db"))
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
