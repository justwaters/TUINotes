package ui

import (
	"context"

	"tuinotes/internal/notes"

	tea "charm.land/bubbletea/v2"
)

func loadAccountsCmd(client *notes.Client) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		accounts, err := client.ListAccounts(ctx)
		if err != nil {
			return accountsLoadedMsg{err: err}
		}
		var folders []folderWithAccount
		for _, a := range accounts {
			fs, err := client.ListFolders(ctx, a.ID)
			if err != nil {
				return accountsLoadedMsg{err: err}
			}
			for _, f := range fs {
				folders = append(folders, folderWithAccount{Folder: f, AccountID: a.ID})
			}
		}
		return accountsLoadedMsg{accounts: accounts, folders: folders}
	}
}

func loadNotesCmd(client *notes.Client, folderID string) tea.Cmd {
	return func() tea.Msg {
		metas, err := client.ListNoteMetas(context.Background(), folderID)
		return notesLoadedMsg{folderID: folderID, metas: metas, err: err}
	}
}

func loadNoteCmd(client *notes.Client, noteID string) tea.Cmd {
	return func() tea.Msg {
		note, err := client.GetNote(context.Background(), noteID)
		return noteLoadedMsg{note: note, err: err}
	}
}

func createNoteCmd(client *notes.Client, folderID, text string) tea.Cmd {
	return func() tea.Msg {
		note, err := client.CreateNote(context.Background(), folderID, text)
		return noteSavedMsg{note: note, isNew: true, folderID: folderID, err: err}
	}
}

func updateNoteCmd(client *notes.Client, noteID, folderID, text string) tea.Cmd {
	return func() tea.Msg {
		note, err := client.UpdateNote(context.Background(), noteID, text)
		return noteSavedMsg{note: note, isNew: false, folderID: folderID, err: err}
	}
}

func deleteNoteCmd(client *notes.Client, noteID, folderID string) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteNote(context.Background(), noteID)
		return noteDeletedMsg{noteID: noteID, folderID: folderID, err: err}
	}
}

func searchAllCmd(client *notes.Client, accountID string) tea.Cmd {
	return func() tea.Msg {
		results, err := client.SearchAll(context.Background(), accountID)
		return searchResultsMsg{results: results, err: err}
	}
}
