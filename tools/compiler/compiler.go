package compiler

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Vec2 represents a 2D point or size
type Vec2 struct {
	X float64
	Y float64
}

// Options holds optional compilation settings
type Options struct {
	Position *Vec2  // -pos x,y - position the drawing at (x,y) in mm
	Size     *Vec2  // -size w,h - scale drawing to fit within width x height in mm
	GenGCode bool   // --gcode - generate G-code output
	GenSVG   bool   // --svg - generate SVG preview
	SubDir   string // subdirectory within outputDir for this compilation
}

// DefaultOptions returns options that generate both SVG and G-code
func DefaultOptions() Options {
	return Options{
		GenGCode: true,
		GenSVG:   true,
	}
}

// Result holds compilation output
type Result struct {
	Success   bool
	SVGPath   string
	GCodePath string
	Errors    []string
	Warnings  []string
	Stdout    string
	Stderr    string
}

// Compiler wraps the external sketchlang compiler
type Compiler struct {
	executablePath string // absolute path to compiler
	outputDir      string // base output directory
}

// New creates a new compiler wrapper
// executablePath can be relative or absolute - it will be converted to absolute
func New(executablePath, outputDir string) (*Compiler, error) {
	// Convert executable path to absolute so it works from any working directory
	absExePath, err := filepath.Abs(executablePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve compiler path: %w", err)
	}

	// Verify the executable exists
	if _, err := os.Stat(absExePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("compiler not found at: %s", absExePath)
	}

	// Convert output directory to absolute as well
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output directory: %w", err)
	}

	return &Compiler{
		executablePath: absExePath,
		outputDir:      absOutputDir,
	}, nil
}

// Compile compiles SketchLang code with default options (both SVG and G-code)
func (c *Compiler) Compile(code string, outputName string) (*Result, error) {
	return c.CompileWithOptions(code, outputName, DefaultOptions())
}

// getWorkDir returns the working directory for compilation
// If opts.SubDir is set, it creates and returns outputDir/SubDir
// Otherwise returns outputDir
func (c *Compiler) getWorkDir(opts Options) (string, error) {
	workDir := c.outputDir
	if opts.SubDir != "" {
		workDir = filepath.Join(c.outputDir, opts.SubDir)
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	return workDir, nil
}

// CompileWithOptions compiles SketchLang code with the specified options
func (c *Compiler) CompileWithOptions(code string, outputName string, opts Options) (*Result, error) {
	// Get working directory (creates subdirectory if needed)
	workDir, err := c.getWorkDir(opts)
	if err != nil {
		return nil, err
	}

	// Write code to input file
	inputPath := filepath.Join(workDir, outputName+".sketch")
	if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write input file: %w", err)
	}

	// Build command arguments - use just filename since we run from workDir
	inputFile := outputName + ".sketch"
	args := []string{inputFile, "-o", outputName}

	// Add position if specified
	if opts.Position != nil {
		args = append(args, "-pos", fmt.Sprintf("%g,%g", opts.Position.X, opts.Position.Y))
	}

	// Add size if specified
	if opts.Size != nil {
		args = append(args, "-size", fmt.Sprintf("%g,%g", opts.Size.X, opts.Size.Y))
	}

	// Add output format flags
	if opts.GenGCode {
		args = append(args, "--gcode")
	}
	if opts.GenSVG {
		args = append(args, "--svg")
	}

	// If neither format specified, default to both
	if !opts.GenGCode && !opts.GenSVG {
		args = append(args, "--gcode", "--svg")
	}

	// Run the compiler with absolute path
	cmd := exec.Command(c.executablePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = workDir

	err = cmd.Run()

	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	// Parse errors and warnings from stderr
	if stderr.Len() > 0 {
		lines := strings.Split(stderr.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(strings.ToLower(line), "warning") {
				result.Warnings = append(result.Warnings, line)
			} else {
				result.Errors = append(result.Errors, line)
			}
		}
	}

	if err != nil {
		result.Success = false
		if len(result.Errors) == 0 {
			result.Errors = append(result.Errors, err.Error())
		}
		return result, nil
	}

	// Check for output files (use full paths)
	svgPath := filepath.Join(workDir, outputName+".svg")
	gcodePath := filepath.Join(workDir, outputName+".txt")

	if _, err := os.Stat(svgPath); err == nil {
		result.SVGPath = svgPath
	}
	if _, err := os.Stat(gcodePath); err == nil {
		result.GCodePath = gcodePath
	}

	result.Success = (opts.GenSVG && result.SVGPath != "") ||
		(opts.GenGCode && result.GCodePath != "") ||
		(!opts.GenSVG && !opts.GenGCode && (result.SVGPath != "" || result.GCodePath != ""))

	return result, nil
}

// CompileToSVG compiles and returns just the SVG content
func (c *Compiler) CompileToSVG(code string, outputName string, subDir string) (string, error) {
	opts := Options{GenSVG: true, SubDir: subDir}
	result, err := c.CompileWithOptions(code, outputName, opts)
	if err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("compilation failed: %v", result.Errors)
	}

	if result.SVGPath == "" {
		return "", fmt.Errorf("no SVG output generated")
	}

	content, err := os.ReadFile(result.SVGPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SVG: %w", err)
	}

	return string(content), nil
}

