// tui_drill.go — outline parser + indented drill-line builder with windowed
// paging. Powers both the inline overlay card (l) and the full page (space).
package main

import (
	"bufio"
	"os"
	"strings"
)

type DrillNode struct {
	Bullet   byte
	Text     string
	Line     int
	Children []*DrillNode
}

type DrillHeading struct {
	Name  string
	Line  int
	Roots []*DrillNode
}

type Drill struct {
	Headings []DrillHeading
}

type rawItem struct {
	indent int
	bullet byte
	text   string
	line   int
}

func parseDrill(path string) *Drill {
	d := &Drill{}
	f, err := os.Open(path)
	if err != nil {
		return d
	}
	defer f.Close()
	fence := false
	var cur *DrillHeading
	var items []rawItem
	flush := func() {
		if cur != nil {
			cur.Roots = buildTree(items)
			d.Headings = append(d.Headings, *cur)
		}
		items = nil
	}
	sc := bufio.NewScanner(f)
	lnNo := 0
	for sc.Scan() {
		lnNo++
		raw := sc.Text()
		t := strings.TrimSpace(raw)
		if strings.HasPrefix(t, "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if strings.HasPrefix(t, "#") && len(t) > 1 && t[1] == ' ' && strings.TrimSpace(t[2:]) != "" {
			flush()
			h := DrillHeading{Name: strings.TrimSpace(t[2:]), Line: lnNo}
			cur = &h
			continue
		}
		if cur == nil {
			continue
		}
		indent := leadIndent(raw)
		rest := strings.TrimLeft(raw, " \t")
		if rest == "" {
			continue
		}
		b := rest[0]
		if b == '+' || b == '-' || b == '*' {
			txt := strings.TrimSpace(rest[1:])
			if txt == "" {
				continue
			}
			items = append(items, rawItem{indent, b, txt, lnNo})
		}
	}
	flush()
	return d
}

func leadIndent(s string) int {
	n := 0
	for _, c := range s {
		if c == ' ' {
			n++
		} else if c == '\t' {
			n += 2
		} else {
			break
		}
	}
	return n
}

func buildTree(items []rawItem) []*DrillNode {
	var roots []*DrillNode
	type frame struct {
		indent int
		node   *DrillNode
	}
	var stack []frame
	for _, it := range items {
		node := &DrillNode{Bullet: it.bullet, Text: it.text, Line: it.line}
		for len(stack) > 0 && stack[len(stack)-1].indent >= it.indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			stack[len(stack)-1].node.Children = append(stack[len(stack)-1].node.Children, node)
		}
		stack = append(stack, frame{it.indent, node})
	}
	return roots
}

func winBullet(nodes []*DrillNode) byte {
	hasP, hasM := false, false
	for _, n := range nodes {
		if n.Bullet == '+' {
			hasP = true
		}
		if n.Bullet == '-' {
			hasM = true
		}
	}
	if hasP {
		return '+'
	}
	if hasM {
		return '-'
	}
	if len(nodes) > 0 {
		return nodes[0].Bullet
	}
	return '-'
}

func filterBy(nodes []*DrillNode, b byte) []*DrillNode {
	var out []*DrillNode
	for _, n := range nodes {
		if n.Bullet == b {
			out = append(out, n)
		}
	}
	return out
}

func sliceWindow(nodes []*DrillNode, start, n int) []*DrillNode {
	if start >= len(nodes) || n <= 0 {
		return nil
	}
	end := start + n
	if end > len(nodes) {
		end = len(nodes)
	}
	return nodes[start:end]
}

type shownHead struct {
	h   DrillHeading
	win []*DrillNode
}

func shownHeads(d *Drill, headStart, hc int) []shownHead {
	end := headStart + hc
	if end > len(d.Headings) {
		end = len(d.Headings)
	}
	if end < headStart {
		end = headStart
	}
	out := make([]shownHead, 0, end-headStart)
	for i := headStart; i < end; i++ {
		h := d.Headings[i]
		top := h.Roots
		out = append(out, shownHead{h: h, win: filterBy(top, winBullet(top))})
	}
	return out
}

func winSubs(t *DrillNode) []*DrillNode { return filterBy(t.Children, winBullet(t.Children)) }

type drillLine struct {
	depth   int
	ancLast []bool
	ownLast bool
	text    string
	bullet  byte
	pbul    byte
	line    int
}

