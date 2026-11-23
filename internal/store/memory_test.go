package store

import (
	"context"
	"errors"
	"testing"
)

func TestFailRecoverCycle(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if _, err := m.Act(ctx, "api", "fail", true); err != nil {
		t.Fatal(err)
	}
	stats, _ := m.Stats(ctx)
	if stats.OpenIncidents != 1 || stats.Healthy != 2 {
		t.Fatalf("unexpected failed stats: %+v", stats)
	}
	if _, err := m.Act(ctx, "api", "recover", true); err != nil {
		t.Fatal(err)
	}
	stats, _ = m.Stats(ctx)
	if stats.OpenIncidents != 0 || stats.Healthy != 3 || stats.Recoveries != 1 {
		t.Fatalf("unexpected recovered stats: %+v", stats)
	}
}
func TestRecoveryLimitAndDisabledActions(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if _, err := m.Act(ctx, "api", "fail", false); !errors.Is(err, ErrDisabled) {
		t.Fatalf("got %v, want disabled", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := m.Act(ctx, "api", "fail", true); err != nil {
			t.Fatal(err)
		}
		if _, err := m.Act(ctx, "api", "recover", true); err != nil {
			t.Fatal(err)
		}
	}
	_, err := m.Act(ctx, "api", "fail", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Act(ctx, "api", "recover", true); !errors.Is(err, ErrPolicyLimit) {
		t.Fatalf("got %v, want policy limit", err)
	}
}
func TestNotFoundAndConflict(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	if _, err := m.Act(ctx, "missing", "fail", true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want not found", err)
	}
	if _, err := m.Act(ctx, "api", "recover", true); !errors.Is(err, ErrConflict) {
		t.Fatalf("got %v, want conflict", err)
	}
}
