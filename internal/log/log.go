package log

import (
	"log"
)

// Info logs informational messages (suppressed when quiet is true)
func Info(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix("INFO: ")
	log.Printf(format, args...)
}

// Error logs error messages (suppressed when quiet is true)
func Error(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix("ERROR: ")
	log.Printf(format, args...)
}

// Success logs success messages (suppressed when quiet is true)
func Success(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix("SUCCESS: ")
	log.Printf(format, args...)
}

// Fatal logs fatal messages and exits
func Fatal(quiet bool, format string, args ...interface{}) {
	log.SetPrefix("FATAL: ")
	log.Fatalf(format, args...)
}
