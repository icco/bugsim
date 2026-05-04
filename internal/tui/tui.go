// Package tui implements the Bubble Tea program for bugsim.
package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/icco/bugsim/internal/engine"
	"github.com/icco/bugsim/internal/pack"
	"github.com/icco/bugsim/internal/runner"
)

// Config controls the interactive session.
type Config struct {
	PacksDir string
	Timeout  time.Duration
}

type screen int

const (
	screenList screen = iota
	screenImplement
	screenBugReview
	screenResult
)

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Run     key.Binding
	Quit    key.Binding
	Help    key.Binding
	Choice1 key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/submit")),
		Back:    key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc", "back")),
		Run:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "run tests")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Choice1: key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"), key.WithHelp("1-9", "choose")),
	}
}

// ShortHelp implements help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Back, k.Run, k.Choice1},
		{k.Help, k.Quit},
	}
}

type packItem struct {
	summary pack.Summary
}

func (p packItem) Title() string       { return p.summary.Title }
func (p packItem) Description() string { return fmt.Sprintf("%s · %s", p.summary.ID, p.summary.Track) }
func (p packItem) FilterValue() string { return p.summary.Title + " " + p.summary.ID }

type model struct {
	cfg      Config
	keys     keyMap
	help     help.Model
	screen   screen
	err      error
	quitting bool

	packsList list.Model
	body      viewport.Model
	renderer  *glamour.TermRenderer

	current *pack.Pack
	workDir string
	running bool
	result  *engine.TestResult

	choice    int
	answered  bool
	correct   bool
	revealLbl string
	status    string
}

// New constructs a Bubble Tea model rooted at the given packs dir.
func New(cfg Config) (tea.Model, error) {
	if cfg.PacksDir == "" {
		return nil, errors.New("packs dir is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Minute
	}

	summaries, err := pack.Discover(cfg.PacksDir)
	if err != nil {
		return nil, err
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].ID < summaries[j].ID })

	items := make([]list.Item, 0, len(summaries))
	for _, s := range summaries {
		items = append(items, packItem{summary: s})
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "bugsim"
	l.SetStatusBarItemName("pack", "packs")

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return nil, fmt.Errorf("init markdown renderer: %w", err)
	}

	return &model{
		cfg:       cfg,
		keys:      defaultKeyMap(),
		help:      help.New(),
		packsList: l,
		body:      viewport.New(),
		renderer:  r,
	}, nil
}

func (m *model) Init() tea.Cmd { return nil }

type testsDoneMsg struct {
	res *engine.TestResult
	err error
}

func (m *model) runTestsCmd() tea.Cmd {
	p := m.current
	workDir := m.workDir
	timeout := m.cfg.Timeout
	return func() tea.Msg {
		rdef, err := runner.Load(p.Manifest.Runner)
		if err != nil {
			return testsDoneMsg{err: err}
		}
		if err := engine.MaterializeWorkspace(p, workDir); err != nil {
			return testsDoneMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := engine.EnsureDocker(ctx); err != nil {
			return testsDoneMsg{err: err}
		}
		res, err := engine.RunTests(ctx, workDir, rdef, engine.DefaultLimits())
		return testsDoneMsg{res: res, err: err}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)
		m.packsList.SetSize(msg.Width, msg.Height-1)
		m.body.SetWidth(msg.Width)
		m.body.SetHeight(max(1, msg.Height-bodyChromeHeight()))
		return m, nil

	case testsDoneMsg:
		m.running = false
		if msg.err != nil {
			m.err = msg.err
			m.status = "error running tests"
			return m, nil
		}
		m.result = msg.res
		m.screen = screenResult
		if msg.res.ExitCode == 0 {
			m.status = "tests passed"
		} else {
			m.status = fmt.Sprintf("tests failed (exit %d)", msg.res.ExitCode)
		}
		m.body.SetContent(formatResult(msg.res))
		return m, nil

	case tea.KeyPressMsg:
		return m.onKey(msg)
	}
	return m, nil
}

func (m *model) onKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, m.keys.Quit):
		// Don't intercept quit when filter input is active.
		if m.screen == screenList && m.packsList.FilterState() == list.Filtering {
			break
		}
		m.quitting = true
		return m, tea.Quit

	case key.Matches(k, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	switch m.screen {
	case screenList:
		return m.onListKey(k)
	case screenImplement:
		return m.onImplementKey(k)
	case screenBugReview:
		return m.onBugReviewKey(k)
	case screenResult:
		return m.onResultKey(k)
	}
	return m, nil
}

func (m *model) onListKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(k, m.keys.Enter) && m.packsList.FilterState() != list.Filtering {
		if it, ok := m.packsList.SelectedItem().(packItem); ok {
			return m.openPack(it.summary)
		}
	}
	var cmd tea.Cmd
	m.packsList, cmd = m.packsList.Update(k)
	return m, cmd
}

func (m *model) openPack(s pack.Summary) (tea.Model, tea.Cmd) {
	p, err := pack.Load(s.Dir)
	if err != nil {
		m.err = err
		return m, nil
	}
	md, err := p.ReadProblemMarkdown()
	if err != nil {
		m.err = err
		return m, nil
	}
	rendered, err := m.renderer.Render(md)
	if err != nil {
		rendered = md
	}

	m.current = p
	m.result = nil
	m.err = nil
	m.answered = false
	m.correct = false
	m.revealLbl = ""
	m.choice = 0
	m.status = ""

	switch p.Manifest.Track {
	case pack.TrackImplement:
		tmp, err := os.MkdirTemp("", fmt.Sprintf("bugsim-%s-", p.Manifest.ID))
		if err != nil {
			m.err = err
			return m, nil
		}
		m.workDir = tmp
		m.body.SetContent(rendered)
		m.body.GotoTop()
		m.screen = screenImplement
	case pack.TrackBugReview:
		corpus, err := p.ReadBugCorpus()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.body.SetContent(rendered + "\n\n" + corpus)
		m.body.GotoTop()
		m.screen = screenBugReview
	}
	return m, nil
}

