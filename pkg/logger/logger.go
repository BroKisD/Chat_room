package logger

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

func New(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "["+prefix+"] ", log.LstdFlags),
	}
}

func (l *Logger) Error(v ...interface{}) {
	l.Println("ERROR:", v)
}

func (l *Logger) Info(v ...interface{}) {
	l.Println("INFO:", v)
}

func (l *Logger) Debug(v ...interface{}) {
	l.Println("DEBUG:", v)
}
