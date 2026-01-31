package main

import (
    "fmt"
    "os"
)

type Vec2 struct{ X, Y float64 }

type SketchCode struct {
    RawCode string
}

type SketchError struct {
    Line int        `json:"line"`
    Column int      `json:"col"`
    Message string  `json:"msg"`
}

type SketchResult struct {
    Code    SketchCode
    Title   string
}

type Logger struct {
    enabled bool
}

func (l *Logger) Info(format string, args ...any) {
    if l.enabled {
        fmt.Printf("[INFO]: " + format, args...)
    }
}

func (l *Logger) Warn(format string, args ...any) {
    if l.enabled {
        fmt.Printf("[WARN]: " + format, args...)
    }
}

func (l *Logger) Debug(format string, args ...any) {
    if l.enabled {
        fmt.Printf("[DEBUG]: " + format, args...)
    }
}