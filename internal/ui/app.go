// Package ui implements the Bubble Tea TUI: a navigation tree (folders,
// with their notes nested underneath) next to a note content pane, plus a
// full-text search overlay.
package ui

import (
	"fmt"
	"strings"

	"github.com/justwaters/TUINotes/internal/notes"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type focus int

const (
	focusNav focus = iota
	focusEditor
	focusSearch
)

// Model is the root Bubble Tea model.
type Model struct {
	client *notes.Client

	focus focus

	accounts []notes.Account
	folders  []folderWithAccount // tree order (see orderFoldersAsTree), Depth set

	navList list.Model
	editor  textarea.Model

	expandedFolders  map[string]bool
	selectedFolderID string
	noteMetaCache    map[string][]notes.NoteMeta

	currentNote notes.Note
	haveNote    bool
	editing     bool
	dirty       bool

	confirmingDelete      bool
	pendingDeleteID       string
	pendingDeleteFolderID string

	searchList     list.Model
	haveSearchList bool

	err    error
	status string

	width, height int
	// contentH is the content height (excluding the pane's own border and
	// title row) that navList/editor were sized to. renderPane clips/pads
	// to exactly this many lines so the two side-by-side panes' borders
	// stay aligned even if a component's rendered line count doesn't
	// exactly match the height it was asked for (e.g. a note with long
	// wrapped lines).
	contentH int
}

// New builds the initial Model. Loading real data happens in Init/Update.
func New(client *notes.Client) Model {
	navDelegate := list.NewDefaultDelegate()
	navDelegate.ShowDescription = false
	navDelegate.SetSpacing(0)
	navList := list.New(nil, navDelegate, 20, 10)
	navList.SetShowTitle(false)
	navList.SetShowStatusBar(false)
	navList.SetShowHelp(false)

	ta := textarea.New()
	ta.Placeholder = "Select a note to view it here."
	ta.ShowLineNumbers = false

	return Model{
		client:          client,
		focus:           focusNav,
		navList:         navList,
		editor:          ta,
		expandedFolders: map[string]bool{},
		noteMetaCache:   map[string][]notes.NoteMeta{},
	}
}

func (m Model) Init() tea.Cmd {
	return loadAccountsCmd(m.client)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil
	case accountsLoadedMsg:
		return m.handleAccountsLoaded(msg)
	case notesLoadedMsg:
		return m.handleNotesLoaded(msg)
	case noteLoadedMsg:
		return m.handleNoteLoaded(msg)
	case noteSavedMsg:
		return m.handleNoteSaved(msg)
	case noteDeletedMsg:
		return m.handleNoteDeleted(msg)
	case searchResultsMsg:
		return m.handleSearchResults(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Anything else (list filter-match results, cursor blink ticks, etc.)
	// belongs to whichever component actually owns it. Fan it out to all of
	// them; each ignores messages it didn't originate.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	cmds = append(cmds, cmd)
	m.navList, cmd = m.navList.Update(msg)
	cmds = append(cmds, cmd)
	if m.haveSearchList {
		m.searchList, cmd = m.searchList.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	if m.confirmingDelete {
		if key.Matches(msg, keyConfirm) {
			id, folderID := m.pendingDeleteID, m.pendingDeleteFolderID
			m.confirmingDelete = false
			m.status = "Deleting..."
			return m, deleteNoteCmd(m.client, id, folderID)
		}
		m.confirmingDelete = false
		m.status = ""
		return m, nil
	}

	if m.focus == focusEditor && m.editing {
		switch {
		case key.Matches(msg, keyCancel):
			m.editing = false
			m.editor.Blur()
			m.editor.SetValue(m.currentNote.PlainBody)
			m.dirty = false
			m.status = "Discarded changes"
			return m, nil
		case key.Matches(msg, keySave):
			text := m.editor.Value()
			if strings.TrimSpace(text) == "" {
				m.status = "Cannot save an empty note"
				return m, nil
			}
			m.status = "Saving..."
			if m.haveNote {
				return m, updateNoteCmd(m.client, m.currentNote.ID, m.currentNote.FolderID, text)
			}
			return m, createNoteCmd(m.client, m.newNoteFolderID(), text)
		default:
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			m.dirty = true
			return m, cmd
		}
	}

	// Viewing (not editing) a note: the textarea ignores input while
	// blurred, so scroll it explicitly instead of leaving the pane inert.
	if m.focus == focusEditor && !m.editing && m.haveNote {
		switch msg.String() {
		case "up", "k":
			m.editor.CursorUp()
			return m, nil
		case "down", "j":
			m.editor.CursorDown()
			return m, nil
		case "pgup":
			m.editor.PageUp()
			return m, nil
		case "pgdown":
			m.editor.PageDown()
			return m, nil
		case "home", "g":
			m.editor.MoveToBegin()
			return m, nil
		case "end", "G":
			m.editor.MoveToEnd()
			return m, nil
		}
	}

	if m.focus == focusSearch {
		if m.searchList.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.searchList, cmd = m.searchList.Update(msg)
			return m, cmd
		}
		switch {
		case key.Matches(msg, keyCancel):
			m.focus = focusNav
			return m, nil
		case key.Matches(msg, keyEnter):
			if it, ok := m.searchList.SelectedItem().(searchItem); ok {
				return m.openSearchResult(it.result)
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.searchList, cmd = m.searchList.Update(msg)
			return m, cmd
		}
	}

	if m.focus == focusNav && m.navList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.navList, cmd = m.navList.Update(msg)
		return m, cmd
	}

	// We're in plain browsing mode (not editing, not confirming a delete,
	// not searching). Clear any leftover toast (e.g. "Saved") so the
	// keybinding hint reappears; whatever this key does will set its own
	// status if it needs to.
	m.status = ""

	switch {
	case key.Matches(msg, keyQuit):
		return m, tea.Quit
	case key.Matches(msg, keyTab):
		m.cycleFocus(1)
		return m, nil
	case key.Matches(msg, keyShiftTab):
		m.cycleFocus(-1)
		return m, nil
	case key.Matches(msg, keySearch):
		if len(m.accounts) == 0 {
			return m, nil
		}
		m.status = "Searching all notes..."
		return m, searchAllCmd(m.client, m.accounts[0].ID)
	case key.Matches(msg, keyRefresh):
		return m.refresh()
	case key.Matches(msg, keyNew):
		folderID := m.newNoteFolderID()
		if folderID == "" {
			m.status = "Select a folder first"
			return m, nil
		}
		m.selectedFolderID = folderID
		m.haveNote = false
		m.currentNote = notes.Note{}
		m.editor.SetValue("")
		m.editor.Focus()
		m.editing = true
		m.dirty = false
		m.focus = focusEditor
		m.status = "New note — ctrl+s to save, esc to cancel"
		return m, nil
	case key.Matches(msg, keyEdit):
		if !m.haveNote {
			m.status = "Open a note first (enter)"
			return m, nil
		}
		if m.currentNote.HasAttachments() {
			m.status = "This note has attachments; editing is disabled to avoid losing them"
			return m, nil
		}
		m.editor.Focus()
		m.editing = true
		m.focus = focusEditor
		m.status = "Editing — ctrl+s to save, esc to cancel"
		return m, nil
	case key.Matches(msg, keyDelete):
		if it, ok := m.navList.SelectedItem().(noteNavItem); ok {
			m.confirmingDelete = true
			m.pendingDeleteID = it.meta.ID
			m.pendingDeleteFolderID = it.FolderID
			m.status = fmt.Sprintf("Delete %q? (y/n)", it.meta.Name)
		}
		return m, nil
	case key.Matches(msg, keyEnter):
		return m.handleEnter()
	default:
		if m.focus == focusNav {
			var cmd tea.Cmd
			m.navList, cmd = m.navList.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

// newNoteFolderID returns the folder a new note (or an in-progress new
// note's save) should go into: the selected folder, or the parent of the
// selected note.
func (m Model) newNoteFolderID() string {
	switch it := m.navList.SelectedItem().(type) {
	case folderNavItem:
		return it.ID
	case noteNavItem:
		return it.FolderID
	}
	return m.selectedFolderID
}

func (m *Model) cycleFocus(dir int) {
	order := []focus{focusNav, focusEditor}
	cur := 0
	for i, f := range order {
		if f == m.focus {
			cur = i
		}
	}
	m.focus = order[(cur+dir+len(order))%len(order)]
}

// handleEnter toggles a folder's expansion (lazily loading its notes) or
// opens a note.
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch it := m.navList.SelectedItem().(type) {
	case folderNavItem:
		m.selectedFolderID = it.ID
		expanding := !m.expandedFolders[it.ID]
		m.expandedFolders[it.ID] = expanding
		if expanding {
			if _, ok := m.noteMetaCache[it.ID]; !ok {
				cmd := m.rebuildNavItems()
				m.status = "Loading notes..."
				return m, tea.Batch(cmd, loadNotesCmd(m.client, it.ID))
			}
		}
		return m, m.rebuildNavItems()
	case noteNavItem:
		m.selectedFolderID = it.FolderID
		m.status = "Loading note..."
		return m, loadNoteCmd(m.client, it.meta.ID)
	}
	return m, nil
}

func (m Model) openSearchResult(r notes.SearchResult) (tea.Model, tea.Cmd) {
	m.selectedFolderID = r.FolderID
	m.expandFolderAndAncestors(r.FolderID)
	m.focus = focusNav
	m.status = "Loading note..."

	var cmds []tea.Cmd
	if _, ok := m.noteMetaCache[r.FolderID]; !ok {
		cmds = append(cmds, loadNotesCmd(m.client, r.FolderID))
	}
	cmds = append(cmds, m.rebuildNavItems())
	for i, item := range m.navList.Items() {
		if fi, ok := item.(folderNavItem); ok && fi.ID == r.FolderID {
			m.navList.Select(i)
			break
		}
	}
	cmds = append(cmds, loadNoteCmd(m.client, r.ID))
	return m, tea.Batch(cmds...)
}

// expandFolderAndAncestors marks folderID and every folder above it in the
// tree as expanded, so a folder reached via search (which may be nested
// several levels deep) is actually visible in the collapsed-by-default tree.
func (m *Model) expandFolderAndAncestors(folderID string) {
	byID := make(map[string]folderWithAccount, len(m.folders))
	for _, f := range m.folders {
		byID[f.ID] = f
	}
	for id := folderID; id != ""; {
		m.expandedFolders[id] = true
		f, ok := byID[id]
		if !ok {
			return
		}
		if _, parentIsFolder := byID[f.ParentID]; !parentIsFolder {
			return // parent is the account itself; id was a top-level folder
		}
		id = f.ParentID
	}
}

// rebuildNavItems recomputes the flattened tree shown in navList. m.folders
// is already in parent-before-children order (orderFoldersAsTree), so a
// collapsed folder's entire subtree — nested folders and notes alike — is
// skipped by tracking the depth we're currently hiding below.
//
// list.Model.SetItems returns a tea.Cmd that recomputes the active filter
// against the new items; callers must return it (or batch it with their
// own Cmd). Dropping it leaves a stale, usually-empty filtered view — the
// list looks "stuck" showing no items until the filter is cleared.
func (m *Model) rebuildNavItems() tea.Cmd {
	var items []list.Item
	skipBelowDepth := -1
	for _, f := range m.folders {
		if skipBelowDepth != -1 && f.Depth > skipBelowDepth {
			continue // inside a collapsed ancestor
		}
		skipBelowDepth = -1

		expanded := m.expandedFolders[f.ID]
		items = append(items, folderNavItem{Folder: f.Folder, Depth: f.Depth, Expanded: expanded})
		if expanded {
			for _, meta := range m.noteMetaCache[f.ID] {
				items = append(items, noteNavItem{meta: meta, FolderID: f.ID, Depth: f.Depth + 1})
			}
		} else {
			skipBelowDepth = f.Depth
		}
	}
	return m.navList.SetItems(items)
}

func (m Model) handleAccountsLoaded(msg accountsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	m.accounts = msg.accounts
	m.folders = orderFoldersAsTree(msg.folders)
	return m, m.rebuildNavItems()
}

func (m Model) handleNotesLoaded(msg notesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Failed to load notes: " + msg.err.Error()
		return m, nil
	}
	m.noteMetaCache[msg.folderID] = msg.metas
	if m.expandedFolders[msg.folderID] {
		m.status = ""
		return m, m.rebuildNavItems()
	}
	return m, nil
}

func (m Model) handleNoteLoaded(msg noteLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Failed to load note: " + msg.err.Error()
		return m, nil
	}
	m.currentNote = msg.note
	m.haveNote = true
	m.editor.SetValue(msg.note.PlainBody)
	m.editor.Blur()
	m.editing = false
	m.dirty = false
	m.focus = focusEditor
	m.status = ""
	return m, nil
}

func (m Model) handleNoteSaved(msg noteSavedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Failed to save: " + msg.err.Error()
		return m, nil
	}
	metas := m.noteMetaCache[msg.folderID]
	found := false
	for i, meta := range metas {
		if meta.ID == msg.note.ID {
			metas[i] = msg.note.NoteMeta
			found = true
			break
		}
	}
	if !found {
		metas = append([]notes.NoteMeta{msg.note.NoteMeta}, metas...)
	}
	m.noteMetaCache[msg.folderID] = metas
	var cmd tea.Cmd
	if m.expandedFolders[msg.folderID] {
		cmd = m.rebuildNavItems()
	}
	m.currentNote = msg.note
	m.haveNote = true
	m.editor.SetValue(msg.note.PlainBody)
	m.editor.Blur()
	m.editing = false
	m.dirty = false
	m.focus = focusEditor
	m.status = "Saved"
	return m, cmd
}

func (m Model) handleNoteDeleted(msg noteDeletedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Failed to delete: " + msg.err.Error()
		return m, nil
	}
	metas := m.noteMetaCache[msg.folderID]
	kept := make([]notes.NoteMeta, 0, len(metas))
	for _, meta := range metas {
		if meta.ID != msg.noteID {
			kept = append(kept, meta)
		}
	}
	m.noteMetaCache[msg.folderID] = kept
	var cmd tea.Cmd
	if m.expandedFolders[msg.folderID] {
		cmd = m.rebuildNavItems()
	}
	if m.haveNote && m.currentNote.ID == msg.noteID {
		m.haveNote = false
		m.currentNote = notes.Note{}
		m.editor.SetValue("")
	}
	m.status = "Deleted"
	return m, cmd
}

