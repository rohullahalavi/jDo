// tui_update.go — all TUI event handling and actions.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		m.clock = time.Now()
		return m, heartbeat()
	case blinkMsg:
		m.cursorVisible = !m.cursorVisible
		if m.mode == "search" || m.mode == "add" {
			return m, blinkCmd()
		}
		return m, nil
	case toastMsg:
		if !time.Now().Before(m.toastExp) {
			m.toast = ""
		}
		return m, nil
	case editorFinishedMsg:
		if msg.err != nil {
			m.toast = "editor error"
		}
		m.history = loadHistory(m.historyPath)
		m.files = scanFiles(m.config, m.history)
		m.applySort()
		m.buildFiltered()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case "search":
		return m.updateSearch(msg)
	case "add":
		return m.updateAdd(msg)
	case "palette":
		return m.updatePalette(msg)
	case "editor":
		return m.updateEditor(msg)
	case "help":
		m.mode = m.prevMode
		return m, nil
	case "pins":
		return m.updatePins(msg)
	case "tasks":
		return m.updateTasks(msg)
	case "drill":
		return m.updateDrill(msg)
	default:
		return m.updateNav(msg)
	}
}

func (m *model) closeInline() {
	m.expandedRow = -1
	m.inline = nil
}

// cardPage scrolls WITHIN the open inline card at its current level.
// The main selection never moves and the card never closes here.
func (m *model) cardPage(dir int) {
	ds := m.inline
	if ds == nil {
		return
	}
	mh, mt, ms := drillCaps(ds.drill, ds.headStart, ds.headCount, ds.taskCount, ds.subCount)
	switch ds.level {
	case 1:
		ds.headStart = clampInt(ds.headStart+dir*3, 0, maxInt(0, mh-1))
	case 2:
		ds.taskStart = clampInt(ds.taskStart+dir*2, 0, maxInt(0, mt-1))
	case 3:
		ds.subStart = clampInt(ds.subStart+dir*2, 0, maxInt(0, ms-1))
	}
}

func (m *model) updateNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	K := m.config.Keys
	m.wrapDir = 0
	if len(m.focus) == 0 {
		m.rebuildFocus()
	}

	if len(key) == 1 && key >= "1" && key <= "9" {
		roots := m.visibleRootPositions()
		n := int(key[0] - '1')
		if n < len(roots) {
			m.selected = roots[n]
		}
		return m, nil
	}

	if key == "g" && K.Top == "g" {
		if m.pendingG {
			m.pendingG = false
			m.selected = 0
			return m, nil
		}
		m.pendingG = true
		return m, nil
	}
	m.pendingG = false

	switch {
	case key == K.Quit || key == "q" || key == "ctrl+c":
		return m, tea.Quit

	case key == K.Down || key == "down":
		m.moveDown()
	case key == K.Up || key == "up":
		m.moveUp()

	case key == K.NextPage || key == "pgdown":
		m.moveFocusClamped(m.pageStep())
	case key == K.PrevPage || key == "pgup":
		m.moveFocusClamped(-m.pageStep())
	case key == K.Bottom || key == "G":
		if last := m.lastRealFocus(); last >= 0 {
			m.selected = last
		}

	case key == "l":
		if node := m.currentNode(); node != nil && node.Kind != "pad" && len(node.Children) > 0 && !node.Expanded {
			node.Expanded = true
			m.rebuildFocus()
		}
		return m, nil

	case key == "h":
		if node := m.currentNode(); node != nil && node.Kind != "pad" {
			if node.Expanded && len(node.Children) > 0 {
				node.Expanded = false
				m.rebuildFocus()
				return m, nil
			}
			if node.Parent != nil {
				parent := node.Parent
				for i, entry := range m.focus {
					if entry.Node == parent {
						m.selected = i
						break
					}
				}
			}
		}
		return m, nil

	case key == K.Open || key == "enter":
		if file, ok := m.currentFile(); ok {
		if node := m.currentNode(); node != nil && node.Kind != "root" {
			return m.openAtLine(file, node.Line)
		}
			return m.openFile(file, m.config.Core.DefaultEditor)
		}
	case key == K.OpenWith || key == "O":
		if _, ok := m.currentFile(); ok {
			m.prevMode = m.mode
			m.mode = "editor"
			m.modalSel = 0
		}
	case key == "a":
		m.addInput = ""
		m.mode = "add"
		return m, blinkCmd()
	case key == "p":
		return m, m.togglePin()
	case key == "P":
		m.prevMode = m.mode
		m.mode = "pins"
		m.modalSel = 0
		return m, nil
	case key == "t":
		m.openTasks()
		return m, nil
	case key == K.Search || key == "/":
		m.prevMode = m.mode
		m.mode = "search"
		m.searchInput = ""
		m.searchIdx = -1
		m.cursorVisible = true
		return m, blinkCmd()
	case key == ":":
		m.prevMode = m.mode
		m.mode = "palette"
		m.palInput = ""
		m.palSel = 0
		return m, nil
	case key == "s":
		m.cycleSort()
		return m, m.emitToast("sort: " + m.config.Core.SortBy)
	case key == "r":
		m.refresh()
		return m, m.emitToast(fmt.Sprintf("rescanned · %d notes", len(m.files)))
	case key == "y":
		return m, m.yank()
	case key == K.Help || key == "?":
		m.prevMode = m.mode
		m.mode = "help"
		return m, nil
	case key == "esc":
		if m.query != "" {
			m.query = ""
			m.buildFiltered()
			return m, m.emitToast("filter cleared")
		}
	}
	return m, nil
}

