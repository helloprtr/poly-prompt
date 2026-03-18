// internal/watcher/event.go
package watcher

import "regexp"

// DetectEvent returns suggested prtr action ("fix", "debug") or "" for no action.
func DetectEvent(exitCode int, output string) string {
	if isGitConflict(output) {
		return "fix"
	}
	if exitCode == 0 {
		return ""
	}
	if isPanic(output) {
		return "debug"
	}
	if isTestFailure(output) || isBuildError(output) {
		return "fix"
	}
	return ""
}

var (
	reTestFail = regexp.MustCompile(`(?i)(FAIL |✕ |failed|Error:)`)
	reBuildErr = regexp.MustCompile(`(?i)(error\[E|cannot find|undefined:)`)
	rePanic    = regexp.MustCompile(`(panic:|Segmentation fault|SIGSEGV)`)
	reConflict = regexp.MustCompile(`(CONFLICT |Automatic merge failed)`)
)

func isTestFailure(output string) bool { return reTestFail.MatchString(output) }
func isBuildError(output string) bool  { return reBuildErr.MatchString(output) }
func isPanic(output string) bool       { return rePanic.MatchString(output) }
func isGitConflict(output string) bool { return reConflict.MatchString(output) }
