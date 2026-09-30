// tui_view.go — palette, width-safe helpers, and every renderer.
// Width rule: every visible line is built to EXACTLY CW cells using runewidth
// (the same width engine the terminal uses), and the panel is assembled with a
// MANUAL box model (no lipgloss Width constraint), so lines can never wrap.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ---- palette (three hues; grays are white at lower intensity) ----
var (
	gCanvas = lipgloss.Color("#0d1117")
	gZebra  = lipgloss.Color("#11161d")
	gHeadBg = lipgloss.Color("#161b22")
	gFg     = lipgloss.Color("#e6edf3")
	gDim    = lipgloss.Color("#8b949e")
	gFaint  = lipgloss.Color("#484f58")
	gBlue   = lipgloss.Color("#58a6ff")
	gSelBg  = lipgloss.Color("#1f6feb")
	gSelFg  = lipgloss.Color("#ffffff")
	gBorder = lipgloss.Color("#30363d")
	gPillBg = lipgloss.Color("#16335c")
)

// ---- styles ----
var (
	tBadge = lipgloss.NewStyle().Background(gBlue).Foreground(gCanvas).Bold(true).Padding(0, 1)
	tTitle = lipgloss.NewStyle().Foreground(gFg).Bold(true)
	tHD    = lipgloss.NewStyle().Foreground(gDim)

	tKbd  = lipgloss.NewStyle().Foreground(gBlue).Bold(true)
	tHint = lipgloss.NewStyle().Foreground(gDim)

	tStatL = lipgloss.NewStyle().Foreground(gDim).Italic(true)
	tStatR = lipgloss.NewStyle().Foreground(gBlue)

	tSearchPr = lipgloss.NewStyle().Foreground(gBlue).Bold(true)
	tSearchTx = lipgloss.NewStyle().Foreground(gFg)

	tMTitle = lipgloss.NewStyle().Foreground(gBlue).Bold(true)
	tMBox   = lipgloss.NewStyle().Background(gCanvas).Border(lipgloss.RoundedBorder()).BorderForeground(gBlue).Padding(1, 2)
	tMHint  = lipgloss.NewStyle().Foreground(gFaint)

	tEmpty = lipgloss.NewStyle().Foreground(gFaint)
)

type treeRenderedLine struct {
	text string
}

// ============================================================
//  WIDTH-SAFE HELPERS (runewidth = terminal-accurate)
// ============================================================

func stripANSI(s string) string {
	var b strings.Builder
	st := 0
	for _, r := range s {
		switch st {
		case 0:
			if r == 0x1b {
				st = 1
			} else {
				b.WriteRune(r)
			}
		case 1:
			if r == '[' {
				st = 2
			} else {
				st = 0
			}
		case 2:
			if r >= 0x40 && r <= 0x7e {
				st = 0
			}
		}
	}
	return b.String()
}

func tw(s string) int { return runewidth.StringWidth(stripANSI(s)) }

func tuiCut(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	if tw(s) <= maxW {
		return s
	}
	var b strings.Builder
	w, st := 0, 0
	done := false
	for _, r := range s {
		if done {
			break
		}
		switch st {
		case 0:
			if r == 0x1b {
				st = 1
				b.WriteRune(r)
				continue
			}
			rw := runewidth.RuneWidth(r)
			if w+rw > maxW {
				done = true
				continue
			}
			b.WriteRune(r)
			w += rw
		case 1:
			b.WriteRune(r)
			if r == '[' {
				st = 2
			} else {
				st = 0
			}
		case 2:
			b.WriteRune(r)
			if r >= 0x40 && r <= 0x7e {
				st = 0
			}
		}
	}
	out := b.String()
	if w+1 <= maxW {
		out += "…"
	}
	return out + "[0m"
}

func splitTextAtWidth(s string, maxW int) (string, string) {
	if maxW < 1 {
		return "", s
	}
	w := 0
	for i, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > maxW {
			part1 := s[:i] + strings.Repeat(" ", maxW-w)
			part2 := s[i:]
			return part1, part2
		}
		w += rw
	}
	return s + strings.Repeat(" ", maxW-w), ""
}

func tuiPad(s string, w int) string {
	cw := tw(s)
	if cw == w {
		return s
	}
	if cw > w {
		return tuiCut(s, w)
	}
	return s + strings.Repeat(" ", w-cw)
}

