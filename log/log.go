package log

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
)

// Level represents a log level
type Level int

func (l Level) String() string {
	switch l {
	case LevelError:
		return "[ ERROR ]"
	case LevelWarn:
		return "[  WARN ]"
	case LevelInfo:
		return "[  INFO ]"
	case LevelDebug:
		return "[ DEBUG ]"
	case LevelTrace:
		return "[ TRACE ]"
	default:
		return "[ TRACE ]"
	}
}

// log level
const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

// LogLevel defines the threshold of entries logged
var LogLevel = LevelInfo

// Log prints a log entry at the specified level
func Log(level Level, format string, a ...interface{}) {
	if level <= LogLevel {
		caller := strings.Split(path.Base(getCaller()), ".")[0]
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s %s: %s\n", level, caller, format), a...)
	}
}

// Error prints a LevelError log entry
func Error(format string, a ...interface{}) {
	Log(LevelError, format, a...)
}

// Warn prints a LevelWarn log entry
func Warn(format string, a ...interface{}) {
	Log(LevelWarn, format, a...)
}

// Info prints a LevelInfo log entry
func Info(format string, a ...interface{}) {
	Log(LevelInfo, format, a...)
}

// Debug prints a LevelDebug log entry
func Debug(format string, a ...interface{}) {
	Log(LevelDebug, format, a...)
}

// Trace prints a LevelTrace log entry
func Trace(format string, a ...interface{}) {
	Log(LevelTrace, format, a...)
}

func getCaller() string {
	self := reflect.TypeOf(LogLevel).PkgPath()
	caller := self
	for skip := 0; strings.HasPrefix(caller, self); skip++ {
		if pc, _, _, ok := runtime.Caller(skip); ok {
			details := runtime.FuncForPC(pc)
			caller = details.Name()
		} else {
			return "unknown"
		}
	}
	return caller
}
