package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sketch-studio/entities/artist"
	"sketch-studio/tools/compiler"
	"sketch-studio/tools/llm"
	"sketch-studio/tools/logger"
)

// Studio orchestrates the sketch generation process
type Studio struct {
	config   StudioConfig
	artist   *artist.Artist
	compiler *compiler.Compiler
	log      *logger.Logger
}

// NewStudio creates a new sketch studio
func NewStudio(config StudioConfig, langSpec string) (*Studio, error) {
	// Set defaults
	if config.OutputDir == "" {
		config.OutputDir = "./output"
	}
	if config.Model == "" {
		config.Model = "claude-opus-4-5"
	}
	if config.MaxIterations == 0 {
		config.MaxIterations = 1
	}

	// Validate
	if config.AnthropicKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}
	if config.CompilerPath == "" {
		return nil, fmt.Errorf("compiler path is required")
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize logger
	logLevel := logger.LevelInfo
	if config.VerboseLogging {
		logLevel = logger.LevelDebug
	}
	log := logger.New(os.Stdout, logLevel, "studio")

	// Initialize LLM client
	llmClient := llm.NewAnthropicClient(config.AnthropicKey, config.Model)

	// Initialize artist
	art := artist.New(llmClient, langSpec, log)

	// Initialize compiler
	comp, err := compiler.New(config.CompilerPath, config.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create compiler: %w", err)
	}

	return &Studio{
		config:   config,
		artist:   art,
		compiler: comp,
		log:      log,
	}, nil
}

