package logger

import (
	"os"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/kdar/factorlog"
)

// Log default method
type Log interface {
	ERROR(v ...interface{})
	INFO(v ...interface{})
	WARN(v ...interface{})
	DEBUG(v ...interface{})
	STACK(v ...string)
}

// FactorLog custom log with factor pkg
type FactorLog struct {
	frmt   string         // format style log
	stacks *AdvanceMap    // save for debug logs
	log    *log.FactorLog // log
	mutex  sync.RWMutex
}

// NewFactorLog return new log with factor pkg
func NewFactorLog() Log {
	// ftm2 := `%{Color "magenta"}[%{Date}] [%{Time}] %{Color "cyan"}[%{FullFunction}:%{Line}]  %{Color "yellow"}[%{SEVERITY}] %{Color "reset"}[%{Message}]`
	// frmt := `%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "STACK"}[%{Date} %{Time}] [%{SEVERITY}:%{File}:%{Line}] %{Message}%{Color "reset"}`

	frmt := `%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "STACK"} [%{Date}] [%{Time "15:04:05.000000000"}] [%{SEVERITY}] [%{Message}%{Color "reset"}]`
	log := log.New(os.Stdout, log.NewStdFormatter(frmt))

	f := &FactorLog{
		frmt:   frmt,
		log:    log,
		stacks: NewAdvanceMap(),
	}
	go f.serve()
	return f
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
func (l *FactorLog) STACK(values ...string) {
	// find exist, if exist incre, not create
	l.stack(values...)
}

// stack linter
func (l *FactorLog) stack(values ...string) {
	for _, id := range values {
		stack := l.getStack(id)
		l.setStack(id, stack+1)
	}
}

func (l *FactorLog) getStacks() *AdvanceMap {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.stacks
}

func (l *FactorLog) getStack(id string) int {
	if stacks := l.getStacks(); stacks != nil {
		t, in := stacks.Get(id)
		if in {
			stack, ok := t.(int)
			if ok {
				return stack
			}
		}
	}
	return 0
}

func (l *FactorLog) setStack(id string, count int) {
	if stacks := l.getStacks(); stacks != nil {
		stacks.Set(id, count)
	}
}

// serve print stacking
func (l *FactorLog) serve() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		if stacks := l.getStacks(); stacks != nil {
			// capture current stacks
			tmp := stacks.Capture()
			spew.Dump(tmp)
		}
	}
}
