package log

import (
	"log"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// Info logs informational messages (suppressed when quiet is true)
func Info(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix(colorBlue + "INFO: " + colorReset)
	log.Printf(format, args...)
}

// Error logs error messages (suppressed when quiet is true)
func Error(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix(colorRed + "ERROR: " + colorReset)
	log.Printf(format, args...)
}

// Success logs success messages (suppressed when quiet is true)
func Success(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix(colorGreen + "SUCCESS: " + colorReset)
	log.Printf(format, args...)
}

// Fatal logs fatal messages and exits
func Fatal(quiet bool, format string, args ...interface{}) {
	log.SetPrefix(colorRed + "FATAL: " + colorReset)
	log.Fatalf(format, args...)
}

// Warning logs warning messages (suppressed when quiet is true)
func Warning(quiet bool, format string, args ...interface{}) {
	if quiet {
		return
	}
	log.SetPrefix(colorYellow + "WARNING: " + colorReset)
	log.Printf(format, args...)
}
