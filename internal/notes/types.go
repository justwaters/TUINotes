package notes

import "time"

// Account is an Apple Notes account, e.g. "iCloud" or "On My Mac".
type Account struct {
	ID   string
	Name string
}

// Folder is a top-level or nested Notes folder.
type Folder struct {
	ID   string
	Name string
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
