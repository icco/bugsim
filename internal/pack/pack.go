package pack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const formatVersion = 1

// Track identifies which training track a pack belongs to.
type Track string

const (
	TrackImplement   Track = "implement"
	TrackBugReview   Track = "bug_review"
)

// Difficulty is a coarse UX label.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Manifest is the machine-readable pack metadata (manifest.yaml).
type Manifest struct {
	PackFormatVersion int        `yaml:"pack_format_version"`
	ID                string     `yaml:"id"`
	Title             string     `yaml:"title"`
	Track             Track      `yaml:"track"`
	Runner            string     `yaml:"runner"`
	Difficulty        Difficulty `yaml:"difficulty"`
	Tags              []string   `yaml:"tags,omitempty"`

	RecommendedMinutes *int `yaml:"recommended_minutes,omitempty"`

	BugReview *BugReview `yaml:"bug_review,omitempty"`
}

// BugReview configures the bug_review track (v1: multiple choice).
type BugReview struct {
	Prompt  string         `yaml:"prompt"`
	Choices []BugChoice    `yaml:"choices"`
}

// BugChoice is one MCQ option.
type BugChoice struct {
	ID      string `yaml:"id"`
	Label   string `yaml:"label"`
	Correct bool   `yaml:"correct"`
}

// Pack is a loaded pack directory plus parsed manifest.
type Pack struct {
	Dir      string
	Manifest Manifest
}

// Summary is a lightweight listing entry.
type Summary struct {
	ID    string
	Title string
	Track Track
	Dir   string
}

// IsTemplatePack returns true for packs used as authoring templates.
func IsTemplatePack(name string) bool {
	return strings.HasPrefix(name, "_")
}

// Discover scans root for immediate child directories that contain manifest.yaml.
func Discover(root string) ([]Summary, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if IsTemplatePack(e.Name()) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		mf := filepath.Join(dir, "manifest.yaml")
		if _, err := os.Stat(mf); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		p, err := Load(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dir, err)
		}
		out = append(out, Summary{
			ID:    p.Manifest.ID,
			Title: p.Manifest.Title,
			Track: p.Manifest.Track,
			Dir:   dir,
		})
	}
	return out, nil
}

// Load reads and validates manifest.yaml in dir.
func Load(dir string) (*Pack, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	p := &Pack{Dir: dir, Manifest: m}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// Validate checks manifest fields and required on-disk layout.
func (p *Pack) Validate() error {
	m := p.Manifest
	if m.PackFormatVersion != formatVersion {
		return fmt.Errorf("unsupported pack_format_version: got %d want %d", m.PackFormatVersion, formatVersion)
	}
	if strings.TrimSpace(m.ID) == "" {
		return errors.New("manifest.id is required")
	}
	if strings.TrimSpace(m.Title) == "" {
		return errors.New("manifest.title is required")
	}
	switch m.Track {
	case TrackImplement, TrackBugReview:
	default:
		return fmt.Errorf("manifest.track must be %q or %q", TrackImplement, TrackBugReview)
	}
	switch m.Difficulty {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
	default:
		return fmt.Errorf("manifest.difficulty must be one of easy|medium|hard")
	}

	switch m.Track {
	case TrackImplement:
		if strings.TrimSpace(m.Runner) == "" {
			return errors.New("manifest.runner is required for implement packs")
		}
		if err := requireDir(filepath.Join(p.Dir, "skeleton")); err != nil {
			return fmt.Errorf("implement packs require skeleton/: %w", err)
		}
		if err := requireDir(filepath.Join(p.Dir, "hidden_tests")); err != nil {
			return fmt.Errorf("implement packs require hidden_tests/: %w", err)
		}
	case TrackBugReview:
		if err := requireDir(filepath.Join(p.Dir, "bug")); err != nil {
			return fmt.Errorf("bug_review packs require bug/: %w", err)
		}
		if m.BugReview == nil {
			return errors.New("manifest.bug_review is required for bug_review packs")
		}
		if strings.TrimSpace(m.BugReview.Prompt) == "" {
			return errors.New("manifest.bug_review.prompt is required")
		}
		if len(m.BugReview.Choices) < 2 {
			return errors.New("manifest.bug_review.choices must include at least 2 options")
		}
		correct := 0
		for _, c := range m.BugReview.Choices {
			if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.Label) == "" {
				return errors.New("each bug_review choice requires id and label")
			}
			if c.Correct {
				correct++
			}
		}
		if correct != 1 {
			return fmt.Errorf("manifest.bug_review.choices must have exactly one correct: true (got %d)", correct)
		}
	}

	if _, err := os.ReadFile(filepath.Join(p.Dir, "problem.md")); err != nil {
		return fmt.Errorf("problem.md: %w", err)
	}
	return nil
}

// corpusExts is the allow-list of file extensions concatenated by
// ReadBugCorpus. Update it (and a test) when adding support for new pack
// languages.
var corpusExts = map[string]struct{}{
	".md":   {},
	".txt":  {},
	".go":   {},
	".json": {},
	".yaml": {},
	".yml":  {},
	".ts":   {},
	".tsx":  {},
	".js":   {},
	".mjs":  {},
	".cjs":  {},
}

func isCorpusExt(ext string) bool {
	_, ok := corpusExts[strings.ToLower(ext)]
	return ok
}

func requireDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}
	return nil
}

// ReadProblemMarkdown returns problem.md contents.
func (p *Pack) ReadProblemMarkdown() (string, error) {
	b, err := os.ReadFile(filepath.Join(p.Dir, "problem.md"))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadBugCorpus returns concatenated Markdown/text files under bug/ for display.
func (p *Pack) ReadBugCorpus() (string, error) {
	if p.Manifest.Track != TrackBugReview {
		return "", errors.New("pack is not bug_review")
	}
	root := filepath.Join(p.Dir, "bug")
	var b strings.Builder
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isCorpusExt(filepath.Ext(path)) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "---- %s ----\n%s\n\n", rel, string(body))
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}
