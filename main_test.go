package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sentinelops/internal/httpapi"
	"sentinelops/internal/store"
)

func TestFoundationRoutes(t *testing.T) {
	handler := httpapi.New(store.NewMemory(), true, slog.New(slog.NewTextHandler(io.Discard, nil)))
	server := httptest.NewServer(handler)
	defer server.Close()

	for _, path := range []string{"/healthz", "/readyz", "/api/services", "/api/v1/services", "/api/incidents", "/api/stats", "/metrics", "/"} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status=%d", path, response.StatusCode)
		}
		if path == "/metrics" {
			body, _ := io.ReadAll(response.Body)
			if !strings.Contains(string(body), "sentinelops_services_total 3") {
				t.Error("expected services metric")
			}
		}
		response.Body.Close()
	}
}
