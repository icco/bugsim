// Package tui implements the Bubble Tea program for bugsim.
package tui

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
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
	screenLanguage screen = iota
	screenDifficulty
	screenImplement
	screenBugReview
	screenResult
)

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	Run       key.Binding
	Next      key.Binding
	Languages key.Binding
	Quit      key.Binding
	Help      key.Binding
	Choice1   key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/submit")),
		Back:      key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc", "back")),
		Run:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "run tests")),
		Next:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next random pack")),
		Languages: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "change language")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Choice1:   key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"), key.WithHelp("1-9", "choose")),
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
		{k.Back, k.Run, k.Next, k.Languages},
		{k.Help, k.Quit, k.Choice1},
	}
}

// languageItem fronts a language option (e.g. "go") in the picker list.
type languageItem struct {
	id    string
	count int
}

func (l languageItem) Title() string       { return languageDisplay(l.id) }
func (l languageItem) Description() string { return fmt.Sprintf("%d pack%s", l.count, plural(l.count)) }
func (l languageItem) FilterValue() string { return l.id + " " + languageDisplay(l.id) }

// difficultyItem fronts a difficulty level + an "any" option.
type difficultyItem struct {
	level pack.Difficulty // empty means "any"
	count int
}

func (d difficultyItem) Title() string {
	if d.level == "" {
		return "Any"
	}
	s := string(d.level)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (d difficultyItem) Description() string {
	return fmt.Sprintf("%d pack%s", d.count, plural(d.count))
}

func (d difficultyItem) FilterValue() string { return string(d.level) }

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// languageDisplay formats a runner id for display: "go" -> "Go",
// "typescript" -> "TypeScript". Unknown ids fall back to the raw id so
// the picker still surfaces them.
func languageDisplay(id string) string {
	switch id {
	case "go":
		return "Go"
	case "typescript":
		return "TypeScript"
	default:
		if id == "" {
			return "(unknown)"
		}
		return id
	}
}

type model struct {
	cfg      Config
	keys     keyMap
	help     help.Model
	screen   screen
	err      error
	quitting bool

	summaries []pack.Summary

	language   string
	difficulty pack.Difficulty // "" => any
	pool       []pack.Summary  // packs filtered by language (+difficulty when set)

	languagesList    list.Model
	difficultiesList list.Model
	body             viewport.Model
	renderer         *glamour.TermRenderer

	rng *rand.Rand

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

	delegate := list.NewDefaultDelegate()
	languagesList := list.New(buildLanguageItems(summaries), delegate, 0, 0)
	languagesList.Title = "bugsim — pick a language"
	languagesList.SetShowStatusBar(false)
	languagesList.SetFilteringEnabled(false)
	languagesList.SetShowHelp(false)

	difficultiesList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	difficultiesList.Title = "pick a difficulty"
	difficultiesList.SetShowStatusBar(false)
	difficultiesList.SetFilteringEnabled(false)
	difficultiesList.SetShowHelp(false)

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return nil, fmt.Errorf("init markdown renderer: %w", err)
	}

	return &model{
		cfg:              cfg,
		keys:             defaultKeyMap(),
		help:             help.New(),
		summaries:        summaries,
		languagesList:    languagesList,
		difficultiesList: difficultiesList,
		body:             viewport.New(),
		renderer:         r,
		rng:              rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		screen:           screenLanguage,
	}, nil
}

func (m *model) Init() tea.Cmd { return nil }

// buildLanguageItems groups summaries by runner id and returns one
// list.Item per language, sorted by id.
func buildLanguageItems(summaries []pack.Summary) []list.Item {
	counts := map[string]int{}
	for _, s := range summaries {
		counts[s.Runner]++
	}
	ids := make([]string, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]list.Item, 0, len(ids))
	for _, id := range ids {
		items = append(items, languageItem{id: id, count: counts[id]})
	}
	return items
}

// buildDifficultyItems returns difficulty options filtered by language.
// The "Any" entry is included when the language has at least one pack.
// Levels with zero packs are still shown (greyed-out by count) so the
// picker is predictable.
func buildDifficultyItems(summaries []pack.Summary, language string) []list.Item {
	counts := map[pack.Difficulty]int{}
	total := 0
	for _, s := range summaries {
		if s.Runner != language {
			continue
		}
		counts[s.Difficulty]++
		total++
	}
	items := []list.Item{
		difficultyItem{level: pack.DifficultyEasy, count: counts[pack.DifficultyEasy]},
		difficultyItem{level: pack.DifficultyMedium, count: counts[pack.DifficultyMedium]},
		difficultyItem{level: pack.DifficultyHard, count: counts[pack.DifficultyHard]},
		difficultyItem{level: "", count: total},
	}
	return items
}

