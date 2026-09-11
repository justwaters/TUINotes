package ui

import "tuinotes/internal/notes"

type accountsLoadedMsg struct {
	accounts []notes.Account
	folders  []folderWithAccount
	err      error
}

type folderWithAccount struct {
	notes.Folder
	AccountID string
	// Depth is the folder's nesting level (0 = top-level). Set by
	// orderFoldersAsTree, not by the load command.
	Depth int
}

type notesLoadedMsg struct {
	folderID string
	metas    []notes.NoteMeta
	err      error
}

type noteLoadedMsg struct {
	note notes.Note
	err  error
}

type noteSavedMsg struct {
	note     notes.Note
	isNew    bool
	folderID string
	err      error
}

type noteDeletedMsg struct {
	noteID   string
	folderID string
	err      error
}

type searchResultsMsg struct {
	results []notes.SearchResult
	err     error
}