func (m Model) handleSearchResults(msg searchResultsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Search failed: " + msg.err.Error()
		return m, nil
	}
	items := make([]list.Item, len(msg.results))
	for i, r := range msg.results {
		items[i] = searchItem{result: r}
	}
	m.searchList = list.New(items, list.NewDefaultDelegate(), m.width, m.height-3)
	m.searchList.Title = "Search all notes (/ to filter, enter to open, esc to close)"
	m.searchList.SetShowStatusBar(false)
	m.searchList.SetShowHelp(false)
	m.haveSearchList = true
	m.focus = focusSearch
	m.status = ""
	return m, nil
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{loadAccountsCmd(m.client)}
	for folderID, expanded := range m.expandedFolders {
		if expanded {
			cmds = append(cmds, loadNotesCmd(m.client, folderID))
		}
	}
	m.status = "Refreshing..."
	return m, tea.Batch(cmds...)
}

func (m *Model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	const chromePerPane = 4                     // 2 border cols + 2 padding cols
	innerH := m.height - 2 /* status bar */ - 2 /* pane border rows */
	if innerH < 3 {
		innerH = 3
	}
	m.contentH = innerH
	remaining := m.width - chromePerPane*2
	if remaining < 30 {
		remaining = 30
	}
	navW := remaining * 32 / 100
	if navW < 24 {
		navW = 24
	}
	editorW := remaining - navW
	if editorW < 20 {
		editorW = 20
	}
	m.navList.SetSize(navW, innerH)
	m.editor.SetWidth(editorW)
	m.editor.SetHeight(innerH)
	if m.haveSearchList {
		m.searchList.SetSize(m.width-2, m.height-3)
	}
}

