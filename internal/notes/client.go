// Package notes talks to Apple Notes by shelling out to `osascript -l
// JavaScript` (JXA). There is no public API or stable file format for Notes
// data, so automation is the only safe way to read and write real notes.
package notes

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os/exec"
)

//go:embed scripts/*.js
var scriptsFS embed.FS

// Client runs JXA scripts against Notes.app.
type Client struct{}

// NewClient returns a ready-to-use Client. Construction never touches
// Notes.app; the first real call is what may trigger the macOS Automation
// permission prompt.
func NewClient() *Client {
	return &Client{}
}

// run executes the named embedded script (e.g. "list_folders.js"), passing
// params JSON-encoded as its single argv entry, and decodes the script's
// JSON stdout into out. If out is nil, the script's output is discarded.
func (c *Client) run(ctx context.Context, script string, params any, out any) error {
	src, err := scriptsFS.ReadFile("scripts/" + script)
	if err != nil {
		return fmt.Errorf("notes: missing embedded script %s: %w", script, err)
	}

	argJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("notes: encoding params for %s: %w", script, err)
	}

	cmd := exec.CommandContext(ctx, "osascript", "-l", "JavaScript", "-", string(argJSON))
	cmd.Stdin = bytes.NewReader(src)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return classifyError(err, stderr.String())
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(stdout.Bytes(), out); err != nil {
		return fmt.Errorf("notes: decoding output of %s: %w", script, err)
	}
	return nil
}
