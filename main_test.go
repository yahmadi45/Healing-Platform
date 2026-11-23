package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSimulatedFailureAndRecovery(t *testing.T) {
	store := newStore()
	if message, status := store.action("api", "fail", true); status != http.StatusOK {
		t.Fatalf("fail: status=%d message=%s", status, message)
	}
	if got := store.serviceList()[0].Status; got == "" {
		t.Fatal("expected services")
	}
	if message, status := store.action("api", "recover", true); status != http.StatusOK {
		t.Fatalf("recover: status=%d message=%s", status, message)
	}
	if got := store.services["api"].Status; got != "healthy" {
		t.Fatalf("status=%s, want healthy", got)
	}
	if got := store.incidents[0].Status; got != "resolved" {
		t.Fatalf("incident status=%s, want resolved", got)
	}
}

func TestRecoveryPolicyLimit(t *testing.T) {
	store := newStore()
	for i := 0; i < 3; i++ {
		if _, status := store.action("api", "fail", true); status != http.StatusOK {
			t.Fatalf("fail status=%d", status)
		}
		if _, status := store.action("api", "recover", true); status != http.StatusOK {
			t.Fatalf("recover status=%d", status)
		}
	}
	_, status := store.action("api", "fail", true)
	if status != http.StatusOK {
		t.Fatalf("fourth demo failure status=%d", status)
	}
	if _, status = store.action("api", "recover", true); status != http.StatusTooManyRequests {
		t.Fatalf("recovery status=%d, want 429", status)
	}
}

func TestActionsCanBeDisabled(t *testing.T) {
	store := newStore()
	if _, status := store.action("api", "fail", false); status != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", status)
	}
}

func TestAPIHealthAndMetrics(t *testing.T) {
	server := httptest.NewServer(mainHandler(newStore(), true))
	defer server.Close()
	for _, path := range []string{"/healthz", "/readyz", "/api/services", "/api/incidents", "/api/stats", "/metrics", "/"} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status=%d", path, response.StatusCode)
		}
		if path == "/api/stats" {
			var value map[string]int
			if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
				t.Errorf("decode stats: %v", err)
			}
			if value["services"] != 3 {
				t.Errorf("services=%d, want 3", value["services"])
			}
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
