package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWritesJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	l := NewLogger(path)
	if err := l.Log("admin", "session.revoke", "token-1", "success", ""); err != nil {
		t.Fatalf("Log() error: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	line := strings.TrimSpace(string(b))
	if line == "" {
		t.Fatalf("expected non-empty audit line")
	}
	var e Event
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		t.Fatalf("decode audit line: %v", err)
	}
	if e.Actor != "admin" || e.Action != "session.revoke" || e.Outcome != "success" {
		t.Fatalf("unexpected audit event content: %+v", e)
	}
}
