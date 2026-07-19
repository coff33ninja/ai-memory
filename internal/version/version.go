package version

import (
	"fmt"
	"runtime/debug"
)

// Set via -ldflags at build time.
var Version = "dev"

func Full() string {
	if bi, ok := debug.ReadBuildInfo(); ok && Version == "dev" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return Version
}

func String() string {
	return fmt.Sprintf("ai-memory %s", Full())
}
