// Package log is from https://github.com/micro/go-micro/blob/master/util/log/log.go
package log

import (
	"fmt"
	stdlog "log"
	"os"
)

// Logger is a generic logging interface
type Logger interface {
	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

// Level is a log level
type Level int

const (
	// LevelFatal fatal level
	LevelFatal Level = iota + 1
	// LevelInfo info level
	LevelInfo
	// LevelError error level
	LevelError
	// LevelDebug debug level
	LevelDebug
)

var (
	// the local logger
	logger Logger = &defaultLogLogger{}

	// default log level is info
	level = LevelInfo

	// prefix for all messages, default is "[Gev]"
	prefix = "[Gev]"
)

type defaultLogLogger struct{}

func (t *defaultLogLogger) Log(v ...interface{}) {
	stdlog.Print(v...)
}

func (t *defaultLogLogger) Logf(format string, v ...interface{}) {
	stdlog.Printf(format, v...)
}

func init() {
	switch os.Getenv("GEV_LOG_LEVEL") {
	case "debug":
		level = LevelDebug
	case "info":
		level = LevelInfo
	case "error":
		level = LevelError
	case "fatal":
		level = LevelFatal
	}
}

// Log makes use of Logger
func Log(v ...interface{}) {
	if len(prefix) > 0 {
		logger.Log(append([]interface{}{prefix, " "}, v...)...)
		return
	}
	logger.Log(v...)
}

// Logf makes use of Logger
func Logf(format string, v ...interface{}) {
	if len(prefix) > 0 {
		format = prefix + " " + format
	}
	logger.Logf(format, v...)
}

// WithLevel logs with the level specified
func WithLevel(l Level, v ...interface{}) {
	if l > level {
		return
	}
	Log(v...)
}

// WithLevelf logs with the level specified
func WithLevelf(l Level, format string, v ...interface{}) {
	if l > level {
		return
	}
	Logf(format, v...)
}

// Debug provides debug level logging
func Debug(v ...interface{}) {
	WithLevel(LevelDebug, v...)
}

// Debugf provides debug level logging
func Debugf(format string, v ...interface{}) {
	WithLevelf(LevelDebug, format, v...)
}

// Info provides info level logging
func Info(v ...interface{}) {
	WithLevel(LevelInfo, v...)
}

// Infof provides info level logging
func Infof(format string, v ...interface{}) {
	WithLevelf(LevelInfo, format, v...)
}

// Error provides warn level logging
func Error(v ...interface{}) {
	WithLevel(LevelError, v...)
}

// Errorf provides warn level logging
func Errorf(format string, v ...interface{}) {
	WithLevelf(LevelError, format, v...)
}

// Fatal logs with Log and then exits with os.Exit(1)
func Fatal(v ...interface{}) {
	WithLevel(LevelFatal, v...)
	os.Exit(1)
}

// Fatalf logs with Logf and then exits with os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	WithLevelf(LevelFatal, format, v...)
	os.Exit(1)
}

// SetLogger sets the local logger
func SetLogger(l Logger) {
	logger = l
}

// GetLogger returns the local logger
func GetLogger() Logger {
	return logger
}

// SetLevel sets the log level
func SetLevel(l Level) {
	level = l
}

// GetLevel returns the current level
func GetLevel() Level {
	return level
}

// SetPrefix sets a prefix for the logger
func SetPrefix(p string) {
	prefix = p
}

// Name sets service name
func Name(name string) {
	prefix = fmt.Sprintf("[%s]", name)
}
