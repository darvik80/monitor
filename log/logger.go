package log

import (
	"github.com/op/go-logging"
	"os"
)

var logger = logging.MustGetLogger("torrent-monitor")

var formatConsole = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

func Setup(path *string) {
	if path == nil {
		backend := logging.NewLogBackend(os.Stderr, "", 0)
		backendFormatter := logging.NewBackendFormatter(backend, formatConsole)
		logging.SetBackend(backendFormatter)
	} else {
		file, err := os.OpenFile(*path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(0666))
		if err != nil {
			logger.Critical(err.Error())
			return
		}
		backend := logging.NewLogBackend(file, "", 0)
		backendFormatter := logging.NewBackendFormatter(backend, formatConsole)
		logging.SetBackend(backendFormatter)
	}
}

// Fatalf is equivalent to CRITICAL followed by a call to os.Exit(1).
func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)
}

// Fatal is equivalent to CRITICAL followed by a call to os.Exit(1).
func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

// Critical logs a message using CRITICAL as log level.
func Critical(message string) {
	if logger.IsEnabledFor(logging.CRITICAL) {
		logger.Critical(message)
	}
}

// Criticalf logs a message using CRITICAL as log level.
func Criticalf(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.CRITICAL) {
		logger.Critical(format, args...)
	}
}

// Error logs a message using ERROR as log level
func Error(message string) {
	if logger.IsEnabledFor(logging.ERROR) {
		logger.Error(message)
	}
}

// Errorf logs a message using ERROR as log level
func Errorf(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.ERROR) {
		logger.Error(format, args...)
	}
}

// Warning logs a message using WARNING as log level
func Warning(message string) {
	if logger.IsEnabledFor(logging.WARNING) {
		logger.Warning(message)
	}
}

// Warningf logs a message using WARNING as log level
func Warningf(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.WARNING) {
		logger.Warning(format, args...)
	}
}

// Notice logs a message using NOTICE as log level.
func Notice(message string) {
	if logger.IsEnabledFor(logging.NOTICE) {
		logger.Notice(message)
	}
}

// Noticef logs a message using NOTICE as log level.
func Noticef(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.NOTICE) {
		logger.Notice(format, args...)
	}
}

// Info logs a message using INFO as log level.
func Info(message string) {
	if logger.IsEnabledFor(logging.INFO) {
		logger.Info(message)
	}
}

// Infof logs a message using INFO as log level.
func Infof(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.INFO) {
		logger.Info(format, args...)
	}
}

// Debug logs a message using DEBUG as log level.
func Debug(message string) {
	if logger.IsEnabledFor(logging.DEBUG) {
		logger.Debug(message)
	}
}

// Debugf logs a message using DEBUG as log level.
func Debugf(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.DEBUG) {
		logger.Debug(format, args...)
	}
}

// Print a wrapper for log.Print function
func Print(message string) {
	if logger.IsEnabledFor(logging.DEBUG) {
		logger.Debug(message)
	}
}

// Printf a wrapper for log.Printf function
func Printf(format string, args ...interface{}) {
	if logger.IsEnabledFor(logging.DEBUG) {
		logger.Debug(format, args...)
	}
}
