// Package session defines the data types for ralphy session registry.
package session

import (
	"fmt"
	"time"
)

// Session represents a running ralphy loop instance.
// The JSON structure matches what the ralphy bash script writes
// to the registry directory.
type Session struct {
	PID              int       `json:"pid"`
	Tool             string    `json:"tool"`
	Project          string    `json:"project"`
	WorkDir          string    `json:"work_dir"`
	Branch           string    `json:"branch"`
	PRDDescription   string    `json:"prd_description"`
	CurrentIteration int       `json:"current_iteration"`
	MaxIterations    int       `json:"max_iterations"`
	UseWorktree      bool      `json:"use_worktree"`
	WorktreeDir      string    `json:"worktree_dir"`
	StartedAt        time.Time `json:"started_at"`
	LastHeartbeat    time.Time `json:"last_heartbeat"`
	Status           string    `json:"status"`
	LogFile          string    `json:"log_file"`

	// Computed fields (not from JSON)
	SessionID string `json:"-"` // extracted from filename
	FilePath  string `json:"-"` // full path to session file
	IsAlive   bool   `json:"-"` // whether PID is actually running

	// User stories progress (computed from .ralph/prd.json)
	UserStoriesTotal int  `json:"-"` // total number of user stories
	UserStoriesDone  int  `json:"-"` // stories with passes: true
	UserStoriesFound bool `json:"-"` // whether prd.json was found and parsed
}

// Status constants
const (
	StatusRunning       = "running"
	StatusCompleted     = "completed"
	StatusMaxIterations = "max_iterations_reached"
	StatusInterrupted   = "interrupted"
	StatusPaused        = "paused"
	StatusStale         = "stale" // computed: heartbeat too old but process alive
	StatusDead          = "dead"  // computed: PID not found and was running

	StaleThreshold = 5 * time.Minute
	SessionTTL     = 24 * time.Hour
)

// IsTerminal returns true if the session status is a final state
// (the ralphy process is no longer running by design).
func (s *Session) IsTerminal() bool {
	switch s.Status {
	case StatusCompleted, StatusMaxIterations, StatusInterrupted:
		return true
	default:
		return false
	}
}

// IsExpired returns true if the session is older than SessionTTL.
func (s *Session) IsExpired() bool {
	return time.Since(s.LastHeartbeat) > SessionTTL
}

// Uptime returns the duration since the session started.
func (s *Session) Uptime() time.Duration {
	return time.Since(s.StartedAt)
}

// TimeSinceHeartbeat returns the duration since the last heartbeat.
func (s *Session) TimeSinceHeartbeat() time.Duration {
	return time.Since(s.LastHeartbeat)
}

// IsStale returns true if the heartbeat is older than StaleThreshold.
func (s *Session) IsStale() bool {
	return s.TimeSinceHeartbeat() > StaleThreshold
}

// DisplayStatus returns the effective status considering liveness and staleness.
func (s *Session) DisplayStatus() string {
	// Terminal statuses are shown as-is
	if s.IsTerminal() {
		return s.Status
	}
	// Non-terminal but process is dead — unexpected death
	if !s.IsAlive {
		return StatusDead
	}
	if s.Status == StatusPaused {
		return StatusPaused
	}
	if s.IsStale() {
		return StatusStale
	}
	return s.Status
}

// IterationProgress returns a string like "3/100".
func (s *Session) IterationProgress() string {
	return fmt.Sprintf("%d/%d", s.CurrentIteration, s.MaxIterations)
}

// FormatUptime returns a human-readable uptime string.
func (s *Session) FormatUptime() string {
	return formatDuration(s.Uptime())
}

// FormatHeartbeat returns a human-readable time since last heartbeat.
func (s *Session) FormatHeartbeat() string {
	d := s.TimeSinceHeartbeat()
	if d < 5*time.Second {
		return "just now"
	}
	return formatDuration(d) + " ago"
}

// UserStoriesProgress returns a string like "5/17" or "-" if no prd.json.
func (s *Session) UserStoriesProgress() string {
	if !s.UserStoriesFound {
		return "-"
	}
	return fmt.Sprintf("%d/%d", s.UserStoriesDone, s.UserStoriesTotal)
}

// UserStoriesPercent returns 0-100 completion percentage.
func (s *Session) UserStoriesPercent() int {
	if !s.UserStoriesFound || s.UserStoriesTotal == 0 {
		return 0
	}
	return s.UserStoriesDone * 100 / s.UserStoriesTotal
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
