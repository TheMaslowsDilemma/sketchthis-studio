package main

import (
    "fmt"
)

type Vec2 struct{ X, Y float64 }

type SketchError struct {
    Line int        `json:"line"`
    Column int      `json:"col"`
    Message string  `json:"msg"`
}

type SketchResult struct {
    Code    string
    Title   string
}

type Logger struct {
    enabled bool
}

func (l *Logger) Info(format string, args ...any) {
    if l.enabled {
        fmt.Printf("[\033[32mINFO\033[0m]: " + format + "\n", args...)
    }
}

func (l *Logger) Warn(format string, args ...any) {
    if l.enabled {
        fmt.Printf("[\033[33mWARN\033[0m]: " + format + "\n", args...)
    }
}

func (l *Logger) Debug(format string, args ...any) {
    if l.enabled {
        fmt.Printf("[DEBUG]: " + format + "\n", args...)
    }
}