func fitLine(s string, w int) string {
	cw := tw(s)
	if cw == w {
		return s
	}
	if cw < w {
		return s + strings.Repeat(" ", w-cw)
	}
	var b strings.Builder
	ww, st := 0, 0
	done := false
	for _, r := range s {
		if done {
			break
		}
		switch st {
		case 0:
			if r == 0x1b {
				st = 1
				b.WriteRune(r)
				continue
			}
			rw := runewidth.RuneWidth(r)
			if ww+rw > w {
				done = true
				continue
			}
			b.WriteRune(r)
			ww += rw
		case 1:
			b.WriteRune(r)
			if r == '[' {
				st = 2
			} else {
				st = 0
			}
		case 2:
			b.WriteRune(r)
			if r >= 0x40 && r <= 0x7e {
				st = 0
			}
		}
	}
	out := b.String()
	if ww < w {
		out += strings.Repeat(" ", w-ww)
	}
	return out + "[0m"
}

func tuiGradient(w int) string {
	if w < 1 {
		w = 1
	}
	var sb strings.Builder
	for i := 0; i < w; i++ {
		t := float64(i) / float64(w)
		c := lerpHex("#58a6ff", "#30363d", t)
		sb.WriteString(lipgloss.NewStyle().Foreground(c).Render("─"))
	}
	return sb.String()
}

func fmtItoa(n int) string { return strconvItoa(n) }

// padBlock forces a multi-line string to exactly h lines, each exactly w wide.
func padBlock(s string, h, w int) []string {
	parts := strings.Split(s, "\n")
	out := make([]string, h)
	for i := 0; i < h; i++ {
		if i < len(parts) {
			out[i] = fitLine(parts[i], w)
		} else {
			out[i] = fitLine("", w)
		}
	}
	return out
}

// windowOff keeps sel inside a [topRes .. vis-1-bottomRes] band (early scroll).
func windowOff(sel, total, vis, bottomRes, topRes, cur int) int {
	if total <= vis {
		return 0
	}
	maxOff := total - vis
	if maxOff < 0 {
		maxOff = 0
	}
	off := cur
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	if sel < 0 {
		return off
	}
	lo := off + topRes
	hi := off + vis - 1 - bottomRes
	if hi < lo {
		hi = lo
	}
	if sel < lo {
		off = sel - topRes
	}
	if sel > hi {
		off = sel - (vis - 1 - bottomRes)
	}
	if off < 0 {
		off = 0
	}
	if off > maxOff {
		off = maxOff
	}
	return off
}

// windowLines windows pre-built lines into exactly h rows of width w.
func (m model) windowLines(lines []string, sel, h, off, topRes, bottomRes, w int) []string {
	total := len(lines)
	off = windowOff(sel, total, h, bottomRes, topRes, off)
	out := make([]string, h)
	for i := 0; i < h; i++ {
		if off+i < total {
			out[i] = fitLine(lines[off+i], w)
		} else {
			out[i] = fitLine("", w)
		}
	}
	return out
}

// ============================================================
//  TREE MAIN VIEW
// ============================================================

func (m model) treeHeader(w int) string {
	badge := tBadge.Render(" JDO ")
	titleTxt := tTitle.Render("  Notes Commander")
	rightInfo := fmt.Sprintf("%d notes", len(m.files))
	if m.config.UI.ShowClock {
		rightInfo += " · " + m.clock.Format("15:04 · Mon 02 Jan")
	}
	right := tHD.Render(rightInfo)
	gap := w - tw(badge+titleTxt) - tw(right)
	if gap < 1 {
		gap = 1
	}
	line1 := fitLine(badge+titleTxt+strings.Repeat(" ", gap)+right, w)

	path := m.config.Core.NotesDir
	if home, err := os.UserHomeDir(); err == nil {
		path = strings.Replace(path, home, "~", 1)
	}
	left := tHD.Render(path)
	right2 := tHD.Render("sort: " + m.config.Core.SortBy)
	if m.query != "" {
		right2 = lipgloss.NewStyle().Foreground(gBlue).Render(fmt.Sprintf("filter: %q", m.query))
	}
	if m.toast != "" && time.Now().Before(m.toastExp) {
		right2 = lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render(m.toast)
	}
	gap2 := w - tw(left) - tw(right2)
	if gap2 < 1 {
		gap2 = 1
	}
	line2 := fitLine(left+strings.Repeat(" ", gap2)+right2, w)
	return line1 + "\n" + line2
}

