package notes

import "time"

// Account is an Apple Notes account, e.g. "iCloud" or "On My Mac".
type Account struct {
	ID   string
	Name string
}

// Folder is a top-level or nested Notes folder. ParentID is either the
// owning Account's ID (a top-level folder) or another Folder's ID (a
// nested folder) — Notes.app returns every folder in an account flattened,
// so this is the only way to reconstruct the real folder tree.
type Folder struct {
	ID       string
	Name     string
	ParentID string
}

// NoteMeta is the lightweight metadata fetched when listing a folder's
// notes: enough to render a list without paying for a full body fetch.
type NoteMeta struct {
	ID              string
	Name            string
	ModifiedAt      time.Time
	AttachmentCount int
}

// HasAttachments reports whether saving over this note's body would risk
// destroying non-text content, since TUINotes only ever writes plain
// per-line HTML and carries no attachment markup.
func (n NoteMeta) HasAttachments() bool {
	return n.AttachmentCount > 0
}

// Note is a fully loaded note, including its plaintext body.
type Note struct {
	NoteMeta
	PlainBody  string
	FolderID   string
	FolderName string
}
