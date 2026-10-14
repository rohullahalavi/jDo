# jDo

**jdo** — a markdown notes commander for the terminal.

jdo is a fast, keyboard-driven CLI + TUI for working with a directory of
markdown notes: fuzzy-find and open them in your editor, quick-add tasks and
list items into the right file and section, track due dates, and inspect
everything without leaving the shell.

## Features

- **Quick launcher** — run `jdo` with no arguments to see your top notes as a
  numbered list; type a number to open one instantly (no TUI, no screen clear).
- **Fuzzy finding** — `jdo <query>` fuzzy-matches note names; when several
  files match, a numbered picker appears. Type e.g. `2d` to make candidate #2
  the *permanent default* for that query.
- **Full interactive TUI** — `jdo i` opens a Bubble Tea-based tree UI with
  live filtering, sorting, previews, pins, quick-add, yank-to-clipboard and
  more.
- **Task management** — `add`, `done`, `next`, `nexts`, `tasks` and `count`
  operate directly on checkboxes (`- [ ]`) and task-ish headings in your
  markdown files, with priorities and natural-language due dates
  (`tomorrow`, `friday`, `+3d`, `2026-08-01`).
- **Inspection commands** — `grep`, `show`, `tree`, `search`, `list`,
  `recent`, `history`, `stats`.
- **Note creation** — `new <name>` from a configurable template, and
  `today`/`daily` to open or create today's note.
- **Maintenance** — `del`, `backup`, `sync`, `doctor`, `edit`/`config`.
- **Editor integration** — opens notes in nvim, vim, helix, zed, VS Code,
  nano, micro, or any configured editor, optionally jumping to a line.

## Requirements

- Go 1.21 or newer
- A terminal (the TUI uses [Bubble Tea](https://github.com/charmbracelet/bubbletea)
  and [Lip Gloss](https://github.com/charmbracelet/lipgloss))

## Installation

```sh
git clone <repo-url> && cd jdo
make build          # equivalent to: go build -o jdo .
sudo cp jdo /usr/local/bin   # or put it anywhere on your PATH
```

## Usage

```
jdo [COMMAND] [ARGS]
```

### Commands

| Command | Description |
| --- | --- |
| *(none)* | Top-N quick launcher; type 1-5 to open |
| `i`, `tui` | Open the full interactive TUI |
| `z`, `last` | Reopen the last note you opened |
| `<query>` | Fuzzy-find a note; picker if several match |
| `add <t> "text"` | Insert an item into the right file + section |
| `done <text>` | Mark a matching task as done |
| `next [n]` | Show tasks due in the next N hours (default 1) |
| `nexts [d\|w\|m\|y]` | Show tasks due today/week/month/year |
| `tasks` | List open tasks across all notes |
| `count` | Count items per section per file |
| `grep <text>` | Search note contents |
| `show <note>` | Print a note or one of its sections |
| `tree` | Print the heading tree with item counts |
| `new <name>`, `n` | Create a note from the template and open it |
| `today`, `daily` | Open or create today's note |
| `search <text>` | Full-text search inside every note |
| `list`, `ls` | Print all notes with descriptions |
| `recent` | Show recently opened notes |
| `hist`, `history` | Show usage counts and last-opened times |
| `stats` | Show a usage statistics dashboard |
| `del <query>` | Delete a note (asks for confirmation) |
| `backup` | Copy all notes into the backup directory |
| `sync` | Run the configured sync command |
| `doctor`, `check` | Check config, editors and permissions |
| `edit`, `config` | Open the jdo config file in your editor |
| `help`, `-h` / `version`, `-v` | Help / version information |

### `add` flags

| Flag | Description |
| --- | --- |
| `-t, --tag <name>` | Heading to insert under (default: target) |
| `-p, --priority <n>` | Priority 1-5 or critical/high/normal/low/someday |
| `-d, --due <date>` | `today`, `tomorrow`, `friday`, `+3d`, `+1w`, `2026-08-01` |
| `-c, --check` | Write as a checkbox `- [ ]` |
| `--top` | Insert at the top of the section |
| `--no-emoji` | Skip the emoji prefix |
| `--dry-run` | Show what would change without writing |

Examples:

```sh
jdo add work "review PR" -p high -d tomorrow --check
jdo done review PR
jdo next 24
jdo show inbox target
```

### Picker defaults

When several files match a query, jdo shows a numbered list. Type a number to
open once, or e.g. `2d` to make candidate #2 the permanent default for that
query. Defaults are stored in `~/.jdo/defaults.json`.

### TUI keys

| Key | Action |
| --- | --- |
| `j` / `↓`, `k` / `↑` | Move down / up |
| `g g`, `G` | Jump to top / bottom |
| `ctrl+d`, `ctrl+u` | Half page down / up |
| `1-9` | Jump to visible root |
| `l` / `h` | Expand / collapse tree |
| `o` / `enter` | Open in the default editor |
| `O` | Choose editor |
| `a` | Quick-add into owning note |
| `p` | Toggle pin on owning note |
| `/` | Filter roots live |
| `s` | Cycle sort: frequency → alpha → recent |
| `r` | Rescan the notes directory |
| `y` | Yank note path to clipboard |
| `?` | Keybinding help overlay |
| `esc` | Clear filter / close dialogs |
| `q` | Quit |

## Configuration

On first run jdo creates `~/.jdo/config.toml` with sensible defaults. The main
sections:

- **`[core]`** — `notes_dir` (where your markdown notes live),
  `default_editor`, `sort_by` (`frequency` / alpha / recent),
  `quick_list_size`, `template_content`, backup & sync settings,
  `confirm_before_delete`, and more.
- **`[ui]`** — descriptions, clock, preview panes, peek counts, row separator
  style, date/time formats, stale-days threshold.
- **`[add]`** — bullet style, indentation, priority format, auto-emoji,
  timestamps.
- **`[editors]`** — name → command map (nvim, vim, hx, zed, code, nano,
  micro by default).
- **`[keys]`** — remap TUI keybindings.

Run `jdo edit` (or `jdo config`) to open the config in your editor, and
`jdo doctor` to validate your setup.

## Data Files

| Path | Contents |
| --- | --- |
| `~/.jdo/config.toml` | Configuration |
| `~/.jdo/history.json` | Usage counts and last-opened times |
| `~/.jdo/defaults.json` | Permanent per-query file pins |

## Project Structure

| File(s) | Purpose |
| --- | --- |
| `main.go` | Entry point and argument dispatch |
| `config.go`, `defaults.go`, `history.go` | Config loading, query pins, usage history |
| `notes.go`, `match.go`, `resolve.go`, `query.go` | Note scanning, fuzzy matching, multi-candidate resolution, read-only inspection |
| `tasks.go`, `add.go` | Task lifecycle and the `add` insertion engine |
| `tui_*.go` | Bubble Tea TUI (model, update, view, drill-down, overlays) |
| `admin.go`, `editor.go`, `util.go`, `styles.go`, `help.go`, `meta.go` | Maintenance commands, editor launching, shared helpers, styling, help/version |

## Development

```sh
make build   # compile the jdo binary
make clean   # remove build artifacts
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