func (m *model) openInline() {
	if len(m.filtered) == 0 {
		return
	}
	f := m.filtered[m.selected]
	d := parseDrill(f.FullPath)
	ds := &drillState{note: f, drill: d, level: 1,
		headCount: m.config.UI.PeekHeadings, taskCount: m.config.UI.PeekTasks, subCount: m.config.UI.PeekSubs}
	if ds.headCount < 3 {
		ds.headCount = 3
	}
	if ds.taskCount < 1 {
		ds.taskCount = 1
	}
	if ds.subCount < 1 {
		ds.subCount = 1
	}
	m.inline = ds
	m.expandedRow = m.selected
}

func (m *model) openDrill() {
	if len(m.filtered) == 0 {
		return
	}
	f := m.filtered[m.selected]
	d := parseDrill(f.FullPath)
	ds := &drillState{note: f, drill: d, level: 1,
		headCount: m.config.UI.PeekHeadings, taskCount: m.config.UI.PeekTasks, subCount: m.config.UI.PeekSubs}
	if ds.headCount < 3 {
		ds.headCount = 3
	}
	if ds.taskCount < 1 {
		ds.taskCount = 1
	}
	if ds.subCount < 1 {
		ds.subCount = 1
	}
	m.full = ds
	m.scroll = 0
	m.mode = "drill"
}

func (m *model) growCount(ds *drillState) {
	mh, mt, ms := drillCaps(ds.drill, ds.headStart, ds.headCount, ds.taskCount, ds.subCount)
	switch ds.level {
	case 1:
		if ds.headCount < mh {
			ds.headCount++
		}
	case 2:
		if ds.taskCount < mt {
			ds.taskCount++
		}
	case 3:
		if ds.subCount < ms {
			ds.subCount++
		}
	}
}

func (m *model) shrinkCount(ds *drillState) {
	switch ds.level {
	case 1:
		if ds.headCount > 3 {
			ds.headCount--
		}
	case 2:
		if ds.taskCount > 1 {
			ds.taskCount--
		}
	case 3:
		if ds.subCount > 1 {
			ds.subCount--
		}
	}
}