func (m model) treeLines(w, h int) []string {
	if len(m.focus) == 0 {
		m.rebuildFocus()
	}
	if len(m.filtered) == 0 {
		msg := "no notes — create one with: jdo new <name>"
		if m.query != "" {
			msg = fmt.Sprintf("no match for %q — esc to clear", m.query)
		}
		empty := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center, tEmpty.Render("◌"), "", tHD.Render(msg)))
		return padBlock(empty, h, w)
	}

	var lines []treeRenderedLine
	focusIdx := 0
	selectedDisp := -1
	for _, f := range m.filtered {
		root := m.trees[f.key]
		if root == nil {
			continue
		}
		lines = append(lines, treeRenderedLine{text: m.renderRootRow(f, focusIdx == m.selected, len(lines)%2 == 1, w)})
		if focusIdx == m.selected {
			selectedDisp = len(lines) - 1
		}
		focusIdx++
		if root.Expanded {
			m.appendTreeLines(&lines, root.Children, nil, &focusIdx, &selectedDisp, w)
		}
	}
	if selectedDisp < 0 {
		selectedDisp = 0
	}

	total := len(lines)
	topRes := maxInt(1, h/3)
	bottomRes := topRes
	off := windowOff(selectedDisp, total, h, bottomRes, topRes, 0)
	out := make([]string, h)
	for i := 0; i < h; i++ {
		if off+i < total {
			out[i] = fitLine(lines[off+i].text, w)
		} else {
			out[i] = fitLine("", w)
		}
	}
	return out
}

func (m model) appendTreeLines(lines *[]treeRenderedLine, nodes []*treeNode, lastStack []bool, focusIdx *int, selectedDisp *int, w int) {
	for i, node := range nodes {
		last := i == len(nodes)-1
		selected := *focusIdx == m.selected
		if *focusIdx == m.selected {
			*selectedDisp = len(*lines)
		}
		*lines = append(*lines, treeRenderedLine{text: m.renderTreeNode(node, lastStack, last, selected, w)})
		*focusIdx++
		if node.Expanded && len(node.Children) > 0 {
			m.appendTreeLines(lines, node.Children, append(lastStack, last), focusIdx, selectedDisp, w)
		}
	}
}

func (m model) renderRootRule(w int) string {
	return fitLine(lipgloss.NewStyle().Foreground(gFaint).Render(strings.Repeat("─", w)), w)
}

func (m model) renderRootRow(f FileInfo, sel bool, zebra bool, w int) string {
	nameW := clampInt((w*30)/100, 12, 24)
	countW := 6
	headW := 6
	sepW := 1
	descW := w - nameW - countW - headW - (sepW * 3)
	if descW < 12 {
		descW = 12
		nameW = w - descW - countW - headW - (sepW * 3)
		if nameW < 10 {
			nameW = 10
		}
	}
	nameSt := lipgloss.NewStyle().Foreground(gFg).Bold(true)
	descSt := lipgloss.NewStyle().Foreground(gDim)
	if f.Stale {
		nameSt = lipgloss.NewStyle().Foreground(gDim)
	}
	if sel {
		nameSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
		descSt = lipgloss.NewStyle().Foreground(gSelFg)
	}
	name := nameSt.Render(tuiPad(tuiCut(f.Title, nameW), nameW))
	desc := f.Description
	if desc == "" {
		desc = "no description"
	}
	descCell := descSt.Render(tuiPad(tuiCut(desc, descW), descW))
	opens := lipgloss.NewStyle().Foreground(gBlue).Render(tuiPad("×"+strconvItoa(f.Count), countW))
	heads := lipgloss.NewStyle().Foreground(gDim).Render(tuiPad("#"+strconvItoa(f.HeadingCnt), headW))
	if sel {
		opens = lipgloss.NewStyle().Foreground(gSelFg).Render(tuiPad("×"+strconvItoa(f.Count), countW))
		heads = lipgloss.NewStyle().Foreground(gSelFg).Render(tuiPad("#"+strconvItoa(f.HeadingCnt), headW))
	}
	line := name + descCell + opens + heads
	line = fitLine(line, w)
	if sel {
		line = lipgloss.NewStyle().Background(gSelBg).Render(line)
	} else if zebra {
		line = lipgloss.NewStyle().Background(gZebra).Render(line)
	}
	return line
}

