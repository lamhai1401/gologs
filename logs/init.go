package logs

import (
	"os"

	"github.com/lamhai1401/gologs/logger"
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
		go Log.ERROR(v...)
	}
}

// Info export none error log
func Info(v ...interface{}) {
	if OffLog != "1" {
		go Log.INFO(v...)
	}
}

// Debug export none error log
func Debug(v ...interface{}) {
	if os.Getenv("DEBUG") == "1" && OffLog != "1" {
		go Log.DEBUG(v...)
	}
}

// Warn export none error log
func Warn(v ...interface{}) {
	if OffLog != "1" {
		go Log.WARN(v...)
	}
}

// Stack linter
func Stack(v ...string) {
	// if OffLog != "1" {
	// 	Log.STACK(v...)
	// }
	go Log.STACK(v...)
}
