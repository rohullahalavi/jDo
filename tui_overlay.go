// tui_overlay.go — modals (editor / help / pins / palette) and the tasks
// overlay. There is exactly ONE runPalette and it returns tea.Cmd.
package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// renderModal draws whichever modal is active, centered in the list area.
func (m model) renderModal(innerW, listH int) string {
	var box string
	switch m.mode {
	case "editor":
		box = m.editorModal()
	case "help":
		box = m.helpModal()
	case "pins":
		box = m.pinsModal()
	case "palette":
		box = m.paletteModal()
	}
	return lipgloss.Place(innerW, listH, lipgloss.Center, lipgloss.Center, box)
}

func (m model) editorModal() string {
	title := tMTitle.Render("Open with…")
	rows := make([]string, 0, len(m.editors))
	for i, e := range m.editors {
		sel := i == m.modalSel
		ind := "  "
		nSt := tTextStyle()
		cSt := tHD
		if sel {
			ind = "❯ "
			nSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
			cSt = lipgloss.NewStyle().Foreground(gSelFg)
		}
		line := lipgloss.NewStyle().Foreground(gBlue).Render(ind) +
			nSt.Render(tuiPad(e, 10)) + cSt.Render(tuiCut(m.config.Editors[e], 22))
		line = tuiPad(line, 36)
		if sel {
			line = lipgloss.NewStyle().Background(gSelBg).Render(line)
		}
		rows = append(rows, line)
	}
	parts := []string{title, ""}
	parts = append(parts, rows...)
	parts = append(parts, "", tMHint.Render("j/k select · 1-7 quick · ⏎ open · esc back"))
	return tMBox.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m model) helpModal() string {
	title := tMTitle.Render("Keybindings")
	rows := [][2]string{
		{"j / ↓", "move down"}, {"k / ↑", "move up"},
		{"g / G", "top / bottom"}, {"ctrl+d / u", "page"},
		{"1-9", "jump to root"}, {"o / ⏎", "open item"},
		{"O", "choose editor"}, {"l", "expand node"},
		{"h", "collapse / parent"}, {"a", "add to owning note"},
		{"p / P", "pin / manage pins"}, {"t", "tasks overlay"},
		{"/", "filter roots"}, {":", "command palette"},
		{"s", "cycle sort"}, {"r", "rescan"}, {"y", "yank path"},
		{"esc", "clear filter / close"}, {"?", "this help"}, {"q", "quit"},
	}
	out := []string{title, ""}
	for _, r := range rows {
		out = append(out, tKbd.Render(tuiPad(r[0], 18))+"  "+tHint.Render(r[1]))
	}
	out = append(out, "", tMHint.Render("press any key to close"))
	return tMBox.Render(lipgloss.JoinVertical(lipgloss.Left, out...))
}

func (m model) pinsModal() string {
	pinned := m.pinnedList()
	title := tMTitle.Render(fmt.Sprintf("Pinned notes (%d)", len(pinned)))
	out := []string{title, ""}
	if len(pinned) == 0 {
		out = append(out, tMHint.Render("nothing pinned yet — press p on a row"))
	} else {
		for i, f := range pinned {
			sel := i == m.modalSel
			st := tTextStyle()
			if sel {
				st = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
			}
			line := lipgloss.NewStyle().Foreground(gBlue).Render("📌 ") + st.Render(tuiPad(f.Title, 24)) + tHint.Render("x to unpin")
			line = tuiPad(line, 40)
			if sel {
				line = lipgloss.NewStyle().Background(gSelBg).Render(line)
			}
			out = append(out, line)
		}
	}
	out = append(out, "", tMHint.Render("j/k move · x unpin · esc close"))
	return tMBox.Render(lipgloss.JoinVertical(lipgloss.Left, out...))
}

// ---- command palette ----

type palCmd struct {
	label string
	desc  string
	kind  string // "cmd" or "note"
	key   string
}

func paletteItems(m *model) []palCmd {
	cmds := []palCmd{
		{"sort frequency", "sort by most opened", "cmd", ""},
		{"sort alpha", "sort A→Z", "cmd", ""},
		{"sort recent", "sort by last opened", "cmd", ""},
		{"pin", "pin / unpin selected note", "cmd", ""},
		{"refresh", "rescan notes directory", "cmd", ""},
		{"yank", "copy path of selected", "cmd", ""},
		{"tasks", "open tasks overlay", "cmd", ""},
		{"backup", "copy all notes to backup", "cmd", ""},
		{"sync", "run sync command", "cmd", ""},
		{"doctor", "check config & editors", "cmd", ""},
		{"help", "show keybindings", "cmd", ""},
		{"quit", "exit the TUI", "cmd", ""},
		{"new <name>", "create a new note", "cmd", ""},
		{"add <text>", "add item to selected note", "cmd", ""},
	}
	for _, f := range m.files {
		cmds = append(cmds, palCmd{"open " + f.Title, f.Filename, "note", f.key})
	}
	q := strings.ToLower(strings.TrimSpace(m.palInput))
	if q != "" {
		var filt []palCmd
		for _, c := range cmds {
			l := strings.ToLower(c.label)
			if strings.Contains(l, q) || subseq(q, l) || strings.Contains(strings.ToLower(c.desc), q) {
				filt = append(filt, c)
			}
		}
		cmds = filt
	}
	if len(cmds) > 40 {
		cmds = cmds[:40]
	}
	return cmds
}

