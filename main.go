package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webFiles embed.FS

type Service struct {
	Name        string    `json:"name"`
	Environment string    `json:"environment"`
	Status      string    `json:"status"`
	Icon        string    `json:"icon"`
	LastCheck   time.Time `json:"last_check"`
	Recoveries  int       `json:"recoveries"`
}
type Incident struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
type Event struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}
type Store struct {
	mu        sync.RWMutex
	services  map[string]*Service
	incidents []Incident
	events    []Event
	sequence  int
}

func newStore() *Store {
	now := time.Now().UTC()
	return &Store{services: map[string]*Service{
		"api":      {Name: "api", Environment: "development", Status: "healthy", Icon: "⌘", LastCheck: now},
		"web":      {Name: "web", Environment: "development", Status: "healthy", Icon: "◫", LastCheck: now},
		"postgres": {Name: "postgres", Environment: "development", Status: "healthy", Icon: "▤", LastCheck: now},
	}}
}
func (s *Store) nextID() string { s.sequence++; return fmt.Sprintf("evt-%04d", s.sequence) }
func (s *Store) serviceList() []Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Service, 0, len(s.services))
	for _, item := range s.services {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (s *Store) incidentList() []Incident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]Incident(nil), s.incidents...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
func (s *Store) stats() map[string]int {
	services := s.serviceList()
	incidents := s.incidentList()
	result := map[string]int{"services": len(services)}
	for _, item := range services {
		if item.Status == "healthy" {
			result["healthy"]++
		}
		result["recoveries"] += item.Recoveries
	}
	for _, item := range incidents {
		if item.Status == "open" {
			result["open_incidents"]++
		}
	}
	return result
}
func (s *Store) action(name, action string, allow bool) (string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	service, ok := s.services[name]
	if !ok {
		return "unknown service", http.StatusNotFound
	}
	if !allow {
		return "demo actions are disabled by policy", http.StatusForbidden
	}
	now := time.Now().UTC()
	service.LastCheck = now
	switch action {
	case "fail":
		if service.Status != "healthy" {
			return "service is already unhealthy", http.StatusConflict
		}
		service.Status = "unhealthy"
		incident := Incident{ID: s.nextID(), Service: name, Title: "Service health check failed", Detail: "Simulated failure; no host process or container was changed", Status: "open", CreatedAt: now}
		s.incidents = append(s.incidents, incident)
		s.events = append(s.events, Event{ID: s.nextID(), Service: name, Action: "simulate_failure", Detail: "Demo-only state change", CreatedAt: now})
		return "Simulated failure recorded; the host was not changed.", http.StatusOK
	case "recover":
		if service.Status != "unhealthy" {
			return "service is not unhealthy", http.StatusConflict
		}
		if service.Recoveries >= 3 {
			return "recovery policy limit reached (3 actions per service)", http.StatusTooManyRequests
		}
		service.Status = "healthy"
		service.Recoveries++
		for i := range s.incidents {
			if s.incidents[i].Service == name && s.incidents[i].Status == "open" {
				s.incidents[i].Status = "resolved"
			}
		}
		s.events = append(s.events, Event{ID: s.nextID(), Service: name, Action: "simulated_recovery", Detail: "Policy-approved demo recovery; no host process or container was changed", CreatedAt: now})
		return "Demo recovery completed; service marked healthy and incident resolved.", http.StatusOK
	default:
		return "unknown action", http.StatusNotFound
	}
}
func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func mainHandler(store *Store, allow bool) http.Handler {
	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(static))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/services", func(w http.ResponseWriter, _ *http.Request) { jsonResponse(w, http.StatusOK, store.serviceList()) })
	mux.HandleFunc("GET /api/incidents", func(w http.ResponseWriter, _ *http.Request) { jsonResponse(w, http.StatusOK, store.incidentList()) })
	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, _ *http.Request) { jsonResponse(w, http.StatusOK, store.stats()) })
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, _ *http.Request) {
		store.mu.RLock()
		defer store.mu.RUnlock()
		jsonResponse(w, http.StatusOK, store.events)
	})
	mux.HandleFunc("POST /api/services/{name}/{action}", func(w http.ResponseWriter, r *http.Request) {
		message, status := store.action(r.PathValue("name"), r.PathValue("action"), allow)
		if status != http.StatusOK {
			jsonResponse(w, status, map[string]string{"error": message})
			return
		}
		jsonResponse(w, status, map[string]string{"message": message})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		stats := store.stats()
		services := store.serviceList()
		unhealthy := 0
		for _, service := range services {
			if service.Status != "healthy" {
				unhealthy++
			}
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintf(w, "# HELP sentinelops_services_total Number of demo services registered.\n# TYPE sentinelops_services_total gauge\nsentinelops_services_total %d\n", stats["services"])
		fmt.Fprintf(w, "# HELP sentinelops_services_unhealthy Number of demo services not healthy.\n# TYPE sentinelops_services_unhealthy gauge\nsentinelops_services_unhealthy %d\n", unhealthy)
		fmt.Fprintf(w, "# HELP sentinelops_incidents_open Number of open demo incidents.\n# TYPE sentinelops_incidents_open gauge\nsentinelops_incidents_open %d\n", stats["open_incidents"])
		fmt.Fprintf(w, "# HELP sentinelops_recoveries_total Demo recovery actions completed.\n# TYPE sentinelops_recoveries_total counter\nsentinelops_recoveries_total %d\n", stats["recoveries"])
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			files.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
	return securityHeaders(logging(mux))
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		response, err := http.Get("http://127.0.0.1" + env("LISTEN_ADDR", ":8080") + "/healthz")
		if err != nil || response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		response.Body.Close()
		return
	}
	allow, err := strconv.ParseBool(env("ALLOW_DEMO_ACTIONS", "true"))
	if err != nil {
		log.Fatal("ALLOW_DEMO_ACTIONS must be true or false")
	}
	addr := env("LISTEN_ADDR", ":8080")
	if !strings.HasPrefix(addr, ":") {
		log.Fatal("LISTEN_ADDR must bind to a port on all interfaces, for example :8080")
	}
	log.Printf("SentinelOps MVP listening on %s; demo_actions=%t", addr, allow)
	if err := http.ListenAndServe(addr, mainHandler(newStore(), allow)); err != nil {
		log.Fatal(err)
	}
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