func (m *model) updateDrill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "q" || key == "ctrl+c":
		return m, tea.Quit
	case key == " ":
		if m.full.level >= 3 {
			m.mode = "main"
			m.full = nil
		} else {
			m.full.level++
		}
		m.scroll = 0
	case key == "l":
		m.growCount(m.full)
	case key == "h":
		m.shrinkCount(m.full)
	case key == "esc":
		if m.full.level <= 1 {
			m.mode = "main"
			m.full = nil
		} else {
			m.full.level--
		}
		m.scroll = 0
	case key == "j" || key == "down":
		m.scroll++
	case key == "k" || key == "up":
		if m.scroll > 0 {
			m.scroll--
		}
	case key == "g":
		m.scroll = 0
	case key == "G":
		m.scroll = 1 << 30
	case key == "o" || key == "enter":
		line := 1
		if m.full != nil && m.full.drill != nil && len(m.full.drill.Headings) > 0 {
			line = m.full.drill.Headings[0].Line
		}
		return m.openAtLine(m.full.note, line)
	case key == "r":
		m.refresh()
		if m.full != nil {
			m.full.drill = parseDrill(m.full.note.FullPath)
		}
		return m, m.emitToast("refreshed")
	case key == "y":
		return m, m.yank()
	case key == "?":
		m.prevMode = m.mode
		m.mode = "help"
	}
	return m, nil
}

func (m *model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = m.prevMode
		return m, nil
	case "enter":
		m.mode = m.prevMode
		m.query = strings.TrimSpace(m.searchInput)
		if m.query != "" {
			m.searchHist = append(m.searchHist, m.query)
			if len(m.searchHist) > 20 {
				m.searchHist = m.searchHist[len(m.searchHist)-20:]
			}
		}
		m.searchIdx = -1
		m.buildFiltered()
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
	case "up":
		if len(m.searchHist) > 0 {
			m.searchIdx++
			if m.searchIdx >= len(m.searchHist) {
				m.searchIdx = len(m.searchHist) - 1
			}
			m.searchInput = m.searchHist[len(m.searchHist)-1-m.searchIdx]
		}
		return m, blinkCmd()
	case "down":
		if m.searchIdx > 0 {
			m.searchIdx--
			m.searchInput = m.searchHist[len(m.searchHist)-1-m.searchIdx]
		} else {
			m.searchIdx = -1
			m.searchInput = ""
		}
		return m, blinkCmd()
	default:
		k := msg.String()
		if len(k) == 1 {
			m.searchInput += k
		}
	}
	return m, blinkCmd()
}

func (m *model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = "main"
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.addInput)
		m.mode = "main"
		if text != "" {
			f, ok := m.currentFile()
			if !ok {
				return m, nil
			}
			res := addCore(m.config, m.history, m.historyPath, f, AddOptions{Items: []string{text}, Tag: firstHeading(f.FullPath)})
			if res.Err == nil {
				m.refresh()
				return m, m.emitToast("✓ added → " + f.Title)
			}
		}
		return m, nil
	case "backspace":
		if len(m.addInput) > 0 {
			m.addInput = m.addInput[:len(m.addInput)-1]
		}
	default:
		k := msg.String()
		if len(k) == 1 {
			m.addInput += k
		}
	}
	return m, blinkCmd()
}

func (m *model) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" || key == "q" {
		m.mode = m.prevMode
		return m, nil
	}
	if key == "j" || key == "down" {
		if m.modalSel < len(m.editors)-1 {
			m.modalSel++
		}
		return m, nil
	}
	if key == "k" || key == "up" {
		if m.modalSel > 0 {
			m.modalSel--
		}
		return m, nil
	}
	if len(key) == 1 && key >= "1" && key <= "9" {
		idx := int(key[0] - '1')
		if idx < len(m.editors) {
			m.modalSel = idx
			m.mode = m.prevMode
			if f, ok := m.currentFile(); ok {
				return m.openFile(f, m.editors[m.modalSel])
			}
		}
		return m, nil
	}
	if key == "o" || key == "enter" {
		ed := m.editors[m.modalSel]
		m.mode = m.prevMode
		if f, ok := m.currentFile(); ok {
			return m.openFile(f, ed)
		}
	}
	return m, nil
}

func (m *model) updatePins(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	pinned := m.pinnedList()
	if key == "esc" || key == "q" {
		m.mode = m.prevMode
		return m, nil
	}
	if key == "j" || key == "down" {
		if m.modalSel < len(pinned)-1 {
			m.modalSel++
		}
		return m, nil
	}
	if key == "k" || key == "up" {
		if m.modalSel > 0 {
			m.modalSel--
		}
		return m, nil
	}
	if key == "x" {
		if m.modalSel < len(pinned) {
			k := pinned[m.modalSel].key
			m.pins = removeString(m.pins, k)
			m.savePins()
			m.buildFiltered()
			if m.modalSel >= len(m.pins) {
				m.modalSel = len(m.pins) - 1
			}
			return m, m.emitToast("unpinned")
		}
	}
	return m, nil
}

