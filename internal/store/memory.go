package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"sentinelops/internal/domain"
)

type Memory struct {
	mu        sync.RWMutex
	services  map[string]*domain.Service
	incidents []domain.Incident
	events    []domain.Event
	sequence  int
}

func NewMemory() *Memory {
	now := time.Now().UTC()
	return &Memory{services: map[string]*domain.Service{
		"api":      {Name: "api", Environment: "development", Status: "healthy", Icon: "⌘", LastCheck: now},
		"web":      {Name: "web", Environment: "development", Status: "healthy", Icon: "◫", LastCheck: now},
		"postgres": {Name: "postgres", Environment: "development", Status: "healthy", Icon: "▤", LastCheck: now},
	}}
}
func (*Memory) Ping(context.Context) error { return nil }
func (m *Memory) ListServices(context.Context) ([]domain.Service, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Service, 0, len(m.services))
	for _, v := range m.services {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
func (m *Memory) ListIncidents(context.Context) ([]domain.Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := append([]domain.Incident(nil), m.incidents...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func (m *Memory) ListEvents(context.Context) ([]domain.Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.Event(nil), m.events...), nil
}
func (m *Memory) Stats(ctx context.Context) (domain.Stats, error) {
	sv, _ := m.ListServices(ctx)
	ins, _ := m.ListIncidents(ctx)
	s := domain.Stats{Services: len(sv)}
	for _, v := range sv {
		if v.Status == "healthy" {
			s.Healthy++
		}
		s.Recoveries += v.Recoveries
	}
	for _, v := range ins {
		if v.Status == "open" {
			s.OpenIncidents++
		}
	}
	return s, nil
}
func (m *Memory) nextID() string { m.sequence++; return fmt.Sprintf("evt-%04d", m.sequence) }
func (m *Memory) Act(_ context.Context, name, action string, allow bool) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	service, ok := m.services[name]
	if !ok {
		return "", ErrNotFound
	}
	if !allow {
		return "", ErrDisabled
	}
	now := time.Now().UTC()
	service.LastCheck = now
	switch action {
	case "fail":
		if service.Status != "healthy" {
			return "", ErrConflict
		}
		service.Status = "unhealthy"
		m.incidents = append(m.incidents, domain.Incident{ID: m.nextID(), Service: name, Title: "Service health check failed", Detail: "Simulated failure; no host process or container was changed", Status: "open", CreatedAt: now})
		m.events = append(m.events, domain.Event{ID: m.nextID(), Service: name, Action: "simulate_failure", Detail: "Demo-only state change", CreatedAt: now})
		return "Simulated failure recorded; the host was not changed.", nil
	case "recover":
		if service.Status != "unhealthy" {
			return "", ErrConflict
		}
		if service.Recoveries >= 3 {
			return "", ErrPolicyLimit
		}
		service.Status = "healthy"
		service.Recoveries++
		for i := range m.incidents {
			if m.incidents[i].Service == name && m.incidents[i].Status == "open" {
				m.incidents[i].Status = "resolved"
			}
		}
		m.events = append(m.events, domain.Event{ID: m.nextID(), Service: name, Action: "simulated_recovery", Detail: "Policy-approved demo recovery; no host process or container was changed", CreatedAt: now})
		return "Demo recovery completed; service marked healthy and incident resolved.", nil
	default:
		return "", ErrNotFound
	}
}
