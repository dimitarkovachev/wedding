package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var bucketName = []byte("invites")

type BBoltStore struct {
	db *bolt.DB
}

func NewBBoltStore(path string) (*BBoltStore, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening bbolt db at %s: %w", path, err)
	}

	// Reason: bucket must exist before any read/write operations
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("creating invites bucket: %w", err)
	}

	return &BBoltStore{db: db}, nil
}

func (s *BBoltStore) GetInvite(_ context.Context, id string) (*InviteRecord, error) {
	var record *InviteRecord

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}

		var r InviteRecord
		if err := json.Unmarshal(data, &r); err != nil {
			return fmt.Errorf("unmarshaling invite %s: %w", id, err)
		}

		r.ViewedAt = append(r.ViewedAt, time.Now().UTC())

		updated, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshaling invite %s: %w", id, err)
		}
		if err := b.Put([]byte(id), updated); err != nil {
			return fmt.Errorf("writing viewed_at for invite %s: %w", id, err)
		}

		record = &r
		return nil
	})
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *BBoltStore) UpdateInvite(_ context.Context, id string, accepted bool, additional []string) (*InviteRecord, error) {
	var record *InviteRecord

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}

		var r InviteRecord
		if err := json.Unmarshal(data, &r); err != nil {
			return fmt.Errorf("unmarshaling invite %s: %w", id, err)
		}

		if !accepted {
			return fmt.Errorf("only accepted=true updates are allowed")
		}

		if len(additional) > r.AdditionalCount {
			return fmt.Errorf(
				"too many additional guests: got %d, max allowed %d",
				len(additional), r.AdditionalCount,
			)
		}

		r.Accepted = true
		r.Additional = additional
		now := time.Now().UTC()
		r.AcceptedAt = &now

		updated, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshaling invite %s: %w", id, err)
		}
		if err := b.Put([]byte(id), updated); err != nil {
			return fmt.Errorf("writing invite %s: %w", id, err)
		}

		record = &r
		return nil
	})
	if err != nil {
		return nil, err
	}

	return record, nil
}

// Seed loads invite records from a map, skipping keys that already exist.
func (s *BBoltStore) Seed(invites map[string]InviteRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		for id, rec := range invites {
			existing := b.Get([]byte(id))
			if existing != nil {
				log.WithField("id", id).Debug("seed: invite already exists, skipping")
				continue
			}
			data, err := json.Marshal(rec)
			if err != nil {
				return fmt.Errorf("marshaling seed invite %s: %w", id, err)
			}
			if err := b.Put([]byte(id), data); err != nil {
				return fmt.Errorf("seeding invite %s: %w", id, err)
			}
			log.WithField("id", id).Info("seeded invite")
		}
		return nil
	})
}

func (s *BBoltStore) GetAllInvites(_ context.Context) (map[string]InviteRecord, error) {
	result := make(map[string]InviteRecord)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.ForEach(func(k, v []byte) error {
			var r InviteRecord
			if err := json.Unmarshal(v, &r); err != nil {
				return fmt.Errorf("unmarshaling invite %s: %w", string(k), err)
			}
			result[string(k)] = r
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *BBoltStore) ReplaceAllInvites(_ context.Context, invites map[string]InviteRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(bucketName); err != nil {
			return fmt.Errorf("deleting invites bucket: %w", err)
		}
		b, err := tx.CreateBucket(bucketName)
		if err != nil {
			return fmt.Errorf("recreating invites bucket: %w", err)
		}
		for id, rec := range invites {
			data, err := json.Marshal(rec)
			if err != nil {
				return fmt.Errorf("marshaling invite %s: %w", id, err)
			}
			if err := b.Put([]byte(id), data); err != nil {
				return fmt.Errorf("writing invite %s: %w", id, err)
			}
		}
		return nil
	})
}

func (s *BBoltStore) Close() error {
	return s.db.Close()
}
