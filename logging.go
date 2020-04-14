package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	// Ldate linter
	Ldate = 1 << iota // the date in the local time zone: 2009/01/23
	// Ltime linter
	Ltime // the time in the local time zone: 01:23:23
	// Lmicroseconds linter
	Lmicroseconds // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	// Llongfile linter
	Llongfile // full file name and line number: /a/b/c/d.go:23
	// Lshortfile linter
	Lshortfile // final file name element and line number: d.go:23. overrides Llongfile
	// LUTC linter
	LUTC // if Ldate or Ltime is set, use UTC rather than the local time zone
	// Lmsgprefix linter
	Lmsgprefix // move the "prefix" from the beginning of the line to before the message
	// LstdFlags linter
	LstdFlags = Ldate | Ltime // initial values for the standard logger
)

// Logging for cmd log
type Logging struct {
	prefix string    // prefix on each line to identify the logger (but see Lmsgprefix)
	buf    []byte    // for accumulating text to write
	out    io.Writer // destination for output
	flag   int       // properties
	mutex  sync.Mutex
}

// New creates a new Logger. The out variable sets the
// destination to which log data will be written.
// The prefix appears at the beginning of each generated log line, or
// after the log header if the Lmsgprefix flag is provided.
// The flag argument defines the logging properties.
func newLogging() *Logging {
	return &Logging{out: os.Stderr, prefix: "", flag: LstdFlags}
}

// Output writes the output for a logging event. The string s contains
// the text to print after the prefix specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Calldepth is used to recover the PC and is
// provided for generality, although at the moment on all pre-defined
// paths it will be 2.
func (l *Logging) Output(calldepth int, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.flag&(Lshortfile) != 0 {
		// Release lock while getting caller info - it's expensive.
		l.mutex.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "???"
			line = 0
		}
		l.mutex.Lock()
	}
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, file, line)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	return err
}

// formatHeader writes log header to buf in following order:
//   * l.prefix (if it's not blank and Lmsgprefix is unset),
//   * date and/or time (if corresponding flags are provided),
//   * file and line number (if corresponding flags are provided),
//   * l.prefix (if it's not blank and Lmsgprefix is set).
func (l *Logging) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	if l.flag&Lmsgprefix == 0 {
		*buf = append(*buf, l.prefix...)
	}
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&LUTC != 0 {
			t = t.UTC()
		}
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if l.flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
	if l.flag&Lmsgprefix != 0 {
		*buf = append(*buf, l.prefix...)
	}
}

// Cheap integer to fixed-width decimal ASCII. Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

// INFO linter
func (l *Logging) INFO(v ...interface{}) {
	l.Output(1, fmt.Sprintln(fmt.Sprintf("[INFO] %s", v...)))
}

// ERROR linter
func (l *Logging) ERROR(v ...interface{}) {
	l.Output(2, fmt.Sprintln(fmt.Sprintf("[ERROR] %s", v...)))
}
