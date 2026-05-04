package tui

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/icco/bugsim/internal/pack"
)

// writePack lays a minimal valid pack on disk under root/<id>.
func writePack(t *testing.T, root, id, runner, track, difficulty string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "pack_format_version: 1\n" +
		"id: " + id + "\n" +
		"title: " + id + " title\n" +
		"track: " + track + "\n" +
		"runner: " + runner + "\n" +
		"difficulty: " + difficulty + "\n"
	if track == "bug_review" {
		manifest += "bug_review:\n" +
			"  prompt: pick\n" +
			"  choices:\n" +
			"    - id: a\n" +
			"      label: a\n" +
			"      correct: true\n" +
			"    - id: b\n" +
			"      label: b\n" +
			"      correct: false\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "problem.md"), []byte("# "+id+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	switch track {
	case "implement":
		if err := os.MkdirAll(filepath.Join(dir, "skeleton"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "skeleton", "x.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "hidden_tests"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "hidden_tests", "x_test.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	case "bug_review":
		if err := os.MkdirAll(filepath.Join(dir, "bug"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "bug", "notes.md"), []byte("notes\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// newTestModel returns the concrete *model so tests can poke at private
// state directly.
func newTestModel(t *testing.T, packsDir string) *model {
	t.Helper()
	mi, err := New(Config{PacksDir: packsDir, Timeout: time.Minute})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	mm, ok := mi.(*model)
	if !ok {
		t.Fatalf("expected *model, got %T", mi)
	}
	mm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return mm
}

func keypress(s string) tea.KeyPressMsg {
	if s == "enter" {
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	}
	if s == "esc" {
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	if s == "down" {
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	if s == "up" {
		return tea.KeyPressMsg{Code: tea.KeyUp}
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

// TestPickerFlowLanguageThenDifficulty walks the new picker pipeline:
// the model starts on the language screen, choosing a language opens
// the difficulty screen, and choosing a difficulty drops the user into
// the random pack matching both filters.
func TestPickerFlowLanguageThenDifficulty(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-easy", "go", "bug_review", "easy")
	writePack(t, root, "go-medium", "go", "implement", "medium")
	writePack(t, root, "ts-hard", "typescript", "implement", "hard")

	m := newTestModel(t, root)
	if m.screen != screenLanguage {
		t.Fatalf("initial screen = %d, want screenLanguage", m.screen)
	}
	// First language item should be "go" (sorted alphabetically).
	m.Update(keypress("enter"))
	if m.screen != screenDifficulty {
		t.Fatalf("after picking language, screen = %d, want screenDifficulty", m.screen)
	}
	if m.language != "go" {
		t.Fatalf("language = %q, want go", m.language)
	}
	// Pick "easy" — first item in our difficulty list.
	m.Update(keypress("enter"))
	if m.difficulty != pack.DifficultyEasy {
		t.Fatalf("difficulty = %q, want easy", m.difficulty)
	}
	if m.current == nil {
		t.Fatal("expected a pack to be opened")
	}
	if m.current.Manifest.ID != "go-easy" {
		t.Fatalf("opened pack = %q, want go-easy", m.current.Manifest.ID)
	}
	if m.screen != screenBugReview {
		t.Fatalf("screen after picking pack = %d, want screenBugReview", m.screen)
	}
}

// TestPickerEmptyPoolShowsError makes sure picking a difficulty that has
// no packs surfaces a clear in-screen error rather than silently
// dropping into a nil pack.
func TestPickerEmptyPoolShowsError(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-easy", "go", "bug_review", "easy")

	m := newTestModel(t, root)
	m.Update(keypress("enter")) // pick go
	// move to "hard"
	m.Update(keypress("down"))
	m.Update(keypress("down"))
	m.Update(keypress("enter"))
	if m.err == nil {
		t.Fatalf("expected error for empty pool, got pool=%v", m.pool)
	}
	if m.screen != screenDifficulty {
		t.Fatalf("screen = %d, want stay on difficulty", m.screen)
	}
}

// TestBackFromDifficultyReturnsToLanguage verifies that pressing esc on
// the difficulty picker walks back to the language picker without
// keeping a stale language selected.
func TestBackFromDifficultyReturnsToLanguage(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-easy", "go", "bug_review", "easy")
	writePack(t, root, "ts-hard", "typescript", "implement", "hard")

	m := newTestModel(t, root)
	m.Update(keypress("enter")) // select language
	if m.screen != screenDifficulty {
		t.Fatalf("expected difficulty screen, got %d", m.screen)
	}
	m.Update(keypress("esc"))
	if m.screen != screenLanguage {
		t.Fatalf("expected language screen, got %d", m.screen)
	}
	if m.language != "" {
		t.Fatalf("language should reset, got %q", m.language)
	}
}

// TestMatchingPoolFilters covers the filter helper directly because the
// rest of the navigation depends on it producing a stable, sorted list
// of summaries for a given (runner, difficulty).
func TestMatchingPoolFilters(t *testing.T) {
	all := []pack.Summary{
		{ID: "go-easy", Runner: "go", Difficulty: pack.DifficultyEasy},
		{ID: "go-hard", Runner: "go", Difficulty: pack.DifficultyHard},
		{ID: "ts-medium", Runner: "typescript", Difficulty: pack.DifficultyMedium},
	}
	t.Run("language only", func(t *testing.T) {
		got := matchingPool(all, "go", "")
		if len(got) != 2 {
			t.Fatalf("want 2 go packs, got %d", len(got))
		}
	})
	t.Run("language + difficulty", func(t *testing.T) {
		got := matchingPool(all, "go", pack.DifficultyEasy)
		if len(got) != 1 || got[0].ID != "go-easy" {
			t.Fatalf("unexpected: %v", got)
		}
	})
	t.Run("no match", func(t *testing.T) {
		got := matchingPool(all, "go", pack.DifficultyMedium)
		if len(got) != 0 {
			t.Fatalf("expected empty, got %v", got)
		}
	})
}

// TestBuildLanguageItemsCountsAndSortsRunners makes sure the picker's
// language list contains exactly the distinct runner ids found in the
// summaries, sorted, with accurate per-language pack counts.
func TestBuildLanguageItemsCountsAndSortsRunners(t *testing.T) {
	all := []pack.Summary{
		{ID: "ts-1", Runner: "typescript", Difficulty: pack.DifficultyEasy},
		{ID: "go-1", Runner: "go", Difficulty: pack.DifficultyEasy},
		{ID: "go-2", Runner: "go", Difficulty: pack.DifficultyHard},
		{ID: "ts-2", Runner: "typescript", Difficulty: pack.DifficultyHard},
		{ID: "ts-3", Runner: "typescript", Difficulty: pack.DifficultyHard},
	}
	items := buildLanguageItems(all)
	if len(items) != 2 {
		t.Fatalf("want 2 languages, got %d", len(items))
	}
	got := []string{items[0].(languageItem).id, items[1].(languageItem).id}
	want := []string{"go", "typescript"}
	sort.Strings(got)
	sort.Strings(want)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("languages[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	for _, it := range items {
		li := it.(languageItem)
		switch li.id {
		case "go":
			if li.count != 2 {
				t.Fatalf("go count = %d, want 2", li.count)
			}
		case "typescript":
			if li.count != 3 {
				t.Fatalf("typescript count = %d, want 3", li.count)
			}
		}
	}
}

// TestNextRandomPackAvoidsImmediateRepeat asserts that calling
// openRandomPack while the previous pack is still loaded picks a
// different pack from the pool when more than one option exists.
func TestNextRandomPackAvoidsImmediateRepeat(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-a", "go", "bug_review", "easy")
	writePack(t, root, "go-b", "go", "bug_review", "easy")

	m := newTestModel(t, root)
	m.Update(keypress("enter")) // language go
	m.Update(keypress("enter")) // difficulty easy

	first := m.current.Manifest.ID
	// Trigger another random pick (still on bug review screen — answer + Enter).
	m.Update(keypress("enter")) // answer
	if !m.answered {
		t.Fatal("expected first MCQ submission to set answered")
	}
	m.Update(keypress("enter")) // next random
	second := m.current.Manifest.ID
	if first == second {
		t.Fatalf("next random returned the same pack twice in a row: %q", first)
	}
}

// TestStaleTestsDoneMsgIgnored simulates the bug we'd hit if a user
// navigated away from a pack while a docker run was in flight: the late
// testsDoneMsg should not paint stale stdout over a different pack (or
// crash on a nil m.current).
func TestStaleTestsDoneMsgIgnored(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-1", "go", "implement", "easy")
	writePack(t, root, "go-2", "go", "implement", "easy")

	m := newTestModel(t, root)
	m.Update(keypress("enter")) // language go
	m.Update(keypress("enter")) // difficulty easy
	if m.current == nil {
		t.Fatal("expected a pack to be opened")
	}
	stalePackID := m.current.Manifest.ID
	// Start a fictitious run, then have the user immediately back out.
	m.running = true
	m.Update(keypress("esc"))
	if m.current != nil {
		t.Fatal("expected back to clear current pack")
	}
	// Late message arrives — must be a no-op.
	res := newModelMessage(m, testsDoneMsg{packID: stalePackID, res: nil, err: nil})
	if res.screen != screenDifficulty {
		t.Fatalf("expected to stay on difficulty screen, got %d", res.screen)
	}
}

// TestRunningResetWhenSwitchingPacks ensures the "running tests..."
// indicator doesn't bleed into the next pack when the user advances
// before the previous run reports back.
func TestRunningResetWhenSwitchingPacks(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-1", "go", "implement", "easy")
	writePack(t, root, "go-2", "go", "implement", "easy")

	m := newTestModel(t, root)
	m.Update(keypress("enter"))
	m.Update(keypress("enter"))
	m.running = true
	m.Update(keypress("n")) // user advances mid-run from screenImplement?
	// 'n' isn't bound on screenImplement so it falls through to viewport;
	// drive openRandomPack manually instead to simulate the result-screen
	// path.
	m.openRandomPack()
	if m.running {
		t.Fatal("running flag must reset when a new pack is opened")
	}
}

// TestResultBackGoesToDifficulty asserts that pressing esc on the
// result screen takes the user to the difficulty picker (not into a
// stale "back to pack" view that still has stdout in the body).
func TestResultBackGoesToDifficulty(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-1", "go", "implement", "easy")

	m := newTestModel(t, root)
	m.Update(keypress("enter")) // language
	m.Update(keypress("enter")) // difficulty
	// Pretend tests just finished.
	m.screen = screenResult
	m.result = nil
	m.Update(keypress("esc"))
	if m.screen != screenDifficulty {
		t.Fatalf("esc on result should land on difficulty, got %d", m.screen)
	}
}

// TestBugReviewIncorrectAdvancesAfterAnswer makes sure picking the wrong
// MCQ option still shows the expected answer label and lets the user
// move on with enter/n.
func TestBugReviewIncorrectAdvancesAfterAnswer(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "br-1", "go", "bug_review", "easy")
	writePack(t, root, "br-2", "go", "bug_review", "easy")

	m := newTestModel(t, root)
	m.Update(keypress("enter")) // language
	m.Update(keypress("enter")) // difficulty
	if m.screen != screenBugReview {
		t.Fatalf("expected bug review screen, got %d", m.screen)
	}
	// Move cursor to the second (incorrect) choice and submit.
	m.Update(keypress("down"))
	m.Update(keypress("enter"))
	if !m.answered {
		t.Fatal("expected answered to be true after submission")
	}
	if m.correct {
		t.Fatal("second choice should be incorrect for the test fixture")
	}
	if m.revealLbl == "" {
		t.Fatal("revealLbl should be populated when answer is incorrect")
	}
	prev := m.current.Manifest.ID
	m.Update(keypress("n"))
	if m.current.Manifest.ID == prev {
		t.Fatal("'n' should advance to the other matching pack")
	}
}

// newModelMessage drives the model's Update and returns the post-state.
func newModelMessage(m *model, msg tea.Msg) *model {
	upd, _ := m.Update(msg)
	mm, _ := upd.(*model)
	if mm == nil {
		return m
	}
	return mm
}

// TestViewSmokeAcrossScreens drives the model through every screen
// transition and renders View() at each step. The intent is to catch
// nil-pointer panics or empty-string crashes when one of the screens
// dereferences an unset field (e.g. m.current).
func TestViewSmokeAcrossScreens(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "go-easy-bug", "go", "bug_review", "easy")
	writePack(t, root, "go-easy-impl", "go", "implement", "easy")

	m := newTestModel(t, root)
	steps := []string{
		"language",
		"language enter",
		"difficulty enter",
		"opened pack",
	}
	for _, step := range steps {
		v := m.View()
		if v.Content == "" {
			t.Fatalf("View at %q returned empty content", step)
		}
		switch step {
		case "language":
			m.Update(keypress("enter"))
		case "language enter":
			m.Update(keypress("enter"))
		case "difficulty enter":
			// On the opened pack screen now.
		}
	}
	// After answering an MCQ pack, View() should render the reveal text
	// without panicking.
	if m.screen == screenBugReview {
		m.Update(keypress("enter"))
		v := m.View()
		if v.Content == "" {
			t.Fatal("View on answered bug_review returned empty content")
		}
	}
	// Force the result screen with a mock result so we hit viewResult.
	m.screen = screenResult
	if m.current == nil {
		m.current = &pack.Pack{Manifest: pack.Manifest{Title: "x", ID: "x"}}
	}
	v := m.View()
	if v.Content == "" {
		t.Fatal("View on result screen returned empty content")
	}
}

// TestLanguageDisplay ensures runner ids show up with friendly casing
// in the picker (and that unknown ids fall back to the raw id).
func TestLanguageDisplay(t *testing.T) {
	cases := map[string]string{
		"go":         "Go",
		"typescript": "TypeScript",
		"":           "(unknown)",
		"rust":       "rust",
	}
	for in, want := range cases {
		if got := languageDisplay(in); got != want {
			t.Fatalf("languageDisplay(%q) = %q, want %q", in, got, want)
		}
	}
}
