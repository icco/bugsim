package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/icco/bugsim/internal/pack"
	"github.com/icco/bugsim/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "play":
		os.Exit(cmdPlay(os.Args[2:]))
	case "list":
		os.Exit(cmdList(os.Args[2:]))
	case "verify-pack":
		os.Exit(cmdVerifyPack(os.Args[2:]))
	case "-h", "--help", "help":
		usage(os.Stdout)
		return
	default:
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w *os.File) {
	fmt.Fprintf(w, `bugsim — terminal flight simulator for software engineers

usage:
  bugsim play [--packs DIR] [--timeout DUR]
  bugsim list [--packs DIR]
  bugsim verify-pack PACK_DIR
`)
}

func cmdPlay(args []string) int {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	packsDir := fs.String("packs", "./packs", "directory containing pack subdirectories")
	timeout := fs.Duration("timeout", 2*time.Minute, "per-test subprocess timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if _, err := os.Stat(*packsDir); err != nil {
		fmt.Fprintf(os.Stderr, "packs dir: %v\n", err)
		return 1
	}
	abs, _ := filepath.Abs(*packsDir)
	m, err := tui.New(tui.Config{PacksDir: abs, Timeout: *timeout})
	if err != nil {
		fmt.Fprintf(os.Stderr, "init: %v\n", err)
		return 1
	}
	prog := tea.NewProgram(m)
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		return 1
	}
	return 0
}

func cmdList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	packsDir := fs.String("packs", "./packs", "directory containing pack subdirectories")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	summaries, err := pack.Discover(*packsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "discover: %v\n", err)
		return 1
	}
	if len(summaries) == 0 {
		fmt.Println("no packs found.")
		return 0
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].ID < summaries[j].ID })
	for _, s := range summaries {
		fmt.Printf("%-30s  %-12s  %s\n", s.ID, s.Track, s.Title)
	}
	return 0
}

func cmdVerifyPack(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: bugsim verify-pack PACK_DIR")
		return 2
	}
	p, err := pack.Load(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
		return 1
	}
	fmt.Printf("ok: %s (%s, runner=%s)\n", p.Manifest.ID, p.Manifest.Track, p.Manifest.Runner)
	return 0
}
