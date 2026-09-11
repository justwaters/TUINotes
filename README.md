# TUINotes

A terminal UI for Apple Notes, written in Go. It reads and writes your real
Notes.app data (via JXA/AppleScript automation) — there's no separate local
notes store.

## Requirements

- macOS with Notes.app
- Go 1.24+

## Build & run

```sh
go build -o bin/tuinotes ./cmd/tuinotes
./bin/tuinotes
```

### Installing so `tuinotes` works from anywhere

Either drop the built binary somewhere already on your `PATH` (e.g.
`~/.local/bin`):

```sh
go build -o ~/.local/bin/tuinotes ./cmd/tuinotes
```

or use `go install`, which puts it in `$(go env GOPATH)/bin` (typically
`~/go/bin`) — add that to your `PATH` once if it isn't already:

```sh
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
go install ./cmd/tuinotes
```

The first time TUINotes talks to Notes.app, macOS will prompt your terminal
app (Terminal, iTerm, etc. — not TUINotes itself) for permission to send it
Apple events. If you miss the prompt or deny it, grant it manually under
**System Settings → Privacy & Security → Automation → (your terminal app) →
Notes**, then restart TUINotes.

## Keybindings

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Switch pane (folders → notes → note) |
| `↑`/`↓`, `j`/`k` | Navigate the focused list |
| `enter` | Open the selected folder or note |
| `/` | Filter the focused list |
| `n` | Create a new note in the selected folder |
| `e` | Edit the open note |
| `ctrl+s` | Save |
| `esc` | Cancel editing / close an overlay |
| `d` then `y` | Delete the selected note (moves to Recently Deleted) |
| `S` | Search all notes in the first account (then `/` to filter, `enter` to open) |
| `r` | Refresh the current pane |
| `q` / `ctrl+c` | Quit |

## How it works

There's no public API or documented file format for Apple Notes, so
TUINotes shells out to `osascript -l JavaScript` (JXA) for every read and
write — see `internal/notes`. Note bodies are HTML internally; TUINotes only
ever reads/writes plain per-line text (the first line is the note's title,
matching how Notes.app itself works), via `internal/convert`.

## v1 limitations

- **No rich formatting.** Saving a note always rewrites its body as plain
  text lines. Bold, italics, headings, and checklists applied in the Notes
  app will be lost if you edit and save that note in TUINotes.
- **No attachment editing.** Notes with attachments (images, scans,
  drawings, tables) open in view-only mode — TUINotes refuses to save over
  them, since it has no way to preserve attachment markup.
- **Search covers one account.** `S` searches the first Notes account only.
- **Search relevance is rough.** The `/` filter in the search overlay does
  fuzzy subsequence matching against full note bodies, so short queries can
  surface loosely-related notes.
- **No folder nesting or note moving in the UI.** `internal/notes` has a
  `MoveNote` function but there's no keybinding for it yet.
