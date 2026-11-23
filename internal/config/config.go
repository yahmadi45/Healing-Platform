package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr       string
	DatabaseURL      string
	AllowDemoActions bool
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ShutdownTimeout  time.Duration
}

func Load() (Config, error) {
	allow, err := strconv.ParseBool(value("ALLOW_DEMO_ACTIONS", "true"))
	if err != nil {
		return Config{}, fmt.Errorf("ALLOW_DEMO_ACTIONS must be true or false: %w", err)
	}
	addr := value("LISTEN_ADDR", ":8080")
	if _, _, err := net.SplitHostPort(strings.TrimPrefix(addr, "tcp://")); err != nil {
		return Config{}, fmt.Errorf("LISTEN_ADDR must be host:port: %w", err)
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" && os.Getenv("POSTGRES_HOST") != "" {
		u := &url.URL{Scheme: "postgres", Host: net.JoinHostPort(value("POSTGRES_HOST", "postgres"), value("POSTGRES_PORT", "5432")), Path: "/" + value("POSTGRES_DB", "sentinelops")}
		u.User = url.UserPassword(value("POSTGRES_USER", "sentinelops"), os.Getenv("POSTGRES_PASSWORD"))
		query := u.Query()
		query.Set("sslmode", value("POSTGRES_SSLMODE", "disable"))
		u.RawQuery = query.Encode()
		databaseURL = u.String()
	}
	return Config{ListenAddr: addr, DatabaseURL: databaseURL, AllowDemoActions: allow, ReadTimeout: 10 * time.Second, WriteTimeout: 20 * time.Second, ShutdownTimeout: 15 * time.Second}, nil
}
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