// Generate creates a sketch from a request
func (s *Studio) Generate(ctx context.Context, req SketchRequest) (*Sketch, error) {
	startTime := time.Now()
	s.log.Info("═══════════════════════════════════════════════════════════════")
	s.log.Info("Starting sketch generation")
	s.log.Info("Description: %s", req.Description)
	s.log.Info("═══════════════════════════════════════════════════════════════")

	// Step 1: Create the initial plan
	s.log.Info("")
	s.log.Info("PHASE 1: Planning")
	s.log.Info("─────────────────────────────────────────────────────────────────")

	plan, planResp, err := s.artist.Plan(ctx, req.Description)
	if err != nil {
		return nil, fmt.Errorf("planning failed: %w", err)
	}

	// Create subdirectory for this sketch
	sketchDir := sanitize(plan.Title)

	s.log.Info("Title: %s", plan.Title)
	s.log.Info("Output folder: %s", sketchDir)
	s.log.Info("Summary: %s", truncate(plan.Summary, 100))
	s.log.Info("Sections: %d", len(plan.Sections))
	for _, sec := range plan.Sections {
		s.log.Section(sec.Title, truncate(sec.Description, 60))
	}

	// Save raw response for debugging
	if s.config.VerboseLogging {
		rawDir := filepath.Join(s.config.OutputDir, sketchDir)
		os.MkdirAll(rawDir, 0755)
		rawPath := filepath.Join(rawDir, "plan_raw.txt")
		os.WriteFile(rawPath, []byte(planResp.Content), 0644)
	}

	// Step 2: Compile initial contours
	s.log.Info("")
	s.log.Info("PHASE 2: Compiling Contours")
	s.log.Info("─────────────────────────────────────────────────────────────────")

	contourName := "contours"
	contourOpts := compiler.Options{
		GenSVG:   true,
		GenGCode: true,
		SubDir:   sketchDir,
	}
	contourResult, err := s.compiler.CompileWithOptions(plan.ContourCode, contourName, contourOpts)
	if err != nil {
		return nil, fmt.Errorf("contour compilation error: %w", err)
	}

	s.log.Compilation(contourResult.Success, contourResult.SVGPath, contourResult.Errors)

	if !contourResult.Success {
		// Save the failed code for inspection
		failedDir := filepath.Join(s.config.OutputDir, sketchDir)
		os.MkdirAll(failedDir, 0755)
		failedPath := filepath.Join(failedDir, contourName+"_failed.sketch")
		os.WriteFile(failedPath, []byte(plan.ContourCode), 0644)
		s.log.Warn("Failed contour code saved to: %s", failedPath)
		return nil, fmt.Errorf("contour compilation failed: %v", contourResult.Errors)
	}

	// Build the sketch object
	sketch := &Sketch{
		Summary: SketchSummary{
			Title:       plan.Title,
			Summary:     plan.Summary,
			Subject:     plan.Subject,
			Perspective: plan.Perspective,
			Style:       plan.Style,
			Metadata:    plan.Metadata,
		},
		Contours: plan.ContourCode,
	}

	for _, sec := range plan.Sections {
		sketch.Sections = append(sketch.Sections, SketchSection{
			Title:       sec.Title,
			Description: sec.Description,
			Neighbors:   sec.Neighbors,
			Expanded:    false,
		})
	}

	// Step 3: Expand each section
	s.log.Info("")
	s.log.Info("PHASE 3: Expanding Sections")
	s.log.Info("─────────────────────────────────────────────────────────────────")

	expandedCode := plan.ContourCode + "\n\n# === EXPANDED DETAILS ===\n"

	for i, section := range plan.Sections {
		s.log.Info("")
		s.log.Info("[%d/%d] Expanding: %s", i+1, len(plan.Sections), section.Title)

		expandedSection, _, err := s.artist.ExpandSection(ctx, plan, section, plan.ContourCode)
		if err != nil {
			s.log.Error("Failed to expand section %s: %v", section.Title, err)
			continue
		}

		// Validate the expanded code compiles
		sectionName := "expanded_" + sanitize(section.Title)
		testCode := expandedCode + "\n\n# Section: " + section.Title + "\n" + expandedSection

		sectionOpts := compiler.Options{
			GenSVG:   true,
			GenGCode: true,
			SubDir:   sketchDir,
		}
		result, err := s.compiler.CompileWithOptions(testCode, sectionName, sectionOpts)
		if err != nil {
			s.log.Error("Compilation error for %s: %v", section.Title, err)
			continue
		}

		if !result.Success {
			s.log.Warn("Section %s failed to compile: %v", section.Title, result.Errors)
			// Save failed code for inspection
			failedPath := filepath.Join(s.config.OutputDir, sketchDir, sectionName+"_failed.sketch")
			os.WriteFile(failedPath, []byte(testCode), 0644)
			continue
		}

		s.log.Compilation(result.Success, result.SVGPath, result.Errors)

		// Add to accumulated code
		expandedCode = testCode
		sketch.Sections[i].Content = expandedSection
		sketch.Sections[i].Expanded = true
	}

	// Step 4: Final compilation
	s.log.Info("")
	s.log.Info("PHASE 4: Final Compilation")
	s.log.Info("─────────────────────────────────────────────────────────────────")

	finalName := "final"
	finalOpts := compiler.Options{
		GenSVG:   true,
		GenGCode: true,
		SubDir:   sketchDir,
	}
	finalResult, err := s.compiler.CompileWithOptions(expandedCode, finalName, finalOpts)
	if err != nil {
		return nil, fmt.Errorf("final compilation error: %w", err)
	}

	s.log.Compilation(finalResult.Success, finalResult.SVGPath, finalResult.Errors)

	// Summary
	s.log.Info("")
	s.log.Info("═══════════════════════════════════════════════════════════════")
	s.log.Info("Generation Complete")
	s.log.Info("Total time: %v", time.Since(startTime).Round(time.Millisecond))
	s.log.Info("Output folder: %s", filepath.Join(s.config.OutputDir, sketchDir))
	s.log.Info("Final SVG: %s", finalResult.SVGPath)
	s.log.Info("═══════════════════════════════════════════════════════════════")

	return sketch, nil
}

// sanitize creates a safe filename from a string
func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	safe := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			safe += string(c)
		}
	}
	if len(safe) > 50 {
		safe = safe[:50]
	}
	return safe
}

// truncate shortens a string with ellipsis
func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}