func (m model) renderTreeNode(node *treeNode, lastStack []bool, last bool, selected bool, w int) string {
	var prefix strings.Builder
	for _, isLast := range lastStack {
		if isLast {
			prefix.WriteString("  ")
		} else {
			prefix.WriteString(lipgloss.NewStyle().Foreground(gFaint).Render("│ "))
		}
	}
	if len(lastStack) > 0 {
		conn := "├ "
		if last {
			conn = "└ "
		}
		prefix.WriteString(lipgloss.NewStyle().Foreground(gFaint).Render(conn))
	}
	glyph := treeGlyph(node)
	glyphW := tw(glyph) + 1
	connW := tw(prefix.String())

	var glyphSt, textSt, connSt lipgloss.Style
	connSt = lipgloss.NewStyle().Foreground(gFaint)
	glyphSt = lipgloss.NewStyle().Foreground(gBlue).Bold(true)
	textSt = lipgloss.NewStyle().Foreground(gFg).Bold(true)
	if node.Kind != "heading" {
		textSt = lipgloss.NewStyle().Foreground(gDim)
		glyphSt = lipgloss.NewStyle().Foreground(gDim)
	}
	if node.Kind == "sub" {
		textSt = lipgloss.NewStyle().Foreground(gFaint)
		glyphSt = lipgloss.NewStyle().Foreground(gFaint)
	}
	if selected {
		glyphSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
		textSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
		connSt = lipgloss.NewStyle().Foreground(lipgloss.Color("#1f6feb"))
	}

	prefixStr := connSt.Render(prefix.String())
	glyphStr := glyphSt.Render(glyph)
	rawText := node.Text
	maxTextW := w - connW - glyphW
	if maxTextW < 1 {
		maxTextW = 1
	}

	if selected {
		selW := clampInt((w*35)/100, 20, maxInt(20, w-connW))
		selTextW := selW - glyphW
		if selTextW < 1 {
			selTextW = 1
		}

		part1, part2 := splitTextAtWidth(rawText, selTextW)
		selBlock := lipgloss.NewStyle().Background(gSelBg).Render(glyphStr + " " + textSt.Render(part1))

		var remStr string
		if part2 != "" {
			remW := maxTextW - selTextW
			if remW > 0 {
				remStr = textSt.Render(tuiCut(part2, remW))
			}
		}

		line := prefixStr + selBlock + remStr
		return fitLine(line, w)
	}

	cutText := tuiCut(rawText, maxTextW)
	textStr := textSt.Render(cutText)
	line := prefixStr + glyphStr + " " + textStr
	line = fitLine(line, w)
	if node.Kind == "heading" {
		line = lipgloss.NewStyle().Background(gHeadBg).Render(line)
	}
	return line
}

func treeGlyph(node *treeNode) string {
	switch node.Kind {
	case "heading":
		if len(node.Children) > 0 {
			return "●"
		}
		return "○"
	case "task":
		if node.Bullet == '+' {
			if len(node.Children) > 0 {
				return "▸"
			}
			return "▹"
		}
		return "•"
	default:
		if node.Parent != nil && node.Parent.Bullet == '+' {
			return "◦"
		}
		return "·"
	}
}

func (m model) treeFooter(w int) string {
	left := tStatL.Render(m.treeStatusText())
	right := tStatR.Render(m.treePosText())
	gap := w - tw(left) - tw(right)
	if gap < 1 {
		gap = 1
	}
	status := fitLine(left+strings.Repeat(" ", gap)+right, w)
	return status + "\n" + fitLine(m.treeKeyHints(), w)
}

func (m model) treeStatusText() string {
	node := m.currentNode()
	if node == nil {
		return "ready"
	}
	if node.Kind == "root" {
		if f, ok := m.currentFile(); ok {
			path := f.FullPath
			if home, err := os.UserHomeDir(); err == nil {
				path = strings.Replace(path, home, "~", 1)
			}
			return tuiCut(path, 60)
		}
		return "ready"
	}
	root := rootOf(node)
	if root == nil {
		return "ready"
	}
	return fmt.Sprintf("%s · %s", root.File.Title, tuiCut(node.Text, 48))
}

func (m model) treePosText() string {
	real := len(m.focus)
	if real < 0 {
		real = 0
	}
	pos := m.selected + 1
	if pos > real {
		pos = real
	}
	if real == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", pos, real)
}

