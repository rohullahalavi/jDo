// admin.go — maintenance commands: delete, backup, sync, doctor.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func runDelete(cfg Config, hist HistoryData, histPath string, files []FileInfo, query string) {
	if query == "" {
		fmt.Println(" usage: jdo del <query>")
		return
	}
	match := findBestMatch(files, query, cfg)
	if match == nil {
		fmt.Printf(" no match for %q\n", query)
		return
	}
	if cfg.Core.ConfirmBeforeDelete {
		fmt.Printf(" delete %s ? (%s) [y/N] ", pName.Render(match.Title), match.Filename)
		answer := strings.ToLower(readLine())
		if answer != "y" && answer != "yes" {
			fmt.Println(" cancelled")
			return
		}
	}
	if err := os.Remove(match.FullPath); err != nil {
		die("could not delete: %v", err)
	}
	delete(hist.Files, match.Filename)
	saveHistory(hist, histPath, cfg.Core.MaxHistoryItems)
	fmt.Printf(" %s deleted %s\n", pOK.Render("✓"), match.Filename)
}

func runBackup(cfg Config, files []FileInfo) {
	stamp := time.Now().Format("2006-01-02-150405")
	dir := filepath.Join(cfg.Core.BackupDir, "jdo-backup-"+stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		die("could not create backup dir: %v", err)
	}
	copied := 0
	for _, f := range files {
		data, err := os.ReadFile(f.FullPath)
		if err != nil {
			continue
		}
		dst := filepath.Join(dir, f.Filename)
		if err := os.WriteFile(dst, data, 0o644); err == nil {
			copied++
		}
	}
	fmt.Printf(" %s backed up %d note(s) to %s\n", pOK.Render("✓"), copied, dir)
}

func runSync(cfg Config) {
	if strings.TrimSpace(cfg.Core.SyncCommand) == "" {
		fmt.Println(" no sync command configured (core.sync_command)")
		return
	}
	cmd := exec.Command("sh", "-c", cfg.Core.SyncCommand)
	cmd.Dir = cfg.Core.NotesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "jdo: sync failed: %v\n", err)
		return
	}
	fmt.Println(" sync finished")
}

func runDoctor(cfg Config, configPath, histPath string) {
	fmt.Println(pHead.Render(" Doctor"))
	fmt.Println(pRule.Render(strings.Repeat("─", 26)))

	if _, err := os.Stat(configPath); err == nil {
		checkOK("config file", configPath)
	} else {
		checkFail("config file", "missing — run jdo once to create it")
	}

	if entries, err := os.ReadDir(cfg.Core.NotesDir); err == nil {
		n := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				n++
			}
		}
		checkOK("notes dir", fmt.Sprintf("%s (%d notes)", cfg.Core.NotesDir, n))
	} else {
		checkFail("notes dir", cfg.Core.NotesDir+" — not readable")
	}

	if _, err := os.Stat(histPath); err == nil {
		checkOK("history file", histPath)
	} else {
		checkFail("history file", "missing — created on first open")
	}

	parts := strings.Fields(cfg.Core.DefaultEditor)
	if len(parts) > 0 {
		if p, err := exec.LookPath(parts[0]); err == nil {
			checkOK("default editor", cfg.Core.DefaultEditor+" → "+p)
		} else {
			checkFail("default editor", cfg.Core.DefaultEditor+" — not in PATH")
		}
	}

	tmp := filepath.Join(filepath.Dir(configPath), ".jdo-doctor-tmp")
	if err := os.WriteFile(tmp, []byte("x"), 0o644); err == nil {
		_ = os.Remove(tmp)
		checkOK("write permission", filepath.Dir(configPath))
	} else {
		checkFail("write permission", filepath.Dir(configPath)+" — not writable")
	}

	fmt.Println()
	fmt.Println(pDim.Render(" tips: run `jdo edit` to change config, and verify `notes_dir` plus `default_editor` if checks fail"))
}

func checkOK(name, detail string) {
	fmt.Printf("  %s %-16s %s\n", pOK.Render("✓"), name, pDim.Render(detail))
}

func checkFail(name, detail string) {
	fmt.Printf("  %s %-16s %s\n", pFail.Render("✗"), name, pDim.Render(detail))
}