func buildDrillLines(d *Drill, level, headStart, hc, taskStart, tc, subStart, sc int) []drillLine {
	var out []drillLine
	for _, sh := range shownHeads(d, headStart, hc) {
		out = append(out, drillLine{depth: 0, text: sh.h.Name, line: sh.h.Line})
		if level >= 2 {
			st := sliceWindow(sh.win, taskStart, tc)
			for ti, t := range st {
				tLast := ti == len(st)-1
				out = append(out, drillLine{depth: 1, ancLast: []bool{false}, ownLast: tLast, text: t.Text, bullet: t.Bullet, line: t.Line})
				if level >= 3 {
					ss := sliceWindow(winSubs(t), subStart, sc)
					for si, s := range ss {
						sLast := si == len(ss)-1
						out = append(out, drillLine{depth: 2, ancLast: []bool{false, tLast}, ownLast: sLast, text: s.Text, bullet: s.Bullet, pbul: t.Bullet, line: s.Line})
					}
				}
			}
		}
	}
	return out
}

func drillCaps(d *Drill, headStart, hc, tc, sc int) (mh, mt, ms int) {
	mh = len(d.Headings)
	for _, sh := range shownHeads(d, headStart, hc) {
		if len(sh.win) > mt {
			mt = len(sh.win)
		}
		for _, t := range sh.win {
			if len(winSubs(t)) > ms {
				ms = len(winSubs(t))
			}
		}
	}
	return
}

// ---- tree model for the main unified list ----

func buildTrees(files []FileInfo) map[string]*treeNode {
	trees := make(map[string]*treeNode, len(files))
	for _, f := range files {
		d := parseDrill(f.FullPath)
		trees[f.key] = buildTreeNode(f, d)
	}
	return trees
}

func buildTreeNode(f FileInfo, d *Drill) *treeNode {
	root := &treeNode{
		Kind:     "root",
		Text:     f.Title,
		FileKey:  f.key,
		FileName: f.Filename,
		File:     f,
	}
	for _, h := range d.Headings {
		hn := &treeNode{
			Kind:     "heading",
			Text:     h.Name,
			Line:     h.Line,
			FileKey:  f.key,
			FileName: f.Filename,
			File:     f,
			Parent:   root,
		}
		for _, child := range h.Roots {
			hn.Children = append(hn.Children, cloneDrillNode(child, f, hn, 1))
		}
		root.Children = append(root.Children, hn)
	}
	return root
}

func cloneDrillNode(src *DrillNode, f FileInfo, parent *treeNode, depth int) *treeNode {
	kind := "task"
	if depth > 1 {
		kind = "sub"
	}
	n := &treeNode{
		Kind:     kind,
		Text:     src.Text,
		Line:     src.Line,
		Bullet:   src.Bullet,
		FileKey:  f.key,
		FileName: f.Filename,
		File:     f,
		Parent:   parent,
	}
	for _, child := range src.Children {
		n.Children = append(n.Children, cloneDrillNode(child, f, n, depth+1))
	}
	return n
}

func (m *model) rebuildFocus() {
	m.focus = buildFocusEntries(m.filtered, m.trees)
	if len(m.focus) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.focus) {
		m.selected = len(m.focus) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func buildFocusEntries(files []FileInfo, trees map[string]*treeNode) []focusEntry {
	var out []focusEntry
	for _, f := range files {
		root := trees[f.key]
		if root == nil {
			continue
		}
		out = append(out, focusEntry{Kind: "root", Root: root, Node: root})
		appendVisibleChildren(&out, root, 1)
	}
	return out
}

func appendVisibleChildren(out *[]focusEntry, node *treeNode, depth int) {
	if node == nil || !node.Expanded {
		return
	}
	for _, child := range node.Children {
		*out = append(*out, focusEntry{Kind: "node", Root: rootOf(child), Node: child})
		appendVisibleChildren(out, child, depth+1)
	}
}

func rootOf(node *treeNode) *treeNode {
	cur := node
	for cur != nil && cur.Parent != nil {
		cur = cur.Parent
	}
	return cur
}

func nodeDepth(node *treeNode) int {
	depth := 0
	for cur := node; cur != nil && cur.Parent != nil; cur = cur.Parent {
		depth++
	}
	return depth
}

func (m *model) focusAt(idx int) focusEntry {
	if idx < 0 || idx >= len(m.focus) {
		return focusEntry{Kind: "pad", Pad: true}
	}
	return m.focus[idx]
}

func (m *model) currentRoot() *treeNode {
	if len(m.focus) == 0 || m.selected < 0 || m.selected >= len(m.focus) {
		return nil
	}
	return rootOf(m.focus[m.selected].Node)
}

func (m *model) currentNode() *treeNode {
	if len(m.focus) == 0 || m.selected < 0 || m.selected >= len(m.focus) {
		return nil
	}
	return m.focus[m.selected].Node
}

func (m *model) currentFile() (FileInfo, bool) {
	root := m.currentRoot()
	if root == nil {
		return FileInfo{}, false
	}
	return root.File, true
}

func (m *model) visibleRootPositions() []int {
	var pos []int
	for i, entry := range m.focus {
		if entry.Kind == "root" {
			pos = append(pos, i)
		}
	}
	return pos
}
