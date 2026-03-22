package session

import "time"

type Mode string
type Status string

const (
	ModeReview Mode = "review"
	ModeEdit   Mode = "edit"
	ModeFix    Mode = "fix"
	ModeDesign Mode = "design"
)

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Checkpoint struct {
	Note   string    `json:"note"`
	GitSHA string    `json:"git_sha"`
	At     time.Time `json:"at"`
}

type Session struct {
	ID           string       `json:"id"`
	Repo         string       `json:"repo"`
	RepoHash     string       `json:"repo_hash"`
	TaskGoal     string       `json:"task_goal"`
	Files        []string     `json:"files"`
	Mode         Mode         `json:"mode"` // "review" | "edit" | "fix" | "design"
	Constraints  []string     `json:"constraints"`
	TargetModel  string       `json:"target_model"`
	Status       Status       `json:"status"` // "active" | "completed"
	StartedAt    time.Time    `json:"started_at"`
	LastActivity time.Time    `json:"last_activity"`
	BaseGitSHA   string       `json:"base_git_sha"`
	Checkpoints  []Checkpoint `json:"checkpoints"`
}
