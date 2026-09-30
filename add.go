// add.go — the `jdo add` insertion engine: a pure core plus a CLI wrapper.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type AddOptions struct {
	Target  string
	Items   []string
	Tag     string
	Priority int
	Due     string
	Top     bool
	Check   bool
	NoEmoji bool
	DryRun  bool
	Verbose bool
	Time    string
}

type AddResult struct {
	File    FileInfo
	Heading string
	Count   int
	Err     error
}

type lineStyle struct {
	indent string
	bullet string
}

func parseAddArgs(args []string) (AddOptions, error) {
	var o AddOptions
	var pos []string
	i := 0
	for i < len(args) {
		a := args[i]
		switch a {
		case "-t", "--tag":
			i++
			if i >= len(args) {
				return o, fmt.Errorf("-t needs a value")
			}
			o.Tag = args[i]
		case "-p", "--priority":
			i++
			if i >= len(args) {
				return o, fmt.Errorf("-p needs a value")
			}
			p, err := parsePriority(args[i])
			if err != nil {
				return o, err
			}
			o.Priority = p
		case "-d", "--due":
			i++
			if i >= len(args) {
				return o, fmt.Errorf("-d needs a value")
			}
			o.Due = args[i]
	case "--top":
		o.Top = true
	case "-m", "--time", "-tt", "--timestr":
		i++
		if i >= len(args) {
			return o, fmt.Errorf("%s needs a value", a)
		}
		o.Time = args[i]
	case "-c", "--check":
		o.Check = true
	case "--no-emoji":
			o.NoEmoji = true
		case "--dry-run":
			o.DryRun = true
		case "-v", "--verbose":
			o.Verbose = true
		default:
			pos = append(pos, a)
		}
		i++
	}
	if len(pos) < 1 {
		return o, fmt.Errorf("usage: jdo add <target> <text...> [flags]")
	}
	o.Target = pos[0]
	o.Items = pos[1:]
	if len(o.Items) == 0 {
		return o, fmt.Errorf("no text given")
	}
	return o, nil
}

// addCore mutates the given file (no resolution, no printing).
func addCore(cfg Config, hist HistoryData, histPath string, f FileInfo, opts AddOptions) AddResult {
	data, err := os.ReadFile(f.FullPath)
	if err != nil {
		return AddResult{Err: err}
	}
	content := string(data)
	hadNL := strings.HasSuffix(content, "\n")
	var lines []string
	if content != "" {
		lines = strings.Split(content, "\n")
		if hadNL {
			lines = lines[:len(lines)-1]
		}
	}
	tag := opts.Tag
	if tag == "" {
		tag = opts.Target
	}
	normTag := normalizeTag(tag)

	hIdx := -1
	for i, ln := range lines {
		if isHeadingLine(ln) && normalizeTag(headingText(ln)) == normTag {
			hIdx = i
			break
		}
	}
	if hIdx == -1 {
		return AddResult{Err: fmt.Errorf("heading not found: %q", tag)}
	}

	st := defaultStyle(cfg)
	if hIdx >= 0 {
		start, end := sectionBounds(lines, hIdx)
		st = detectStyle(lines, start, end, cfg)
		items := buildItems(opts, st, cfg)
		insertAt := start
		if !opts.Top {
			last := start
			for i := start; i < end; i++ {
				if strings.TrimSpace(lines[i]) != "" {
					last = i + 1
				}
			}
			for last > start && isEmptyBullet(lines[last-1]) {
				last--
			}
			insertAt = last
		}
		lines = insertSlice(lines, insertAt, items)
	} else {
		items := buildItems(opts, st, cfg)
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, "", buildHeading(tag, cfg, lines))
		lines = append(lines, items...)
		for p := 0; p < cfg.Add.PlaceholderBullets; p++ {
			lines = append(lines, st.indent+st.bullet)
		}
	}

	if !opts.DryRun {
		if err := writeFileAtomic(f.FullPath, strings.Join(lines, "\n")+"\n"); err != nil {
			return AddResult{Err: err}
		}
		recordHistory(hist, histPath, f.Filename, cfg.Core.MaxHistoryItems)
	}
	return AddResult{File: f, Heading: normTag, Count: len(opts.Items)}
}

