package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	// log "github.com/kdar/factorlog"

	log "github.com/sirupsen/logrus"
)

var OffLog string

func init() {
	// logging = newLogger()
	OffLog = os.Getenv("OFF_LOG")

}

type LogMsg struct {
	level log.Level
	msg   []interface{}
}

// Log default method
type Log interface {
	ERROR(v ...interface{})
	INFO(v ...interface{})
	WARN(v ...interface{})
	DEBUG(v ...interface{})
	DEBUGSPEW(v ...interface{})
	STACK(v ...string)
	AddTag(tag string) *log.Entry
	AddCustomTag(tagName, value string) *log.Entry
}

// FactorLog custom log with factor pkg
type FactorLog struct {
	level log.Level
	f     *os.File
	// frmt   string      // format style log
	stacks *AdvanceMap // save for debug logs
	log    *log.Logger // log
	// chann  chan int
	// mutex sync.RWMutex

	msgChann chan *LogMsg
}

// func (f *FactorLog) setFile(g)

// // NewFactorLog return new log with factor pkg
// func NewFactorLog() Log {
// 	// ftm2 := `%{Color "magenta"}[%{Date}] [%{Time}] %{Color "cyan"}[%{FullFunction}:%{Line}]  %{Color "yellow"}[%{SEVERITY}] %{Color "reset"}[%{Message}]`
// 	// frmt := `%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "STACK"}[%{Date} %{Time}] [%{SEVERITY}:%{File}:%{Line}] %{Message}%{Color "reset"}`

// 	frmt := `%{Color "red" "ERROR"}%{Color "yellow" "WARN"}%{Color "green" "INFO"}%{Color "cyan" "DEBUG"}%{Color "blue" "STACK"} [%{Date}] [%{Time "15:04:05.000000000"}] [%{SEVERITY}] [%{Message}%{Color "reset"}]`
// 	log := log.New(os.Stdout, log.NewStdFormatter(frmt))

// 	f := &FactorLog{
// 		frmt:   frmt,
// 		log:    log,
// 		stacks: NewAdvanceMap(),
// 	}
// 	go f.serve()
// 	return f
// }

// var (
// 	ctx    context.Context
// 	cancel context.CancelFunc
// 	temp   chan int
// )

const (
	sdtErrMode = "stderr"
	sdtOutMode = "stdout"
	fileMode   = "file"
)

func NewFactorLog() Log {
	LogLevel := log.DebugLevel
	myLog := log.New()

	// add context
	// ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	// myLog.WithContext(ctx)

	// setup log level
	LogMode := sdtOutMode
	if mode := os.Getenv("LOG_MODE"); mode != "" {
		LogMode = mode
	}
	if lv := os.Getenv("LOG_LEVEL"); lv != "" {
		LogLV, err := log.ParseLevel(lv)
		if err == nil {
			LogLevel = LogLV
		}
	}
	myLog.SetLevel(LogLevel)

	factorlog := &FactorLog{
		level:    LogLevel,
		log:      myLog,
		stacks:   NewAdvanceMap(),
		msgChann: make(chan *LogMsg, 1024),
	}

	// setting mode
	switch LogMode {
	case sdtOutMode:
		myLog.SetOutput(os.Stdout)
	case sdtErrMode:
		myLog.SetOutput(os.Stderr)
	case fileMode:
		logPath := os.Getenv("LOG_PATH")
		if logPath == "" {
			logPath = "/var/log/classroom-core/"
		}
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			nodeID = "nodeID"
		}
		roomID := os.Getenv("SIGNAL_ROOM_ID")
		if roomID == "" {
			roomID = "roomID"
		}
		// add folder Log
		newpath := filepath.Join(logPath, "/", roomID)
		os.MkdirAll(newpath, os.ModePerm)

		filename := fmt.Sprintf("./%s/%s-%s-out.log", newpath, roomID, nodeID)
		f, _ := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		// myLog.ReportCaller = true
		// setting out put file
		myLog.SetOutput(f)
		factorlog.f = f
		myLog.Debug("Start writing log file: ", filename)
	default:
		myLog.SetOutput(os.Stdout)
	}

	// format type
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05.000000000"
	Formatter.FullTimestamp = true
	Formatter.DisableLevelTruncation = true
	Formatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
		// this function is required when you want to introduce your custom format.
		// In my case I wanted file and line to look like this `file="engine.go:141`
		// but f.File provides a full path along with the file name.
		// So in `formatFilePath()` function I just trimmet everything before the file name
		// and added a line number in the end
		return "", fmt.Sprintf("%s:%d", formatFilePath(f.File), f.Line)
	}
	myLog.SetFormatter(Formatter)

	go factorlog.serve()
	go factorlog.readMsg()
	return factorlog
}

