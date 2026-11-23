package config

import (
	"net/url"
	"strings"
	"testing"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"ALLOW_DEMO_ACTIONS", "LISTEN_ADDR", "DATABASE_URL", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_SSLMODE"} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaultsToMemoryDemo(t *testing.T) {
	clearConfigEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("listen address=%q", cfg.ListenAddr)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("database URL should be empty, got %q", cfg.DatabaseURL)
	}
	if !cfg.AllowDemoActions {
		t.Fatal("demo actions should default to enabled for local demo")
	}
}

func TestLoadBuildsEscapedPostgresURL(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("POSTGRES_HOST", "db.example")
	t.Setenv("POSTGRES_DB", "sentinel ops")
	t.Setenv("POSTGRES_USER", "ops@user")
	t.Setenv("POSTGRES_PASSWORD", "p:a/ss word")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if password != "p:a/ss word" {
		t.Fatalf("password not escaped correctly: %q", password)
	}
	if parsed.User.Username() != "ops@user" {
		t.Fatalf("username=%q", parsed.User.Username())
	}
	if parsed.Path != "/sentinel ops" {
		t.Fatalf("database path=%q", parsed.Path)
	}
	if parsed.Query().Get("sslmode") != "disable" {
		t.Fatalf("sslmode=%q", parsed.Query().Get("sslmode"))
	}
}

func TestLoadRejectsInvalidSettings(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("ALLOW_DEMO_ACTIONS", "sometimes")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ALLOW_DEMO_ACTIONS") {
		t.Fatalf("expected invalid boolean error, got %v", err)
	}
	clearConfigEnv(t)
	t.Setenv("LISTEN_ADDR", "not-a-host-port")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "LISTEN_ADDR") {
		t.Fatalf("expected invalid address error, got %v", err)
	}
}
