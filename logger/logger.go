package logger

import (
	"fmt"
	"os"
	"time"
)

func logf(level, format string, v ...any) {
	fmt.Fprintf(os.Stderr, "%s %s: %s\n", time.Now().Format("15:04:05"), level, fmt.Sprintf(format, v...))
}

// Info logs an informational message.
func Info(msg string) { logf("INFO", "%s", msg) }

// Infof logs a formatted informational message.
func Infof(format string, v ...any) { logf("INFO", format, v...) }

// Warn logs a warning message.
func Warn(msg string) { logf("WARN", "%s", msg) }

// Warnf logs a formatted warning message.
func Warnf(format string, v ...any) { logf("WARN", format, v...) }

// Error logs an error message.
func Error(msg string) { logf("ERROR", "%s", msg) }

// Errorf logs a formatted error message.
func Errorf(format string, v ...any) { logf("ERROR", format, v...) }
