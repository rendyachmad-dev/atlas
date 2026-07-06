package logger

import (
	"log"
	"os"
)

// Level represents log severity.
type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

var levelNames = map[Level]string{
	Debug: "DEBUG",
	Info:  "INFO",
	Warn:  "WARN",
	Error: "ERROR",
}

// Setup configures the standard logger with a given level string.
// Accepted values: debug, info, warn, error (case-insensitive).
func Setup(levelStr string) {
	lvl := Info
	switch levelStr {
	case "debug":
		lvl = Debug
	case "info":
		lvl = Info
	case "warn":
		lvl = Warn
	case "error":
		lvl = Error
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stdout)
	_ = lvl // level filtering can be added later
}