// This file exists so `go test ./...` and `go vet ./...` from the bugsim
// module root do not descend into pack source trees. Pack content is only
// expected to compile after materialization into a workspace, where the
// pack's skeleton/go.mod (or equivalent toolchain config) takes effect.

module bugsim.local/packs

go 1.26