func (m *model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "esc" || key == "q":
		m.mode = "main"
		return m, nil
	case key == "j" || key == "down":
		if m.tasksSel < len(m.tasksSnap)-1 {
			m.tasksSel++
		}
	case key == "k" || key == "up":
		if m.tasksSel > 0 {
			m.tasksSel--
		}
	case key == "x" || key == " ":
		if m.tasksSel < len(m.tasksSnap) {
			it := m.tasksSnap[m.tasksSel]
			_ = toggleDoneAt(it.file.FullPath, it.line, true)
			m.refreshTasks()
			return m, m.emitToast("✓ done")
		}
	case key == "u":
		if m.tasksSel < len(m.tasksSnap) {
			it := m.tasksSnap[m.tasksSel]
			_ = toggleDoneAt(it.file.FullPath, it.line, false)
			m.refreshTasks()
			return m, m.emitToast("reopened")
		}
	case key == "o" || key == "enter":
		if m.tasksSel < len(m.tasksSnap) {
			it := m.tasksSnap[m.tasksSel]
			return m.openAtLine(it.file, it.line+1)
		}
	}
	return m, nil
}

func (m *model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := paletteItems(m)
	key := msg.String()
	switch key {
	case "esc", "ctrl+c":
		m.mode = m.prevMode
		return m, nil
	case "down":
		if m.palSel < len(items)-1 {
			m.palSel++
		}
		return m, nil
	case "up":
		if m.palSel > 0 {
			m.palSel--
		}
		return m, nil
	case "tab":
		if m.palSel < len(items) {
			m.palInput = items[m.palSel].label
		}
		return m, nil
	case "enter":
		var cmd tea.Cmd
		if m.palSel < len(items) {
			cmd = m.runPalette(items[m.palSel])
		}
		return m, cmd
	case "backspace":
		if len(m.palInput) > 0 {
			m.palInput = m.palInput[:len(m.palInput)-1]
		}
		m.palSel = 0
		return m, nil
	default:
		if len(key) == 1 {
			m.palInput += key
			m.palSel = 0
		}
	}
	return m, nil
}

func (m *model) openFile(f FileInfo, editorKey string) (tea.Model, tea.Cmd) {
	recordHistory(m.history, m.historyPath, f.Filename, m.config.Core.MaxHistoryItems)
	editorCmd := m.config.Core.DefaultEditor
	if val, ok := m.config.Editors[editorKey]; ok {
		editorCmd = val
	} else if editorKey != "" {
		editorCmd = editorKey
	}
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return m, m.emitToast("no editor configured")
	}
	cmd := exec.Command(parts[0], append(parts[1:], f.FullPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editorFinishedMsg{err} })
}

func (m *model) openAtLine(f FileInfo, line int) (tea.Model, tea.Cmd) {
	recordHistory(m.history, m.historyPath, f.Filename, m.config.Core.MaxHistoryItems)
	parts := strings.Fields(m.config.Core.DefaultEditor)
	if len(parts) == 0 {
		return m, m.emitToast("no editor configured")
	}
	arg := strings.ReplaceAll(m.config.Core.EditorJumpArg, "{line}", strconvItoa(line))
	args := append(parts[1:], arg, f.FullPath)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editorFinishedMsg{err} })
}

func (m *model) yank() tea.Cmd {
	f, ok := m.currentFile()
	if !ok {
		return nil
	}
	c := exec.Command("pbcopy")
	c.Stdin = strings.NewReader(f.FullPath)
	_ = c.Run()
	return m.emitToast("✓ path copied")
}

func (m *model) togglePin() tea.Cmd {
	f, ok := m.currentFile()
	if !ok {
		return nil
	}
	if m.isPinned(f.key) {
		m.pins = removeString(m.pins, f.key)
		m.savePins()
		m.buildFiltered()
		return m.emitToast("unpinned " + f.Title)
	}
	m.pins = append(m.pins, f.key)
	m.savePins()
	m.buildFiltered()
	return m.emitToast("pinned " + f.Title)
}

