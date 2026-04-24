package observability

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Observer struct {
	logger  *slog.Logger
	started time.Time

	mu                 sync.Mutex
	lastHeartbeat      time.Time
	claimed            int64
	completed          int64
	retried            int64
	terminalFailures   int64
	noJobs             int64
	completionOutcomes map[string]int64
}

func New(logger *slog.Logger) *Observer {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	now := time.Now().UTC()
	return &Observer{
		logger:             logger,
		started:            now,
		lastHeartbeat:      now,
		completionOutcomes: make(map[string]int64),
	}
}

func (observer *Observer) Heartbeat() {
	observer.mu.Lock()
	observer.lastHeartbeat = time.Now().UTC()
	observer.mu.Unlock()
}

func (observer *Observer) JobClaimed(submissionID string, language string) {
	observer.mu.Lock()
	observer.claimed++
	observer.lastHeartbeat = time.Now().UTC()
	observer.mu.Unlock()

	observer.logger.Info("judge_job_claimed",
		slog.String("submission_id", submissionID),
		slog.String("language", language),
	)
}

func (observer *Observer) JobCompleted(submissionID string, status string) {
	observer.mu.Lock()
	observer.completed++
	observer.lastHeartbeat = time.Now().UTC()
	observer.completionOutcomes[status]++
	observer.mu.Unlock()

	observer.logger.Info("judge_job_completed",
		slog.String("submission_id", submissionID),
		slog.String("status", status),
	)
}

func (observer *Observer) JobRetried(submissionID string, lastError string) {
	observer.mu.Lock()
	observer.retried++
	observer.lastHeartbeat = time.Now().UTC()
	observer.mu.Unlock()

	observer.logger.Warn("judge_job_retried",
		slog.String("submission_id", submissionID),
		slog.String("error", lastError),
	)
}

func (observer *Observer) JobTerminalFailure(submissionID string, lastError string) {
	observer.mu.Lock()
	observer.terminalFailures++
	observer.lastHeartbeat = time.Now().UTC()
	observer.mu.Unlock()

	observer.logger.Error("judge_job_terminal_failure",
		slog.String("submission_id", submissionID),
		slog.String("error", lastError),
	)
}

func (observer *Observer) NoJobs() {
	observer.mu.Lock()
	observer.noJobs++
	observer.lastHeartbeat = time.Now().UTC()
	observer.mu.Unlock()
}

func (observer *Observer) Handler() http.Handler {
	mux := http.NewServeMux()
	observer.AttachRoutes(mux)
	return mux
}

func (observer *Observer) AttachRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", observer.healthzHandler)
	mux.HandleFunc("GET /metricsz", observer.metricsHandler)
}

func (observer *Observer) healthzHandler(writer http.ResponseWriter, _ *http.Request) {
	observer.writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (observer *Observer) metricsHandler(writer http.ResponseWriter, _ *http.Request) {
	payload := observer.snapshot()
	observer.writeJSON(writer, http.StatusOK, payload)
}

func (observer *Observer) snapshot() map[string]any {
	observer.mu.Lock()
	defer observer.mu.Unlock()

	now := time.Now().UTC()
	outcomes := make(map[string]int64, len(observer.completionOutcomes))
	for key, value := range observer.completionOutcomes {
		outcomes[key] = value
	}

	return map[string]any{
		"startedAt": observer.started,
		"heartbeat": map[string]any{
			"lastSeenAt": observer.lastHeartbeat,
			"ageMs":      now.Sub(observer.lastHeartbeat).Milliseconds(),
		},
		"worker": map[string]any{
			"claimed":          observer.claimed,
			"completed":        observer.completed,
			"retried":          observer.retried,
			"terminalFailures": observer.terminalFailures,
			"noJobs":           observer.noJobs,
			"outcomes":         outcomes,
		},
	}
}

func (observer *Observer) writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
