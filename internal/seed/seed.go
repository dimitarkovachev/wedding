package seed

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/dimitarkovachev/wedding/internal/store"
)

type SeedData struct {
	Invites map[string]store.InviteRecord `json:"invites"`
}

// LoadFromFile reads seed data from a JSON file and populates the store.
// Returns nil if path is empty (seeding disabled).
func LoadFromFile(path string, s *store.BBoltStore) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading seed file %s: %w", path, err)
	}

	var sd SeedData
	if err := json.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("parsing seed file %s: %w", path, err)
	}

	log.WithField("count", len(sd.Invites)).Info("seeding invites from file")

	return s.Seed(sd.Invites)
}
