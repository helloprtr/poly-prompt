package event

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Type identifies the kind of event written to events.jsonl.
type Type string

const (
	RunStarted        Type = "run.started"
	ContextCompiled   Type = "context.compiled"
	PlanCreated       Type = "plan.created"
	TodoUpdated       Type = "todo.updated"
	WorkerStarted     Type = "worker.started"
	WorkerCompleted   Type = "worker.completed"
	ArtifactReady     Type = "artifact.ready"
	ApprovalRequested Type = "approval.requested"
	ApprovalGranted   Type = "approval.granted"
	DeliveryStarted   Type = "delivery.started"
	DeliveryCompleted Type = "delivery.completed"
	MemorySuggested   Type = "memory.update.suggested"
	RunCompleted      Type = "run.completed"
	RunFailed         Type = "run.failed"
	LLMEnhanceFailed  Type = "llm.enhance.failed"
)

// Event is a single structured entry appended to a run's events.jsonl log.
type Event struct {
	Type      Type           `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data,omitempty"`
}

// Append serialises e as a JSON line and appends it to path.
// A zero Timestamp is replaced with the current UTC time.
// A blank path is a no-op.
func Append(path string, e Event) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create event log directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer f.Close()
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	return nil
}