func removeString(s []string, v string) []string {
	out := s[:0]
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// buildFiltered closes any open inline card because row indices may change.
func (m *model) buildFiltered() {
	list := m.files
	if m.query != "" {
		q := m.query
		cs := m.config.Core.CaseSensitiveSearch
		var res []FileInfo
		for _, f := range list {
			if fuzzyMatch(q, f.Filename, cs) || fuzzyMatch(q, f.Title, cs) ||
				strings.Contains(strings.ToLower(f.Description), strings.ToLower(q)) {
				res = append(res, f)
			}
		}
		list = res
	}
	pinned := make([]FileInfo, 0)
	rest := make([]FileInfo, 0, len(list))
	for _, f := range list {
		if m.isPinned(f.key) {
			pinned = append(pinned, f)
		} else {
			rest = append(rest, f)
		}
	}
	sortPinned := make([]FileInfo, 0, len(pinned))
	for _, k := range m.pins {
		for _, f := range pinned {
			if f.key == k {
				sortPinned = append(sortPinned, f)
				break
			}
		}
	}
	m.sortSlice(rest)
	m.filtered = append(sortPinned, rest...)
	m.rebuildFocus()
}

func (m *model) sortSlice(s []FileInfo) {
	switch m.config.Core.SortBy {
	case "alpha":
		sortByAlpha(s)
	case "recent":
		sortByRecent(s)
	default:
		sortByFrequency(s)
	}
}

func (m *model) applySort() { m.sortSlice(m.files) }

func (m *model) cycleSort() {
	switch m.config.Core.SortBy {
	case "frequency":
		m.config.Core.SortBy = "alpha"
	case "alpha":
		m.config.Core.SortBy = "recent"
	default:
		m.config.Core.SortBy = "frequency"
	}
	m.buildFiltered()
}

func (m *model) refresh() {
	m.history = loadHistory(m.historyPath)
	m.files = scanFiles(m.config, m.history)
	m.trees = buildTrees(m.files)
	m.applySort()
	m.buildFiltered()
}

func (m *model) moveDown() {
	m.wrapNav(+1)
}

func (m *model) moveUp() {
	m.wrapNav(-1)
}

func (m *model) wrapNav(dir int) {
	n := len(m.focus)
	if n == 0 {
		return
	}
	newSel := m.selected + dir
	if newSel < 0 || newSel >= n {
		if m.selected == newSel {
			return
		}
		if m.wrapDir == dir && time.Since(m.wrapAt) < 500*time.Millisecond {
			m.selected = (newSel + n) % n
			m.wrapDir = 0
			return
		}
		m.wrapDir = dir
		m.wrapAt = time.Now()
		return
	}
	m.selected = newSel
}

func (m *model) moveFocusClamped(delta int) {
	n := len(m.focus)
	if n == 0 {
		return
	}
	m.selected = clampInt(m.selected+delta, 0, n-1)
}

func (m *model) lastRealFocus() int {
	n := len(m.focus)
	if n == 0 {
		return -1
	}
	return n - 1
}

func (m *model) pageStep() int {
	p := m.height / 4
	if p < 1 {
		p = 1
	}
	return p
}

func (m *model) pinnedList() []FileInfo {
	var out []FileInfo
	for _, k := range m.pins {
		for _, f := range m.files {
			if f.key == k {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

func (m *model) openTasks() {
	m.tasksSnap = listOpenTasks(m.files, "")
	m.tasksSel = 0
	m.scroll = 0
	m.mode = "tasks"
}

func (m *model) refreshTasks() {
	sel := m.tasksSel
	m.tasksSnap = listOpenTasks(m.files, "")
	m.history = loadHistory(m.historyPath)
	m.files = scanFiles(m.config, m.history)
	m.applySort()
	m.buildFiltered()
	if sel >= len(m.tasksSnap) {
		sel = len(m.tasksSnap) - 1
	}
	if sel < 0 {
		sel = 0
	}
	m.tasksSel = sel
}
