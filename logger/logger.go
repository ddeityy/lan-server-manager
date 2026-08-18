package logger

import (
	"log"
	"os"
)

var (
	infoLog  = log.New(os.Stderr, "", log.Ltime)
	warnLog  = log.New(os.Stderr, "", log.Ltime)
	errorLog = log.New(os.Stderr, "", log.Ltime)
)

// Info logs an informational message.
func Info(msg string) { infoLog.Printf("INFO:  %s", msg) }

// Infof logs a formatted informational message.
func Infof(format string, v ...any) { infoLog.Printf("INFO:  "+format, v...) }

// Warn logs a warning message.
func Warn(msg string) { warnLog.Printf("WARN:  %s", msg) }

// Warnf logs a formatted warning message.
func Warnf(format string, v ...any) { warnLog.Printf("WARN:  "+format, v...) }

// Error logs an error message.
func Error(msg string) { errorLog.Printf("ERROR: %s", msg) }

// Errorf logs a formatted error message.
func Errorf(format string, v ...any) { errorLog.Printf("ERROR: "+format, v...) }
