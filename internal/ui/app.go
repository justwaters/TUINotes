// Package ui implements the Bubble Tea TUI: a three-pane Notes.app-style
// layout (folders | notes | note content) plus a full-text search overlay.
package ui

import (
	"fmt"
	"strings"

	"tuinotes/internal/notes"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type focus int

const (
	focusFolders focus = iota
	focusNotes
	focusEditor
	focusSearch
)

// Model is the root Bubble Tea model.
type Model struct {
	client *notes.Client

	focus focus

	accounts []notes.Account
	folders  []folderWithAccount

	folderList list.Model
	noteList   list.Model
	editor     textarea.Model

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
}

// New builds the initial Model. Loading real data happens in Init/Update.
func New(client *notes.Client) Model {
	folderList := list.New(nil, list.NewDefaultDelegate(), 20, 10)
	folderList.SetShowTitle(false)
	folderList.SetShowStatusBar(false)
	folderList.SetShowHelp(false)

	noteList := list.New(nil, list.NewDefaultDelegate(), 20, 10)
	noteList.SetShowTitle(false)
	noteList.SetShowStatusBar(false)
	noteList.SetShowHelp(false)

	ta := textarea.New()
	ta.Placeholder = "Select a note to view it here."
	ta.ShowLineNumbers = false

	return Model{
		client:        client,
		focus:         focusFolders,
		folderList:    folderList,
		noteList:      noteList,
		editor:        ta,
		noteMetaCache: map[string][]notes.NoteMeta{},
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
	m.folderList, cmd = m.folderList.Update(msg)
	cmds = append(cmds, cmd)
	m.noteList, cmd = m.noteList.Update(msg)
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
			return m, createNoteCmd(m.client, m.selectedFolderID, text)
		default:
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			m.dirty = true
			return m, cmd
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
			m.focus = focusNotes
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

	if m.focusedListFilterState() == list.Filtering {
		return m.updateFocusedList(msg)
	}

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
		if m.selectedFolderID == "" {
			m.status = "Select a folder first"
			return m, nil
		}
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
		if m.focus == focusNotes {
			if it, ok := m.noteList.SelectedItem().(noteItem); ok {
				m.confirmingDelete = true
				m.pendingDeleteID = it.meta.ID
				m.pendingDeleteFolderID = m.selectedFolderID
				m.status = fmt.Sprintf("Delete %q? (y/n)", it.meta.Name)
			}
		}
		return m, nil
	case key.Matches(msg, keyEnter):
		return m.handleEnter()
	default:
		return m.updateFocusedList(msg)
	}
}

func (m Model) focusedListFilterState() list.FilterState {
	switch m.focus {
	case focusFolders:
		return m.folderList.FilterState()
	case focusNotes:
		return m.noteList.FilterState()
	}
	return list.Unfiltered
}

func (m Model) updateFocusedList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusFolders:
		m.folderList, cmd = m.folderList.Update(msg)
	case focusNotes:
		m.noteList, cmd = m.noteList.Update(msg)
	}
	return m, cmd
}

func (m *Model) cycleFocus(dir int) {
	order := []focus{focusFolders, focusNotes, focusEditor}
	cur := 0
	for i, f := range order {
		if f == m.focus {
			cur = i
		}
	}
	m.focus = order[(cur+dir+len(order))%len(order)]
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFolders:
		if it, ok := m.folderList.SelectedItem().(folderItem); ok {
			m.selectedFolderID = it.ID
			m.focus = focusNotes
			if metas, ok := m.noteMetaCache[it.ID]; ok {
				m.setNoteListItems(metas)
				return m, nil
			}
			m.status = "Loading notes..."
			return m, loadNotesCmd(m.client, it.ID)
		}
	case focusNotes:
		if it, ok := m.noteList.SelectedItem().(noteItem); ok {
			m.status = "Loading note..."
			return m, loadNoteCmd(m.client, it.meta.ID)
		}
	}
	return m, nil
}

