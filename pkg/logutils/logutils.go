package logutils

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"
)

var log = logrus.StandardLogger()

// Color definitions for log levels
var (
	debugColor = lipgloss.Color("#6B7280") // Gray
	infoColor  = lipgloss.Color("#3B82F6") // Blue
	warnColor  = lipgloss.Color("#F59E0B") // Amber
	errorColor = lipgloss.Color("#EF4444") // Red
)

// Level styles
var (
	debugStyle = lipgloss.NewStyle().Foreground(debugColor)
	infoStyle  = lipgloss.NewStyle().Foreground(infoColor)
	warnStyle  = lipgloss.NewStyle().Foreground(warnColor)
	errorStyle = lipgloss.NewStyle().Foreground(errorColor)
	fatalStyle = lipgloss.NewStyle().Foreground(errorColor).Bold(true)
)

// ColoredFormatter is a logrus formatter that outputs colored log levels
type ColoredFormatter struct {
	// TimestampFormat is the format for timestamps
	TimestampFormat string
	// DisableColors disables colored output
	DisableColors bool
	// DisableTimestamp disables timestamp output
	DisableTimestamp bool
}

// Format implements logrus.Formatter
func (f *ColoredFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02 15:04:05"
	}

	// Timestamp
	if !f.DisableTimestamp {
		b.WriteString(entry.Time.Format(timestampFormat))
		b.WriteString(" ")
	}

	// Level with color
	levelText := strings.ToUpper(entry.Level.String())
	if len(levelText) < 5 {
		levelText = levelText + strings.Repeat(" ", 5-len(levelText))
	}

	if !f.DisableColors && isTerminal() {
		switch entry.Level {
		case logrus.DebugLevel, logrus.TraceLevel:
			b.WriteString(debugStyle.Render(levelText))
		case logrus.InfoLevel:
			b.WriteString(infoStyle.Render(levelText))
		case logrus.WarnLevel:
			b.WriteString(warnStyle.Render(levelText))
		case logrus.ErrorLevel:
			b.WriteString(errorStyle.Render(levelText))
		case logrus.FatalLevel, logrus.PanicLevel:
			b.WriteString(fatalStyle.Render(levelText))
		default:
			b.WriteString(levelText)
		}
	} else {
		b.WriteString(levelText)
	}
	b.WriteString(" ")

	// Package field if present
	if pkg, ok := entry.Data["package"]; ok {
		b.WriteString("[")
		b.WriteString(fmt.Sprintf("%v", pkg))
		b.WriteString("] ")
	}

	// Message
	b.WriteString(entry.Message)

	// Other fields
	for k, v := range entry.Data {
		if k == "package" {
			continue
		}
		b.WriteString(fmt.Sprintf(" %s=%v", k, v))
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// SetLoggerLevel sets the logger level from a string
func SetLoggerLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)
}

// SetupLogger configures the global logger with colored output
func SetupLogger(level string) {
	SetLoggerLevel(level)
	log.SetFormatter(&ColoredFormatter{})
}