func (m model) treeKeyHints() string {
	hints := [][2]string{
		{"j k", "move"}, {"g G", "jump"}, {"1-9", "root"}, {"l h", "open/close"},
		{"o", "open"}, {"O", "editor"}, {"a", "add"}, {"p", "pin"},
		{"P", "pins"}, {"t", "tasks"}, {"/", "filter"}, {":", "palette"},
		{"s", "sort"}, {"r", "refresh"}, {"y", "yank"}, {"?", "help"}, {"q", "quit"},
	}
	var sb strings.Builder
	for i, h := range hints {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(tKbd.Render(h[0]))
		sb.WriteString(" ")
		sb.WriteString(tHint.Render(h[1]))
	}
	return sb.String()
}

// ============================================================
//  MASTER VIEW — manual box model (no wrap possible)
// ============================================================

func (m model) View() string {
	if m.width < 40 || m.height < 12 {
		return lipgloss.NewStyle().Foreground(gDim).Background(gCanvas).Render("  jdo — enlarge the terminal…")
	}
	CW := clampInt(m.width-8, 30, 160)
	const headerH, footerH = 3, 2
	contentH := m.height - 2 - 2 - headerH - footerH
	if contentH < 3 {
		contentH = 3
	}

	header := m.treeHeader(CW)
	var clines []string
	switch m.mode {
	case "editor", "help", "pins", "palette":
		clines = padBlock(m.renderModal(CW, contentH), contentH, CW)
	case "tasks":
		clines = m.windowLines(m.tasksLines(CW), m.tasksSel, contentH, m.scroll, 0, 0, CW)
	default:
		clines = m.treeLines(CW, contentH)
	}
	footer := m.treeFooter(CW)

	inner := header + "\n" + strings.Join(clines, "\n") + "\n" + footer
	var margined []string
	for _, ln := range strings.Split(inner, "\n") {
		margined = append(margined, "  "+tuiPad(ln, CW)+"  ")
	}
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(gBorder).
		Render(strings.Join(margined, "\n"))

	screen := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
	return lipgloss.NewStyle().Background(gCanvas).Render(screen)
}

// ============================================================
//  HEADER
// ============================================================

func (m model) renderHeader(w int) string {
	if m.mode == "drill" && m.full != nil {
		return m.renderDrillHeader(w)
	}
	badge := tBadge.Render(" JDO ")
	titleTxt := tTitle.Render("  Notes Commander")
	rightInfo := fmt.Sprintf("%d notes", len(m.files))
	if m.config.UI.ShowClock {
		rightInfo += " · " + m.clock.Format("15:04 · Mon 02 Jan")
	}
	right := tHD.Render(rightInfo)
	left := badge + titleTxt
	gap := w - tw(left) - tw(right)
	if gap < 1 {
		gap = 1
	}
	line1 := fitLine(left+strings.Repeat(" ", gap)+right, w)

	path := m.config.Core.NotesDir
	if home, err := os.UserHomeDir(); err == nil {
		path = strings.Replace(path, home, "~", 1)
	}
	pathTxt := tHD.Render(path)
	right2 := tHD.Render("sort: " + m.config.Core.SortBy)
	if m.query != "" {
		right2 = lipgloss.NewStyle().Foreground(gBlue).Render(fmt.Sprintf("filter: %q", m.query))
	}
	if m.toast != "" && time.Now().Before(m.toastExp) {
		right2 = lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render(m.toast)
	}
	gap2 := w - tw(pathTxt) - tw(right2)
	if gap2 < 1 {
		gap2 = 1
	}
	line2 := fitLine(pathTxt+strings.Repeat(" ", gap2)+right2, w)
	return line1 + "\n" + line2
}

func (m model) renderDrillHeader(w int) string {
	f := m.full.note
	left := lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render("▌ ") +
		tTitle.Render(tuiCut(f.Title, w-30))
	right := tHD.Render(fmt.Sprintf("%d headings", f.HeadingCnt))
	gap := w - tw(left) - tw(right)
	if gap < 1 {
		gap = 1
	}
	line1 := fitLine(left+strings.Repeat(" ", gap)+right, w)
	line2 := fitLine(tHD.Render(f.Filename)+"  "+
		lipgloss.NewStyle().Foreground(gBlue).Render(fmt.Sprintf("drill · L%d · %d/%d/%d",
			m.full.level, m.full.headCount, m.full.taskCount, m.full.subCount)), w)
	return line1 + "\n" + line2
}

// ============================================================
//  MAIN LIST — rows + floating borderless overlay card
// ============================================================

