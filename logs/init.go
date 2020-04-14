package logs

import (
	"os"
)

// Log linter
var logging *Logging

func init() {
	logging = newLogging()
	// logging = newLogger()
}

// Error export error log
func Error(v ...interface{}) {
	logging.ERROR(v...)
}

// Info export none error log
func Info(v ...interface{}) {
	logging.INFO(v...)
}

// Fatal linter
func Fatal(v ...interface{}) {
	logging.ERROR(v...)
	os.Exit(1)
}
