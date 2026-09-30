// main.go — entry point and argument dispatch.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		die("cannot find home directory: %v", err)
	}

	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "-h", "--help", "help":
			printHelp()
			return
		case "-v", "--version", "version":
			printVersion(home)
			return
		}
	}

	configDir := filepath.Join(home, ".jdo")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		die("cannot create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	defsPath := filepath.Join(configDir, "defaults.json")

	cfg := loadConfig(configPath)
	histPath := expandHome(cfg.Core.HistoryFile, home)
	hist := loadHistory(histPath)
	defs := loadDefaults(defsPath)

	if cfg.Core.AutoCreateMissing {
		_ = os.MkdirAll(cfg.Core.NotesDir, 0o755)
	}

	files := scanFiles(cfg, hist)

	if len(os.Args) == 1 {
		runQuickLaunch(cfg, hist, histPath, files)
		return
	}

	arg := os.Args[1]
	rest := strings.Join(os.Args[2:], " ")

	switch arg {
	case "i", "tui", "interactive":
		runFullTUI(files, cfg, hist, histPath)
	case "z", "last":
		openLastOpened(cfg, hist, histPath)
	case "add", "a":
		opts, err := parseAddArgs(os.Args[2:])
		if err != nil {
			fmt.Println(" " + err.Error())
			return
		}
		runAdd(cfg, hist, histPath, &defs, defsPath, files, opts)
	case "done":
		runDone(cfg, files, rest)
	case "next":
		runNext(files, os.Args[2:])
	case "nexts":
		runNexts(files, os.Args[2:])
	case "tasks":
		runTasks(files, rest)
	case "count":
		runCount(files, rest)
	case "grep":
		runGrep(files, rest)
	case "show":
		runShow(files, cfg, &defs, defsPath, rest, "")
	case "tree":
		runTree(files, cfg, &defs, defsPath, rest)
	case "new", "n":
		if rest == "" {
			fmt.Println(" usage: jdo new <name>")
			return
		}
		createNewNote(cfg, hist, histPath, rest)
	case "today", "daily":
		createOrOpenDailyNote(cfg, hist, histPath)
	case "search":
		fullTextSearch(cfg, files, rest)
	case "list", "ls":
		runList(files)
	case "recent":
		runRecent(files)
	case "hist", "history":
		runHistory(files)
	case "stats":
		runStats(cfg, files)
	case "del", "rm", "delete":
		runDelete(cfg, hist, histPath, files, rest)
	case "backup":
		runBackup(cfg, files)
	case "sync":
		runSync(cfg)
	case "doctor", "check":
		runDoctor(cfg, configPath, histPath)
	case "edit", "config":
		openInEditor(configPath, cfg.Core.DefaultEditor)
	case "-h", "--help", "help":
		printHelp()
	case "-v", "--version", "version":
		printVersion(home)
	default:
		query := strings.Join(os.Args[1:], " ")
		f, ok := resolveTarget(query, files, cfg, &defs, defsPath)
		if !ok {
			os.Exit(1)
		}
		openTracked(cfg, hist, histPath, f)
	}
}