func (m *model) mainLines(w, h int) []string {
	rows, lineRow := m.rowLines(w)
	total := len(rows)
	if total == 0 {
		msg := "no notes — create one with: jdo new <name>"
		if m.query != "" {
			msg = fmt.Sprintf("no match for %q — esc to clear", m.query)
		}
		m.visRows = m.visRows[:0]
		empty := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center, tEmpty.Render("◌"), "", tHD.Render(msg)))
		return padBlock(empty, h, w)
	}

	var card []string
	cardOpen := m.expandedRow == m.selected && m.inline != nil
	if cardOpen {
		card = m.cardLines(m.inline, w)
	}

	// early scrolling: keep the cursor in a comfortable middle band
	base := maxInt(1, h/3)
	if maxb := (h - 1) / 2; base > maxb {
		base = maxb
	}
	topRes, bottomRes := base, base
	if cardOpen {
		bottomRes = maxInt(bottomRes, len(card)+1)
	}

	selLine := -1
	for i, fr := range lineRow {
		if fr == m.selected {
			selLine = i
			break
		}
	}
	off := windowOff(selLine, total, h, bottomRes, topRes, m.offset)
	m.offset = off

	vis := make([]string, h)
	for i := 0; i < h; i++ {
		if off+i < total {
			vis[i] = rows[off+i]
		} else {
			vis[i] = fitLine("", w)
		}
	}
	m.visRows = m.visRows[:0]
	for i := 0; i < h; i++ {
		if idx := off + i; idx < total && lineRow[idx] >= 0 {
			m.visRows = append(m.visRows, lineRow[idx])
		}
	}

	// paint the floating card over the rows around the selection
	if cardOpen && selLine >= 0 {
		selPos := selLine - off
		if selPos >= 0 && selPos < h {
			start := selPos + 1
			if h-(selPos+1) < len(card) {
				if selPos >= len(card) {
					start = selPos - len(card) // not enough room below -> above
				}
			}
			for i, cl := range card {
				if pos := start + i; pos >= 0 && pos < h {
					vis[pos] = cl
				}
			}
		}
	}
	return vis
}

func (m *model) rowLines(w int) ([]string, []int) {
	pc := m.pinnedCount()
	hasSep := pc > 0 && pc < len(m.filtered)
	var rows []string
	var lineRow []int
	rowsSeen := 0
	push := func(s string, row int) {
		rows = append(rows, s)
		lineRow = append(lineRow, row)
	}
	for i, f := range m.filtered {
		if hasSep && i == pc {
			push(m.renderSepLine(w), -1)
		}
		zebra := rowsSeen%2 == 1
		sel := i == m.selected
		push(m.renderRow(f, sel, zebra && !sel, w), i)
		rowsSeen++
	}
	return rows, lineRow
}

func (m model) pinnedCount() int {
	n := 0
	for _, f := range m.filtered {
		if m.isPinned(f.key) {
			n++
		} else {
			break
		}
	}
	return n
}

func (m model) renderSepLine(w int) string {
	label := "pinned"
	return fitLine(lipgloss.NewStyle().Foreground(gFaint).Render("◆ "+label), w)
}

