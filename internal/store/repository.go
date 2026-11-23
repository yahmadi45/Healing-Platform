package store

import (
	"context"
	"errors"

	"sentinelops/internal/domain"
)

var (
	ErrNotFound    = errors.New("resource not found")
	ErrConflict    = errors.New("operation conflicts with current state")
	ErrPolicyLimit = errors.New("recovery policy limit reached")
	ErrDisabled    = errors.New("demo actions are disabled by policy")
)

type Repository interface {
	Ping(context.Context) error
	ListServices(context.Context) ([]domain.Service, error)
	ListIncidents(context.Context) ([]domain.Incident, error)
	ListEvents(context.Context) ([]domain.Event, error)
	Stats(context.Context) (domain.Stats, error)
	Act(context.Context, string, string, bool) (string, error)
}
