package main

import (
	"fmt"
	"os"

	"github.com/janyksteenbeek/autoploi/internal/actions"
)

func main() {
	if err := actions.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
