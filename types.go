package main

import "time"

// StudioConfig holds configuration for the studio.
type StudioConfig struct {
    CompilerPath   string
    OutputDir      string
    AnthropicKey   string
    Model          string
    EnableLogging  bool
    VerboseLogging bool
}

// SketchRequest is the input for sketch generation.
type SketchRequest struct {
    Description string
    RequestFrom string
    CreatedAt   time.Time
}

// Sketch is the final output of generation.
type Sketch struct {
    Title    string
    Summary  string
    Metadata map[string]string
    Code     string
    SVGPath  string
}

// SketchResult is the parsed response from the artist.
type SketchResult struct {
    Title    string
    Summary  string
    Metadata map[string]string
    Code     string
}