package logs

import (
	"os"
)

// Log linter
var Log *Logging

func init() {
	Log = newLogging()
	// logging = newLogger()
}

// Error export error log
func Error(v ...interface{}) {
	Log.ERROR(v...)
}

// Info export none error log
func Info(v ...interface{}) {
	Log.INFO(v...)
}

// Fatal linter
func Fatal(v ...interface{}) {
	Log.ERROR(v...)
	os.Exit(1)
}
