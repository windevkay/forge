package logllama

import (
	"log/slog"
	"testing"
	"time"
)

func TestSpanHistoryStoreLogToHistory(t *testing.T) {
	history := &spanHistory{}

	entry := logEntry{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test message",
		Attrs:   []slog.Attr{slog.String("key", "value")},
	}

	history.storeLogToHistory(entry)

	if len(history.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(history.logs))
	}

	if history.logs[0].Message != "test message" {
		t.Errorf("expected message 'test message', got %q", history.logs[0].Message)
	}
}

func TestSpanHistoryMaxSize(t *testing.T) {
	history := &spanHistory{}

	// Add more than maxHistorySize entries
	for i := 0; i < maxHistorySize+10; i++ {
		entry := logEntry{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "message " + string(rune('0'+i%10)),
		}
		history.storeLogToHistory(entry)
	}

	if len(history.logs) != maxHistorySize {
		t.Errorf("expected history size %d, got %d", maxHistorySize, len(history.logs))
	}

	// Check that oldest entries were dropped
	if history.logs[0].Message != "message 0" {
		t.Errorf("expected oldest message to start from 'message 0', got %q", history.logs[0].Message)
	}
}

func TestRandID(t *testing.T) {
	id1 := randID()
	id2 := randID()

	if len(id1) != 15 {
		t.Errorf("expected ID length 15, got %d", len(id1))
	}

	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}

	// Verify it's alphanumeric
	for _, ch := range id1 {
		if !((ch >= '0' && ch <= '9') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= 'a' && ch <= 'z')) {
			t.Errorf("ID contains non-alphanumeric character: %c", ch)
		}
	}
}

func TestLogEntryStructure(t *testing.T) {
	now := time.Now()
	entry := logEntry{
		Time:    now,
		Level:   slog.LevelError,
		Message: "error occurred",
		Attrs: []slog.Attr{
			slog.String("user", "admin"),
			slog.Int("code", 500),
		},
	}

	if entry.Time != now {
		t.Error("time mismatch")
	}

	if entry.Level != slog.LevelError {
		t.Errorf("expected level ERROR, got %v", entry.Level)
	}

	if entry.Message != "error occurred" {
		t.Errorf("expected message 'error occurred', got %q", entry.Message)
	}

	if len(entry.Attrs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(entry.Attrs))
	}
}

func TestSpanHistoryConcurrency(t *testing.T) {
	history := &spanHistory{}
	done := make(chan bool, 10)

	// Simulate concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			entry := logEntry{
				Time:    time.Now(),
				Level:   slog.LevelInfo,
				Message: "concurrent message",
			}
			history.storeLogToHistory(entry)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(history.logs) != 10 {
		t.Errorf("expected 10 log entries after concurrent writes, got %d", len(history.logs))
	}
}
