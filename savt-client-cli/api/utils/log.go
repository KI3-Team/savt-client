// Copyright 2025 KI3
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"io"
	"log"
	"os"
	"sync"

	"github.com/natefinch/lumberjack"
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	SetConsole(enable bool)
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

type FileLogger struct {
	debugLogger   *log.Logger
	infoLogger    *log.Logger
	warnLogger    *log.Logger
	errorLogger   *log.Logger
	level         LogLevel
	logFile       string
	console       bool
	logFileOutput io.Writer
	mutex         sync.Mutex
}

type LoggerConfig struct {
	LogFile    string
	Console    bool
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

type LoggerFactory struct{}

var sysConfig, _ = GetSysConfig()
var userConfig, _ = GetUserConfig()
var (
	defaultWorkerLoggerConfig = LoggerConfig{
		LogFile:    sysConfig.LogDir + "/savt-client-worker.log",
		Console:    true,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	defaultGUILoggerConfig = LoggerConfig{
		LogFile:    userConfig.LogDir + "/savt-client-gui.log",
		Console:    true,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	workerLoggerInstance Logger
	guiLoggerInstance    Logger
	logOnce              sync.Once

	LoggerFactoryInstance = &LoggerFactory{}
)

// NewFileLogger creates a new file logger instance
func NewFileLogger(config LoggerConfig, level LogLevel) *FileLogger {
	logFileOutput := &lumberjack.Logger{
		Filename:   config.LogFile,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	var logOutput io.Writer

	if config.Console {
		logOutput = io.MultiWriter(logFileOutput, os.Stdout)
	} else {
		logOutput = logFileOutput
	}

	return &FileLogger{
		debugLogger:   log.New(logOutput, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		infoLogger:    log.New(logOutput, "[INFO]  ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger:    log.New(logOutput, "[WARN]  ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger:   log.New(logOutput, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		level:         level,
		logFile:       config.LogFile,
		console:       config.Console,
		logFileOutput: logFileOutput,
	}
}

func (l *FileLogger) Debug(format string, v ...interface{}) {
	if l.level <= Debug {
		if len(v) == 0 {
			l.debugLogger.Print(format)
		} else {
			l.debugLogger.Printf(format, v...)
		}
	}
}

func (l *FileLogger) Info(format string, v ...interface{}) {
	if l.level <= Info {
		if len(v) == 0 {
			l.infoLogger.Print(format)
		} else {
			l.infoLogger.Printf(format, v...)
		}
	}
}

func (l *FileLogger) Warn(format string, v ...interface{}) {
	if l.level <= Warn {
		if len(v) == 0 {
			l.warnLogger.Print(format)
		} else {
			l.warnLogger.Printf(format, v...)
		}
	}
}

func (l *FileLogger) Error(format string, v ...interface{}) {
	if l.level <= Error {
		if len(v) == 0 {
			l.errorLogger.Print(format)
		} else {
			l.errorLogger.Printf(format, v...)
		}
	}
}

// SetConsole sets whether to enable console output
func (l *FileLogger) SetConsole(enable bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.console == enable {
		return
	}

	l.console = enable

	if enable {
		logOutput := io.MultiWriter(l.logFileOutput, os.Stdout)
		l.updateLoggers(logOutput)
	} else {
		logOutput := l.logFileOutput
		l.updateLoggers(logOutput)
	}
}

func (l *FileLogger) updateLoggers(logOutput io.Writer) {
	l.debugLogger.SetOutput(logOutput)
	l.infoLogger.SetOutput(logOutput)
	l.warnLogger.SetOutput(logOutput)
	l.errorLogger.SetOutput(logOutput)
}

// SetLevel sets the log level
func (l *FileLogger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.level = level
}

// GetLevel returns the current log level
func (l *FileLogger) GetLevel() LogLevel {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.level
}

// GetWorkerLogger returns the worker process logger instance
func (f *LoggerFactory) GetWorkerLogger() Logger {
	logOnce.Do(func() {
		workerLoggerInstance = NewFileLogger(defaultWorkerLoggerConfig, Info)
	})
	return workerLoggerInstance
}

// GetGUILogger returns the GUI program logger instance
func (f *LoggerFactory) GetGUILogger() Logger {
	logOnce.Do(func() {
		guiLoggerInstance = NewFileLogger(defaultGUILoggerConfig, Info)
	})
	return guiLoggerInstance
}