func (m model) renderRow(f FileInfo, sel bool, zebra bool, w int) string {
	indW, divW, metaW := 2, 3, 10
	nameW := clampInt((w-indW-divW-metaW)*38/100, 8, 22)
	descW := w - indW - divW - metaW - nameW
	if descW < 4 {
		descW = 4
		nameW = w - indW - divW - metaW - descW
	}

	indRaw := " "
	indSt := lipgloss.NewStyle()
	if sel {
		indRaw = "▌"
		indSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
	} else if m.isPinned(f.key) {
		indRaw = "▌"
		indSt = lipgloss.NewStyle().Foreground(gBlue)
	}
	indCell := indSt.Render(tuiPad(indRaw, indW))

	nameSt := lipgloss.NewStyle().Foreground(gFg).Bold(true)
	if sel {
		nameSt = lipgloss.NewStyle().Foreground(gSelFg).Bold(true)
	} else if f.Stale {
		nameSt = lipgloss.NewStyle().Foreground(gDim)
	}
	nameCell := nameSt.Render(tuiPad(tuiCut(f.Title, nameW), nameW))

	divSt := lipgloss.NewStyle().Foreground(gBorder)
	if sel {
		divSt = lipgloss.NewStyle().Foreground(gSelFg)
	}
	divCell := divSt.Render(tuiPad(" ", divW))
	descBody := f.Description
	if descBody == "" {
		descBody = "no description"
	}
	descSt := lipgloss.NewStyle().Foreground(gDim)
	hashSt := lipgloss.NewStyle().Foreground(gFaint)
	if sel {
		descSt = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd9e5"))
		hashSt = lipgloss.NewStyle().Foreground(gSelFg)
	}
	descCell := hashSt.Render("# ") + descSt.Render(tuiPad(tuiCut(descBody, descW-2), descW-2))

	// meta: ASCII-safe count + a glyph-free pill for open tasks (no wrap risk)
	cnt := lipgloss.NewStyle().Foreground(gBlue).Render(fmt.Sprintf("×%d", f.Count))
	if sel {
		cnt = lipgloss.NewStyle().Foreground(gSelFg).Render(fmt.Sprintf("×%d", f.Count))
	}
	meta := cnt
	if f.OpenTasks > 0 {
		pill := lipgloss.NewStyle().Background(gPillBg).Foreground(gBlue).Bold(true).
			Render(fmt.Sprintf(" %d ", f.OpenTasks))
		if sel {
			pill = lipgloss.NewStyle().Background(lipgloss.Color("#3a6ea5")).Foreground(gSelFg).Bold(true).
				Render(fmt.Sprintf(" %d ", f.OpenTasks))
		}
		meta = cnt + " " + pill
	}
	metaCell := tuiPad(meta, metaW)

	line := indCell + nameCell + divCell + descCell + metaCell
	line = fitLine(line, w)
	if sel {
		line = lipgloss.NewStyle().Background(gSelBg).Render(line)
	} else if zebra {
		line = lipgloss.NewStyle().Background(gZebra).Render(line)
	}
	return line
}

// ============================================================
//  INLINE CARD — borderless, nested, conceptual glyphs
// ============================================================

func (m model) cardLines(ds *drillState, w int) []string {
	dls := buildDrillLines(ds.drill, ds.level, ds.headStart, ds.headCount,
		ds.taskStart, ds.taskCount, ds.subStart, ds.subCount)
	out := make([]string, 0, len(dls))
	for _, dl := range dls {
		out = append(out, fitLine("  "+drillLineText(dl), w))
	}
	if len(out) == 0 {
		out = append(out, fitLine("  "+tMHint.Render("(no headings)"), w))
	}
	return out
}

func drillLineText(dl drillLine) string {
	if dl.depth == 0 {
		return lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render("►") + " " +
			lipgloss.NewStyle().Foreground(gFg).Bold(true).Render(dl.text)
	}
	var g strings.Builder
	for k := 0; k < dl.depth; k++ {
		if k < len(dl.ancLast) && dl.ancLast[k] {
			g.WriteString("  ")
		} else {
			g.WriteString(lipgloss.NewStyle().Foreground(gFaint).Render("│ "))
		}
	}
	if dl.ownLast {
		g.WriteString(lipgloss.NewStyle().Foreground(gFaint).Render("└ "))
	} else {
		g.WriteString(lipgloss.NewStyle().Foreground(gFaint).Render("├ "))
	}
	bulSt := lipgloss.NewStyle().Foreground(gDim)
	txtSt := lipgloss.NewStyle().Foreground(gDim)
	if dl.depth == 1 && dl.bullet == '+' {
		bulSt = lipgloss.NewStyle().Foreground(gBlue).Bold(true)
	}
	if dl.depth == 2 {
		bulSt = lipgloss.NewStyle().Foreground(gFaint)
		txtSt = lipgloss.NewStyle().Foreground(gFaint)
	}
	return g.String() + bulSt.Render(drillGlyph(dl)) + " " + txtSt.Render(dl.text)
}

func drillGlyph(dl drillLine) string {
	switch dl.depth {
	case 0:
		return "◆"
	case 1:
		if dl.bullet == '+' {
			return "▸"
		}
		return "•"
	default:
		if dl.pbul == '+' {
			return "◦"
		}
		return "·"
	}
}

// ============================================================
//  FULL-SCREEN DRILL PAGE
// ============================================================