func (l *FactorLog) readMsg() {
	var open bool
	var msg *LogMsg
	for {
		msg, open = <-l.msgChann
		if !open {
			return
		}
		l.log.Log(msg.level, msg.msg...)

		msg = nil
	}
}

func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}

func (l *FactorLog) AddCustomTag(tagName, value string) *log.Entry {
	return l.log.WithFields(log.Fields{
		tagName: value,
	})
}

// AddTag add Tag for check log
func (l *FactorLog) AddTag(tag string) *log.Entry {
	return l.log.WithFields(log.Fields{
		"tag": tag,
	})
}

func (l *FactorLog) writeLog(level log.Level, args ...interface{}) {
	if OffLog != "1" {
		// go l.log.Log(level, args...)
		l.msgChann <- &LogMsg{
			level: level,
			msg:   args,
		}
	}
}

// DEBUG linter auto println
func (l *FactorLog) DEBUG(v ...interface{}) {
	l.writeLog(log.DebugLevel, v...)
}

// DEBUG linter auto println
func (l *FactorLog) DEBUGSPEW(v ...interface{}) {
	// l.writeLog(log.DebugLevel, v...)
	if l.level != log.DebugLevel {
		return
	}
	if l.f != nil {
		spew.Fdump(l.f, v...)
	} else {
		spew.Dump(v...)
	}
}

// ERROR linter auto println
func (l *FactorLog) ERROR(v ...interface{}) {
	l.writeLog(log.ErrorLevel, v...)
}

// INFO linter auto println
func (l *FactorLog) INFO(v ...interface{}) {
	l.writeLog(log.InfoLevel, v...)
}

// WARN linter auto println
func (l *FactorLog) WARN(v ...interface{}) {
	l.writeLog(log.WarnLevel, v...)
}

// STACK linter auto println
func (l *FactorLog) STACK(values ...string) {
	// find exist, if exist incre, not create
	go l.stack(values...)
}

// stack linter
func (l *FactorLog) stack(values ...string) {
	for _, id := range values {
		stack := l.getStack(id)
		l.setStack(id, stack+1)
	}
}

func (l *FactorLog) getStacks() *AdvanceMap {
	// l.mutex.RLock()
	// defer l.mutex.RUnlock()
	return l.stacks
}

func (l *FactorLog) getStack(id string) int {
	t, in := l.stacks.Get(id)
	if in {
		stack, ok := t.(int)
		if ok {
			return stack
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
	ticker := time.NewTicker(time.Duration(getInterval()) * time.Second)
	for range ticker.C {
		if stacks := l.getStacks(); stacks != nil {
			// capture current stacks
			tmp := stacks.Capture()

			if l.f != nil {
				spew.Fdump(l.f, time.Now(), tmp)
			} else {
				spew.Dump(time.Now(), tmp)
			}
		}
	}
}

func getInterval() int {
	i := 15
	if interval := os.Getenv("LOG_INTERVAL"); interval != "" {
		j, err := strconv.Atoi(interval)
		if err == nil {
			i = j
		}
	}
	return i
}
