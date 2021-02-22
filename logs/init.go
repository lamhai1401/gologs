package logs

import (
	"os"

	"github.com/lamhai1401/gologs/logger"
)

// Log linter
var Log logger.Log

func init() {
	Log = logger.NewFactorLog()
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

// Debug export none error log
func Debug(v ...interface{}) {
	if os.Getenv("DEBUG") == "1" {
		Log.DEBUG(v...)
	}
}

// Warn export none error log
func Warn(v ...interface{}) {
	Log.WARN(v...)
}

// Stack linter
func Stack(v ...string) {
	Log.STACK(v...)
}
