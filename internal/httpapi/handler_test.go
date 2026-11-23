package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sentinelops/internal/store"
)

func TestRoutesAndFailureRecovery(t *testing.T) {
	repo := store.NewMemory()
	h := New(repo, true, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s := httptest.NewServer(h)
	defer s.Close()
	for _, path := range []string{"/healthz", "/readyz", "/api/services", "/api/v1/services", "/metrics"} {
		res, err := http.Get(s.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d", path, res.StatusCode)
		}
		res.Body.Close()
	}
	res, err := http.Post(s.URL+"/api/v1/services/api/fail", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("fail status %d", res.StatusCode)
	}
	res.Body.Close()
	res, err = http.Post(s.URL+"/api/v1/services/api/recover", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("recover status %d", res.StatusCode)
	}
	res.Body.Close()
	res, err = http.Get(s.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), "sentinelops_services_total 3") {
		t.Fatalf("metrics missing series: %s", body)
	}
}

func TestDashboardFallbackPreservesAPIAndAsset404s(t *testing.T) {
	h := New(store.NewMemory(), true, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s := httptest.NewServer(h)
	defer s.Close()

	for _, path := range []string{"/dashboard", "/api/unknown", "/api/v1/unknown", "/healthz/unknown", "/readyz/unknown", "/metrics/unknown", "/assets/missing.js"} {
		res, err := http.Get(s.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		want := http.StatusNotFound
		if path == "/dashboard" {
			want = http.StatusOK
		}
		if res.StatusCode != want {
			t.Errorf("GET %s: status=%d, want %d", path, res.StatusCode, want)
		}
		res.Body.Close()
	}
}