func (m *model) onImplementKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, m.keys.Back):
		return m.backToList(), nil
	case key.Matches(k, m.keys.Run):
		if m.running {
			return m, nil
		}
		m.running = true
		m.status = "running tests..."
		return m, m.runTestsCmd()
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(k)
	return m, cmd
}

func (m *model) onBugReviewKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	br := m.current.Manifest.BugReview
	switch {
	case key.Matches(k, m.keys.Back):
		return m.backToList(), nil
	case key.Matches(k, m.keys.Up):
		if m.choice > 0 {
			m.choice--
		}
		return m, nil
	case key.Matches(k, m.keys.Down):
		if m.choice < len(br.Choices)-1 {
			m.choice++
		}
		return m, nil
	case key.Matches(k, m.keys.Enter):
		if m.answered {
			return m.backToList(), nil
		}
		m.answered = true
		picked := br.Choices[m.choice]
		m.correct = picked.Correct
		if !picked.Correct {
			for _, c := range br.Choices {
				if c.Correct {
					m.revealLbl = c.Label
					break
				}
			}
		}
		if m.correct {
			m.status = "correct"
		} else {
			m.status = "incorrect"
		}
		return m, nil
	case key.Matches(k, m.keys.Choice1):
		idx := int(k.String()[0] - '1')
		if idx >= 0 && idx < len(br.Choices) {
			m.choice = idx
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(k)
	return m, cmd
}

func (m *model) onResultKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, m.keys.Back), key.Matches(k, m.keys.Enter):
		switch m.current.Manifest.Track {
		case pack.TrackImplement:
			m.screen = screenImplement
		case pack.TrackBugReview:
			m.screen = screenBugReview
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(k)
	return m, cmd
}

func (m *model) backToList() *model {
	m.screen = screenList
	if m.workDir != "" && strings.Contains(m.workDir, "bugsim-") {
		_ = os.RemoveAll(m.workDir)
		m.workDir = ""
	}
	m.current = nil
	m.result = nil
	m.err = nil
	m.status = ""
	return m
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	okStyle     = lipgloss.NewStyle().Bold(true)
	errStyle    = lipgloss.NewStyle().Bold(true)
	cursorStyle = lipgloss.NewStyle().Bold(true)
)

func bodyChromeHeight() int { return 6 }

func (m *model) View() tea.View {
	if m.quitting {
		return tea.View{Content: "bye.\n"}
	}
	var body string
	switch m.screen {
	case screenList:
		body = m.packsList.View()
	case screenImplement:
		body = m.viewImplement()
	case screenBugReview:
		body = m.viewBugReview()
	case screenResult:
		body = m.viewResult()
	}
	return tea.View{Content: body, AltScreen: true}
}

func (m *model) viewImplement() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.current.Manifest.Title))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("track: implement   runner: %s   difficulty: %s",
		m.current.Manifest.Runner, m.current.Manifest.Difficulty)))
	b.WriteString("\n\n")
	b.WriteString(m.body.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("workspace: %s", m.workDir)))
	b.WriteString("\n")
	if m.running {
		b.WriteString("running tests...\n")
	} else if m.status != "" {
		b.WriteString(m.status + "\n")
	}
	if m.err != nil {
		b.WriteString(errStyle.Render("error: "+m.err.Error()) + "\n")
	}
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func (m *model) viewBugReview() string {
	var b strings.Builder
	br := m.current.Manifest.BugReview
	b.WriteString(headerStyle.Render(m.current.Manifest.Title))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("track: bug review   difficulty: %s", m.current.Manifest.Difficulty)))
	b.WriteString("\n\n")
	b.WriteString(m.body.View())
	b.WriteString("\n")
	b.WriteString(headerStyle.Render(br.Prompt))
	b.WriteString("\n")
	for i, c := range br.Choices {
		marker := "  "
		if i == m.choice {
			marker = "> "
		}
		line := fmt.Sprintf("%s%d. %s", marker, i+1, c.Label)
		if i == m.choice {
			line = cursorStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if m.answered {
		if m.correct {
			b.WriteString("\n" + okStyle.Render("correct.") + "\n")
		} else {
			b.WriteString("\n" + errStyle.Render("incorrect."))
			if m.revealLbl != "" {
				b.WriteString(" expected: " + m.revealLbl)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func (m *model) viewResult() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.current.Manifest.Title + " — results"))
	b.WriteString("\n")
	if m.result != nil {
		if m.result.ExitCode == 0 {
			b.WriteString(okStyle.Render("PASS"))
		} else {
			b.WriteString(errStyle.Render(fmt.Sprintf("FAIL (exit %d)", m.result.ExitCode)))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(m.body.View())
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("workspace: %s", m.workDir)))
	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func formatResult(r *engine.TestResult) string {
	if r == nil {
		return "no result."
	}
	var b strings.Builder
	if s := strings.TrimRight(r.Stdout, "\n"); s != "" {
		b.WriteString("stdout\n")
		b.WriteString(s)
		b.WriteString("\n\n")
	}
	if s := strings.TrimRight(r.Stderr, "\n"); s != "" {
		b.WriteString("stderr\n")
		b.WriteString(s)
		b.WriteString("\n")
	}
	return b.String()
}
