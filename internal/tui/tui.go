package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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

type model struct {
	cfg      Config
	packs    []pack.Summary
	cursor   int
	width    int
	height   int
	screen   screen
	err      error
	quitting bool

	current   *pack.Pack
	problemMD string

	// implement
	workDir string
	running bool
	result  *engine.TestResult

	// bug review
	bugCorpus string
	choice    int
	answered  bool
	correct   bool
	revealLbl string

	status string
}

// New returns a Bubble Tea program model.
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
	return &model{cfg: cfg, packs: summaries}, nil
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
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.onKey(msg)
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
		return m, nil
	}
	return m, nil
}

func (m *model) onKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := k.String()

	if key == "ctrl+c" || (m.screen == screenList && key == "q") {
		m.quitting = true
		return m, tea.Quit
	}

	switch m.screen {
	case screenList:
		return m.onListKey(key)
	case screenImplement:
		return m.onImplementKey(key)
	case screenBugReview:
		return m.onBugReviewKey(key)
	case screenResult:
		switch key {
		case "enter", "esc", "backspace", "b":
			return m.backToProblem()
		case "q":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) onListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.packs)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.packs) == 0 {
			return m, nil
		}
		return m.openPack(m.packs[m.cursor])
	}
	return m, nil
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
	m.current = p
	m.problemMD = md
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
		m.screen = screenImplement
	case pack.TrackBugReview:
		corpus, err := p.ReadBugCorpus()
		if err != nil {
			m.err = err
			return m, nil
		}
		m.bugCorpus = corpus
		m.screen = screenBugReview
	}
	return m, nil
}

func (m *model) onImplementKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "b":
		return m.backToList(), nil
	case "r":
		if m.running {
			return m, nil
		}
		m.running = true
		m.status = "running tests..."
		return m, m.runTestsCmd()
	case "w":
		m.status = "workspace: " + m.workDir
	}
	return m, nil
}

func (m *model) onBugReviewKey(key string) (tea.Model, tea.Cmd) {
	br := m.current.Manifest.BugReview
	if br == nil {
		return m, nil
	}
	switch key {
	case "esc", "b":
		return m.backToList(), nil
	case "up", "k":
		if m.choice > 0 {
			m.choice--
		}
	case "down", "j":
		if m.choice < len(br.Choices)-1 {
			m.choice++
		}
	case "enter":
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
	}
	// numeric shortcut 1..9
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		idx := int(key[0] - '1')
		if idx < len(br.Choices) {
			m.choice = idx
		}
	}
	return m, nil
}

func (m *model) backToProblem() (tea.Model, tea.Cmd) {
	switch m.current.Manifest.Track {
	case pack.TrackImplement:
		m.screen = screenImplement
	case pack.TrackBugReview:
		m.screen = screenBugReview
	}
	return m, nil
}

func (m *model) backToList() *model {
	m.screen = screenList
	m.current = nil
	m.problemMD = ""
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
	boxStyle    = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1)
)

func (m *model) View() tea.View {
	if m.quitting {
		return tea.View{Content: "bye.\n"}
	}
	var body string
	switch m.screen {
	case screenList:
		body = m.viewList()
	case screenImplement:
		body = m.viewImplement()
	case screenBugReview:
		body = m.viewBugReview()
	case screenResult:
		body = m.viewResult()
	}
	return tea.View{Content: body, AltScreen: true}
}

func (m *model) viewList() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("bugsim"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("packs dir: %s", m.cfg.PacksDir)))
	b.WriteString("\n\n")
	if len(m.packs) == 0 {
		b.WriteString("no packs discovered.\n")
		b.WriteString(dimStyle.Render("place pack directories under --packs and run again.\n"))
	} else {
		for i, p := range m.packs {
			line := fmt.Sprintf("  %s  [%s]  %s", p.ID, p.Track, p.Title)
			if i == m.cursor {
				line = cursorStyle.Render("> " + strings.TrimPrefix(line, "  "))
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter: open   up/down or j/k: move   q: quit"))
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errStyle.Render("error: " + m.err.Error()))
	}
	return b.String()
}

func (m *model) viewImplement() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.current.Manifest.Title))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("track: implement   runner: %s   difficulty: %s", m.current.Manifest.Runner, m.current.Manifest.Difficulty)))
	b.WriteString("\n\n")
	b.WriteString(m.problemMD)
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(fmt.Sprintf("workspace: %s", m.workDir)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("r: run tests   w: show workspace path   esc/b: back   q: quit"))
	if m.running {
		b.WriteString("\n")
		b.WriteString("running tests...")
	}
	if m.status != "" && !m.running {
		b.WriteString("\n")
		b.WriteString(m.status)
	}
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errStyle.Render("error: " + m.err.Error()))
	}
	return b.String()
}

func (m *model) viewBugReview() string {
	var b strings.Builder
	br := m.current.Manifest.BugReview
	b.WriteString(headerStyle.Render(m.current.Manifest.Title))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("track: bug review   difficulty: %s", m.current.Manifest.Difficulty)))
	b.WriteString("\n\n")
	b.WriteString(m.problemMD)
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(m.bugCorpus))
	b.WriteString("\n\n")
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
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.answered {
		if m.correct {
			b.WriteString(okStyle.Render("correct."))
		} else {
			b.WriteString(errStyle.Render("incorrect."))
			if m.revealLbl != "" {
				b.WriteString(" expected: " + m.revealLbl)
			}
		}
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("enter: back to list   q: quit"))
	} else {
		b.WriteString(dimStyle.Render("1-9 or up/down: choose   enter: submit   esc/b: back   q: quit"))
	}
	return b.String()
}

func (m *model) viewResult() string {
	var b strings.Builder
	title := m.current.Manifest.Title
	b.WriteString(headerStyle.Render(title + " — results"))
	b.WriteString("\n\n")
	if m.result == nil {
		b.WriteString("no result.\n")
	} else {
		if m.result.ExitCode == 0 {
			b.WriteString(okStyle.Render("PASS"))
		} else {
			b.WriteString(errStyle.Render(fmt.Sprintf("FAIL (exit %d)", m.result.ExitCode)))
		}
		b.WriteString("\n\n")
		if strings.TrimSpace(m.result.Stdout) != "" {
			b.WriteString(headerStyle.Render("stdout"))
			b.WriteString("\n")
			b.WriteString(m.result.Stdout)
			b.WriteString("\n")
		}
		if strings.TrimSpace(m.result.Stderr) != "" {
			b.WriteString(headerStyle.Render("stderr"))
			b.WriteString("\n")
			b.WriteString(m.result.Stderr)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("workspace: %s", m.workDir)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("enter/b: back to problem   q: quit"))
	return b.String()
}

// MustCleanup removes a workspace directory, ignoring errors.
func MustCleanup(workDir string) {
	if workDir == "" {
		return
	}
	if !strings.Contains(filepath.Base(workDir), "bugsim-") {
		return
	}
	_ = os.RemoveAll(workDir)
}
