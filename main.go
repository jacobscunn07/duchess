package main

import (
	"os"

	"github.com/jacobscunn07/duchess/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
