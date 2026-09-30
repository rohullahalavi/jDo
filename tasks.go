// tasks.go — task lifecycle: pure cores plus CLI wrappers, and the open-task
// counter used for the TUI badge. A task is an unchecked "- [ ]" checkbox
// anywhere, or a +/- bullet inside a "task-ish" heading.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type taskItem struct {
	file    FileInfo
	heading string
	text    string
	due     string
	line    int // 0-based line index in the file
	done    bool
	state   byte // ' ', 'x', '.', '+', '<', '>', '^', '~', '=', '#', '-', '!', '*', '?'
}

func isTaskHeading(name string) bool {
	n := normalizeTag(name)
	for _, k := range []string{"todo", "today", "task", "important", "check", "plan", "routine", "asap"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// collectFileTasks returns every task in one file (pure).
func collectFileTasks(f FileInfo) []taskItem {
	var out []taskItem
	data, err := os.ReadFile(f.FullPath)
	if err != nil {
		return out
	}
	lines := strings.Split(string(data), "\n")
	cur := ""
	taskish := false
	fence := false
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		if isHeadingLine(ln) {
			cur = headingText(ln)
			taskish = isTaskHeading(cur)
			continue
		}
		if t == "" {
			continue
		}
		// checkbox with any state marker: - [ ] - [x] - [.] - [+] etc
		if strings.HasPrefix(t, "- [") && len(t) >= 5 && t[3] != ' ' {
			state := byte(' ')
			if len(t) >= 5 {
				state = t[3]
			}
			done := state == 'x' || state == 'X'
			rest := t[5:]
			var due string
			if idx := strings.Index(rest, "(📅 "); idx >= 0 {
				end := strings.Index(rest[idx:], ")")
				if end > 0 {
					due = rest[idx+3 : idx+end]
					rest = strings.TrimSpace(rest[:idx] + rest[idx+end+1:])
				}
			}
			text := strings.TrimSpace(rest)
			out = append(out, taskItem{f, cur, text, due, i, done, state})
			continue
		}
		// +/- bullet inside a task-ish heading
		if taskish {
			if _, bul, ok := splitIndentBullet(ln); ok && (bul == "+" || bul == "-") {
				rest := strings.TrimSpace(strings.TrimPrefix(t, bul))
				if rest != "" {
					out = append(out, taskItem{f, cur, rest, "", i, false, ' '})
				}
			}
		}
	}
	return out
}

// countOpenInFile counts open tasks in a file (pure, used at scan time).
func countOpenInFile(path string) int {
	st := FileInfo{FullPath: path}
	n := 0
	for _, it := range collectFileTasks(st) {
		if !it.done {
			n++
		}
	}
	return n
}

// listOpenTasks returns open tasks across files (pure).
func listOpenTasks(files []FileInfo, onlyFile string) []taskItem {
	var out []taskItem
	for _, f := range files {
		if onlyFile != "" {
			base := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename))
			if !strings.EqualFold(f.Filename, onlyFile) && !strings.EqualFold(base, onlyFile) {
				continue
			}
		}
		for _, it := range collectFileTasks(f) {
			if !it.done {
				out = append(out, it)
			}
		}
	}
	return out
}

// toggleDoneAt sets the checkbox state of one line (pure, atomic write).
func toggleDoneAt(path string, line int, done bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if line < 0 || line >= len(lines) {
		return fmt.Errorf("line out of range")
	}
	ln := lines[line]
	ind, _, ok := splitIndentBullet(ln)
	if !ok {
		return fmt.Errorf("not a task line")
	}
	t := strings.TrimSpace(ln)
	_, bul, _ := splitIndentBullet(ln)
	text := stripDecorations(strings.TrimSpace(strings.TrimPrefix(t, bul)))
	if done {
		lines[line] = ind + "- [x] " + text
	} else {
		lines[line] = ind + "- [ ] " + text
	}
	return writeFileAtomic(path, strings.Join(lines, "\n"))
}

// markDoneCore fuzzy-finds a task by text and marks it done (pure).
type DoneResult struct {
	File FileInfo
	Line int
	Ok   bool
}

