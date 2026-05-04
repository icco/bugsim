// Command bugsim is a terminal flight simulator for software engineers.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/icco/bugsim/internal/pack"
	"github.com/icco/bugsim/internal/tui"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "bugsim",
		Short:         "Terminal flight simulator for software engineers",
		Long:          "bugsim runs short, repeatable practice scenarios for implementation and debugging skills.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(newPlayCmd(), newListCmd(), newVerifyCmd())
	return root
}

func newPlayCmd() *cobra.Command {
	var packsDir string
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "play",
		Short: "Open the interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			abs, err := filepath.Abs(packsDir)
			if err != nil {
				return err
			}
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("packs dir: %w", err)
			}
			m, err := tui.New(tui.Config{PacksDir: abs, Timeout: timeout})
			if err != nil {
				return err
			}
			prog := tea.NewProgram(m, tea.WithContext(cmd.Context()))
			_, err = prog.Run()
			return err
		},
	}
	cmd.Flags().StringVar(&packsDir, "packs", "./packs", "directory containing pack subdirectories")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "per-test subprocess timeout")
	return cmd
}

func newListCmd() *cobra.Command {
	var packsDir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered packs",
		RunE: func(cmd *cobra.Command, args []string) error {
			summaries, err := pack.Discover(packsDir)
			if err != nil {
				return fmt.Errorf("discover: %w", err)
			}
			if len(summaries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no packs found.")
				return nil
			}
			sort.Slice(summaries, func(i, j int) bool { return summaries[i].ID < summaries[j].ID })
			for _, s := range summaries {
				fmt.Fprintf(cmd.OutOrStdout(), "%-30s  %-12s  %s\n", s.ID, s.Track, s.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&packsDir, "packs", "./packs", "directory containing pack subdirectories")
	return cmd
}

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify-pack PACK_DIR",
		Short: "Validate a pack manifest and on-disk layout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := pack.Load(args[0])
			if err != nil {
				return fmt.Errorf("invalid: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ok: %s (%s, runner=%s)\n", p.Manifest.ID, p.Manifest.Track, p.Manifest.Runner)
			return nil
		},
	}
}
