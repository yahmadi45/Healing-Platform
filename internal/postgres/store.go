package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"sentinelops/internal/domain"
	"sentinelops/internal/store"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store                     { return &Store{db: db} }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *Store) ListServices(ctx context.Context) ([]domain.Service, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name,environment,status,icon,last_check,recoveries FROM services ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Service
	for rows.Next() {
		var v domain.Service
		if err := rows.Scan(&v.Name, &v.Environment, &v.Status, &v.Icon, &v.LastCheck, &v.Recoveries); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) ListIncidents(ctx context.Context) ([]domain.Incident, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,service_name,title,detail,status,created_at FROM incidents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Incident
	for rows.Next() {
		var v domain.Incident
		if err := rows.Scan(&v.ID, &v.Service, &v.Title, &v.Detail, &v.Status, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) ListEvents(ctx context.Context) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id::text,service_name,action,detail,created_at FROM operation_events ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Event
	for rows.Next() {
		var v domain.Event
		if err := rows.Scan(&v.ID, &v.Service, &v.Action, &v.Detail, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Stats(ctx context.Context) (domain.Stats, error) {
	var out domain.Stats
	err := s.db.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE status='healthy'),coalesce(sum(recoveries),0) FROM services`).Scan(&out.Services, &out.Healthy, &out.Recoveries)
	if err != nil {
		return out, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM incidents WHERE status='open'`).Scan(&out.OpenIncidents)
	return out, err
}
func (s *Store) Act(ctx context.Context, name, action string, allow bool) (string, error) {
	if !allow {
		return "", store.ErrDisabled
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var status string
	var recoveries int
	err = tx.QueryRowContext(ctx, `SELECT status,recoveries FROM services WHERE name=$1 FOR UPDATE`, name).Scan(&status, &recoveries)
	if errors.Is(err, sql.ErrNoRows) {
		return "", store.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	switch action {
	case "fail":
		if status != "healthy" {
			return "", store.ErrConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE services SET status='unhealthy',last_check=$2,updated_at=$2 WHERE name=$1`, name, now)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO incidents(service_name,title,detail,status,created_at) VALUES($1,$2,$3,'open',$4)`, name, "Service health check failed", "Simulated failure; no host process or container was changed", now)
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO operation_events(service_name,action,detail,created_at) VALUES($1,$2,$3,$4)`, name, "simulate_failure", "Demo-only state change", now)
		}
		if err != nil {
			return "", fmt.Errorf("record simulated failure: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return "", err
		}
		return "Simulated failure recorded; the host was not changed.", nil
	case "recover":
		if status != "unhealthy" {
			return "", store.ErrConflict
		}
		if recoveries >= 3 {
			return "", store.ErrPolicyLimit
		}
		_, err = tx.ExecContext(ctx, `UPDATE services SET status='healthy',recoveries=recoveries+1,last_check=$2,updated_at=$2 WHERE name=$1`, name, now)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE incidents SET status='resolved',resolved_at=$2 WHERE service_name=$1 AND status='open'`, name, now)
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO operation_events(service_name,action,detail,created_at) VALUES($1,$2,$3,$4)`, name, "simulated_recovery", "Policy-approved demo recovery; no host process or container was changed", now)
		}
		if err != nil {
			return "", fmt.Errorf("record simulated recovery: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return "", err
		}
		return "Demo recovery completed; service marked healthy and incident resolved.", nil
	default:
		return "", store.ErrNotFound
	}
}
