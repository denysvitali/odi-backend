package ui

import (
	"fmt"
	"os"
)

const (
	checkMark   = "✓"
	crossMark   = "✗"
	warningMark = "⚠"
	infoMark    = "ℹ"
)

// PrintSuccess prints a success message with a green checkmark
func PrintSuccess(msg string) {
	fmt.Fprintln(os.Stderr, SuccessStyle.Render(checkMark+" "+msg))
}

// PrintSuccessf prints a formatted success message with a green checkmark
func PrintSuccessf(format string, args ...any) {
	PrintSuccess(fmt.Sprintf(format, args...))
}

// PrintError prints an error message with a red X
func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, ErrorStyle.Render(crossMark+" "+msg))
}

// PrintErrorf prints a formatted error message with a red X
func PrintErrorf(format string, args ...any) {
	PrintError(fmt.Sprintf(format, args...))
}

// PrintWarning prints a warning message with a yellow warning sign
func PrintWarning(msg string) {
	fmt.Fprintln(os.Stderr, WarningStyle.Render(warningMark+" "+msg))
}

// PrintWarningf prints a formatted warning message with a yellow warning sign
func PrintWarningf(format string, args ...any) {
	PrintWarning(fmt.Sprintf(format, args...))
}

// PrintInfo prints an info message with a blue info symbol
func PrintInfo(msg string) {
	fmt.Fprintln(os.Stderr, InfoStyle.Render(infoMark+" "+msg))
}

// PrintInfof prints a formatted info message with a blue info symbol
func PrintInfof(format string, args ...any) {
	PrintInfo(fmt.Sprintf(format, args...))
}

// PrintHeader prints a bold header
func PrintHeader(title string) {
	fmt.Fprintln(os.Stderr, HeaderStyle.Render(title))
}

// PrintMuted prints muted/dimmed text
func PrintMuted(msg string) {
	fmt.Fprintln(os.Stderr, MutedStyle.Render(msg))
}

// PrintMutedf prints formatted muted/dimmed text
func PrintMutedf(format string, args ...any) {
	PrintMuted(fmt.Sprintf(format, args...))
}
