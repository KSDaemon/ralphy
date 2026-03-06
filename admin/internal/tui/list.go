package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ksdaemon/ralphy-admin/internal/session"
)

// Minimum column widths for fixed-width columns
const (
	minColTool        = 10
	minColIteration   = 10
	minColStatus      = 12
	minColHeartbeat   = 12
	minColUptime      = 8
	minColProject     = 10
	minColBranch      = 12
	minColUserStories = 16
	tablePadding      = 4 // 2 chars left margin + gaps between columns
)

// columnWidths holds the computed column widths for the current terminal size.
type columnWidths struct {
	project     int
	branch      int
	tool        int
	iteration   int
	userStories int
	status      int
	heartbeat   int
	uptime      int
}

// computeColumns calculates column widths based on available terminal width
// and actual session data. It examines the longest project/branch names and
// distributes the available space proportionally so that columns are only
// truncated when the terminal is genuinely too narrow.
func computeColumns(termWidth int, sessions []*session.Session) columnWidths {
	if termWidth <= 0 {
		termWidth = 120 // reasonable default
	}

	cw := columnWidths{
		tool:        minColTool,
		iteration:   minColIteration,
		userStories: minColUserStories,
		status:      minColStatus,
		heartbeat:   minColHeartbeat,
		uptime:      minColUptime,
	}

	// Fixed columns total + padding (2 left margin + column gaps)
	fixedTotal := cw.tool + cw.iteration + cw.userStories + cw.status + cw.heartbeat + cw.uptime
	overhead := tablePadding + 6 // 6 gaps between 7 remaining fixed cols
	remaining := termWidth - fixedTotal - overhead

	// Find the longest project and branch names across all sessions.
	maxProject := len("PROJECT") // at least as wide as the header
	maxBranch := len("BRANCH")
	for _, s := range sessions {
		if len(s.Project) > maxProject {
			maxProject = len(s.Project)
		}
		if len(s.Branch) > maxBranch {
			maxBranch = len(s.Branch)
		}
	}

	if remaining < minColProject+minColBranch+2 {
		// Terminal is very narrow — give minimums
		cw.project = minColProject
		cw.branch = minColBranch
	} else if maxProject+maxBranch <= remaining {
		// Everything fits — no truncation needed, give exact widths.
		cw.project = maxProject
		cw.branch = maxBranch
		// Distribute any leftover space proportionally.
		leftover := remaining - maxProject - maxBranch
		if leftover > 0 {
			projExtra := leftover * maxProject / (maxProject + maxBranch)
			cw.project += projExtra
			cw.branch += leftover - projExtra
		}
	} else {
		// Not enough room for both — split proportionally to content needs.
		cw.project = remaining * maxProject / (maxProject + maxBranch)
		cw.branch = remaining - cw.project
		// Enforce minimums, giving the remainder to the other column.
		if cw.project < minColProject {
			cw.project = minColProject
			cw.branch = remaining - cw.project
		}
		if cw.branch < minColBranch {
			cw.branch = minColBranch
			cw.project = remaining - cw.branch
		}
	}

	return cw
}

// updateList handles key events in the list view.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, DefaultKeyMap.Quit):
		return m, tea.Quit

	case keyMatches(msg, DefaultKeyMap.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case keyMatches(msg, DefaultKeyMap.Down):
		if m.cursor < len(m.sessions)-1 {
			m.cursor++
		}

	case keyMatches(msg, DefaultKeyMap.Enter):
		if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
			m.selected = m.sessions[m.cursor]
			m.currentView = viewDetail
			m.statusMsg = ""
		}

	case keyMatches(msg, DefaultKeyMap.Kill):
		sess := m.getSelectedSession()
		if sess != nil {
			m.confirming = confirmKill
			m.confirmText = fmt.Sprintf("Kill session %s (PID %d)? [y/n]", sess.Project, sess.PID)
		}

	case keyMatches(msg, DefaultKeyMap.Pause):
		sess := m.getSelectedSession()
		if sess != nil {
			return m, m.pauseResumeSession(sess)
		}

	case keyMatches(msg, DefaultKeyMap.Refresh):
		m.statusMsg = ""
		return m, m.refreshSessions
	}

	return m, nil
}

