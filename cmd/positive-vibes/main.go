package main

import (
	"fmt"
	"os"

	"github.com/chaz8081/positive-vibes/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		// Keep output chill but useful
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
