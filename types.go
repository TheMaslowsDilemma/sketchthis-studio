package main

import "time"

// SketchRequest represents an incoming request to generate a sketch
type SketchRequest struct {
    Description string
    RequestFrom string // X user handle
    CreatedAt   time.Time
}

// SketchSummary contains high-level metadata about the sketch
type SketchSummary struct {
    Title       string
    Summary     string
    Subject     string
    Perspective string
    Style       string
    Metadata    map[string]string // lighting direction, mood, etc. left to the LLM
}

// SketchSection represents a portion of the sketch that can be worked on independently
type SketchSection struct {
    Title       string
    Description string
    Neighbors   []string // titles of adjacent sections for coordination
    Content     string   // the SketchLang code for this section
    Expanded    bool     // whether this section has been detailed by a sub-artist
}

// Sketch is the complete representation of a sketch in progress
type Sketch struct {
    Summary  SketchSummary
    Sections []SketchSection
    Contours string // the initial contour SketchLang code
}

// StudioConfig holds configuration for the sketch studio
type StudioConfig struct {
    CompilerPath   string // path to sketchlang compiler executable
    OutputDir      string // directory for output files
    AnthropicKey   string // API key for Claude
    Model          string // model to use (e.g., "claude-opus-4-5")
    MaxIterations  int    // max iterations per section
    EnableLogging  bool
    VerboseLogging bool
}

// CompilationResult holds the result of compiling SketchLang code
type CompilationResult struct {
    Success   bool
    SVGPath   string
    GCodePath string
    Errors    []string
    Warnings  []string
}

// ArtistResponse represents what comes back from the LLM artist
type ArtistResponse struct {
    Summary     SketchSummary
    Sections    []SketchSection
    ContourCode string
    TokensUsed  int
    Duration    time.Duration
    RawResponse string
}

// SubArtistResponse represents what comes back from a sub-artist expanding a section
type SubArtistResponse struct {
    SectionTitle string
    ExpandedCode string
    TokensUsed   int
    Duration     time.Duration
    RawResponse  string
}