// viewList renders the list screen.
func (m Model) viewList() string {
	var b strings.Builder

	cw := computeColumns(m.width, m.sessions)

	// Title with summary
	activeCount := 0
	pausedCount := 0
	finishedCount := 0
	for _, s := range m.sessions {
		switch s.DisplayStatus() {
		case "running", "stale":
			activeCount++
		case "paused":
			pausedCount++
		case "completed", "interrupted", "max_iterations_reached", "dead":
			finishedCount++
		}
	}

	title := fmt.Sprintf("Ralphy Admin - %d session(s)", len(m.sessions))
	var parts []string
	if activeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d active", activeCount))
	}
	if pausedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d paused", pausedCount))
	}
	if finishedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d finished", finishedCount))
	}
	if len(parts) > 0 {
		title += " (" + strings.Join(parts, ", ") + ")"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if len(m.sessions) == 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("  No ralphy sessions found (recent 24h)."))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render("  Start a ralphy loop in any project and it will appear here."))
		b.WriteString("\n")
	} else {
		// Table header
		header := renderRow(cw, "PROJECT", "BRANCH", "TOOL", "ITERATION", "USER STORIES", "STATUS", "HEARTBEAT", "UPTIME", lipgloss.Style{})
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")

		// Table rows
		for i, sess := range m.sessions {
			status := sess.DisplayStatus()

			project := truncate(sess.Project, cw.project)
			branch := truncate(sess.Branch, cw.branch)
			tool := truncate(sess.Tool, cw.tool)
			iter := pad(sess.IterationProgress(), cw.iteration)
			usProgress := renderUserStoriesCell(sess, cw.userStories)
			stText := pad(status, cw.status)
			hb := pad(sess.FormatHeartbeat(), cw.heartbeat)
			uptime := pad(sess.FormatUptime(), cw.uptime)

			if i == m.cursor {
				// Build entire row as plain text, then apply selectedStyle uniformly.
				// Status and user stories get merged styles to preserve colors
				// on the selected background.
				selStatus := selectedStyle.Foreground(statusColor(status))
				before := fmt.Sprintf("  %-*s %-*s %-*s %s ",
					cw.project, project,
					cw.branch, branch,
					cw.tool, tool,
					iter)
				after := fmt.Sprintf(" %s %s", hb, uptime)
				// User stories with selected background
				selUS := renderUserStoriesCellSelected(sess, cw.userStories)
				// Pad the trailing part to fill remaining width
				usedWidth := lipgloss.Width(before) + cw.userStories + 1 + lipgloss.Width(stText) + lipgloss.Width(after)
				trailing := ""
				if m.width > usedWidth {
					trailing = strings.Repeat(" ", m.width-usedWidth)
				}
				row := selectedStyle.Render(before) +
					selUS + " " +
					selStatus.Render(stText) +
					selectedStyle.Render(after+trailing)
				b.WriteString(row)
			} else {
				row := fmt.Sprintf("  %-*s %-*s %-*s %s ",
					cw.project, project,
					cw.branch, branch,
					cw.tool, tool,
					iter,
				)
				row += usProgress + " " +
					statusStyle(status).Render(stText) +
					fmt.Sprintf(" %s %s", hb, uptime)
				b.WriteString(row)
			}
			b.WriteString("\n")
		}
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString("\n")
		if m.statusError {
			b.WriteString(errorStyle.Render("  " + m.statusMsg))
		} else {
			b.WriteString(infoStyle.Render("  " + m.statusMsg))
		}
		b.WriteString("\n")
	}

	// Confirmation dialog
	if m.confirming != confirmNone {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render("  " + m.confirmText))
		b.WriteString("\n")
		b.WriteString(ConfirmHelp())
	} else {
		b.WriteString("\n")
		b.WriteString(ListHelp())
	}

	return b.String()
}

// renderRow builds a plain-text table row with the given column widths.
func renderRow(cw columnWidths, project, branch, tool, iteration, userStories, status, heartbeat, uptime string, _ lipgloss.Style) string {
	return fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		cw.project, project,
		cw.branch, branch,
		cw.tool, tool,
		cw.iteration, iteration,
		cw.userStories, userStories,
		cw.status, status,
		cw.heartbeat, heartbeat,
		cw.uptime, uptime,
	)
}

// renderProgressBar builds a text progress bar like: [████░░░░] 5/17
// totalWidth is the full cell width including the count text.
func renderProgressBar(done, total, totalWidth int, barFilled, barEmpty lipgloss.Style) string {
	if total == 0 {
		return pad("-", totalWidth)
	}

	countText := fmt.Sprintf(" %d/%d", done, total)
	barWidth := totalWidth - len(countText)
	if barWidth < 3 {
		// Not enough room for a bar, just show the count
		return pad(fmt.Sprintf("%d/%d", done, total), totalWidth)
	}

	filled := 0
	if total > 0 {
		filled = done * barWidth / total
	}
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	bar := barFilled.Render(strings.Repeat("█", filled)) +
		barEmpty.Render(strings.Repeat("░", empty)) +
		countText

	return bar
}

// renderUserStoriesCell renders the user stories progress bar for list rows.
func renderUserStoriesCell(sess *session.Session, width int) string {
	if !sess.UserStoriesFound {
		return pad("-", width)
	}
	return renderProgressBar(
		sess.UserStoriesDone, sess.UserStoriesTotal, width,
		lipgloss.NewStyle().Foreground(colorGreen),
		lipgloss.NewStyle().Foreground(colorDim),
	)
}

// renderUserStoriesCellSelected renders the user stories progress bar
// for the selected (highlighted) row.
func renderUserStoriesCellSelected(sess *session.Session, width int) string {
	if !sess.UserStoriesFound {
		return selectedStyle.Render(pad("-", width))
	}
	return renderProgressBar(
		sess.UserStoriesDone, sess.UserStoriesTotal, width,
		lipgloss.NewStyle().Foreground(colorGreen).Background(lipgloss.Color("#333366")),
		lipgloss.NewStyle().Foreground(colorDim).Background(lipgloss.Color("#333366")),
	)
}

// truncate shortens a string to maxLen, adding "..." if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// pad right-pads a string to exactly width characters.
func pad(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