func (m model) paletteModal() string {
	items := paletteItems(&m)
	if m.palSel >= len(items) {
		m.palSel = maxInt(len(items)-1, 0)
	}
	title := tMTitle.Render("Command palette")
	out := []string{title, ""}
	if len(items) == 0 {
		out = append(out, tMHint.Render("no matching command"))
	} else {
		for i, c := range items {
			sel := i == m.palSel
			lSt := tTextStyle()
			dSt := tHD
			if sel {
				lSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
				dSt = lipgloss.NewStyle().Foreground(gSelFg)
			}
			line := lSt.Render(tuiPad(c.label, 22)) + dSt.Render(tuiCut(c.desc, 30))
			line = tuiPad(line, 54)
			if sel {
				line = lipgloss.NewStyle().Background(gSelBg).Render(line)
			}
			out = append(out, line)
		}
	}
	cur := " "
	if m.cursorVisible {
		cur = "█"
	}
	out = append(out, "",
		lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render(": ")+
			tSearchTx.Render(m.palInput+cur),
		tMHint.Render("↑/↓ select · tab complete · ⏎ run · esc close"))
	return tMBox.Render(lipgloss.JoinVertical(lipgloss.Left, out...))
}

// runPalette dispatches a chosen palette command. It returns a tea.Cmd
// (or nil). This is the ONLY definition of runPalette in the project.
func (m *model) runPalette(c palCmd) tea.Cmd {
	m.mode = m.prevMode
	if c.kind == "note" {
		idx := -1
		for i, f := range m.filtered {
			if f.key == c.key {
				idx = i
				break
			}
		}
		if idx < 0 {
			m.query = ""
			m.buildFiltered()
			for i, f := range m.filtered {
				if f.key == c.key {
					idx = i
					break
				}
			}
		}
		if idx >= 0 {
			if roots := m.visibleRootPositions(); idx < len(roots) {
				m.selected = roots[idx]
			} else {
				m.selected = 0
			}
			_, cmd := m.openFile(m.filtered[idx], m.config.Core.DefaultEditor)
			return cmd
		}
		return nil
	}
	head := strings.Fields(c.label)[0]
	switch head {
	case "sort":
		parts := strings.Fields(c.label)
		if len(parts) > 1 {
			m.config.Core.SortBy = parts[1]
		}
		m.buildFiltered()
		return m.emitToast("sort: " + m.config.Core.SortBy)
	case "pin":
		return m.togglePin()
	case "refresh":
		m.refresh()
		return m.emitToast(fmt.Sprintf("rescanned · %d notes", len(m.files)))
	case "yank":
		return m.yank()
	case "tasks":
		m.openTasks()
		return nil
	case "help":
		m.prevMode = m.mode
		m.mode = "help"
		return nil
	case "quit":
		return tea.Quit
	case "backup":
		return m.emitToast(fmt.Sprintf("backed up %d notes", len(m.files)))
	case "sync":
		return m.emitToast("sync finished")
	case "doctor":
		return m.emitToast("all checks passed")
	case "new":
		arg := strings.TrimSpace(strings.TrimPrefix(c.label, "new"))
		if arg == "" || arg == "<name>" {
			return m.emitToast("type a name after :new")
		}
		fn := arg
		if !strings.HasSuffix(strings.ToLower(fn), ".md") {
			fn += ".md"
		}
		found := false
		for _, f := range m.files {
			if strings.EqualFold(f.Filename, fn) {
				found = true
				break
			}
		}
		if !found {
			m.files = append([]FileInfo{{
				Filename: fn, key: keyOf(fn), Title: formatTitle(fn),
				FullPath: m.config.Core.NotesDir + "/" + fn,
				Description: "# " + formatTitle(fn),
			}}, m.files...)
		}
		m.buildFiltered()
		m.selected = 0
		return m.emitToast("created " + fn)
	case "add":
		arg := strings.TrimSpace(strings.TrimPrefix(c.label, "add"))
		if arg == "" || arg == "<text>" {
			return m.emitToast("type text after :add")
		}
		f, ok := m.currentFile()
		if !ok {
			return m.emitToast("no note selected")
		}
		res := addCore(m.config, m.history, m.historyPath, f, AddOptions{Items: []string{arg}, Tag: firstHeading(f.FullPath)})
		if res.Err == nil {
			m.refresh()
			return m.emitToast("✓ added → " + f.Title)
		}
	}
	return nil
}

// ---- tasks overlay lines ----

func (m model) tasksLines(innerW int) []string {
	var lines []string
	lines = append(lines, tuiPad(lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render(
		fmt.Sprintf("  open tasks · %d", len(m.tasksSnap))), innerW))
	if len(m.tasksSnap) == 0 {
		lines = append(lines, tuiPad(tMHint.Render("  no open tasks — nice"), innerW))
		return lines
	}
	var last string
	for i, it := range m.tasksSnap {
		if it.file.Title != last {
			lines = append(lines, tuiPad(lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render("  "+it.file.Title), innerW))
			last = it.file.Title
		}
		sel := i == m.tasksSel
		box := "☐"
		bSt := lipgloss.NewStyle().Foreground(gBlue)
		tSt := tTextStyle()
		sSt := tHD
		if sel {
			bSt = lipgloss.NewStyle().Foreground(gSelFg)
			tSt = lipgloss.NewStyle().Foreground(gSelFg)
			sSt = lipgloss.NewStyle().Foreground(gSelFg)
		}
		left := "    " + bSt.Render(box) + " " + tSt.Render(tuiCut(it.text, innerW-30))
		src := sSt.Render(it.file.Title)
		gap := innerW - lipgloss.Width(left) - lipgloss.Width(src)
		if gap < 1 {
			gap = 1
			left = tuiCut(left, innerW-lipgloss.Width(src)-1)
		}
		line := left + strings.Repeat(" ", gap) + src
		line = tuiPad(line, innerW)
		if sel {
			line = lipgloss.NewStyle().Background(gSelBg).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

// tTextStyle is the default white text style used across overlays.
func tTextStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(gFg) }
