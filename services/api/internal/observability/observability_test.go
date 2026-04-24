package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubQueueDepthProvider struct {
	depth int
}

func (provider stubQueueDepthProvider) QueueDepth(_ context.Context) (int, error) {
	return provider.depth, nil
}

func TestMiddlewareAddsRequestIDAndRecordsRequest(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	observer := New(logger)

	handler := observer.Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/problems", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(requestIDHeader); got == "" {
		t.Fatal("expected middleware to add request id header")
	}
	if !strings.Contains(logs.String(), `"request_id"`) || !strings.Contains(logs.String(), `"method":"GET"`) || !strings.Contains(logs.String(), `"path":"/v1/problems"`) || !strings.Contains(logs.String(), `"route":"/v1/problems"`) || !strings.Contains(logs.String(), `"status":201`) || !strings.Contains(logs.String(), `"latency_ms"`) {
		t.Fatalf("logs = %q, want structured request fields", logs.String())
	}
	if observer.totalRequests() != 1 {
		t.Fatalf("totalRequests = %d, want %d", observer.totalRequests(), 1)
	}
}

func TestMetricsHandlerReturnsRequestCountsAndQueueDepth(t *testing.T) {
	observer := New(nil)
	observer.recordRequest(http.MethodGet, "/v1/problems", http.StatusOK)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metricsz", nil)
	observer.MetricsHandler(stubQueueDepthProvider{depth: 3}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /metricsz status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	requests := payload["requests"].(map[string]any)
	if requests["total"].(float64) != 1 {
		t.Fatalf("requests.total = %v, want 1", requests["total"])
	}
	queue := payload["submissionQueue"].(map[string]any)
	if queue["depth"].(float64) != 3 {
		t.Fatalf("submissionQueue.depth = %v, want 3", queue["depth"])
	}
}
