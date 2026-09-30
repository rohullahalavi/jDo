// commands.go — primary CLI commands (quick launch, capture, listings).
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

// runQuickLaunch is what bare `jdo` does: a plain numbered top-N list.
// No TUI, the screen is never cleared.
func runQuickLaunch(cfg Config, hist HistoryData, histPath string, files []FileInfo) {
	size := cfg.Core.QuickListSize
	if size < 1 {
		size = 5
	}
	if size > 9 {
		size = 9
	}
	if len(files) == 0 {
		fmt.Println(" no notes found in:", cfg.Core.NotesDir)
		fmt.Println(" create one with:   jdo new <name>")
		return
	}
	limit := len(files)
	if limit > size {
		limit = size
	}
	top := files[:limit]

	title := cfg.Core.QuickListTitle
	if title == "" {
		title = "Top notes"
	}
	fmt.Println(pHead.Render(" " + title))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))
	for i, f := range top {
		fmt.Printf("  %s  %s\n", pNum.Render(strconv.Itoa(i+1)), pName.Render(f.Title))
	}
	fmt.Println()
	for {
		fmt.Print(pPrompt.Render(fmt.Sprintf(" open 1–%d (q to cancel) › ", limit)))
		line := readLine()
		if line == "" || line == "q" || line == "Q" {
			return
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > limit {
			fmt.Println(pFail.Render(" invalid — type a number between 1 and " + strconv.Itoa(limit)))
			continue
		}
		openTracked(cfg, hist, histPath, top[n-1])
		return
	}
}

func openLastOpened(cfg Config, hist HistoryData, histPath string) {
	bestName := ""
	var bestTime time.Time
	for name, st := range hist.Files {
		if st.LastOpened.After(bestTime) {
			bestTime = st.LastOpened
			bestName = name
		}
	}
	if bestName == "" {
		fmt.Println(" no history yet — open a note first")
		return
	}
	path := filepath.Join(cfg.Core.NotesDir, bestName)
	if _, err := os.Stat(path); err != nil {
		fmt.Printf(" last note %q no longer exists\n", bestName)
		return
	}
	fmt.Printf(" reopening %s %s\n", pName.Render(formatTitle(bestName)), pDim.Render("("+relTime(bestTime)+")"))
	recordHistory(hist, histPath, bestName, cfg.Core.MaxHistoryItems)
	openInEditor(path, cfg.Core.DefaultEditor)
}

func createNewNote(cfg Config, hist HistoryData, histPath, name string) {
	ext := cfg.Core.DefaultExtension
	if ext == "" {
		ext = ".md"
	}
	if !strings.HasSuffix(strings.ToLower(name), ext) {
		name += ext
	}
	path := filepath.Join(cfg.Core.NotesDir, name)
	if _, err := os.Stat(path); err == nil {
		fmt.Printf(" %s already exists — opening it\n", pName.Render(formatTitle(name)))
		openInEditor(path, cfg.Core.DefaultEditor)
		return
	}
	content := templateFill(cfg.Core.TemplateContent, formatTitle(name))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		die("could not create note: %v", err)
	}
	fmt.Printf(" %s created %s\n", pOK.Render("✓"), pName.Render(formatTitle(name)))
	recordHistory(hist, histPath, name, cfg.Core.MaxHistoryItems)
	openInEditor(path, cfg.Core.DefaultEditor)
}

func createOrOpenDailyNote(cfg Config, hist HistoryData, histPath string) {
	name := time.Now().Format("2006-01-02")
	createNewNote(cfg, hist, histPath, name)
}

func templateFill(tpl, title string) string {
	out := strings.ReplaceAll(tpl, "{{title}}", title)
	out = strings.ReplaceAll(out, "{{date}}", time.Now().Format("2006-01-02 15:04"))
	return out
}