func (m model) drillLines(w int) []string {
	ds := m.full
	dls := buildDrillLines(ds.drill, ds.level, 0, ds.headCount, 0, ds.taskCount, 0, ds.subCount)
	var lines []string
	first := true
	for _, dl := range dls {
		if dl.depth == 0 && !first {
			lines = append(lines, fitLine("", w))
		}
		first = false
		if dl.depth == 0 {
			lines = append(lines, lipgloss.NewStyle().Background(gHeadBg).Render(tuiPad("  "+drillLineText(dl), w)))
		} else {
			lines = append(lines, fitLine("  "+drillLineText(dl), w))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, fitLine("  (no headings in this note)", w))
	}
	return lines
}

// ============================================================
//  FOOTER
// ============================================================

func (m model) renderFooter(w int) string {
	return m.renderStatus(w) + "\n" + m.renderLast(w)
}

func (m model) renderStatus(w int) string {
	left := tStatL.Render(m.statusText())
	right := ""
	switch m.mode {
	case "drill":
		if m.full != nil {
			right = tStatR.Render(fmt.Sprintf("L%d · %d/%d/%d", m.full.level, m.full.headCount, m.full.taskCount, m.full.subCount))
		}
	case "tasks":
		right = tStatR.Render(fmt.Sprintf("%d open", len(m.tasksSnap)))
	default:
		if len(m.filtered) > 0 && m.selected < len(m.filtered) {
			f := m.filtered[m.selected]
			right = tStatR.Render(fmt.Sprintf("%s · %d/%d", relTime(f.LastOpened), m.selected+1, len(m.filtered)))
		}
	}
	gap := w - tw(left) - tw(right)
	if gap < 1 {
		gap = 1
	}
	return fitLine(left+strings.Repeat(" ", gap)+right, w)
}

func (m model) renderLast(w int) string {
	var s string
	switch m.mode {
	case "search":
		cur := " "
		if m.cursorVisible {
			cur = "█"
		}
		s = tSearchPr.Render("/ ") + tSearchTx.Render(m.searchInput+cur)
	case "add":
		cur := " "
		if m.cursorVisible {
			cur = "█"
		}
		tgt := ""
		if len(m.filtered) > 0 {
			tgt = m.filtered[m.selected].Title
		}
		s = lipgloss.NewStyle().Foreground(gBlue).Bold(true).Render("+ ") +
			tSearchTx.Render(m.addInput+cur) + tHD.Render("  → "+tgt)
	default:
		s = m.renderKeyHints()
	}
	return fitLine(s, w)
}

func (m model) statusText() string {
	if m.expandedRow == m.selected && m.inline != nil {
		ds := m.inline
		return fmt.Sprintf("peek L%d · %d headings · %d tasks · %d subs", ds.level, ds.headCount, ds.taskCount, ds.subCount)
	}
	if m.mode == "drill" && m.full != nil {
		return "space deeper · l/h count · esc back · ⏎ open at heading"
	}
	if m.mode == "tasks" {
		return "x done · u reopen · ⏎ jump · esc close"
	}
	if len(m.filtered) > 0 && m.selected < len(m.filtered) {
		d := m.filtered[m.selected].Description
		if d == "" {
			d = "no description"
		}
		return tuiCut(strings.TrimPrefix(d, "# "), 60)
	}
	return "ready"
}

func (m model) renderKeyHints() string {
	var hints [][2]string
	switch m.mode {
	case "drill":
		hints = [][2]string{{"space", "deeper"}, {"l h", "count"}, {"j k", "scroll"}, {"esc", "back"}, {"⏎", "open"}, {"q", "quit"}}
	case "tasks":
		hints = [][2]string{{"j k", "move"}, {"x", "done"}, {"u", "undo"}, {"⏎", "jump"}, {"esc", "close"}}
	default:
		if m.expandedRow == m.selected && m.inline != nil {
			hints = [][2]string{{"l h", "level"}, {"space", "+count"}, {"j k", "scroll"}, {"esc", "close"}, {"q", "quit"}}
		} else {
			hints = [][2]string{{"j k", "move"}, {"g G", "jump"}, {"1-9", "open #"}, {"o", "open"}, {"O", "editor"}, {"/", "find"}, {"space", "page"}, {"l", "peek"}, {"s", "sort"}, {"r", "refresh"}, {"y", "yank"}, {"?", "help"}, {"q", "quit"}}
		}
	}
	var sb strings.Builder
	for i, h := range hints {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(tKbd.Render(h[0]))
		sb.WriteString(" ")
		sb.WriteString(tHint.Render(h[1]))
	}
	return sb.String()
}
