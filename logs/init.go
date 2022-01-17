package logs

import (
	"os"

	"github.com/lamhai1401/gologs/logger"
	"github.com/sirupsen/logrus"
)

// Log linter
var Log logger.Log
var OffLog string

func init() {
	Log = logger.NewFactorLog()
	// logging = newLogger()
	OffLog = os.Getenv("OFF_LOG")
}

// Error export error log
func Error(v ...interface{}) {
	if OffLog != "1" {
		Log.ERROR(v...)
	}
}

// Info export none error log
func Info(v ...interface{}) {
	if OffLog != "1" {
		Log.INFO(v...)
	}
}

// Debug export none error log
func Debug(v ...interface{}) {
	if OffLog != "1" {
		Log.DEBUG(v...)
	}
}

// Warn export none error log
func Warn(v ...interface{}) {
	if OffLog != "1" {
		Log.WARN(v...)
	}
}

// Stack linter
func Stack(v ...string) {
	if OffLog != "1" {
		Log.STACK(v...)
	}
	// Log.STACK(v...)
}

func AddTag(tag string) *logrus.Entry {
	return Log.AddTag(tag)
}

func AddCustomTag(tagName, value string) *logrus.Entry {
	return Log.AddCustomTag(tagName, value)
}
