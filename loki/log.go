package loki

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	log "github.com/sirupsen/logrus"
)

const (
	sdtErrMode = "stderr"
	sdtOutMode = "stdout"
	fileMode   = "file"
)

// Log default method
type Log interface {
	ERROR(msg string, tags map[string]any)
	INFO(msg string, tags map[string]any)
	WARN(msg string, tags map[string]any)
	DEBUG(msg string, tags map[string]any)
	STACK(msg ...string)
	DEBUGSPEW(msg string, tags map[string]any)
	AddHook(log.Hook)
}

type LogMsg struct {
	tag   map[string]any
	level log.Level
	msg   string
}

var _ Log = (*logger)(nil)

// FactorLog custom log with factor pkg
type logger struct {
	level log.Level
	f     *os.File
	// frmt   string      // format style log
	stacks *AdvanceMap // save for debug logs
	log    *log.Logger // log
	// chann  chan int
	// mutex sync.RWMutex

	hostname string
	msgChann chan *LogMsg
}

func NewLoggerWithLoki(URL string, batchSize int, batchWait time.Duration) (*logger, error) {
	client, err := NewLoki(URL, batchSize, batchWait)
	if err != nil {
		panic(err.Error())
	}
	logger := NewLogger()
	logger.AddHook(client)
	return logger, nil
}

func NewLogger() *logger {

	LogLevel := log.DebugLevel
	myLog := log.New()

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

	logger := &logger{
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
		logger.f = f
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

	host, _ := os.Hostname()
	logger.hostname = host

	go logger.serve()
	go logger.readMsg()
	return logger

}

func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}

// DEBUG linter auto println
func (l *logger) DEBUG(msg string, tags map[string]any) {
	l.writeLog(log.DebugLevel, msg, tags)
}

// DEBUG linter auto println
func (l *logger) DEBUGSPEW(msg string, tags map[string]any) {
	if l.f != nil {
		spew.Fdump(l.f, msg, tags)
	} else {
		spew.Dump(msg, tags)
	}
}

// ERROR linter auto println
func (l *logger) ERROR(msg string, tags map[string]any) {
	l.writeLog(log.ErrorLevel, msg, tags)
}

// INFO linter auto println
func (l *logger) INFO(msg string, tags map[string]any) {
	l.writeLog(log.InfoLevel, msg, tags)
}

// WARN linter auto println
func (l *logger) WARN(msg string, tags map[string]any) {
	l.writeLog(log.WarnLevel, msg, tags)
}

func (l *logger) writeLog(level log.Level, msg string, tags map[string]any) {
	// go l.log.Log(level, args...)
	l.msgChann <- &LogMsg{
		level: level,
		msg:   msg,
		tag:   tags,
	}
}

func (l *logger) readMsg() {
	var open bool
	var msg *LogMsg
	for {
		msg, open = <-l.msgChann
		if !open {
			return
		}
		// additional host name
		msg.tag["host"] = l.hostname
		l.log.WithFields(msg.tag).Log(msg.level, msg.msg)
		msg = nil
	}
}

// serve print stacking
func (l *logger) serve() {
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
			for key, value := range tmp {
				l.INFO(fmt.Sprintf("%s: %v", key, value), map[string]any{
					"data": "Data stacking from",
				})
			}
		}
	}
}

func getInterval() int {
	i := 1
	if interval := os.Getenv("LOG_INTERVAL"); interval != "" {
		j, err := strconv.Atoi(interval)
		if err == nil {
			i = j
		}
	}
	return i
}

// stack linter
func (l *logger) stack(values ...string) {
	for _, id := range values {
		stack := l.getStack(id)
		l.setStack(id, stack+1)
	}
}

func (l *logger) getStacks() *AdvanceMap {
	return l.stacks
}

func (l *logger) getStack(id string) int {
	t, in := l.stacks.Get(id)
	if in {
		stack, ok := t.(int)
		if ok {
			return stack
		}
	}
	return 0
}

func (l *logger) setStack(id string, count int) {
	if stacks := l.getStacks(); stacks != nil {
		stacks.Set(id, count)
	}
}

// STACK linter auto println
func (l *logger) STACK(values ...string) {
	// find exist, if exist incre, not create
	l.stack(values...)
}
func (l *logger) AddHook(hook log.Hook) {
	l.log.AddHook(hook)
}
