// resolve.go — multi-candidate resolution with permanent default pins.
// Powers both `jdo <query>` and the target lookup of `jdo add`.
package main

import (
	"fmt"
	"strconv"
	"strings"
)

// resolveTarget picks a file for a query. If a default was pinned earlier
// it wins. One candidate opens directly. Several candidates open a picker.
func resolveTarget(query string, files []FileInfo, cfg Config, defs *QueryDefaults, defsPath string) (FileInfo, bool) {
	key := normalizeTag(query)
	if fn, ok := defs.Pins[key]; ok {
		for _, f := range files {
			if f.Filename == fn {
				return f, true
			}
		}
	}
	cands := resolveCandidates(query, files, cfg)
	switch len(cands) {
	case 0:
		fmt.Printf(" no match for %q\n", query)
		return FileInfo{}, false
	case 1:
		return cands[0], true
	default:
		return runPicker(query, cands, defs, defsPath)
	}
}

// runPicker shows a numbered list. Typing "N" opens candidate N.
// Typing "Nd" also saves it as the permanent default for this query.
func runPicker(query string, cands []FileInfo, defs *QueryDefaults, defsPath string) (FileInfo, bool) {
	fmt.Printf(" %s %q — %d matches:\n", pHead.Render("multiple for"), query, len(cands))
	for i, c := range cands {
		fmt.Printf("   %s  %s  %s\n",
			pNum.Render(strconv.Itoa(i+1)),
			pName.Render(padRight(c.Title, 22)),
			pDim.Render(c.Filename))
	}
	for {
		fmt.Print(pPrompt.Render(fmt.Sprintf(" pick 1-%d (Nd = make default, q = cancel) › ", len(cands))))
		line := strings.ToLower(strings.TrimSpace(readLine()))
		if line == "" || line == "q" {
			return FileInfo{}, false
		}
		pin := false
		if strings.HasSuffix(line, "d") {
			pin = true
			line = strings.TrimSuffix(line, "d")
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || n < 1 || n > len(cands) {
			fmt.Println(pFail.Render(" invalid choice"))
			continue
		}
		chosen := cands[n-1]
		if pin {
			if defs.Pins == nil {
				defs.Pins = map[string]string{}
			}
			defs.Pins[normalizeTag(query)] = chosen.Filename
			saveDefaults(*defs, defsPath)
			fmt.Printf(" %s default for %q → %s\n", pOK.Render("✓"), query, chosen.Filename)
		}
		return chosen, true
	}
}
