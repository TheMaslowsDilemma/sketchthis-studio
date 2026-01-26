package main

import (
    "fmt"
    "os"
)

type Vec2 struct{ X, Y float64 }

type SketchResult struct {
    Code    string
    Title   string
    Summary string
}

type Logger struct {
    enabled bool
}

func (l *Logger) Info(format string, args ...any) {
    if l.enabled {
        printf("INFO: "+format, args...)
    }
}

func (l *Logger) Warn(format string, args ...any) {
    if l.enabled {
        printf("WARN: "+format, args...)
    }
}

func (l *Logger) Debug(format string, args ...any) {
    if l.enabled {
        printf("DEBUG: "+format, args...)
    }
}

func printf(format string, args ...any) {
    fmt.Fprintf(os.Stderr, format+"\n", args...)
}