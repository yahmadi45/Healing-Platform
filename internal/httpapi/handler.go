package httpapi

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"sentinelops/internal/store"
)

//go:embed web/*
var webFiles embed.FS

type Handler struct {
	repo         store.Repository
	allowActions bool
	logger       *slog.Logger
}

func New(repo store.Repository, allow bool, logger *slog.Logger) http.Handler {
	h := &Handler{repo: repo, allowActions: allow, logger: logger}
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	static := http.FileServer(http.FS(web))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /readyz", h.ready)
	mux.HandleFunc("GET /api/services", h.services)
	mux.HandleFunc("GET /api/incidents", h.incidents)
	mux.HandleFunc("GET /api/events", h.events)
	mux.HandleFunc("GET /api/stats", h.stats)
	mux.HandleFunc("POST /api/services/{name}/{action}", h.action)
	mux.HandleFunc("GET /api/v1/services", h.services)
	mux.HandleFunc("GET /api/v1/incidents", h.incidents)
	mux.HandleFunc("GET /api/v1/events", h.events)
	mux.HandleFunc("GET /api/v1/stats", h.stats)
	mux.HandleFunc("POST /api/v1/services/{name}/{action}", h.action)
	mux.HandleFunc("GET /metrics", h.metrics)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if requestPath == "" {
			requestPath = "index.html"
		}
		if requestPath == "api" || strings.HasPrefix(requestPath, "api/") ||
			requestPath == "healthz" || strings.HasPrefix(requestPath, "healthz/") ||
			requestPath == "readyz" || strings.HasPrefix(requestPath, "readyz/") ||
			requestPath == "metrics" || strings.HasPrefix(requestPath, "metrics/") {
			http.NotFound(w, r)
			return
		}
		if file, err := web.Open(requestPath); err == nil {
			info, statErr := file.Stat()
			file.Close()
			if statErr == nil && !info.IsDir() {
				static.ServeHTTP(w, r)
				return
			}
		}
		if requestPath == "assets" || strings.HasPrefix(requestPath, "assets/") || path.Ext(requestPath) != "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, web, "index.html")
	}))
	return securityHeaders(requestLog(logger, mux))
}
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.repo.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "error": "database unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (h *Handler) services(w http.ResponseWriter, r *http.Request) {
	v, err := h.repo.ListServices(r.Context())
	h.reply(w, v, err)
}
func (h *Handler) incidents(w http.ResponseWriter, r *http.Request) {
	v, err := h.repo.ListIncidents(r.Context())
	h.reply(w, v, err)
}
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	v, err := h.repo.ListEvents(r.Context())
	h.reply(w, v, err)
}
func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	v, err := h.repo.Stats(r.Context())
	h.reply(w, v, err)
}
func (h *Handler) action(w http.ResponseWriter, r *http.Request) {
	message, err := h.repo.Act(r.Context(), r.PathValue("name"), r.PathValue("action"), h.allowActions)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, store.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(err, store.ErrConflict):
			status = http.StatusConflict
		case errors.Is(err, store.ErrPolicyLimit):
			status = http.StatusTooManyRequests
		case errors.Is(err, store.ErrDisabled):
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": message})
}
func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.Stats(r.Context())
	if err != nil {
		http.Error(w, "metrics unavailable", http.StatusServiceUnavailable)
		return
	}
	unhealthy := stats.Services - stats.Healthy
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP sentinelops_services_total Number of tracked services.\n# TYPE sentinelops_services_total gauge\nsentinelops_services_total %d\n", stats.Services)
	fmt.Fprintf(w, "# HELP sentinelops_services_unhealthy Number of services not healthy.\n# TYPE sentinelops_services_unhealthy gauge\nsentinelops_services_unhealthy %d\n", unhealthy)
	fmt.Fprintf(w, "# HELP sentinelops_incidents_open Number of open incidents.\n# TYPE sentinelops_incidents_open gauge\nsentinelops_incidents_open %d\n", stats.OpenIncidents)
	fmt.Fprintf(w, "# HELP sentinelops_recoveries_total Completed demo recoveries.\n# TYPE sentinelops_recoveries_total counter\nsentinelops_recoveries_total %d\n", stats.Recoveries)
}

func (h *Handler) reply(w http.ResponseWriter, v any, err error) {
	if err != nil {
		h.logger.Error("request failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
func requestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).Round(time.Millisecond))
	})
}
