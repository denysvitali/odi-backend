package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	Primary = lipgloss.Color("#7C3AED") // Purple
	Success = lipgloss.Color("#10B981") // Green
	Warning = lipgloss.Color("#F59E0B") // Amber
	Error   = lipgloss.Color("#EF4444") // Red
	Info    = lipgloss.Color("#3B82F6") // Blue
	Muted   = lipgloss.Color("#6B7280") // Gray
)

// Text styles
var (
	// SuccessStyle is used for success messages
	SuccessStyle = lipgloss.NewStyle().Foreground(Success)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().Foreground(Error)

	// WarningStyle is used for warning messages
	WarningStyle = lipgloss.NewStyle().Foreground(Warning)

	// InfoStyle is used for informational messages
	InfoStyle = lipgloss.NewStyle().Foreground(Info)

	// MutedStyle is used for less important text
	MutedStyle = lipgloss.NewStyle().Foreground(Muted)

	// HeaderStyle is used for section headers
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(Primary)

	// BoldStyle is used for emphasized text
	BoldStyle = lipgloss.NewStyle().Bold(true)
)

// Log level styles for the custom formatter
var (
	DebugLevelStyle = lipgloss.NewStyle().Foreground(Muted)
	InfoLevelStyle  = lipgloss.NewStyle().Foreground(Info)
	WarnLevelStyle  = lipgloss.NewStyle().Foreground(Warning)
	ErrorLevelStyle = lipgloss.NewStyle().Foreground(Error)
	FatalLevelStyle = lipgloss.NewStyle().Foreground(Error).Bold(true)
	PanicLevelStyle = lipgloss.NewStyle().Foreground(Error).Bold(true)
)
