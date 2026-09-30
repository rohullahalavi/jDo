// tui_model.go — TUI model, runners, timers, pins, toasts.
// TUI CONTRACT: the rest of the program only calls runFullTUI below.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type drillState struct {
	note      FileInfo
	drill     *Drill
	level     int // 1 headings, 2 tasks, 3 sub-tasks
	headCount int
	taskCount int
	subCount  int
	headStart int // window offset for paging headings (inline card)
	taskStart int // window offset for paging tasks   (inline card)
	subStart  int // window offset for paging subs     (inline card)
}

type treeNode struct {
	Kind     string
	Text     string
	Line     int
	Bullet   byte
	FileKey  string
	FileName string
	File     FileInfo
	Parent   *treeNode
	Children []*treeNode
	Expanded bool
}

type focusEntry struct {
	Kind string // root | node | pad
	Root *treeNode
	Node *treeNode
	Pad  bool
}

type model struct {
	files       []FileInfo
	filtered    []FileInfo
	trees       map[string]*treeNode
	focus       []focusEntry
	editors     []string
	config      Config
	history     HistoryData
	historyPath string
	pinsPath    string
	pins        []string

	mode     string // main | drill | search | add | palette | tasks | editor | help | pins
	prevMode string
	selected int
	offset   int // line-window offset for main
	scroll   int // line-window offset for drill / tasks
	query    string

	// inline card (main mode): attached to expandedRow; closes on cursor move
	expandedRow int
	inline      *drillState
	// full-screen drill page
	full *drillState

	// inputs / overlays
	searchInput string
	searchHist []string
	searchIdx int
	addInput    string
	addMode     string // "task" | "category" | "rename"
	addTarget   string
	renameNode  *treeNode
	palInput    string
	palSel      int
	tasksSnap   []taskItem
	tasksSel    int
	modalSel    int

	// feedback / live state
	toast string
	toastExp time.Time
	clock time.Time
	cursorVisible bool
	pendingG  bool
	showEmpty bool
	wrapDir    int
	wrapAt     time.Time

	// computed during render
	lineRow   []int // parallel to last mainLines(): filtered index per line, -1 otherwise
	visRows   []int // filtered indices currently visible (for 1-9)
	width     int
	height    int
}

type editorFinishedMsg struct{ err error }
type blinkMsg struct{}
type tickMsg struct{}
type toastMsg struct{}

type tuiPins struct {
	Pinned []string `json:"pinned"`
}

func (m *model) loadPins() {
	var p tuiPins
	if data, err := os.ReadFile(m.pinsPath); err == nil {
		_ = json.Unmarshal(data, &p)
	}
	m.pins = p.Pinned
}

func (m *model) savePins() {
	data, _ := json.MarshalIndent(tuiPins{Pinned: m.pins}, "", "  ")
	_ = os.WriteFile(m.pinsPath, data, 0o644)
}

func (m *model) isPinned(key string) bool {
	for _, p := range m.pins {
		if p == key {
			return true
		}
	}
	return false
}

func (m *model) emitToast(msg string) tea.Cmd {
	m.toast = msg
	m.toastExp = time.Now().Add(2200 * time.Millisecond)
	return tea.Tick(2200*time.Millisecond, func(time.Time) tea.Msg { return toastMsg{} })
}

func runFullTUI(files []FileInfo, cfg Config, hist HistoryData, histPath string) {
	m := model{
		files:         files,
		filtered:      files,
		trees:         buildTrees(files),
		config:        cfg,
		history:       hist,
		historyPath:   histPath,
		pinsPath:      filepath.Join(filepath.Dir(histPath), "pins.json"),
		mode:          "main",
		clock:         time.Now(),
		cursorVisible: true,
	}
	m.loadPins()
	var eds []string
	for k := range cfg.Editors {
		eds = append(eds, k)
	}
	sort.Strings(eds)
	m.editors = eds
	m.buildFiltered()
	if len(m.filtered) > 0 {
		m.toast = ""
	}
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		die("tui error: %v", err)
	}
}

func (m model) Init() tea.Cmd {
	return heartbeat()
}

func heartbeat() tea.Cmd {
	return tea.Tick(15*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func blinkCmd() tea.Cmd {
	return tea.Tick(530*time.Millisecond, func(time.Time) tea.Msg { return blinkMsg{} })
}
