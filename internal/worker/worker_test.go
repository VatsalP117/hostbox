package worker

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- SSE Hub Tests ---

func TestSSEHub_PublishSubscribe(t *testing.T) {
	hub := NewSSEHub()

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	hub.Publish("deploy-1", SSEEventLog, "hello world")

	select {
	case event := <-ch:
		if event.ID != 1 {
			t.Errorf("expected event ID 1, got %d", event.ID)
		}
		if event.Type != SSEEventLog {
			t.Errorf("expected log type, got %s", event.Type)
		}
		if event.Data != "hello world" {
			t.Errorf("expected 'hello world', got %s", event.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSSEHub_MultipleSubscribers(t *testing.T) {
	hub := NewSSEHub()

	ch1, unsub1 := hub.Subscribe("deploy-1")
	defer unsub1()
	ch2, unsub2 := hub.Subscribe("deploy-1")
	defer unsub2()

	hub.Publish("deploy-1", SSEEventStatus, "building")

	for _, ch := range []<-chan SSEEvent{ch1, ch2} {
		select {
		case event := <-ch:
			if event.Data != "building" {
				t.Errorf("expected 'building', got %s", event.Data)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	}
}

func TestSSEHub_Unsubscribe(t *testing.T) {
	hub := NewSSEHub()

	ch, unsub := hub.Subscribe("deploy-1")
	unsub()

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestSSEHub_IsolatedDeployments(t *testing.T) {
	hub := NewSSEHub()

	ch1, unsub1 := hub.Subscribe("deploy-1")
	defer unsub1()
	ch2, unsub2 := hub.Subscribe("deploy-2")
	defer unsub2()

	hub.Publish("deploy-1", SSEEventLog, "only for deploy-1")

	select {
	case <-ch1:
		// expected
	case <-time.After(time.Second):
		t.Fatal("deploy-1 should have received event")
	}

	select {
	case <-ch2:
		t.Fatal("deploy-2 should not receive deploy-1 events")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestSSEHub_MonotonicEventIDs(t *testing.T) {
	hub := NewSSEHub()

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	hub.Publish("deploy-1", SSEEventLog, "first")
	hub.Publish("deploy-1", SSEEventLog, "second")
	hub.Publish("deploy-1", SSEEventLog, "third")

	var lastID int64
	for i := 0; i < 3; i++ {
		select {
		case event := <-ch:
			if event.ID <= lastID {
				t.Errorf("event ID %d not monotonically increasing (last was %d)", event.ID, lastID)
			}
			lastID = event.ID
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	}
}

func TestSSEHub_NonBlockingPublish(t *testing.T) {
	hub := NewSSEHub()

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	// Fill the buffer (100 events)
	for i := 0; i < 100; i++ {
		hub.Publish("deploy-1", SSEEventLog, "filling buffer")
	}

	// This should not block, even with a full channel
	done := make(chan struct{})
	go func() {
		hub.Publish("deploy-1", SSEEventLog, "overflow")
		close(done)
	}()

	select {
	case <-done:
		// expected: publish completed without blocking
	case <-time.After(time.Second):
		t.Fatal("publish blocked on full channel")
	}

	// Drain to avoid leaking
	for len(ch) > 0 {
		<-ch
	}
}

func TestSSEHub_Cleanup(t *testing.T) {
	hub := NewSSEHub()
	hub.Publish("deploy-1", SSEEventLog, "hello")
	hub.Cleanup("deploy-1")

	// After cleanup, event counter should reset
	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	hub.Publish("deploy-1", SSEEventLog, "after cleanup")

	select {
	case event := <-ch:
		if event.ID != 1 {
			t.Errorf("expected event ID 1 after cleanup, got %d", event.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSSEHub_PublishJSON(t *testing.T) {
	hub := NewSSEHub()

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	hub.PublishJSON("deploy-1", SSEEventStatus, map[string]string{"status": "ready"})

	select {
	case event := <-ch:
		if !strings.Contains(event.Data, `"status":"ready"`) {
			t.Errorf("expected JSON data, got %s", event.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestSSEHub_ConcurrentAccess(t *testing.T) {
	hub := NewSSEHub()
	var wg sync.WaitGroup

	// Multiple goroutines subscribing, publishing, and unsubscribing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsub := hub.Subscribe("deploy-1")
			for j := 0; j < 50; j++ {
				hub.Publish("deploy-1", SSEEventLog, "concurrent")
			}
			unsub()
			_ = ch
		}()
	}

	wg.Wait()
}

// --- Build Logger Tests ---

func TestBuildLogger_WritesToFile(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("hello from test")
	logger.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "[INFO] hello from test") {
		t.Errorf("expected log line in file, got: %s", content)
	}
}

func TestBuildLogger_PublishesToSSE(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("sse test")
	logger.Close()

	select {
	case event := <-ch:
		if !strings.Contains(event.Data, "[INFO] sse test") {
			t.Errorf("expected sse event with log line, got: %s", event.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}

func TestBuildLogger_AllLevels(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")
	logger.Infof("formatted %s", "info")
	logger.Errorf("formatted %s", "error")
	logger.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	for _, expected := range []string{"[INFO] info msg", "[WARN] warn msg", "[ERROR] error msg", "[INFO] formatted info", "[ERROR] formatted error"} {
		if !strings.Contains(content, expected) {
			t.Errorf("expected %q in log file", expected)
		}
	}
}

func TestBuildLogger_FileSizeTruncation(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	// Max size: 100 bytes — will be exceeded quickly
	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 100)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 20; i++ {
		logger.Infof("line %d with some content to fill the buffer", i)
	}
	logger.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "truncated") {
		t.Error("expected truncation notice in log file")
	}
}

func TestBuildLogger_SSEStillWorksAfterTruncation(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 50)
	if err != nil {
		t.Fatal(err)
	}

	// Write enough to exceed limit
	for i := 0; i < 10; i++ {
		logger.Infof("line %d", i)
	}
	logger.Close()

	// SSE should still receive all events (even after file truncation)
	eventCount := 0
	for {
		select {
		case <-ch:
			eventCount++
		case <-time.After(100 * time.Millisecond):
			if eventCount < 10 {
				t.Errorf("expected at least 10 SSE events, got %d", eventCount)
			}
			return
		}
	}
}

func TestBuildLogger_StreamWriter(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	w := logger.StreamWriter(LogInfo)
	w.Write([]byte("partial "))
	w.Write([]byte("line\n"))
	logger.Close()

	select {
	case event := <-ch:
		if !strings.Contains(event.Data, "partial line") {
			t.Errorf("expected combined line, got: %s", event.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}
}

func TestBuildLogger_StreamWriter_MultipleLinesInOneWrite(t *testing.T) {
	hub := NewSSEHub()
	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "test.log")

	ch, unsub := hub.Subscribe("deploy-1")
	defer unsub()

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	w := logger.StreamWriter(LogInfo)
	w.Write([]byte("line1\nline2\nline3\n"))
	logger.Close()

	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(100 * time.Millisecond):
			if count != 3 {
				t.Errorf("expected 3 lines, got %d", count)
			}
			return
		}
	}
}

func TestBuildLogger_CreatesDirectory(t *testing.T) {
	hub := NewSSEHub()
	logDir := filepath.Join(t.TempDir(), "nested", "dir")
	logPath := filepath.Join(logDir, "test.log")

	logger, err := NewBuildLogger(logPath, hub, "deploy-1", 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	logger.Close()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected log file to be created")
	}
}
