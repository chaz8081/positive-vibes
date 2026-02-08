package main

import (
	"fmt"
	"github.com/chaz8081/positive-vibes/internal/cli"
)

func main() {
	// Friendly startup message
	fmt.Println("positive-vibes — harmonizing your AI tooling ✨")
	if err := cli.Execute(); err != nil {
		// Keep output chill but useful
		fmt.Printf("error: %v\n", err)
	}
}
