package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sketch-studio/tools/compiler"
	"sketch-studio/tools/llm"
	"sketch-studio/tools/logger"
)

// Studio orchestrates sketch generation.
type Studio struct {
	config   StudioConfig
	artist   *Artist
	compiler *compiler.Compiler
	log      *logger.Logger
}

// NewStudio creates a new Studio instance.
func NewStudio(config StudioConfig, langSpec string) (*Studio, error) {
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	log := logger.New(os.Stdout, logLevel(config.VerboseLogging), "studio")
	llmClient := llm.NewAnthropicClient(config.AnthropicKey, config.Model)

	comp, err := compiler.New(config.CompilerPath, config.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("create compiler: %w", err)
	}

	return &Studio{
		config:   config,
		artist:   NewArtist(llmClient, langSpec, log),
		compiler: comp,
		log:      log,
	}, nil
}

// Generate creates a complete sketch from a request.
func (s *Studio) Generate(ctx context.Context, req SketchRequest) (*Sketch, error) {
	return s.generate(ctx, req, false)
}

// GenerateWithValidation creates a sketch with compile-time validation feedback.
func (s *Studio) GenerateWithValidation(ctx context.Context, req SketchRequest) (*Sketch, error) {
	return s.generate(ctx, req, true)
}

func (s *Studio) generate(ctx context.Context, req SketchRequest, validate bool) (*Sketch, error) {
	start := time.Now()
	s.logHeader("Starting sketch generation", req.Description)

	// Generate sketch
	s.log.Info("Generating sketch...")
	s.logDivider()

	var result *SketchResult
	var resp *llm.Response
	var err error

	if validate {
		result, resp, err = s.artist.CreateSketchWithValidation(ctx, req.Description, s.validateCode)
	} else {
		result, resp, err = s.artist.CreateSketch(ctx, req.Description)
	}
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	sketchDir := sanitize(result.Title)
	s.log.Info("Title: %s", result.Title)
	s.log.Info("Output: %s", sketchDir)

	if s.config.VerboseLogging {
		s.saveDebugFile(sketchDir, "response_raw.txt", resp.Content)
	}

	// Compile
	s.log.Info("")
	s.log.Info("Compiling...")
	s.logDivider()

	compResult, err := s.compiler.CompileWithOptions(result.Code, "sketch", compiler.Options{
		GenSVG:   true,
		GenGCode: true,
		SubDir:   sketchDir,
	})
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}

	s.log.Compilation(compResult.Success, compResult.SVGPath, compResult.Errors)

	if !compResult.Success {
		s.saveDebugFile(sketchDir, "failed.sketch", result.Code)
		return nil, fmt.Errorf("compilation failed: %v", compResult.Errors)
	}

	s.logFooter(start, filepath.Join(s.config.OutputDir, sketchDir), compResult.SVGPath)

	return &Sketch{
		Title:    result.Title,
		Summary:  result.Summary,
		Metadata: result.Metadata,
		Code:     result.Code,
		SVGPath:  compResult.SVGPath,
	}, nil
}

func (s *Studio) validateCode(code string) (bool, []string) {
	result, err := s.compiler.CompileWithOptions(code, "validate", compiler.Options{})
	if err != nil {
		return false, []string{err.Error()}
	}
	return result.Success, result.Errors
}

func (s *Studio) saveDebugFile(dir, name, content string) {
	path := filepath.Join(s.config.OutputDir, dir)
	os.MkdirAll(path, 0755)
	os.WriteFile(filepath.Join(path, name), []byte(content), 0644)
}

func (s *Studio) logHeader(title, description string) {
	s.log.Info("═══════════════════════════════════════════════════════════════")
	s.log.Info(title)
	s.log.Info("Description: %s", description)
	s.log.Info("═══════════════════════════════════════════════════════════════")
	s.log.Info("")
}

func (s *Studio) logDivider() {
	s.log.Info("───────────────────────────────────────────────────────────────")
}

func (s *Studio) logFooter(start time.Time, outputDir, svgPath string) {
	s.log.Info("")
	s.log.Info("═══════════════════════════════════════════════════════════════")
	s.log.Info("Complete in %v", time.Since(start).Round(time.Millisecond))
	s.log.Info("Output: %s", outputDir)
	s.log.Info("SVG: %s", svgPath)
	s.log.Info("═══════════════════════════════════════════════════════════════")
}

// --- Helpers ---

func validateConfig(c *StudioConfig) error {
	if c.OutputDir == "" {
		c.OutputDir = "./output"
	}
	if c.Model == "" {
		c.Model = "claude-sonnet-4-5"
	}
	if c.AnthropicKey == "" {
		return fmt.Errorf("Anthropic API key required")
	}
	if c.CompilerPath == "" {
		return fmt.Errorf("compiler path required")
	}
	return nil
}

func logLevel(verbose bool) logger.Level {
	if verbose {
		return logger.LevelDebug
	}
	return logger.LevelInfo
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if len(result) > 50 {
		return result[:50]
	}
	return result
}