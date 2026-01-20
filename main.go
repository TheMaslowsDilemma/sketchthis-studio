package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Default SketchLang specification
const defaultLangSpec = `# Sketch DSL Language Specification

A minimal language for generating pen plotter artwork via G-code.

## Types

- ` + "`number`" + ` - floating point value
- ` + "`vec`" + ` - 2D point (x, y)
- ` + "`sketch`" + ` - drawable primitive or list of sketches

## Syntax

statement := let_binding | render_command
let_binding := "let" IDENT ":" type "=" expr
render_command := ("trace" | "draw" | "scribble") sketch_expr

type := "number" | "vec" | "sketch"

## Expressions

### Numbers
num_expr := NUMBER | IDENT | "-" num_expr
          | num_expr ("+" | "-" | "*" | "/") num_expr
          | "(" num_expr ")"

### Vectors
vec_expr := "(" num_expr "," num_expr ")"  -- construct
          | IDENT                           -- variable
          | "origin"                        -- (0, 0)
          | "center" "of" sketch_expr       -- centroid
          | "flow" "at" vec_expr            -- flow field direction
          | vec_expr ("+" | "-") vec_expr   -- arithmetic
          | vec_expr "*" num_expr           -- scale

### Sketches
sketch_expr := primitive | IDENT | "[" sketch_list "]"
sketch_list := sketch_expr ("," sketch_expr)*

primitive := "dot" "at" vec_expr
           | "dash" "at" vec_expr
           | "stroke" "from" vec_expr "to" vec_expr ["via" vec_list]

vec_list := "[" vec_expr ("," vec_expr)* "]"

## Render Commands

| Command | Effect |
|---------|--------|
| trace | Exact rendering, no noise |
| draw | Slight wobble, hand-drawn feel |
| scribble | Heavy noise, sketchy style |

## Flow Field

dash orientation is determined by nearby stroke directions. Strokes contribute to a flow field weighted by inverse-square distance. Default direction is horizontal if no strokes exist.

## Important Notes

- dot notation such as vec1.x or vec1.y is NOT SUPPORTED
- variable re-assignment is NOT SUPPORTED
- Dashes can be helpful with shading
- Comments start with #
- Comments are helpful to plan and label sections
- Coordinates are in mm
- Newlines separate statements
- Flow field only affects dash, not stroke or dot
- via points create smooth Catmull-Rom splines
- Noise magnitude: scribble > draw > trace (none)
`

func main() {
	// CLI flags
	description := flag.String("d", "", "Description of the sketch to generate")
	descFile := flag.String("f", "", "File containing the sketch description")
	compilerPath := flag.String("compiler", "./output/main.exe", "Path to sketchlang compiler")
	outputDir := flag.String("output", "./output", "Output directory for generated files")
	apiKey := flag.String("key", "", "Anthropic API key (or set ANTHROPIC_API_KEY env)")
	model := flag.String("model", "claude-opus-4-5", "Model to use")
	verbose := flag.Bool("v", false, "Verbose logging")
	langFile := flag.String("lang", "", "Path to SketchLang specification file")
	requestFrom := flag.String("from", "", "Source handle (e.g., X username)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Sketch Studio - AI-powered sketch generation

Usage:
  sketch-studio -d "description" [options]
  sketch-studio -f description.txt [options]

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  sketch-studio -d "a cat sitting on a windowsill"
  sketch-studio -f prompt.txt -output ./sketches -v

Environment:
  ANTHROPIC_API_KEY - API key for Claude (alternative to -key flag)

Output Structure:
  Each sketch is saved to its own subdirectory under the output directory:
    output/
      sketch_title/
        sketch_title_contours.sketch
        sketch_title_contours.svg
        sketch_title_contours.txt
`)
	}

	flag.Parse()

	// Get description
	desc := *description
	if desc == "" && *descFile != "" {
		content, err := os.ReadFile(*descFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading description file: %v\n", err)
			os.Exit(1)
		}
		desc = string(content)
	}

	if desc == "" {
		fmt.Fprintln(os.Stderr, "Error: description required (-d or -f)")
		flag.Usage()
		os.Exit(1)
	}

	// Get API key
	key := *apiKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: Anthropic API key required (-key or ANTHROPIC_API_KEY env)")
		os.Exit(1)
	}

	// Load language spec
	langSpec := defaultLangSpec
	if *langFile != "" {
		content, err := os.ReadFile(*langFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading language spec: %v\n", err)
			os.Exit(1)
		}
		langSpec = string(content)
	}

	// Create config
	config := StudioConfig{
		CompilerPath:   *compilerPath,
		OutputDir:      *outputDir,
		AnthropicKey:   key,
		Model:          *model,
		MaxIterations:  1,
		EnableLogging:  true,
		VerboseLogging: *verbose,
	}

	// Create studio
	studio, err := NewStudio(config, langSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating studio: %v\n", err)
		os.Exit(1)
	}

	// Create request
	request := SketchRequest{
		Description: desc,
		RequestFrom: *requestFrom,
		CreatedAt:   time.Now(),
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInterrupted, shutting down...")
		cancel()
	}()

	// Generate!
	sketch, err := studio.Generate(ctx, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating sketch: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSketch '%s' generated successfully!\n", sketch.Summary.Title)
}