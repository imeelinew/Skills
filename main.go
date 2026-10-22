package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"skills/internal/catalog"
	"skills/internal/ui"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "skills:", err)
		os.Exit(1)
	}
	model := ui.New(catalog.Sources(home, os.Getenv("XDG_CONFIG_HOME")), home)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "skills:", err)
		os.Exit(1)
	}
}