func fullTextSearch(cfg Config, files []FileInfo, query string) {
	if query == "" {
		fmt.Println(" usage: jdo search <text>")
		return
	}
	q := query
	if !cfg.Core.CaseSensitiveSearch {
		q = strings.ToLower(q)
	}
	hits := 0
	for _, f := range files {
		fdata, err := os.Open(f.FullPath)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(fdata)
		for scanner.Scan() {
			line := scanner.Text()
			cmp := line
			if !cfg.Core.CaseSensitiveSearch {
				cmp = strings.ToLower(cmp)
			}
			if strings.Contains(cmp, q) {
				snippet := strings.TrimSpace(line)
				snippet = strings.TrimLeft(snippet, "#")
				snippet = strings.TrimSpace(snippet)
				fmt.Printf("  %s  %s\n",
					pName.Render(padRight(f.Title, 22)),
					pDim.Render("# "+truncate(snippet, 52)))
				hits++
				break
			}
		}
		fdata.Close()
	}
	if hits == 0 {
		fmt.Printf(" no note contains %q\n", query)
	} else {
		fmt.Printf("\n %d note(s) match\n", hits)
	}
}

func runList(files []FileInfo) {
	if len(files) == 0 {
		fmt.Println(" no notes found")
		return
	}
	sorted := make([]FileInfo, len(files))
	copy(sorted, files)
	sortByAlpha(sorted)
	fmt.Println(pHead.Render(" All notes"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))
	for _, f := range sorted {
		desc := f.Description
		if desc == "" {
			desc = "no description"
		}
		fmt.Printf("  %s %s %s\n",
			pName.Render(padRight(f.Title, 22)),
			pFaint.Render("│"),
			pDim.Render("# "+truncate(desc, 34)))
	}
	fmt.Printf("\n %d note(s)\n", len(sorted))
}

func runRecent(files []FileInfo) {
	sorted := make([]FileInfo, 0, len(files))
	for _, f := range files {
		if !f.LastOpened.IsZero() {
			sorted = append(sorted, f)
		}
	}
	sortByRecent(sorted)
	fmt.Println(pHead.Render(" Recently opened"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))
	if len(sorted) == 0 {
		fmt.Println("  nothing opened yet")
		return
	}
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		f := sorted[i]
		fmt.Printf("  %s  %s\n",
			pDim.Render(padRight(relTime(f.LastOpened), 12)),
			pName.Render(f.Title))
	}
}

func runHistory(files []FileInfo) {
	sorted := make([]FileInfo, 0, len(files))
	for _, f := range files {
		if f.Count > 0 {
			sorted = append(sorted, f)
		}
	}
	sortByFrequency(sorted)
	fmt.Println(pHead.Render(" Usage history"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))
	if len(sorted) == 0 {
		fmt.Println("  no usage recorded yet")
		return
	}
	for _, f := range sorted {
		fmt.Printf("  %s  %s  %s\n",
			pNum.Render(padRight("×"+strconv.Itoa(f.Count), 5)),
			pDim.Render(padRight(relTime(f.LastOpened), 12)),
			pName.Render(f.Title))
	}
}

func runStats(cfg Config, files []FileInfo) {
	totalOpens := 0
	neverOpened := 0
	mostUsed := ""
	mostCount := 0
	var lastFile FileInfo
	for _, f := range files {
		totalOpens += f.Count
		if f.Count == 0 {
			neverOpened++
		}
		if f.Count > mostCount {
			mostCount = f.Count
			mostUsed = f.Title
		}
		if f.LastOpened.After(lastFile.LastOpened) {
			lastFile = f
		}
	}
	fmt.Println(pHead.Render(" Statistics"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))
	statLine("notes", strconv.Itoa(len(files)))
	statLine("total opens", strconv.Itoa(totalOpens))
	if mostUsed != "" {
		statLine("most used", fmt.Sprintf("%s (×%d)", mostUsed, mostCount))
	}
	if !lastFile.LastOpened.IsZero() {
		statLine("last opened", fmt.Sprintf("%s (%s)", lastFile.Title, relTime(lastFile.LastOpened)))
	}
	statLine("never opened", strconv.Itoa(neverOpened))
	statLine("notes dir", cfg.Core.NotesDir)
	_ = sort.Strings
}

func statLine(key, val string) {
	fmt.Printf("  %s  %s\n", pDim.Render(padRight(key, 13)), pName.Render(val))
}
