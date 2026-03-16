package schema

// PatchDraft is the raw output from the patcher worker before reconciliation.
type PatchDraft struct {
	Summary             string   `json:"summary"`
	TouchedFiles        []string `json:"touched_files,omitempty"`
	ImplementationNotes []string `json:"implementation_notes,omitempty"`
	Constraints         []string `json:"constraints,omitempty"`
	Diff                string   `json:"diff"`
}

// RiskItem describes a single risk identified by the critic worker.
type RiskItem struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Detail     string `json:"detail"`
	Confidence string `json:"confidence,omitempty"`
}

// RiskReport is the full output from the critic worker.
type RiskReport struct {
	TopRisks      []RiskItem `json:"top_risks,omitempty"`
	MissingChecks []string   `json:"missing_checks,omitempty"`
	OverallRisk   string     `json:"overall_risk,omitempty"`
}

// TestPlan is the output from the tester worker.
type TestPlan struct {
	TestCases         []string `json:"test_cases,omitempty"`
	EdgeCases         []string `json:"edge_cases,omitempty"`
	VerificationSteps []string `json:"verification_steps,omitempty"`
}

// PatchBundle is the final reconciled artifact produced by a deep patch run.
type PatchBundle struct {
	Summary       string     `json:"summary"`
	Diff          string     `json:"diff"`
	TouchedFiles  []string   `json:"touched_files,omitempty"`
	Risks         []RiskItem `json:"risks,omitempty"`
	TestPlan      TestPlan   `json:"test_plan"`
	OpenQuestions []string   `json:"open_questions,omitempty"`
	Warnings      []string   `json:"warnings,omitempty"`
}