// runAdd is the CLI wrapper: resolves the target (with the picker) then prints.
func runAdd(cfg Config, hist HistoryData, histPath string, defs *QueryDefaults, defsPath string, files []FileInfo, opts AddOptions) {
	f, ok := resolveTarget(opts.Target, files, cfg, defs, defsPath)
	if !ok {
		return
	}
	res := addCore(cfg, hist, histPath, f, opts)
	if res.Err != nil {
		die("%v", res.Err)
	}
	if opts.DryRun {
		fmt.Printf(" ~ %s → # %s → would write %d item(s)\n", f.Filename, res.Heading, res.Count)
		return
	}
	fmt.Printf(" %s %s → # %s → %d item(s)\n",
		pOK.Render("✓"), pName.Render(f.Filename), pHead.Render(res.Heading), res.Count)
}

func buildItems(opts AddOptions, st lineStyle, cfg Config) []string {
	items := make([]string, 0, len(opts.Items))
	for _, text := range opts.Items {
		items = append(items, formatItem(text, opts, st, cfg))
	}
	return items
}

func formatItem(text string, opts AddOptions, st lineStyle, cfg Config) string {
	var sb strings.Builder
	sb.WriteString(st.indent)
	sb.WriteString("- [ ] ")
	if opts.Priority > 0 {
		sb.WriteString(renderPriority(opts.Priority, cfg) + " ")
	}
	if !opts.NoEmoji {
		sb.WriteString(emojiFor(text, opts, cfg) + " ")
	}
	sb.WriteString(text)
	if opts.Time != "" {
		sb.WriteString(" (🕐 " + opts.Time + ")")
	}
	if opts.Due != "" {
		if d, ok := parseDue(opts.Due); ok {
			sb.WriteString(" (📅 " + d + ")")
		}
	}
	if cfg.Add.TimestampOnAdd {
		sb.WriteString(" (" + time.Now().Format("02 Jan 15:04") + ")")
	}
	return sb.String()
}

func renderPriority(n int, cfg Config) string {
	f := cfg.Add.PriorityFormat
	if f == "" {
		f = "P{n}"
	}
	return strings.ReplaceAll(f, "{n}", strconv.Itoa(n))
}

func emojiFor(text string, opts AddOptions, cfg Config) string {
	if opts.Priority > 0 {
		return priorityEmoji(opts.Priority)
	}
	if cfg.Add.AutoEmoji {
		if e := keywordEmoji(text); e != "" {
			return e
		}
	}
	if cfg.Add.DefaultEmoji != "" {
		return cfg.Add.DefaultEmoji
	}
	return "🟡"
}

func priorityEmoji(n int) string {
	switch n {
	case 1:
		return "🔴"
	case 2:
		return "🟠"
	case 3:
		return "🟡"
	case 4:
		return "🔵"
	default:
		return "⚪"
	}
}

var emojiKeywords = []struct {
	keys  []string
	emoji string
}{
	{[]string{"buy", "purchase", "pay", "order"}, "🛒"},
	{[]string{"call", "phone", "email", "message"}, "📞"},
	{[]string{"learn", "study", "read", "course"}, "📖"},
	{[]string{"fix", "bug", "debug", "error"}, "🔧"},
	{[]string{"watch", "video", "film"}, "🎬"},
	{[]string{"trade", "crypto", "btc", "stock"}, "📈"},
	{[]string{"vps", "server", "deploy", "docker"}, "🖥️"},
	{[]string{"idea", "think", "brainstorm"}, "💡"},
	{[]string{"download", "dl", "fetch"}, "⬇️"},
	{[]string{"youtube", "channel", "upload"}, "▶️"},
	{[]string{"math", "calculate", "division"}, "🔢"},
	{[]string{"write", "blog", "post", "article"}, "✍️"},
	{[]string{"meeting", "appointment", "dentist"}, "📅"},
	{[]string{"gym", "workout", "run", "sport"}, "🏋️"},
	{[]string{"travel", "flight", "visa"}, "✈️"},
	{[]string{"money", "budget", "save"}, "💰"},
	{[]string{"code", "program", "app", "website"}, "💻"},
}

