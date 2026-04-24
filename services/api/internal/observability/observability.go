package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const requestIDHeader = "X-Request-ID"

type QueueDepthProvider interface {
	QueueDepth(context.Context) (int, error)
}

type Observer struct {
	logger  *slog.Logger
	started time.Time

	mu           sync.Mutex
	total        int64
	byRoute      map[requestMetricKey]int64
	lastObserved time.Time
}

type requestMetricKey struct {
	Method string
	Route  string
	Status int
}

type requestMetric struct {
	Method string `json:"method"`
	Route  string `json:"route"`
	Status int    `json:"status"`
	Count  int64  `json:"count"`
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func New(logger *slog.Logger) *Observer {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}

	return &Observer{
		logger:       logger,
		started:      time.Now().UTC(),
		byRoute:      make(map[requestMetricKey]int64),
		lastObserved: time.Now().UTC(),
	}
}

func (observer *Observer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get(requestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}

		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: writer, status: http.StatusOK}
		recorder.Header().Set(requestIDHeader, requestID)

		next.ServeHTTP(recorder, request)

		route := request.Pattern
		if route == "" {
			route = request.URL.Path
		}
		observer.recordRequest(request.Method, route, recorder.status)
		observer.logger.Info("http_request",
			slog.String("request_id", requestID),
			slog.String("method", request.Method),
			slog.String("path", request.URL.Path),
			slog.String("route", route),
			slog.Int("status", recorder.status),
			slog.Int64("latency_ms", time.Since(started).Milliseconds()),
		)
	})
}

func (observer *Observer) MetricsHandler(queueDepthProvider QueueDepthProvider) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		queueDepth := -1
		if queueDepthProvider != nil {
			if depth, err := queueDepthProvider.QueueDepth(request.Context()); err == nil {
				queueDepth = depth
			}
		}

		payload := map[string]any{
			"startedAt": observer.started,
			"requests": map[string]any{
				"total":   observer.totalRequests(),
				"byRoute": observer.requestMetrics(),
			},
			"submissionQueue": map[string]any{
				"depth": queueDepth,
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(payload)
	}
}

func (observer *Observer) recordRequest(method string, route string, status int) {
	observer.mu.Lock()
	defer observer.mu.Unlock()

	observer.total++
	observer.lastObserved = time.Now().UTC()
	observer.byRoute[requestMetricKey{Method: method, Route: route, Status: status}]++
}

func (observer *Observer) totalRequests() int64 {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	return observer.total
}

func (observer *Observer) requestMetrics() []requestMetric {
	observer.mu.Lock()
	defer observer.mu.Unlock()

	metrics := make([]requestMetric, 0, len(observer.byRoute))
	for key, count := range observer.byRoute {
		metrics = append(metrics, requestMetric{
			Method: key.Method,
			Route:  key.Route,
			Status: key.Status,
			Count:  count,
		})
	}

	return metrics
}

func (recorder *responseRecorder) WriteHeader(status int) {
	if recorder.wroteHeader {
		return
	}

	recorder.wroteHeader = true
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func generateRequestID() string {
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(value)
}