func markDoneCore(files []FileInfo, query string) DoneResult {
	if query == "" {
		return DoneResult{}
	}
	q := strings.ToLower(query)
	best := -1
	var bestFile FileInfo
	var bestLine int
	for _, f := range files {
		for _, it := range collectFileTasks(f) {
			if it.done {
				continue
			}
			lt := strings.ToLower(it.text)
			s := -1
			switch {
			case lt == q:
				s = 1000
			case strings.Contains(lt, q):
				s = 500
			case subseq(q, lt):
				s = 200
			}
			if s > best {
				best = s
				bestFile = f
				bestLine = it.line
			}
		}
	}
	if best < 0 {
		return DoneResult{}
	}
	if err := toggleDoneAt(bestFile.FullPath, bestLine, true); err != nil {
		return DoneResult{}
	}
	return DoneResult{File: bestFile, Line: bestLine, Ok: true}
}

func runTasks(files []FileInfo, onlyFile string) {
	items := listOpenTasks(files, onlyFile)
	if len(items) == 0 {
		fmt.Println(" no open tasks")
		return
	}
	sort.Slice(items, func(i, j int) bool { return items[i].file.Title < items[j].file.Title })
	fmt.Println(pHead.Render(" Open tasks"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))
	lastFile := ""
	for _, it := range items {
		if it.file.Title != lastFile {
			fmt.Printf("  %s %s\n", pName.Render(it.file.Title), pDim.Render("· "+it.heading))
			lastFile = it.file.Title
		}
		fmt.Printf("    %s %s\n", pFaint.Render("☐"), pText.Render(it.text))
	}
	fmt.Printf("\n %d open task(s)\n", len(items))
}

func runDone(cfg Config, files []FileInfo, query string) {
	res := markDoneCore(files, query)
	if !res.Ok {
		fmt.Printf(" no open task matching %q\n", query)
		return
	}
	fmt.Printf(" %s done in %s line %d\n", pOK.Render("✓"), pName.Render(res.File.Filename), res.Line+1)
	_ = cfg
}

func runCount(files []FileInfo, target string) {
	for _, f := range files {
		if target != "" {
			base := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename))
			if !strings.EqualFold(f.Filename, target) && !strings.EqualFold(base, target) {
				continue
			}
		}
		data, err := os.ReadFile(f.FullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		fmt.Println(pHead.Render(" " + f.Title))
		any := false
		for i, ln := range lines {
			if isHeadingLine(ln) {
				start, end := sectionBounds(lines, i)
				n := 0
				for j := start; j < end; j++ {
					if _, _, ok := splitIndentBullet(lines[j]); ok && !isEmptyBullet(lines[j]) {
						n++
					}
				}
				fmt.Printf("   %s %s\n", pDim.Render(headingText(ln)), pNum.Render(strconv.Itoa(n)))
				any = true
			}
		}
		if !any {
			fmt.Println(pDim.Render("   (no headings)"))
		}
	}
}

func stripDecorations(s string) string {
	s = strings.TrimSpace(s)
	for {
		before := s
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "[ ]")
		s = strings.TrimPrefix(s, "[x]")
		s = strings.TrimPrefix(s, "[X]")
		s = strings.TrimPrefix(s, "[.]")
		s = strings.TrimPrefix(s, "[+]")
		s = strings.TrimPrefix(s, "[<]")
		s = strings.TrimPrefix(s, "[>]")
		s = strings.TrimPrefix(s, "[^]")
		s = strings.TrimPrefix(s, "[~]")
		s = strings.TrimPrefix(s, "[=]")
		s = strings.TrimPrefix(s, "[#]")
		s = strings.TrimPrefix(s, "[-]")
		s = strings.TrimPrefix(s, "[!]")
		s = strings.TrimPrefix(s, "[*]")
		s = strings.TrimPrefix(s, "[?]")
		s = strings.TrimSpace(s)
		if len(s) >= 2 && (s[0] == 'P' || s[0] == 'p') && s[1] >= '0' && s[1] <= '9' {
			j := 1
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			s = strings.TrimSpace(s[j:])
			continue
		}
		r := []rune(s)
		k := 0
		for k < len(r) && !(unicodeIsLetterOrDigit(r[k]) || r[k] == '"' || r[k] == '\'') {
			k++
		}
		if k > 0 {
			s = string(r[k:])
			continue
		}
		if s == before {
			break
		}
	}
	return strings.TrimSpace(s)
}

func unicodeIsLetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r > 127
}

// keep bufio referenced for files that only import it transitively
var _ = bufio.Scanner{}

func taskStateIcon(state byte) string {
	switch state {
	case 'x', 'X':
		return "☑"
	case '.':
		return "☐"
	case '+':
		return "➕"
	case '<':
		return "📅"
	case '>':
		return "⏳"
	case '^':
		return "⭐"
	case '~':
		return "▶️"
	case '=':
		return "🔄"
	case '#':
		return "⏳"
	case '-':
		return "❌"
	case '!':
		return "⚡"
	case '*':
		return "⭐"
	case '?':
		return "❓"
	default:
		return "☐"
	}
}

