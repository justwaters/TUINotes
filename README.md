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

TUINotes has two panes: a **Navigation** tree (folders, with each expanded
folder's notes nested underneath) and a **Note** pane showing the open
note's content.

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Switch pane (navigation ↔ note) |
| `↑`/`↓`, `j`/`k` | Navigate the tree |
| `enter` | Expand/collapse a folder, or open a note |
| `/` | Filter the focused list |
| `n` | Create a new note in the selected/parent folder |
| `e` | Edit the open note |
| `ctrl+s` | Save |
| `esc` | Cancel editing / close an overlay |
| `d` then `y` | Delete the selected note (moves to Recently Deleted) |
| `S` | Search all notes in the first account (then `/` to filter, `enter` to open) |
| `r` | Refresh the tree and any expanded folders |
| `q` / `ctrl+c` | Quit |

## How it works

There's no public API or documented file format for Apple Notes, so
TUINotes shells out to `osascript -l JavaScript` (JXA) for every read and
write — see `internal/notes`. Note bodies are HTML internally; TUINotes
reads and writes a plain-text representation of that HTML (the first line
is the note's title, matching how Notes.app itself works) via
`internal/convert`.

Lists round-trip: prefix a line with `* ` for a bulleted list, `- ` for a
dashed list, or `1. ` for a numbered list, and consecutive lines of the
same kind become one list when saved — matching the actual (undocumented)
HTML Notes.app uses for each. Existing lists in a note show up the same
way when you open it.

## v1 limitations

- **No checklists yet, and no other rich formatting.** Bold, italics,
  headings, colors, and checklists applied in the Notes app are dropped to
  plain text if you edit and save that note in TUINotes (lists are the
  exception — see above).
- **No attachment editing.** Notes with attachments (images, scans,
  drawings, tables) open in view-only mode — TUINotes refuses to save over
  them, since it has no way to preserve attachment markup.
- **Search covers one account.** `S` searches the first Notes account only.
- **Search relevance is rough.** The `/` filter in the search overlay does
  fuzzy subsequence matching against full note bodies, so short queries can
  surface loosely-related notes.
- **No note moving in the UI.** `internal/notes` has a `MoveNote` function
  but there's no keybinding for it yet. Nested folders do display correctly
  in the navigation tree (indented under their parent, matching Notes.app).
