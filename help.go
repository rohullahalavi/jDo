// help.go — help text and version information.
package main

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func printHelp() {
	fmt.Printf("%s %s — %s\n\n", pHead.Render(appName), appVersion, appTagline)

	fmt.Println(pHead.Render("USAGE"))
	fmt.Println("    jdo [COMMAND] [ARGS]")
	fmt.Println()
	fmt.Println("With no command, jdo prints your most used notes as a numbered")
	fmt.Println("list and opens the one you type. No TUI, the screen is not cleared.")
	fmt.Println()

	fmt.Println(pHead.Render("COMMANDS"))
	cmds := [][2]string{
		{"(none)", "top-N quick launcher; type 1-5 to open"},
		{"i, tui", "open the full interactive TUI"},
		{"z, last", "reopen the last note you opened"},
		{"<query>", "fuzzy-find a note; picker if several match"},
		{"add <t> \"text\"", "insert an item into the right file + section"},
	{"done <text>", "mark a matching task as done"},
	{"next [n]", "show tasks due in next N hours (default 1)"},
	{"nexts [d|w|m|y]", "show tasks due today/week/month/year"},
	{"tasks", "list open tasks across all notes"},
	{"count", "count items per section per file"},

		{"grep <text>", "search note contents"},
		{"show <note>", "print a note or one of its sections"},
		{"tree", "print the heading tree with item counts"},
		{"new <name>", "create a note from the template and open it"},
		{"today, daily", "open or create today's note"},
		{"search <text>", "full-text search inside every note"},
		{"list, ls", "print all notes with descriptions"},
		{"recent", "show recently opened notes"},
		{"hist, history", "show usage counts and last-opened times"},
		{"stats", "show a usage statistics dashboard"},
		{"del <query>", "delete a note (asks for confirmation)"},
		{"backup", "copy all notes into the backup directory"},
		{"sync", "run the configured sync command"},
		{"doctor", "check config, editors and permissions"},
		{"edit, config", "open the jdo config file in your editor"},
		{"help, -h, --help", "show this help"},
		{"version, -v", "show version information"},
	}
	for _, kv := range cmds {
		fmt.Printf("    %s   %s\n", pCmd.Render(padRight(kv[0], 22)), kv[1])
	}
	fmt.Println()

	fmt.Println(pHead.Render("ADD FLAGS"))
	flags := [][2]string{
		{"-t, --tag <name>", "heading to insert under (default: target)"},
		{"-p, --priority <n>", "priority 1-5 or critical/high/normal/low/someday"},
		{"-d, --due <date>", "today, tomorrow, friday, +3d, +1w, 2026-08-01"},
		{"-c, --check", "write as a checkbox - [ ]"},
		{"--top", "insert at the top of the section"},
		{"--no-emoji", "skip the emoji prefix"},
		{"--dry-run", "show what would change without writing"},
	}
	for _, kv := range flags {
		fmt.Printf("    %s   %s\n", pCmd.Render(padRight(kv[0], 22)), kv[1])
	}
	fmt.Println()

	fmt.Println(pHead.Render("PICKER DEFAULTS"))
	fmt.Println("    When several files match a query, jdo shows a numbered list.")
	fmt.Println("    Type a number to open once, or e.g. \"2d\" to make candidate #2")
	fmt.Println("    the permanent default for that query. Stored in defaults.json.")
	fmt.Println()

	fmt.Println(pHead.Render("TUI KEYS"))
	keys := [][2]string{
		{"j / ↓, k / ↑", "move down / up"},
		{"g g, G", "jump to top / bottom"},
		{"ctrl+d, ctrl+u", "half page down / up"},
		{"1-9", "jump to visible root"},
		{"l / h", "expand / collapse tree"},
		{"o / enter", "open in the default editor"},
		{"O", "choose editor"},
		{"a", "quick-add into owning note"},
		{"p", "toggle pin on owning note"},
		{"/", "filter roots live"},
		{"s", "cycle sort: frequency → alpha → recent"},
		{"r", "rescan the notes directory"},
		{"y", "yank note path to clipboard"},
		{"?", "keybinding help overlay"},
		{"esc", "clear filter / close dialogs"},
		{"q", "quit"},
	}
	for _, kv := range keys {
		fmt.Printf("    %s   %s\n", pCmd.Render(padRight(kv[0], 22)), kv[1])
	}
	fmt.Println()

	fmt.Println(pHead.Render("FILES"))
	fmt.Println("    ~/.jdo/config.toml    configuration")
	fmt.Println("    ~/.jdo/history.json   usage counts and last-opened times")
	fmt.Println("    ~/.jdo/defaults.json  permanent per-query file pins")
}

func printVersion(home string) {
	fmt.Printf("%s %s\n", appName, appVersion)
	fmt.Printf("%s\n", appTagline)
	fmt.Printf("  go        %s\n", runtime.Version())
	fmt.Printf("  platform  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  config    %s\n", filepath.Join(home, ".jdo", "config.toml"))
	fmt.Printf("  data      %s\n", filepath.Join(home, ".jdo", "history.json"))
	fmt.Printf("  defaults  %s\n", filepath.Join(home, ".jdo", "defaults.json"))
}
