package store

import (
	"context"
	"time"
)

type InviteRecord struct {
	People          []string    `json:"people"`
	AdditionalCount int         `json:"additional_count"`
	Additional      []string    `json:"additional"`
	Accepted        bool        `json:"accepted"`
	ViewedAt        []time.Time `json:"viewed_at"`
	AcceptedAt      *time.Time  `json:"accepted_at"`
}

type InviteStore interface {
	GetInvite(ctx context.Context, id string) (*InviteRecord, error)
	UpdateInvite(ctx context.Context, id string, accepted bool, additional []string) (*InviteRecord, error)
	Close() error
}
