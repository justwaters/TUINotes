package ui

import (
	"fmt"
	"time"

	"tuinotes/internal/notes"
)

// folderItem adapts notes.Folder to list.Item / list.DefaultItem.
type folderItem struct {
	notes.Folder
}

func (f folderItem) FilterValue() string { return f.Name }
func (f folderItem) Title() string       { return f.Name }
func (f folderItem) Description() string { return "" }

// noteItem adapts notes.NoteMeta to list.Item / list.DefaultItem.
type noteItem struct {
	meta notes.NoteMeta
}

func (n noteItem) FilterValue() string { return n.meta.Name }

func (n noteItem) Title() string {
	if n.meta.HasAttachments() {
		return n.meta.Name + " 📎"
	}
	return n.meta.Name
}

func (n noteItem) Description() string {
	return timeAgo(n.meta.ModifiedAt)
}

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
