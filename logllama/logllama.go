// Package logllama provides HTTP middleware and a slog.Handler wrapper that attaches
// per-request span IDs to log records and buffers non-error logs so that an error
// record can include the span's prior log history.
package logllama

import (
	"context"
	"crypto/rand"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"
)

// logEntry is a snapshot of a single log record captured for span history.
// It is JSON-annotated because instances are embedded in error records.
type logEntry struct {
	Time    time.Time   `json:"time"`
	Level   slog.Level  `json:"level"`
	Message string      `json:"message"`
	Attrs   []slog.Attr `json:"attrs"`
}

// spanHistory holds the bounded in-memory log history for a single span.
// Access is guarded by mu because multiple goroutines may log to the same span.
type spanHistory struct {
	logs []logEntry
	mu   sync.RWMutex
}

const maxHistorySize = 100

// storeLogToHistory appends a log entry to the span history and trims the
// buffer to maxHistorySize by dropping the oldest entry.
func (s *spanHistory) storeLogToHistory(log logEntry) {
	s.mu.Lock()
	s.logs = append(s.logs, log)
	if len(s.logs) > maxHistorySize {
		s.logs = s.logs[1:] // remove oldest log
	}
	s.mu.Unlock()
}

// tracingHandler wraps an underlying slog.Handler by:
// 1) injecting a span_id into each record (when present in context), and
// 2) maintaining a per-span history of non-error logs for inclusion on errors.
type tracingHandler struct {
	slog.Handler
	histories sync.Map
}

// processLog records non-error logs into the span history and, for error-level
// logs, attaches a snapshot of the history to the record and clears the history.
func (t *tracingHandler) processLog(r slog.Record, spanID string) {
	var history *spanHistory
	// load existing history
	existingHistory, ok := t.histories.Load(spanID)
	if !ok {
		history = &spanHistory{}
	} else {
		history = existingHistory.(*spanHistory)
	}
	// take action based on current record level
	if r.Level == slog.LevelError {
		history.mu.RLock()
		historySnapshot := make([]logEntry, len(history.logs))
		copy(historySnapshot, history.logs)
		history.mu.RUnlock()

		r.AddAttrs(slog.Any("span_history", historySnapshot))
		// Trigger Ollama analysis in background
		AnalyzeErrorWithHistory(spanID, r.Message, historySnapshot)
		// clear history buffer for span
		// note: this assumes an application starts to return upon encountering an error
		t.histories.Delete(spanID)
	} else {
		history.storeLogToHistory(logEntry{
			Time:    r.Time,
			Level:   r.Level,
			Message: r.Message,
			Attrs: func() []slog.Attr {
				attrs := make([]slog.Attr, 0, r.NumAttrs())

				r.Attrs(func(attr slog.Attr) bool {
					attrs = append(attrs, attr)
					return true
				})

				return attrs
			}(),
		})
		t.histories.Store(spanID, history)
	}
}

// spanIDKey is the context key type used to store and retrieve span IDs.
type spanIDKey struct{}

// Handle enriches the record with span_id and routes it through processLog,
// then forwards to the wrapped handler. It expects spanIDKey to be present
// in the context for per-request tracing.
func (t *tracingHandler) Handle(ctx context.Context, r slog.Record) error {
	if v := ctx.Value(spanIDKey{}); v != nil {
		if spanID, ok := v.(string); ok {
			r.AddAttrs(slog.String("span_id", spanID))
			t.processLog(r, spanID)
		}
	}

	return t.Handler.Handle(ctx, r)
}

// Setup installs a global JSON slog logger that captures span histories and
// returns an HTTP middleware that assigns a unique span_id per request.
// Histories are cleared on error or at request completion via defer.
func Setup() func(next http.Handler) http.Handler {
	handler := &tracingHandler{
		Handler: slog.NewJSONHandler(os.Stdout, nil),
	}
	slog.SetDefault(slog.New(handler))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spanID := randID()
			ctx := context.WithValue(r.Context(), spanIDKey{}, spanID)

			defer func() {
				// clean up history for successful requests
				handler.histories.Delete(spanID)
			}()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var alphanum = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

// randID returns a cryptographically secure, 15-character alphanumeric ID
// used as the span identifier for HTTP requests.
func randID() string {
	const size = 15
	b := make([]rune, size)

	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphanum))))
		b[i] = alphanum[idx.Int64()]
	}
	return string(b)
}
