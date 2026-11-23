CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE services (
    name TEXT PRIMARY KEY,
    environment TEXT NOT NULL DEFAULT 'development',
    status TEXT NOT NULL CHECK (status IN ('healthy', 'unhealthy', 'unknown')),
    icon TEXT NOT NULL DEFAULT '',
    last_check TIMESTAMPTZ NOT NULL DEFAULT now(),
    recoveries INTEGER NOT NULL DEFAULT 0 CHECK (recoveries >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name TEXT NOT NULL REFERENCES services(name) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('open', 'resolved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX incidents_status_created_idx ON incidents(status, created_at DESC);
CREATE INDEX incidents_service_created_idx ON incidents(service_name, created_at DESC);

CREATE TABLE operation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name TEXT NOT NULL REFERENCES services(name) ON DELETE RESTRICT,
    action TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX operation_events_created_idx ON operation_events(created_at DESC);

INSERT INTO services(name, environment, status, icon) VALUES
    ('api', 'development', 'healthy', '⌘'),
    ('web', 'development', 'healthy', '◫'),
    ('postgres', 'development', 'healthy', '▤')
ON CONFLICT (name) DO NOTHING;
