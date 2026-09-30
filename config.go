// config.go — configuration structures, defaults, and TOML loading.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Core    CoreConfig
	UI      UIConfig
	Add     AddConfig
	Editors map[string]string
	Keys    KeysConfig
}

type CoreConfig struct {
	NotesDir            string `toml:"notes_dir"`
	DefaultEditor       string `toml:"default_editor"`
	HistoryFile         string `toml:"history_file"`
	MaxHistoryItems     int    `toml:"max_history_items"`
	AutoCreateMissing   bool   `toml:"auto_create_missing_files"`
	SortBy              string `toml:"sort_by"`
	CaseSensitiveSearch bool   `toml:"case_sensitive_search"`
	IgnoreHidden        bool   `toml:"ignore_hidden"`
	DefaultExtension    string `toml:"default_extension"`
	TemplateContent     string `toml:"template_content"`
	BackupEnabled       bool   `toml:"backup_enabled"`
	BackupDir           string `toml:"backup_dir"`
	SyncEnabled         bool   `toml:"sync_enabled"`
	SyncCommand         string `toml:"sync_command"`
	ConfirmBeforeDelete bool   `toml:"confirm_before_delete"`
	QuickListSize       int    `toml:"quick_list_size"`
	QuickListTitle      string `toml:"quick_list_title"`
	WrapScroll          bool   `toml:"wrap_scroll"`
	EditorJumpArg       string `toml:"editor_jump_arg"`
}

type UIConfig struct {
	ShowDescriptions bool   `toml:"show_descriptions"`
	MaxDescLength    int    `toml:"max_description_length"`
	ShowShortcuts    bool   `toml:"show_shortcuts_in_footer"`
	ShowUsageCounter bool   `toml:"show_usage_counter"`
	ShowClock        bool   `toml:"show_clock"`
	ShowPreview      bool   `toml:"show_preview"`
	RowSpacing       int    `toml:"row_spacing"`
	PromptSymbol     string `toml:"prompt_symbol"`
	DateFormat       string `toml:"date_format"`
	TimeFormat       string `toml:"time_format"`
	StaleDays        int    `toml:"stale_days"`
	PeekHeadings     int    `toml:"peek_headings"`
	PeekTasks        int    `toml:"peek_tasks"`
	PeekSubs         int    `toml:"peek_subs"`
	RowSeparator     string `toml:"row_separator"`
	ShowPinLabel     bool   `toml:"show_pin_label"`
}

type AddConfig struct {
	DefaultBullet      string `toml:"default_bullet"`
	DefaultIndent      int    `toml:"default_indent"`
	PlaceholderBullets int    `toml:"placeholder_bullets"`
	AutoEmoji          bool   `toml:"auto_emoji"`
	PriorityFormat     string `toml:"priority_format"`
	DefaultEmoji       string `toml:"default_emoji"`
	HeadingColon       string `toml:"heading_colon"`
	TimestampOnAdd     bool   `toml:"timestamp_on_add"`
}

type KeysConfig struct {
	Up       string `toml:"up"`
	Down     string `toml:"down"`
	Open     string `toml:"open"`
	OpenWith string `toml:"open_with"`
	Quit     string `toml:"quit"`
	Search   string `toml:"search"`
	Help     string `toml:"help"`
	NextPage string `toml:"next_page"`
	PrevPage string `toml:"prev_page"`
	Top      string `toml:"top"`
	Bottom   string `toml:"bottom"`
}

func defaultConfig(configDir string) Config {
	var cfg Config
	cfg.Core.NotesDir = "/Users/nox/Desktop/Notes/Tasks"
	cfg.Core.DefaultEditor = "nvim"
	cfg.Core.HistoryFile = filepath.Join(configDir, "history.json")
	cfg.Core.MaxHistoryItems = 200
	cfg.Core.AutoCreateMissing = true
	cfg.Core.SortBy = "frequency"
	cfg.Core.CaseSensitiveSearch = false
	cfg.Core.IgnoreHidden = true
	cfg.Core.DefaultExtension = ".md"
	cfg.Core.TemplateContent = "# {{title}}\n\ncreated: {{date}}\n\n"
	cfg.Core.BackupEnabled = true
	cfg.Core.BackupDir = filepath.Join(configDir, "backups")
	cfg.Core.SyncEnabled = false
	cfg.Core.SyncCommand = "git add . && git commit -m 'jdo sync'"
	cfg.Core.ConfirmBeforeDelete = true
	cfg.Core.QuickListSize = 5
	cfg.Core.QuickListTitle = "Top notes"
	cfg.Core.WrapScroll = true
	cfg.Core.EditorJumpArg = "+{line}"

	cfg.UI.ShowDescriptions = true
	cfg.UI.MaxDescLength = 28
	cfg.UI.ShowShortcuts = true
	cfg.UI.ShowUsageCounter = true
	cfg.UI.ShowClock = true
	cfg.UI.ShowPreview = false
	cfg.UI.RowSpacing = 0
	cfg.UI.PromptSymbol = "❯"
	cfg.UI.DateFormat = "2006-01-02"
	cfg.UI.TimeFormat = "15:04"
	cfg.UI.StaleDays = 30
	cfg.UI.PeekHeadings = 3
	cfg.UI.PeekTasks = 2
	cfg.UI.PeekSubs = 2
	cfg.UI.RowSeparator = "zebra"
	cfg.UI.ShowPinLabel = true

	cfg.Add.DefaultBullet = "-"
	cfg.Add.DefaultIndent = 2
	cfg.Add.PlaceholderBullets = 2
	cfg.Add.AutoEmoji = false
	cfg.Add.PriorityFormat = "P{n}"
	cfg.Add.DefaultEmoji = "🟡"
	cfg.Add.HeadingColon = "auto"
	cfg.Add.TimestampOnAdd = false

	cfg.Editors = map[string]string{
		"nvim": "nvim", "vim": "vim", "hx": "hx", "zed": "zed --wait",
		"code": "code --wait", "nano": "nano", "micro": "micro",
	}
	cfg.Keys.Up = "k"
	cfg.Keys.Down = "j"
	cfg.Keys.Open = "o"
	cfg.Keys.OpenWith = "O"
	cfg.Keys.Quit = "q"
	cfg.Keys.Search = "/"
	cfg.Keys.Help = "?"
	cfg.Keys.NextPage = "ctrl+d"
	cfg.Keys.PrevPage = "ctrl+u"
	cfg.Keys.Top = "g"
	cfg.Keys.Bottom = "G"
	return cfg
}

func loadConfig(path string) Config {
	configDir := filepath.Dir(path)
	cfg := defaultConfig(configDir)
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			fmt.Fprintf(os.Stderr, "jdo: config parse error: %v\n", err)
		}
		return cfg
	}
	if err := os.MkdirAll(configDir, 0o755); err == nil {
		if f, err := os.Create(path); err == nil {
			_ = toml.NewEncoder(f).Encode(cfg)
			f.Close()
		}
	}
	return cfg
}
