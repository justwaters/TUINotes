package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/justwaters/TUINotes/internal/notes"
)

// folderNavItem is a folder row in the navigation tree. Depth is the
// folder's nesting level (0 = top-level); Expanded controls the
// disclosure marker and whether its notes are shown beneath it.
type folderNavItem struct {
	notes.Folder
	Depth    int
	Expanded bool
}

func (f folderNavItem) FilterValue() string { return f.Name }

func (f folderNavItem) Title() string {
	marker := "▸"
	if f.Expanded {
		marker = "▾"
	}
	return strings.Repeat("  ", f.Depth) + marker + " " + f.Name
}

func (f folderNavItem) Description() string { return "" }

// noteNavItem is a note row nested beneath its (expanded) folder in the
// navigation tree. Depth is the note's own indentation level, i.e. one
// deeper than its parent folder's Depth.
type noteNavItem struct {
	meta     notes.NoteMeta
	FolderID string
	Depth    int
}

func (n noteNavItem) FilterValue() string { return n.meta.Name }

func (n noteNavItem) Title() string {
	name := n.meta.Name
	if n.meta.HasAttachments() {
		name += " 📎"
	}
	return strings.Repeat("  ", n.Depth) + "  " + name + "  " + timeAgo(n.meta.ModifiedAt)
}

func (n noteNavItem) Description() string { return "" }

// searchItem adapts a notes.SearchResult to list.Item / list.DefaultItem for
// the global search overlay.
type searchItem struct {
	result notes.SearchResult
}

func (s searchItem) FilterValue() string { return s.result.Name + " " + s.result.PlainBody }
func (s searchItem) Title() string       { return s.result.Name }
func (s searchItem) Description() string {
	return fmt.Sprintf("%s · %s", s.result.FolderName, timeAgo(s.result.ModifiedAt))
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}
