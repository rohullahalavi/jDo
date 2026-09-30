// util.go — shared text, color, IO, and markdown helpers.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func truncate(s string, max int) string {
	if max < 1 {
		max = 1
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func padRight(s string, w int) string {
	if n := lipgloss.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func relTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("02 Jan 06")
	}
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	return hexVal(hex[0:2]), hexVal(hex[2:4]), hexVal(hex[4:6])
}

func hexVal(s string) int {
	v := 0
	for _, c := range s {
		v *= 16
		switch {
		case c >= '0' && c <= '9':
			v += int(c - '0')
		case c >= 'a' && c <= 'f':
			v += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v += int(c-'A') + 10
		}
	}
	return v
}

func lerpHex(a, b string, t float64) lipgloss.Color {
	ar, ag, ab := hexToRGB(a)
	br, bg, bb := hexToRGB(b)
	r := int(float64(ar) + (float64(br)-float64(ar))*t)
	g := int(float64(ag) + (float64(bg)-float64(ag))*t)
	bl := int(float64(ab) + (float64(bb)-float64(ab))*t)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))
}

// gradientRule draws a header rule fading from blue to faint white.
func gradientRule(w int) string {
	var sb strings.Builder
	for i := 0; i < w; i++ {
		t := float64(i) / float64(w)
		c := lerpHex("#4c9aff", "#3a3a3a", t)
		sb.WriteString(lipgloss.NewStyle().Foreground(c).Render("─"))
	}
	return sb.String()
}

func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "jdo: "+format+"\n", args...)
	os.Exit(1)
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// writeFileAtomic writes via a temp file + rename so a crash never
// corrupts the original note.
func writeFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".jdo-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// ---- markdown structure helpers ----

func isHeadingLine(s string) bool {
	return strings.HasPrefix(strings.TrimLeft(s, " "), "#")
}

func headingLevel(s string) int {
	t := strings.TrimLeft(s, " ")
	n := 0
	for n < len(t) && t[n] == '#' {
		n++
	}
	return n
}

func headingText(s string) string {
	t := strings.TrimLeft(s, " ")
	t = strings.TrimLeft(t, "#")
	return strings.TrimSpace(t)
}

// normalizeTag lowercases and strips #, :, _, - and extra spaces so that
// "# ToDo:", "todo" and "To-Do" all compare equal.
func normalizeTag(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "#", "")
	return strings.Join(strings.Fields(s), " ")
}

// splitIndentBullet splits a line into leading whitespace and its bullet
// character (-, + or *). Returns ok=false if there is no bullet.
func splitIndentBullet(ln string) (string, string, bool) {
	i := 0
	for i < len(ln) && (ln[i] == ' ' || ln[i] == '\t') {
		i++
	}
	indent := ln[:i]
	rest := ln[i:]
	if rest == "" {
		return "", "", false
	}
	b := rest[0]
	if b == '-' || b == '+' || b == '*' {
		return indent, string(b), true
	}
	return "", "", false
}

// isEmptyBullet reports whether a line is a placeholder bullet such as
// "  - " or "  - [ ] " with no real content.
func isEmptyBullet(ln string) bool {
	t := strings.TrimSpace(ln)
	t = strings.TrimLeft(t, "-+*")
	t = strings.TrimSpace(t)
	t = strings.TrimPrefix(t, "[ ]")
	t = strings.TrimPrefix(t, "[x]")
	t = strings.TrimPrefix(t, "[X]")
	return strings.TrimSpace(t) == ""
}
