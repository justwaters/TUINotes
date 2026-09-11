package notes

import (
	"context"
	"fmt"
	"time"

	"tuinotes/internal/convert"
)

// dateLayout matches the ISO 8601 strings produced by JSON.stringify(Date)
// in JXA, e.g. "2026-09-11T02:30:13.000Z".
const dateLayout = time.RFC3339Nano

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("notes: parsing modification date %q: %w", s, err)
	}
	return t, nil
}

// wireNoteMeta mirrors the JSON shape returned by list_notes.js.
type wireNoteMeta struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ModificationDate string `json:"modificationDate"`
	AttachmentCount  int    `json:"attachmentCount"`
}

func (w wireNoteMeta) toNoteMeta() (NoteMeta, error) {
	modAt, err := parseDate(w.ModificationDate)
	if err != nil {
		return NoteMeta{}, err
	}
	return NoteMeta{
		ID:              w.ID,
		Name:            w.Name,
		ModifiedAt:      modAt,
		AttachmentCount: w.AttachmentCount,
	}, nil
}

// ListNoteMetas returns lightweight metadata (no body) for every note in a
// folder, cheapest-first so a note list can render immediately.
func (c *Client) ListNoteMetas(ctx context.Context, folderID string) ([]NoteMeta, error) {
	var wire []wireNoteMeta
	params := map[string]string{"folderId": folderID}
	if err := c.run(ctx, "list_notes.js", params, &wire); err != nil {
		return nil, err
	}
	metas := make([]NoteMeta, len(wire))
	for i, w := range wire {
		m, err := w.toNoteMeta()
		if err != nil {
			return nil, err
		}
		metas[i] = m
	}
	return metas, nil
}

// wireNote mirrors the JSON shape returned by get_note.js, create_note.js
// and update_note.js: the raw HTML body, so list markup (bullets, dashes,
// numbers) survives into the editable text via convert.HTMLToLines. Plain
// plaintext would lose it — Notes.app strips all list structure from that
// property.
type wireNote struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Body             string `json:"body"`
	ModificationDate string `json:"modificationDate"`
	AttachmentCount  int    `json:"attachmentCount"`
	FolderID         string `json:"folderId"`
	FolderName       string `json:"folderName"`
}

func (w wireNote) toNote() (Note, error) {
	modAt, err := parseDate(w.ModificationDate)
	if err != nil {
		return Note{}, err
	}
	lines, err := convert.HTMLToLines(w.Body)
	if err != nil {
		return Note{}, fmt.Errorf("notes: parsing body of %q: %w", w.Name, err)
	}
	return Note{
		NoteMeta: NoteMeta{
			ID:              w.ID,
			Name:            w.Name,
			ModifiedAt:      modAt,
			AttachmentCount: w.AttachmentCount,
		},
		PlainBody:  trimTrailingNewline(lines),
		FolderID:   w.FolderID,
		FolderName: w.FolderName,
	}, nil
}

func trimTrailingNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}

// GetNote fetches a single note's full plaintext body.
func (c *Client) GetNote(ctx context.Context, noteID string) (Note, error) {
	var w wireNote
	params := map[string]string{"noteId": noteID}
	if err := c.run(ctx, "get_note.js", params, &w); err != nil {
		return Note{}, err
	}
	return w.toNote()
}

// CreateNote creates a new note in folderID. text is the note's full
// content as the user edits it, where the first line becomes the note's
// title — this mirrors how Notes.app itself derives a note's name from the
// first line of its body. Only `body` is ever written; Notes.app derives
// `name` from it (setting `name` directly causes it to be duplicated as an
// extra leading line).
func (c *Client) CreateNote(ctx context.Context, folderID, text string) (Note, error) {
	var w wireNote
	params := map[string]string{
		"folderId": folderID,
		"body":     convert.LinesToHTML(text),
	}
	if err := c.run(ctx, "create_note.js", params, &w); err != nil {
		return Note{}, err
	}
	return w.toNote()
}

// UpdateNote overwrites an existing note's body. See CreateNote for the
// title-is-first-line convention.
func (c *Client) UpdateNote(ctx context.Context, noteID, text string) (Note, error) {
	var w wireNote
	params := map[string]string{
		"noteId": noteID,
		"body":   convert.LinesToHTML(text),
	}
	if err := c.run(ctx, "update_note.js", params, &w); err != nil {
		return Note{}, err
	}
	return w.toNote()
}

// DeleteNote moves a note to Notes.app's "Recently Deleted" folder.
func (c *Client) DeleteNote(ctx context.Context, noteID string) error {
	params := map[string]string{"noteId": noteID}
	return c.run(ctx, "delete_note.js", params, nil)
}

// MoveNote relocates a note to a different folder.
func (c *Client) MoveNote(ctx context.Context, noteID, destFolderID string) error {
	params := map[string]string{"noteId": noteID, "destFolderId": destFolderID}
	return c.run(ctx, "move_note.js", params, nil)
}

// SearchResult is one hit from SearchAll: note metadata plus the folder it
// lives in and its full plaintext, for client-side filtering.
type SearchResult struct {
	Note
}

// wireSearchResult mirrors the JSON shape returned by search_all.js. Unlike
// wireNote, this uses the plaintext property directly (list markup is
// irrelevant for a search index, and plaintext is far cheaper to fetch in
// bulk than parsing every note's HTML body).
type wireSearchResult struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Plaintext        string `json:"plaintext"`
	ModificationDate string `json:"modificationDate"`
	FolderID         string `json:"folderId"`
	FolderName       string `json:"folderName"`
}

func (w wireSearchResult) toSearchResult() (SearchResult, error) {
	modAt, err := parseDate(w.ModificationDate)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Note: Note{
		NoteMeta:   NoteMeta{ID: w.ID, Name: w.Name, ModifiedAt: modAt},
		PlainBody:  trimTrailingNewline(w.Plaintext),
		FolderID:   w.FolderID,
		FolderName: w.FolderName,
	}}, nil
}

// SearchAll fetches id/name/plaintext/modified-date/folder for every note
// in an account in one pass (one Apple Event per folder), so free-text
// search can run entirely client-side against the result.
func (c *Client) SearchAll(ctx context.Context, accountID string) ([]SearchResult, error) {
	var wire []wireSearchResult
	params := map[string]string{"accountId": accountID}
	if err := c.run(ctx, "search_all.js", params, &wire); err != nil {
		return nil, err
	}
	results := make([]SearchResult, len(wire))
	for i, w := range wire {
		r, err := w.toSearchResult()
		if err != nil {
			return nil, err
		}
		results[i] = r
	}
	return results, nil
}
