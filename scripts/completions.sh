#!/bin/sh
# Generate shell completions for the bugsim CLI.
# Invoked by GoReleaser's before.hooks; safe to run by hand too.
set -eu

mkdir -p completions
go run ./cmd/bugsim completion bash > completions/bugsim.bash
go run ./cmd/bugsim completion zsh  > completions/_bugsim
go run ./cmd/bugsim completion fish > completions/bugsim.fish
