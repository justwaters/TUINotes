// Package convert reconstructs the minimal per-line HTML that Notes.app
// stores internally from the plain text a user edits in the TUI, and vice
// versa. It has no dependency on Notes.app itself.
package convert

import (
	"html"
	"strings"
)

// LinesToHTML converts multi-line plain text — where, matching Notes.app's
// own convention, the first line is the note's title — into the per-line
// <div> HTML Notes.app expects for a note body. Notes.app derives both the
// note's name and its plaintext property from this structure, so callers
// should never also set the note's name directly.
func LinesToHTML(text string) string {
	lines := strings.Split(text, "\n")
	divs := make([]string, len(lines))
	for i, line := range lines {
		if line == "" {
			divs[i] = "<div><br></div>"
			continue
		}
		divs[i] = "<div>" + html.EscapeString(line) + "</div>"
	}
	return strings.Join(divs, "\n")
}
