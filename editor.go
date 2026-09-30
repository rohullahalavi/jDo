// editor.go — launching external editors, optionally at a line number.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func openInEditor(path string, editor string) {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		die("no editor configured")
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "jdo: editor error: %v\n", err)
	}
}

func openTracked(cfg Config, hist HistoryData, histPath string, f FileInfo) {
	recordHistory(hist, histPath, f.Filename, cfg.Core.MaxHistoryItems)
	openInEditor(f.FullPath, cfg.Core.DefaultEditor)
}

// openAtLine opens the editor jumped to a 1-based line number.
func openAtLine(path string, editor string, line int, jumpArg string) {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		die("no editor configured")
	}
	if line < 1 {
		line = 1
	}
	arg := strings.ReplaceAll(jumpArg, "{line}", strconvItoa(line))
	args := append(parts[1:], arg, path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "jdo: editor error: %v\n", err)
	}
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
