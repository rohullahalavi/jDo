// notes.go — note scanning, titles, descriptions, and per-note stats.
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// FileInfo is one note plus everything the UI needs to know about it.
// The "key" field is the lowercased, extension-less filename and is used
// for pins and palette jumps.
type FileInfo struct {
	Filename    string
	key         string
	Title       string
	Description string
	FullPath    string
	Count       int
	LastOpened  time.Time
	HeadingCnt  int
	OpenTasks   int
	Stale       bool
}

// keyOf returns the normalized lookup key for a filename.
func keyOf(filename string) string {
	return strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
}

func scanFiles(cfg Config, hist HistoryData) []FileInfo {
	var files []FileInfo
	entries, err := os.ReadDir(cfg.Core.NotesDir)
	if err != nil {
		return files
	}
	ext := strings.ToLower(cfg.Core.DefaultExtension)
	if ext == "" {
		ext = ".md"
	}
	staleDays := cfg.UI.StaleDays
	if staleDays <= 0 {
		staleDays = 30
	}
	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if cfg.Core.IgnoreHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ext) {
			continue
		}
		fullPath := filepath.Join(cfg.Core.NotesDir, name)
		st := hist.Files[name]
		stale := false
		if !st.LastOpened.IsZero() {
			stale = now.Sub(st.LastOpened) >= time.Duration(staleDays)*24*time.Hour
		}
		files = append(files, FileInfo{
			Filename:    name,
			key:         keyOf(name),
			Title:       formatTitle(name),
			Description: getFirstLine(fullPath),
			FullPath:    fullPath,
			Count:       st.Count,
			LastOpened:  st.LastOpened,
			HeadingCnt:  countHeadings(fullPath),
			OpenTasks:   countOpenInFile(fullPath),
			Stale:       stale,
		})
	}
	sortByFrequency(files)
	return files
}

// countHeadings counts markdown headings ("# ...") outside code fences.
func countHeadings(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	n := 0
	fence := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(t, "```") {
			fence = !fence
			continue
		}
		if !fence && strings.HasPrefix(t, "#") && len(t) > 1 && t[1] == ' ' {
			n++
		}
	}
	return n
}

func sortByFrequency(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Count != files[j].Count {
			return files[i].Count > files[j].Count
		}
		if !files[i].LastOpened.Equal(files[j].LastOpened) {
			return files[i].LastOpened.After(files[j].LastOpened)
		}
		return files[i].Title < files[j].Title
	})
}

func sortByAlpha(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool { return files[i].Title < files[j].Title })
}

func sortByRecent(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		zi := files[i].LastOpened.IsZero()
		zj := files[j].LastOpened.IsZero()
		if zi != zj {
			return zj
		}
		return files[i].LastOpened.After(files[j].LastOpened)
	})
}

func formatTitle(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = splitCamel(name)
	words := strings.Fields(name)
	for i, w := range words {
		words[i] = titleWord(w)
	}
	return strings.Join(words, " ")
}

func splitCamel(s string) string {
	var sb strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				sb.WriteRune(' ')
			}
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func titleWord(w string) string {
	if w == "" {
		return w
	}
	if len(w) <= 4 && strings.ToUpper(w) == w && !allDigits(w) {
		return w
	}
	r := []rune(strings.ToLower(w))
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func getFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// firstHeading returns the raw text of the first heading, or "".
func firstHeading(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	fence := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(t, "```") {
			fence = !fence
			continue
		}
		if !fence && strings.HasPrefix(t, "#") && len(t) > 1 && t[1] == ' ' {
			return strings.TrimSpace(t[2:])
		}
	}
	return ""
}