func (m Model) View() tea.View {
	if m.err != nil {
		v := tea.NewView(errStyle.Render("Error: "+m.err.Error()) + "\n\nPress q to quit.")
		v.AltScreen = true
		return v
	}
	if m.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	if m.focus == focusSearch {
		v := tea.NewView(m.searchList.View() + "\n" + m.statusBar())
		v.AltScreen = true
		return v
	}

	navPane := m.renderPane("Navigation", m.navList.View(), m.focus == focusNav)

	editorContent := m.editor.View()
	if !m.haveNote && !m.editing {
		editorContent = statusStyle.Render("Select a note (enter) or create one (n).")
	}
	editorLabel := "Note"
	if m.editing {
		editorLabel = "Editing"
	}
	editorPane := m.renderPane(editorLabel, editorContent, m.focus == focusEditor)

	row := lipgloss.JoinHorizontal(lipgloss.Top, navPane, editorPane)
	v := tea.NewView(row + "\n" + m.statusBar())
	v.AltScreen = true
	return v
}

func (m Model) renderPane(title, content string, focused bool) string {
	style := paneStyle
	if focused {
		style = paneStyleFocused
	}
	return style.Render(titleStyle.Render(title) + "\n" + fitHeight(content, m.contentH))
}

// fitHeight clips or blank-pads content to exactly n lines.
func fitHeight(content string, n int) string {
	if n < 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) statusBar() string {
	if m.status != "" {
		if m.confirmingDelete {
			return errStyle.Render(m.status)
		}
		return statusStyle.Render(m.status)
	}
	return statusStyle.Render("tab: switch pane · enter: open/expand · n: new · e: edit · ctrl+s: save · d: delete · S: search · r: refresh · q: quit")
}
