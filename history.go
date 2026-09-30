// history.go — usage counts and last-opened timestamps, stored as JSON.
package main

import (
	"encoding/json"
	"os"
	"sort"
	"time"
)

type HistoryData struct {
	Files map[string]FileStats `json:"files"`
}

type FileStats struct {
	Count      int       `json:"count"`
	LastOpened time.Time `json:"last_opened"`
}

func loadHistory(path string) HistoryData {
	var hist HistoryData
	hist.Files = make(map[string]FileStats)
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &hist)
	}
	if hist.Files == nil {
		hist.Files = make(map[string]FileStats)
	}
	return hist
}

func saveHistory(hist HistoryData, path string, maxItems int) {
	if maxItems > 0 && len(hist.Files) > maxItems {
		type kv struct {
			name  string
			count int
		}
		all := make([]kv, 0, len(hist.Files))
		for name, st := range hist.Files {
			all = append(all, kv{name, st.Count})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].count < all[j].count })
		extra := len(all) - maxItems
		for i := 0; i < extra; i++ {
			delete(hist.Files, all[i].name)
		}
	}
	data, err := json.MarshalIndent(hist, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func recordHistory(hist HistoryData, path, filename string, maxItems int) {
	st := hist.Files[filename]
	st.Count++
	st.LastOpened = time.Now()
	hist.Files[filename] = st
	saveHistory(hist, path, maxItems)
}
