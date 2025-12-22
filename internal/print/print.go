package print

import (
	"fmt"
	"io"
	"strings"
)

// ANSI color codes using opencode theme colors
const (
	SuccessColor = "7fd88f" // darkGreen from opencode theme
	ErrorColor   = "e06c75" // darkRed from opencode theme
	WarningColor = "f5a742" // darkOrange from opencode theme
	InfoColor    = "56b6c2" // darkCyan from opencode theme
	GrayColor    = "6c757d" // medium gray
	ResetColor   = "\x1b[0m"
)

// Common symbols for composition
const (
	Tick       = "✓"
	Cross      = "✗"
	WarningSym = "⚠"
	InfoSym    = "ℹ"
	Arrow      = "→"
	Bullet     = "•"
)

// colorize wraps text with ANSI color codes
func colorize(color, text string) string {
	var code string
	switch color {
	case SuccessColor:
		code = "32" // Green
	case ErrorColor:
		code = "31" // Red
	case WarningColor:
		code = "33" // Yellow
	case InfoColor:
		code = "36" // Cyan
	case GrayColor:
		code = "90" // Bright Black (Dark Gray)
	default:
		return text
	}
	return fmt.Sprintf("\x1b[%sm%s%s", code, text, ResetColor)
}

// Success prints a success message with variadic parts for composition
func Success(w io.Writer, parts ...string) {
	fmt.Fprintln(w, colorize(SuccessColor, strings.Join(parts, " ")))
}

// Error prints an error message with variadic parts for composition
func Error(w io.Writer, parts ...string) {
	fmt.Fprintln(w, colorize(ErrorColor, strings.Join(parts, " ")))
}

// Warning prints a warning message with variadic parts for composition
func Warning(w io.Writer, parts ...string) {
	fmt.Fprintln(w, colorize(WarningColor, strings.Join(parts, " ")))
}

// Info prints a plain text message (no color)
func Info(w io.Writer, parts ...string) {
	fmt.Fprintln(w, strings.Join(parts, " "))
}

// Note prints a colored note message with variadic parts for composition
func Note(w io.Writer, parts ...string) {
	fmt.Fprintln(w, colorize(InfoColor, strings.Join(parts, " ")))
}

// Successf prints a formatted success message in green
func Successf(w io.Writer, format string, args ...any) {
	fmt.Fprint(w, colorize(SuccessColor, fmt.Sprintf(format, args...)))
}

// Errorf prints a formatted error message in red
func Errorf(w io.Writer, format string, args ...any) {
	fmt.Fprint(w, colorize(ErrorColor, fmt.Sprintf(format, args...)))
}

// Warningf prints a formatted warning message in orange
func Warningf(w io.Writer, format string, args ...any) {
	fmt.Fprint(w, colorize(WarningColor, fmt.Sprintf(format, args...)))
}

// Infof prints a formatted plain text message (no color)
func Infof(w io.Writer, format string, args ...any) (int, error) {
	return fmt.Fprintf(w, format, args...)
}

// Notef prints a formatted colored note message
func Notef(w io.Writer, format string, args ...any) {
	fmt.Fprint(w, colorize(InfoColor, fmt.Sprintf(format, args...)))
}

// ColorSuccess returns a green colored string
func ColorSuccess(text string) string {
	return colorize(SuccessColor, text)
}

// ColorError returns a red colored string
func ColorError(text string) string {
	return colorize(ErrorColor, text)
}

// ColorWarning returns a yellow colored string
func ColorWarning(text string) string {
	return colorize(WarningColor, text)
}

// ColorNote returns a cyan colored string
func ColorNote(text string) string {
	return colorize(InfoColor, text)
}

// ColorSubtle returns a subtle gray colored string
func ColorSubtle(text string) string {
	return colorize(GrayColor, text)
}

// Wrap adds newlines before and after text
func Wrap(parts ...string) string {
	return "\n" + strings.Join(parts, " ") + "\n"
}

// WrapTop adds newline before text
func WrapTop(parts ...string) string {
	return "\n" + strings.Join(parts, " ")
}

// WrapBottom adds newline after text
func WrapBottom(parts ...string) string {
	return strings.Join(parts, " ") + "\n"
}

// WrapMulti adds multiple newlines before and after text
func WrapMulti(lines int, parts ...string) string {
	return strings.Repeat("\n", lines) + strings.Join(parts, " ") + strings.Repeat("\n", lines)
}