// CompileToGCode compiles and returns just the G-code content
func (c *Compiler) CompileToGCode(code string, outputName string, subDir string) (string, error) {
	opts := Options{GenGCode: true, SubDir: subDir}
	result, err := c.CompileWithOptions(code, outputName, opts)
	if err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("compilation failed: %v", result.Errors)
	}

	if result.GCodePath == "" {
		return "", fmt.Errorf("no G-code output generated")
	}

	content, err := os.ReadFile(result.GCodePath)
	if err != nil {
		return "", fmt.Errorf("failed to read G-code: %w", err)
	}

	return string(content), nil
}

// CompileBoth compiles and returns both SVG and G-code content
func (c *Compiler) CompileBoth(code string, outputName string, subDir string) (svg string, gcode string, err error) {
	opts := Options{GenSVG: true, GenGCode: true, SubDir: subDir}
	result, err := c.CompileWithOptions(code, outputName, opts)
	if err != nil {
		return "", "", err
	}

	if !result.Success {
		return "", "", fmt.Errorf("compilation failed: %v", result.Errors)
	}

	if result.SVGPath != "" {
		content, err := os.ReadFile(result.SVGPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to read SVG: %w", err)
		}
		svg = string(content)
	}

	if result.GCodePath != "" {
		content, err := os.ReadFile(result.GCodePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to read G-code: %w", err)
		}
		gcode = string(content)
	}

	return svg, gcode, nil
}

// Validate checks if code compiles without keeping outputs
func (c *Compiler) Validate(code string) (bool, []string) {
	result, err := c.Compile(code, "_validate_temp")
	if err != nil {
		return false, []string{err.Error()}
	}

	// Clean up temp files
	c.cleanupTempFiles("_validate_temp", "")

	return result.Success, result.Errors
}

// cleanupTempFiles removes temporary compilation artifacts
func (c *Compiler) cleanupTempFiles(baseName string, subDir string) {
	workDir := c.outputDir
	if subDir != "" {
		workDir = filepath.Join(c.outputDir, subDir)
	}

	extensions := []string{".sketch", ".svg", ".txt"}
	for _, ext := range extensions {
		os.Remove(filepath.Join(workDir, baseName+ext))
	}
}

// GetOutputDir returns the compiler's base output directory
func (c *Compiler) GetOutputDir() string {
	return c.outputDir
}

// GetExecutablePath returns the absolute path to the compiler executable
func (c *Compiler) GetExecutablePath() string {
	return c.executablePath
}