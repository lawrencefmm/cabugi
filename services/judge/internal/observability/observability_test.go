package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandlerReturnsWorkerCountersAndHeartbeat(t *testing.T) {
	var logs bytes.Buffer
	observer := New(slog.New(slog.NewJSONHandler(&logs, nil)))
	observer.JobClaimed("submission-1", "cpp17")
	observer.JobCompleted("submission-1", "accepted")
	observer.JobRetried("submission-2", "temporary error")
	observer.JobTerminalFailure("submission-3", "fatal error")
	observer.NoJobs()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metricsz", nil)
	observer.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /metricsz status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	worker := payload["worker"].(map[string]any)
	if worker["claimed"].(float64) != 1 || worker["completed"].(float64) != 1 || worker["retried"].(float64) != 1 || worker["terminalFailures"].(float64) != 1 || worker["noJobs"].(float64) != 1 {
		t.Fatalf("worker metrics = %#v, want counters recorded", worker)
	}
	outcomes := worker["outcomes"].(map[string]any)
	if outcomes["accepted"].(float64) != 1 {
		t.Fatalf("worker outcomes = %#v, want accepted count", outcomes)
	}
	heartbeat := payload["heartbeat"].(map[string]any)
	if heartbeat["lastSeenAt"] == "" {
		t.Fatalf("heartbeat payload = %#v, want lastSeenAt", heartbeat)
	}
	for _, event := range []string{"judge_job_claimed", "judge_job_completed", "judge_job_retried", "judge_job_terminal_failure"} {
		if !strings.Contains(logs.String(), event) {
			t.Fatalf("logs = %q, want event %q", logs.String(), event)
		}
	}
}

func TestHealthzHandler(t *testing.T) {
	observer := New(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	observer.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("GET /healthz body = %q, want ok status", got)
	}
}
