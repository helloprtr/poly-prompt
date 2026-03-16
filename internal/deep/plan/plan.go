package plan

// TodoStatus tracks the lifecycle of a single todo item.
type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
	TodoFailed     TodoStatus = "failed"
	TodoSkipped    TodoStatus = "skipped"
)

// TodoItem is one unit of work inside a WorkPlan, executed by a named worker.
type TodoItem struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Worker       string     `json:"worker"`
	Status       TodoStatus `json:"status"`
	DependsOn    []string   `json:"depends_on,omitempty"`
	InputRefs    []string   `json:"input_refs,omitempty"`
	OutputRefs   []string   `json:"output_refs,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// WorkerSpec declares how a named worker should be invoked.
type WorkerSpec struct {
	Name             string   `json:"name"`
	Objective        string   `json:"objective"`
	Inputs           []string `json:"inputs,omitempty"`
	AllowedArtifacts []string `json:"allowed_artifacts,omitempty"`
	OutputType       string   `json:"output_type"`
	DependsOn        []string `json:"depends_on,omitempty"`
	Required         bool     `json:"required"`
}

// WorkPlan is the static execution graph produced during the planning phase.
type WorkPlan struct {
	Version      int          `json:"version"`
	Action       string       `json:"action"`
	ResultType   string       `json:"result_type"`
	Summary      string       `json:"summary"`
	EvidenceRefs []string     `json:"evidence_refs,omitempty"`
	Todos        []TodoItem   `json:"todos"`
	WorkerGraph  []WorkerSpec `json:"worker_graph"`
}