// matchingPool returns summaries with the given language and (if set)
// difficulty.
func matchingPool(summaries []pack.Summary, language string, difficulty pack.Difficulty) []pack.Summary {
	out := make([]pack.Summary, 0, len(summaries))
	for _, s := range summaries {
		if s.Runner != language {
			continue
		}
		if difficulty != "" && s.Difficulty != difficulty {
			continue
		}
		out = append(out, s)
	}
	return out
}

type testsDoneMsg struct {
	// packID identifies the pack that was running when the command was
	// dispatched. The TUI uses it to ignore late results from a pack the
	// user has already navigated away from.
	packID string
	res    *engine.TestResult
	err    error
}

func (m *model) runTestsCmd() tea.Cmd {
	p := m.current
	workDir := m.workDir
	timeout := m.cfg.Timeout
	return func() tea.Msg {
		rdef, err := runner.Load(p.Manifest.Runner)
		if err != nil {
			return testsDoneMsg{packID: p.Manifest.ID, err: err}
		}
		if err := engine.MaterializeWorkspace(p, workDir); err != nil {
			return testsDoneMsg{packID: p.Manifest.ID, err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := engine.EnsureDocker(ctx); err != nil {
			return testsDoneMsg{packID: p.Manifest.ID, err: err}
		}
		res, err := engine.RunTests(ctx, workDir, rdef, engine.DefaultLimits())
		return testsDoneMsg{packID: p.Manifest.ID, res: res, err: err}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)
		m.languagesList.SetSize(msg.Width, msg.Height-1)
		m.difficultiesList.SetSize(msg.Width, msg.Height-1)
		m.body.SetWidth(msg.Width)
		m.body.SetHeight(max(1, msg.Height-bodyChromeHeight()))
		return m, nil

	case testsDoneMsg:
		// If the user navigated to a different pack while tests were
		// running, drop the result on the floor. Without this guard the
		// TUI would either panic on m.current.Manifest.* or paint stale
		// output on top of the new pack.
		if m.current == nil || m.current.Manifest.ID != msg.packID {
			return m, nil
		}
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
		m.quitting = true
		return m, tea.Quit

	case key.Matches(k, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	switch m.screen {
	case screenLanguage:
		return m.onLanguageKey(k)
	case screenDifficulty:
		return m.onDifficultyKey(k)
	case screenImplement:
		return m.onImplementKey(k)
	case screenBugReview:
		return m.onBugReviewKey(k)
	case screenResult:
		return m.onResultKey(k)
	}
	return m, nil
}

func (m *model) onLanguageKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(k, m.keys.Enter) {
		if it, ok := m.languagesList.SelectedItem().(languageItem); ok {
			m.language = it.id
			m.difficulty = ""
			m.difficultiesList.SetItems(buildDifficultyItems(m.summaries, m.language))
			m.difficultiesList.Title = fmt.Sprintf("%s — pick a difficulty", languageDisplay(m.language))
			m.difficultiesList.Select(0)
			m.screen = screenDifficulty
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.languagesList, cmd = m.languagesList.Update(k)
	return m, cmd
}

func (m *model) onDifficultyKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, m.keys.Back):
		return m.backToLanguage(), nil
	case key.Matches(k, m.keys.Enter):
		if it, ok := m.difficultiesList.SelectedItem().(difficultyItem); ok {
			m.difficulty = it.level
			m.pool = matchingPool(m.summaries, m.language, m.difficulty)
			if len(m.pool) == 0 {
				m.err = fmt.Errorf("no packs match %s · %s", languageDisplay(m.language), difficultyLabel(m.difficulty))
				return m, nil
			}
			m.err = nil
			return m, m.openRandomPack()
		}
	}
	var cmd tea.Cmd
	m.difficultiesList, cmd = m.difficultiesList.Update(k)
	return m, cmd
}

// openRandomPack picks a fresh pack from m.pool and opens it. If the
// previous pack is in the pool and there are multiple options, it is
// excluded so the user does not get the same pack twice in a row.
func (m *model) openRandomPack() tea.Cmd {
	if len(m.pool) == 0 {
		return nil
	}
	candidates := m.pool
	if m.current != nil && len(m.pool) > 1 {
		filtered := candidates[:0:0]
		for _, s := range m.pool {
			if s.ID != m.current.Manifest.ID {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}
	pick := candidates[m.rng.IntN(len(candidates))]
	m.openPack(pick)
	return nil
}

func (m *model) openPack(s pack.Summary) {
	p, err := pack.Load(s.Dir)
	if err != nil {
		m.err = err
		return
	}
	md, err := p.ReadProblemMarkdown()
	if err != nil {
		m.err = err
		return
	}
	rendered, err := m.renderer.Render(md)
	if err != nil {
		rendered = md
	}

	// Reset per-pack state; clean up previous workspace if any.
	if m.workDir != "" {
		_ = os.RemoveAll(m.workDir)
		m.workDir = ""
	}
	m.current = p
	m.result = nil
	m.err = nil
	m.answered = false
	m.correct = false
	m.revealLbl = ""
	m.choice = 0
	m.status = ""
	m.running = false

	switch p.Manifest.Track {
	case pack.TrackImplement:
		tmp, err := os.MkdirTemp("", fmt.Sprintf("bugsim-%s-", p.Manifest.ID))
		if err != nil {
			m.err = err
			return
		}
		m.workDir = tmp
		m.body.SetContent(rendered)
		m.body.GotoTop()
		m.screen = screenImplement
	case pack.TrackBugReview:
		corpus, err := p.ReadBugCorpus()
		if err != nil {
			m.err = err
			return
		}
		m.body.SetContent(rendered + "\n\n" + corpus)
		m.body.GotoTop()
		m.screen = screenBugReview
	}
}

func (m *model) onImplementKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, m.keys.Back):
		return m.backToDifficulty(), nil
	case key.Matches(k, m.keys.Languages):
		return m.backToLanguage(), nil
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
		return m.backToDifficulty(), nil
	case key.Matches(k, m.keys.Languages):
		return m.backToLanguage(), nil
	case m.answered && key.Matches(k, m.keys.Next):
		return m, m.openRandomPack()
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
			return m, m.openRandomPack()
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
	case key.Matches(k, m.keys.Languages):
		return m.backToLanguage(), nil
	case key.Matches(k, m.keys.Back):
		return m.backToDifficulty(), nil
	case key.Matches(k, m.keys.Run):
		// Re-run tests against the same workspace (useful if the user
		// edited skeleton files in another window between runs).
		if m.running || m.current == nil {
			return m, nil
		}
		m.running = true
		m.status = "running tests..."
		m.screen = screenImplement
		// Restore the problem markdown — formatResult had taken over
		// the body when the previous run finished.
		md, err := m.current.ReadProblemMarkdown()
		if err == nil {
			rendered, rerr := m.renderer.Render(md)
			if rerr != nil {
				rendered = md
			}
			m.body.SetContent(rendered)
			m.body.GotoTop()
		}
		return m, m.runTestsCmd()
	case key.Matches(k, m.keys.Enter), key.Matches(k, m.keys.Next):
		return m, m.openRandomPack()
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(k)
	return m, cmd
}

// backToDifficulty resets per-pack state and returns to the difficulty
// picker, keeping the selected language so the user can pick a different
// difficulty without re-choosing the language.
func (m *model) backToDifficulty() *model {
	m.cleanupRun()
	m.current = nil
	m.result = nil
	m.err = nil
	m.status = ""
	m.running = false
	m.screen = screenDifficulty
	m.difficultiesList.SetItems(buildDifficultyItems(m.summaries, m.language))
	return m
}

// backToLanguage drops both filters and returns to the language picker.
func (m *model) backToLanguage() *model {
	m.cleanupRun()
	m.current = nil
	m.result = nil
	m.err = nil
	m.status = ""
	m.running = false
	m.language = ""
	m.difficulty = ""
	m.pool = nil
	m.screen = screenLanguage
	m.languagesList.SetItems(buildLanguageItems(m.summaries))
	return m
}

func (m *model) cleanupRun() {
	if m.workDir != "" && strings.Contains(m.workDir, "bugsim-") {
		_ = os.RemoveAll(m.workDir)
		m.workDir = ""
	}
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
	case screenLanguage:
		body = m.viewLanguage()
	case screenDifficulty:
		body = m.viewDifficulty()
	case screenImplement:
		body = m.viewImplement()
	case screenBugReview:
		body = m.viewBugReview()
	case screenResult:
		body = m.viewResult()
	}
	return tea.View{Content: body, AltScreen: true}
}

