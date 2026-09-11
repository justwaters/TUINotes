package main

import (
	"context"
	"fmt"
	"os"

	"github.com/justwaters/TUINotes/internal/notes"
	"github.com/justwaters/TUINotes/internal/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	client := notes.NewClient()

	if _, err := client.ListAccounts(context.Background()); err != nil {
		if notes.IsNotAuthorized(err) {
			fmt.Fprintln(os.Stderr, "TUINotes needs permission to control Notes.\n"+
				"Open System Settings → Privacy & Security → Automation → (your terminal app) → enable Notes, then restart TUINotes.")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "tuinotes: failed to reach Notes.app:", err)
		os.Exit(1)
	}

	m := ui.New(client)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tuinotes:", err)
		os.Exit(1)
	}
}
