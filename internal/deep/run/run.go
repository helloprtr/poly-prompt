package run

import (
	"time"

	"github.com/helloprtr/poly-prompt/internal/deep/plan"
	"github.com/helloprtr/poly-prompt/internal/deep/schema"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

// Status is the lifecycle state of a DeepRun.
type Status string

const (
	StatusCreated              Status = "created"
	StatusPlanning             Status = "planning"
	StatusRunning              Status = "running"
	StatusAwaitingApproval     Status = "awaiting_approval"
	StatusCompleted            Status = "completed"
	StatusCompletedWithWarnings Status = "completed_with_warnings"
	StatusFailed               Status = "failed"
)

// DeepRun is the canonical in-memory and on-disk record for a single deep execution.
type DeepRun struct {
	ID              string        `json:"id"`
	Version         int           `json:"version"`
	Action          string        `json:"action"`
	Engine          string        `json:"engine"`
	Status          Status        `json:"status"`
	ResultType      string        `json:"result_type"`
	TargetApp       string        `json:"target_app,omitempty"`
	DeliveryMode    string        `json:"delivery_mode,omitempty"`
	SourceKind      string        `json:"source_kind,omitempty"`
	ParentHistoryID string        `json:"parent_history_id,omitempty"`
	RepoRoot        string        `json:"repo_root,omitempty"`
	ArtifactRoot    string        `json:"artifact_root"`
	EventLogPath    string        `json:"event_log_path"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
	Plan            plan.WorkPlan `json:"plan"`
	ResultRef       string        `json:"result_ref,omitempty"`
	WarningCount    int           `json:"warning_count,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
}

// Progress carries incremental status updates emitted during execution.
type Progress struct {
	Step    string
	Index   int
	Total   int
	Message string
}

// Options are the caller-supplied inputs for starting a deep run.
type Options struct {
	Action          string
	Source          string
	SourceKind      string
	TargetApp       string
	DeliveryMode    string
	RepoRoot        string
	ParentHistoryID string
	ProtectedTerms  []string
	HistoryEntry    *history.Entry
	RepoSummary     repoctx.Summary
	Progress        func(Progress)
}

// Result is returned by ExecutePatchRun on success.
type Result struct {
	Run            DeepRun
	Bundle         schema.PatchBundle
	DeliveryPrompt string
}
