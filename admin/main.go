// ralphy-admin is a TUI dashboard for monitoring running ralphy loop sessions.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ksdaemon/ralphy-admin/internal/registry"
	"github.com/ksdaemon/ralphy-admin/internal/tui"
)

func main() {
	reg := registry.New()

	// Clean up expired sessions (>24h) on startup
	if removed, err := reg.CleanupExpired(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to clean expired sessions: %v\n", err)
	} else if removed > 0 {
		fmt.Fprintf(os.Stderr, "Cleaned up %d expired session(s)\n", removed)
	}

	model := tui.NewModel(reg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running ralphy-admin: %v\n", err)
		os.Exit(1)
	}
}