func (m Model) openSearchResult(r notes.SearchResult) (tea.Model, tea.Cmd) {
	m.selectedFolderID = r.FolderID
	for i, f := range m.folders {
		if f.ID == r.FolderID {
			m.folderList.Select(i)
			break
		}
	}
	m.focus = focusNotes
	m.status = "Loading note..."
	var cmds []tea.Cmd
	if metas, ok := m.noteMetaCache[r.FolderID]; ok {
		m.setNoteListItems(metas)
	} else {
		cmds = append(cmds, loadNotesCmd(m.client, r.FolderID))
	}
	cmds = append(cmds, loadNoteCmd(m.client, r.ID))
	return m, tea.Batch(cmds...)
}

func (m *Model) setNoteListItems(metas []notes.NoteMeta) {
	items := make([]list.Item, len(metas))
	for i, meta := range metas {
		items[i] = noteItem{meta: meta}
	}
	m.noteList.SetItems(items)
}

func (m Model) handleAccountsLoaded(msg accountsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	m.accounts = msg.accounts
	m.folders = msg.folders
	items := make([]list.Item, len(msg.folders))
	for i, f := range msg.folders {
		items[i] = folderItem{Folder: f.Folder}
	}
	m.folderList.SetItems(items)
	if len(msg.folders) > 0 && m.selectedFolderID == "" {
		m.selectedFolderID = msg.folders[0].ID
		return m, loadNotesCmd(m.client, msg.folders[0].ID)
	}
	return m, nil
}

func (m Model) handleNotesLoaded(msg notesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Failed to load notes: " + msg.err.Error()
		return m, nil
	}
	m.noteMetaCache[msg.folderID] = msg.metas
	if msg.folderID == m.selectedFolderID {
		m.setNoteListItems(msg.metas)
		m.status = ""
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
	if msg.folderID == m.selectedFolderID {
		m.setNoteListItems(metas)
	}
	m.currentNote = msg.note
	m.haveNote = true
	m.editor.SetValue(msg.note.PlainBody)
	m.editor.Blur()
	m.editing = false
	m.dirty = false
	m.focus = focusEditor
	m.status = "Saved"
	return m, nil
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
	if msg.folderID == m.selectedFolderID {
		m.setNoteListItems(kept)
	}
	if m.haveNote && m.currentNote.ID == msg.noteID {
		m.haveNote = false
		m.currentNote = notes.Note{}
		m.editor.SetValue("")
	}
	m.status = "Deleted"
	return m, nil
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
	if m.focus == focusFolders {
		m.status = "Refreshing folders..."
		return m, loadAccountsCmd(m.client)
	}
	if m.selectedFolderID == "" {
		return m, nil
	}
	m.status = "Refreshing notes..."
	return m, loadNotesCmd(m.client, m.selectedFolderID)
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
	remaining := m.width - chromePerPane*3
	if remaining < 30 {
		remaining = 30
	}
	folderW := remaining * 22 / 100
	if folderW < 16 {
		folderW = 16
	}
	noteW := remaining * 30 / 100
	if noteW < 20 {
		noteW = 20
	}
	editorW := remaining - folderW - noteW
	if editorW < 20 {
		editorW = 20
	}
	m.folderList.SetSize(folderW, innerH)
	m.noteList.SetSize(noteW, innerH)
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

	folderPane := m.renderPane("Folders", m.folderList.View(), m.focus == focusFolders)
	notePane := m.renderPane("Notes", m.noteList.View(), m.focus == focusNotes)

	editorContent := m.editor.View()
	if !m.haveNote && !m.editing {
		editorContent = statusStyle.Render("Select a note (enter) or create one (n).")
	}
	editorLabel := "Note"
	if m.editing {
		editorLabel = "Editing"
	}
	editorPane := m.renderPane(editorLabel, editorContent, m.focus == focusEditor)

	row := lipgloss.JoinHorizontal(lipgloss.Top, folderPane, notePane, editorPane)
	v := tea.NewView(row + "\n" + m.statusBar())
	v.AltScreen = true
	return v
}

func (m Model) renderPane(title, content string, focused bool) string {
	style := paneStyle
	if focused {
		style = paneStyleFocused
	}
	return style.Render(titleStyle.Render(title) + "\n" + content)
}

func (m Model) statusBar() string {
	if m.status != "" {
		if m.confirmingDelete {
			return errStyle.Render(m.status)
		}
		return statusStyle.Render(m.status)
	}
	return statusStyle.Render("tab: switch pane · enter: open · n: new · e: edit · ctrl+s: save · d: delete · S: search · r: refresh · q: quit")
}