func keywordEmoji(text string) string {
	low := strings.ToLower(text)
	for _, group := range emojiKeywords {
		for _, k := range group.keys {
			if strings.Contains(low, k) {
				return group.emoji
			}
		}
	}
	return ""
}

func parsePriority(s string) (int, error) {
	if n, err := strconv.Atoi(s); err == nil {
		if n < 1 {
			n = 1
		}
		if n > 5 {
			n = 5
		}
		return n, nil
	}
	switch strings.ToLower(s) {
	case "critical", "crit":
		return 1, nil
	case "high":
		return 2, nil
	case "normal", "med", "medium":
		return 3, nil
	case "low":
		return 4, nil
	case "someday", "backlog":
		return 5, nil
	}
	return 0, fmt.Errorf("invalid priority %q", s)
}

func parseDue(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	now := time.Now()
	const dayFmt = "Mon 02 Jan"
	switch s {
	case "today":
		return now.Format(dayFmt), true
	case "tomorrow":
		return now.AddDate(0, 0, 1).Format(dayFmt), true
	case "next week":
		return now.AddDate(0, 0, 7).Format(dayFmt), true
	}
	if strings.HasPrefix(s, "+") {
		rest := s[1:]
		if strings.HasSuffix(rest, "d") {
			if n, err := strconv.Atoi(strings.TrimSuffix(rest, "d")); err == nil {
				return now.AddDate(0, 0, n).Format(dayFmt), true
			}
		}
		if strings.HasSuffix(rest, "w") {
			if n, err := strconv.Atoi(strings.TrimSuffix(rest, "w")); err == nil {
				return now.AddDate(0, 0, 7*n).Format(dayFmt), true
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
			return now.AddDate(0, 0, add).Format(dayFmt), true
		}
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format(dayFmt), true
	}
	return "", false
}

func sectionBounds(lines []string, hIdx int) (int, int) {
	lvl := headingLevel(lines[hIdx])
	start := hIdx + 1
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if isHeadingLine(lines[i]) && headingLevel(lines[i]) <= lvl {
			end = i
			break
		}
	}
	return start, end
}

func defaultStyle(cfg Config) lineStyle {
	bullet := cfg.Add.DefaultBullet
	if bullet == "" {
		bullet = "-"
	}
	return lineStyle{indent: strings.Repeat(" ", maxInt(cfg.Add.DefaultIndent, 0)), bullet: bullet}
}

func detectStyle(lines []string, start, end int, cfg Config) lineStyle {
	def := defaultStyle(cfg)
	for i := end - 1; i >= start; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		indent, bullet, ok := splitIndentBullet(lines[i])
		if ok {
			return lineStyle{indent, bullet}
		}
	}
	return def
}

func buildHeading(tag string, cfg Config, lines []string) string {
	title := formatTitle(tag + ".md")
	colon := ""
	switch cfg.Add.HeadingColon {
	case "always":
		colon = ":"
	case "never":
		colon = ""
	default:
		withC, withoutC := 0, 0
		for _, ln := range lines {
			if isHeadingLine(ln) {
				if strings.HasSuffix(strings.TrimSpace(ln), ":") {
					withC++
				} else {
					withoutC++
				}
			}
		}
		if withC > withoutC {
			colon = ":"
		}
	}
	return "# " + title + colon
}

func insertSlice(lines []string, at int, items []string) []string {
	out := make([]string, 0, len(lines)+len(items))
	out = append(out, lines[:at]...)
	out = append(out, items...)
	out = append(out, lines[at:]...)
	return out
}
