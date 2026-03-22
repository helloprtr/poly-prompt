// Package approval will contain the human-in-the-loop approval gate.
//
// Implement this when prtr gains the ability to take irreversible local actions, such as:
//   - applying patch.diff to the working tree directly
//   - creating git commits or pushing branches
//   - executing shell commands on behalf of the user
//
// Until then, the delivery step (clipboard copy + AI app launch) serves as the
// natural approval gate — the user decides what the AI does with the prompt.
//
// Planned gate position: after planner completes (plan.json available), before
// patcher runs. Events [approval.requested, approval.granted] are already defined
// in internal/deep/event/event.go.
package approval
