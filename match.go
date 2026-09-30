// match.go — fuzzy matching, scoring, and candidate resolution.
package main

import (
	"path/filepath"
	"sort"
	"strings"
)

func subseq(query, target string) bool {
	qi := 0
	for i := 0; i < len(target) && qi < len(query); i++ {
		if target[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

func fuzzyMatch(query, target string, caseSensitive bool) bool {
	if !caseSensitive {
		query = strings.ToLower(query)
		target = strings.ToLower(target)
	}
	if strings.Contains(target, query) {
		return true
	}
	return subseq(query, target)
}

// matchScore rates how well a query matches a note. Higher is better,
// -1 means no match. Usage count breaks ties so frequently used notes win.
func matchScore(query string, f FileInfo, caseSensitive bool) int {
	name := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename))
	title := f.Title
	q := query
	if !caseSensitive {
		name = strings.ToLower(name)
		title = strings.ToLower(title)
		q = strings.ToLower(q)
	}
	score := -1
	switch {
	case name == q || title == q:
		score = 1000
	case strings.HasPrefix(name, q) || strings.HasPrefix(title, q):
		score = 700
	case strings.Contains(name, q) || strings.Contains(title, q):
		score = 500
	case subseq(q, name) || subseq(q, title):
		score = 200
	}
	if score >= 0 {
		score += f.Count
	}
	return score
}

func findBestMatch(files []FileInfo, query string, cfg Config) *FileInfo {
	best := -1
	var bestIdx int
	for i, f := range files {
		s := matchScore(query, f, cfg.Core.CaseSensitiveSearch)
		if s > best {
			best = s
			bestIdx = i
		}
	}
	if best < 0 {
		return nil
	}
	return &files[bestIdx]
}

// resolveCandidates returns every matching file sorted by score.
func resolveCandidates(query string, files []FileInfo, cfg Config) []FileInfo {
	type scored struct {
		f FileInfo
		s int
	}
	var res []scored
	for _, f := range files {
		s := matchScore(query, f, cfg.Core.CaseSensitiveSearch)
		if s >= 0 {
			res = append(res, scored{f, s})
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].s != res[j].s {
			return res[i].s > res[j].s
		}
		return res[i].f.Title < res[j].f.Title
	})
	out := make([]FileInfo, 0, len(res))
	for _, r := range res {
		out = append(out, r.f)
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}
