// Package logger provides colored, leveled logging for CLI output.
// It uses ANSI escape codes for color, which work cross-platform on modern
// terminals (Linux, macOS, and Windows 10+). Colors are automatically disabled
// when output is not a terminal (e.g., when piping to a file).
package logger

import (
	"fmt"
	"io"
	"os"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// colorEnabled is true when stdout is a real terminal (not a pipe or file).
// It is set once at startup in init().
var colorEnabled bool

// passFailWriter is the destination for PASS/FAIL output. It defaults to
// os.Stdout but can be redirected to os.Stderr via UseStderrForPassFail().
// This is necessary in post-renderer mode, where stdout carries YAML that
// Helm reads — mixing diagnostic output into it would corrupt the stream.
var passFailWriter io.Writer = os.Stdout

// UseStderrForPassFail redirects PASS/FAIL output to stderr. Call this once
// at startup when running as a Helm post-renderer.
func UseStderrForPassFail() {
	passFailWriter = os.Stderr
}

func init() {
	fi, err := os.Stdout.Stat()
	// Enable colors only when stdout is a character device (i.e. a terminal).
	if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		colorEnabled = true
	}
}

// colorize wraps text in the given ANSI color code, if colors are enabled.
func colorize(code, text string) string {
	if !colorEnabled {
		return text
	}
	return code + text + colorReset
}

// Log writes an informational (cyan) message to stdout.
// Use for general progress/status messages.
func Log(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "%s: %s\n", colorize(colorCyan, "LOG"), fmt.Sprintf(format, args...))
}

// Warn writes a warning (yellow) message to stderr.
// Use for non-fatal issues that the user should be aware of.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", colorize(colorYellow, "WARN"), fmt.Sprintf(format, args...))
}

// Error writes an error (red) message to stderr.
// Use for failures that don't immediately stop execution.
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", colorize(colorRed, "ERROR"), fmt.Sprintf(format, args...))
}

// Fatal writes an error (red) message to stderr and exits with code 1.
// Use for unrecoverable errors.
func Fatal(format string, args ...any) {
	Error(format, args...)
	os.Exit(1)
}

// Pass writes a passing (green) assertion result to passFailWriter (stdout by
// default, stderr in post-renderer mode).
func Pass(format string, args ...any) {
	fmt.Fprintf(passFailWriter, "%s: %s\n", colorize(colorGreen, "PASS"), fmt.Sprintf(format, args...))
}

// Fail writes a failing (red) assertion result to passFailWriter (stdout by
// default, stderr in post-renderer mode).
func Fail(format string, args ...any) {
	fmt.Fprintf(passFailWriter, "%s: %s\n", colorize(colorRed, "FAIL"), fmt.Sprintf(format, args...))
}
