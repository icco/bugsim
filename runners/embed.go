package runners

import "embed"

// Files contains built-in runner definitions (JSON).
//
//go:embed *.json
var Files embed.FS
