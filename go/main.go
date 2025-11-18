package main

import (
	"fmt"
	"runtime"
)

// Version is injected via ldflags in default.nix
var Version = "dev"

func main() {
	fmt.Printf("Hello from Nix! Version: %s, Go: %s\n", Version, runtime.Version())
}
