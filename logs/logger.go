package logs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger to export log into a file
type Logger struct {
	info *log.Logger
	err  *log.Logger
}

func (l *Logger) init() error {
	var generalLog *os.File

	absPath, err := filepath.Abs("./logs")
	if err != nil {
		return err
	}
	generalLog, err = os.OpenFile(absPath+fmt.Sprintf("/%s.log", time.Now().Local().String()), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	l.info = log.New(
		generalLog,
		"[INFO]",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	l.err = log.New(
		generalLog,
		"[ERROR]",
		log.Ldate|log.Ltime|log.Lshortfile,
	)
	return nil
}

// INFO write info log
func (l *Logger) INFO(str string) {
	l.info.Println(str)
}

// ERROR write error log
func (l *Logger) ERROR(str string) {
	l.err.Println(str)
}

func newLogger() *Logger {
	l := &Logger{}
	l.init()
	return l
}
