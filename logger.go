package nlock

import (
	"fmt"
	"log"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
)

func (ll LogLevel) String() string {
	switch ll {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("LEVEL %d", ll)
	}

}

type logging struct {
	level LogLevel
}

var logger *logging = &logging{level: LevelDebug}

func (l *logging) log(severity LogLevel, format string, a ...any) {
	if l.level > severity {
		// log.Printf("level:%d, severity: %d", l.level, severity)
		return
	}
	prefix := fmt.Sprintf("[%s]\t", severity)
	log.Printf(prefix+format, a...)
}

func (l *logging) Debug(format string, a ...any) {
	// log.Printf("[DEBUG]\t"+format, a...)
	l.log(LevelDebug, format, a...)
}

func (l *logging) Info(format string, a ...any) {
	l.log(LevelInfo, format, a...)
}

func (l *logging) Warning(format string, a ...any) {
	l.log(LevelWarning, format, a...)
}

func (l *logging) Error(format string, a ...any) {
	l.log(LevelError, format, a...)
}