func (m *model) viewLanguage() string {
	var b strings.Builder
	b.WriteString(m.languagesList.View())
	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("error: "+m.err.Error()))
	}
	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func (m *model) viewDifficulty() string {
	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("language: %s", languageDisplay(m.language))))
	b.WriteString("\n")
	b.WriteString(m.difficultiesList.View())
	if m.err != nil {
		b.WriteString("\n" + errStyle.Render("error: "+m.err.Error()))
	}
	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

// difficultyLabel renders a difficulty value for status lines.
func difficultyLabel(d pack.Difficulty) string {
	if d == "" {
		return "any"
	}
	return string(d)
}

func (m *model) sessionLine() string {
	return fmt.Sprintf("language: %s · difficulty: %s · pack: %s",
		languageDisplay(m.language), difficultyLabel(m.difficulty), m.current.Manifest.ID)
}

func (m *model) viewImplement() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.current.Manifest.Title))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.sessionLine()))
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
	b.WriteString(dimStyle.Render(m.sessionLine()))
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
		b.WriteString(dimStyle.Render("[enter/n] next random  [esc] change difficulty  [L] change language") + "\n")
	}
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func (m *model) viewResult() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.current.Manifest.Title + " — results"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.sessionLine()))
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
	b.WriteString(dimStyle.Render("[enter/n] next random  [r] re-run tests  [esc] change difficulty  [L] change language") + "\n")
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
