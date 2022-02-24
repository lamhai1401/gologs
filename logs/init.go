package logs

import (
	"github.com/lamhai1401/gologs/logger"
	"github.com/sirupsen/logrus"
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
	Log.DEBUG(v...)
}

// DebugSpew export none error log
func DebugSpew(v ...interface{}) {
	Log.DEBUGSPEW(v...)
}

// Warn export none error log
func Warn(v ...interface{}) {
	Log.WARN(v...)
}

// Stack linter
func Stack(v ...string) {
	Log.STACK(v...)
}

func AddTag(tag string) *logrus.Entry {
	return Log.AddTag(tag)
}

func AddCustomTag(tagName, value string) *logrus.Entry {
	return Log.AddCustomTag(tagName, value)
}
