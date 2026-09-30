// query.go — read-only inspection commands: grep, show, tree.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runGrep(files []FileInfo, query string) {
	if query == "" {
		fmt.Println(" usage: jdo grep <text>")
		return
	}
	q := strings.ToLower(query)
	hits := 0
	for _, f := range files {
		data, err := os.ReadFile(f.FullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		var matched []string
		for i, ln := range lines {
			if strings.Contains(strings.ToLower(ln), q) {
				matched = append(matched, fmt.Sprintf("   %s %s",
					pFaint.Render(fmt.Sprintf("%3d", i+1)), strings.TrimSpace(ln)))
			}
		}
		if len(matched) > 0 {
			fmt.Println(pHead.Render(" " + f.Title))
			for _, m := range matched {
				fmt.Println(m)
			}
			hits += len(matched)
		}
	}
	if hits == 0 {
		fmt.Printf(" no match for %q\n", query)
	} else {
		fmt.Printf("\n %d hit(s)\n", hits)
	}
}

func runShow(files []FileInfo, cfg Config, defs *QueryDefaults, defsPath, target, heading string) {
	if target == "" {
		fmt.Println(" usage: jdo show <note>")
		return
	}
	f, ok := resolveTarget(target, files, cfg, defs, defsPath)
	if !ok {
		return
	}
	data, err := os.ReadFile(f.FullPath)
	if err != nil {
		die("cannot read %s: %v", f.Filename, err)
	}
	lines := strings.Split(string(data), "\n")
	if heading == "" {
		fmt.Println(pHead.Render(" " + f.Title))
		for _, ln := range lines {
			fmt.Println(ln)
		}
		return
	}
	nh := normalizeTag(heading)
	for i, ln := range lines {
		if isHeadingLine(ln) && normalizeTag(headingText(ln)) == nh {
			start, end := sectionBounds(lines, i)
			fmt.Println(pHead.Render(" " + headingText(ln)))
			for j := start; j < end; j++ {
				fmt.Println(lines[j])
			}
			return
		}
	}
	fmt.Printf(" no heading %q in %s\n", heading, f.Filename)
}

func runTree(files []FileInfo, cfg Config, defs *QueryDefaults, defsPath, target string) {
	var list []FileInfo
	if target != "" {
		f, ok := resolveTarget(target, files, cfg, defs, defsPath)
		if !ok {
			return
		}
		list = []FileInfo{f}
	} else {
		list = files
	}
	for _, f := range list {
		data, err := os.ReadFile(f.FullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		fmt.Println(pHead.Render(" " + f.Filename))
		for i, ln := range lines {
			if isHeadingLine(ln) {
				lvl := headingLevel(ln)
				start, end := sectionBounds(lines, i)
				n := 0
				for j := start; j < end; j++ {
					if _, _, ok := splitIndentBullet(lines[j]); ok && !isEmptyBullet(lines[j]) {
						n++
					}
				}
				indent := strings.Repeat("  ", lvl)
				fmt.Printf("   %s%s %s\n",
					pFaint.Render(indent+"├─"),
					pName.Render(headingText(ln)),
					pDim.Render(fmt.Sprintf("(%d)", n)))
			}
		}
	}
}

func runBacklinks(files []FileInfo, cfg Config, defs *QueryDefaults, defsPath, target string) {
	if target == "" {
		fmt.Println(" usage: jdo backlinks <note>")
		return
	}
	f, ok := resolveTarget(target, files, cfg, defs, defsPath)
	if !ok {
		return
	}
	stem := strings.ToLower(strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename)))
	title := strings.ToLower(f.Title)
	needle := strings.ToLower(target)
	if needle == "" {
		needle = title
	}
	hits := 0
	for _, other := range files {
		if other.Filename == f.Filename {
			continue
		}
		data, err := os.ReadFile(other.FullPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		found := false
		for i, ln := range lines {
			low := strings.ToLower(ln)
			if strings.Contains(low, needle) || strings.Contains(low, title) || strings.Contains(low, stem) {
				if !found {
					fmt.Println(pHead.Render(" " + other.Title))
					found = true
				}
				fmt.Printf("   %s %s\n", pFaint.Render(fmt.Sprintf("%3d", i+1)), strings.TrimSpace(ln))
				hits++
				break
			}
		}
	}
	if hits == 0 {
		fmt.Printf(" no backlinks found for %s\n", f.Title)
	}
}

func runWeeklyReview(files []FileInfo) {
	fmt.Println(pHead.Render(" Weekly review"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))

	recent := 0
	overdue := 0
	stale := 0
	taskTotal := 0
	now := time.Now()
	for _, f := range files {
		taskTotal += f.OpenTasks
		if !f.LastOpened.IsZero() && now.Sub(f.LastOpened) <= 7*24*time.Hour {
			recent++
		}
		if f.Stale {
			stale++
		}
		if f.OpenTasks > 0 && !f.LastOpened.IsZero() && now.Sub(f.LastOpened) > 7*24*time.Hour {
			overdue++
		}
	}
	statLine("recent notes", fmt.Sprintf("%d (7d)", recent))
	statLine("open tasks", fmt.Sprintf("%d", taskTotal))
	statLine("stale notes", fmt.Sprintf("%d", stale))
	statLine("review backlog", fmt.Sprintf("%d", overdue))
}
