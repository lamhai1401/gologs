package logger

import (
	"os"

	log "github.com/kdar/factorlog"
)

// Log default method
type Log interface {
	ERROR(v ...interface{})
	INFO(v ...interface{})
	WARN(v ...interface{})
	STACK(v ...interface{})
	DEBUG(v ...interface{})
}

// FactorLog custom log with factor pkg
type FactorLog struct {
	frmt string         // format style log
	log  *log.FactorLog // log
}

// NewFactorLog return new log with factor pkg
func NewFactorLog() Log {
	// ftm2 := `%{Color "magenta"}[%{Date}] [%{Time}] %{Color "cyan"}[%{FullFunction}:%{Line}]  %{Color "yellow"}[%{SEVERITY}] %{Color "reset"}[%{Message}]`
	// frmt := `%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "STACK"}[%{Date} %{Time}] [%{SEVERITY}:%{File}:%{Line}] %{Message}%{Color "reset"}`

	frmt := `%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "STACK"} [%{Date}] [%{Time}] [%{SEVERITY}] [%{Message}%{Color "reset"}]`
	log := log.New(os.Stdout, log.NewStdFormatter(frmt))

	return &FactorLog{
		frmt: frmt,
		log:  log,
	}
}

// DEBUG linter auto println
func (l *FactorLog) DEBUG(v ...interface{}) {
	l.log.Debugln(v...)
}

// ERROR linter auto println
func (l *FactorLog) ERROR(v ...interface{}) {
	l.log.Errorln(v...)
}

// INFO linter auto println
func (l *FactorLog) INFO(v ...interface{}) {
	l.log.Infoln(v...)
}

// WARN linter auto println
func (l *FactorLog) WARN(v ...interface{}) {
	l.log.Warnln(v...)
}

// STACK linter auto println
func (l *FactorLog) STACK(v ...interface{}) {
	l.log.Stackln(v...)
}