func runNext(files []FileInfo, args []string) {
	hours := 1
	if len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			hours = n
		}
	}
	now := time.Now()
	deadline := now.Add(time.Duration(hours) * time.Hour)
	var items []taskItem
	for _, f := range files {
		for _, it := range collectFileTasks(f) {
			if it.done || it.due == "" {
				continue
			}
			if t, ok := parseTaskDue(it.due); ok {
				if t.After(now) && t.Before(deadline) {
					items = append(items, it)
				}
			}
		}
	}
	if len(items) == 0 {
		fmt.Printf(" no tasks due in the next %d hour(s)\n", hours)
		return
	}
	fmt.Println(pHead.Render(fmt.Sprintf(" Due in next %d hour(s)", hours)))
	fmt.Println(pRule.Render(strings.Repeat("─", 40)))
	lastFile := ""
	for _, it := range items {
		if it.file.Title != lastFile {
			fmt.Printf(" %s %s\n", pName.Render(it.file.Title), pDim.Render("· "+it.heading))
			lastFile = it.file.Title
		}
		fmt.Printf(" %s %s (📅 %s)\n", pFaint.Render(taskStateIcon(it.state)), pText.Render(it.text), pDim.Render(it.due))
	}
	fmt.Printf("\n %d task(s)\n", len(items))
}

func parseTaskDue(s string) (time.Time, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	now := time.Now()
	const dayFmt = "Mon 02 Jan"
	switch s {
	case "today":
		return now, true
	case "tomorrow":
		return now.AddDate(0, 0, 1), true
	case "next week":
		return now.AddDate(0, 0, 7), true
	}
	if strings.HasPrefix(s, "+") {
		rest := s[1:]
		if strings.HasSuffix(rest, "d") {
			if n, err := strconv.Atoi(strings.TrimSuffix(rest, "d")); err == nil {
				return now.AddDate(0, 0, n), true
			}
		}
		if strings.HasSuffix(rest, "w") {
			if n, err := strconv.Atoi(strings.TrimSuffix(rest, "w")); err == nil {
				return now.AddDate(0, 0, 7*n), true
			}
		}
	}
	days := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	for i, d := range days {
		if s == d {
			wd := time.Weekday(i)
			add := (int(wd) - int(now.Weekday()) + 7) % 7
			if add == 0 {
				add = 7
			}
			return now.AddDate(0, 0, add), true
		}
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	if t, err := time.Parse(dayFmt, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func runNexts(files []FileInfo, args []string) {
	period := "day"
	if len(args) > 0 {
		switch args[0] {
		case "d", "day":
			period = "day"
		case "w", "week":
			period = "week"
		case "m", "month":
			period = "month"
		case "y", "year":
			period = "year"
		default:
			fmt.Println(" usage: jdo nexts [d|w|m|y]")
			return
		}
	}
	now := time.Now()
	var end time.Time
	switch period {
	case "day":
		end = now.AddDate(0, 0, 1)
	case "week":
		end = now.AddDate(0, 0, 7)
	case "month":
		end = now.AddDate(0, 1, 0)
	case "year":
		end = now.AddDate(1, 0, 0)
	}
	var items []taskItem
	for _, f := range files {
		for _, it := range collectFileTasks(f) {
			if it.done || it.due == "" {
				continue
			}
			if t, ok := parseTaskDue(it.due); ok {
				if t.After(now) && t.Before(end) {
					items = append(items, it)
				}
			}
		}
	}
	if len(items) == 0 {
		fmt.Printf(" no tasks due in the next %s\n", period)
		return
	}
	title := "This week"
	switch period {
	case "day":
		title = "Today"
	case "week":
		title = "This week"
	case "month":
		title = "This month"
	case "year":
		title = "This year"
	}
	fmt.Println(pHead.Render(" " + title))
	fmt.Println(pRule.Render(strings.Repeat("─", 40)))
	lastFile := ""
	for _, it := range items {
		if it.file.Title != lastFile {
			fmt.Printf(" %s %s\n", pName.Render(it.file.Title), pDim.Render("· "+it.heading))
			lastFile = it.file.Title
		}
		fmt.Printf(" %s %s (📅 %s)\n", pFaint.Render(taskStateIcon(it.state)), pText.Render(it.text), pDim.Render(it.due))
	}
	fmt.Printf("\n %d task(s)\n", len(items))
}
