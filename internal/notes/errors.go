package notes

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotAuthorized is returned when macOS has not granted this terminal
// permission to send Apple events to Notes.app (Automation/TCC).
var ErrNotAuthorized = errors.New("not authorized to control Notes.app; grant permission in System Settings → Privacy & Security → Automation")

// AppleScriptError wraps a non-zero osascript exit that isn't a recognized
// permission failure.
type AppleScriptError struct {
	ExitCode int
	Stderr   string
}

func (e *AppleScriptError) Error() string {
	return fmt.Sprintf("osascript failed (exit %d): %s", e.ExitCode, e.Stderr)
}

// classifyError inspects a failed osascript invocation and returns a typed
// error: ErrNotAuthorized for the well-known "not authorized to send Apple
// events" failure (OSA error -1743), or an *AppleScriptError otherwise.
func classifyError(runErr error, stderr string) error {
	if strings.Contains(stderr, "-1743") || strings.Contains(stderr, "Not authorized to send Apple events") {
		return ErrNotAuthorized
	}
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	return &AppleScriptError{ExitCode: exitCode, Stderr: strings.TrimSpace(stderr)}
}

// IsNotAuthorized reports whether err is (or wraps) ErrNotAuthorized.
func IsNotAuthorized(err error) bool {
	return errors.Is(err, ErrNotAuthorized)
}
