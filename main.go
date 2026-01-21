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

func main() {
	cfg := parseFlags()

	studio, err := NewStudio(cfg.studio, langSpec)
	if err != nil {
		fatal("create studio: %v", err)
	}

	req := SketchRequest{
		Description: cfg.description,
		RequestFrom: cfg.requestFrom,
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupt(cancel)

	var sketch *Sketch
	if cfg.validate {
		sketch, err = studio.GenerateWithValidation(ctx, req)
	} else {
		sketch, err = studio.Generate(ctx, req)
	}
	if err != nil {
		fatal("generate: %v", err)
	}

	fmt.Printf("\n✓ Sketch '%s' generated\n", sketch.Title)
	fmt.Printf("  SVG: %s\n", sketch.SVGPath)
}

type config struct {
	studio      StudioConfig
	description string
	requestFrom string
	validate    bool
}

func parseFlags() config {
	var c config

	flag.StringVar(&c.description, "d", "", "Description of the sketch (required)")
	flag.StringVar(&c.studio.CompilerPath, "compiler", "./output/main.exe", "Path to sketchlang compiler")
	flag.StringVar(&c.studio.OutputDir, "output", "./output", "Output directory")
	flag.StringVar(&c.studio.AnthropicKey, "key", "", "Anthropic API key (or ANTHROPIC_API_KEY env)")
	flag.StringVar(&c.studio.Model, "model", "claude-sonnet-4-5", "Model to use")
	flag.BoolVar(&c.studio.VerboseLogging, "v", false, "Verbose logging")
	flag.BoolVar(&c.validate, "validate", false, "Enable compile validation feedback")
	flag.StringVar(&c.requestFrom, "from", "", "Source handle")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Sketch Studio - AI-powered sketch generation\n\n")
		fmt.Fprintf(os.Stderr, "Usage: sketch-studio -d \"description\" [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if c.description == "" {
		fmt.Fprintln(os.Stderr, "Error: -d description required")
		flag.Usage()
		os.Exit(1)
	}

	if c.studio.AnthropicKey == "" {
		c.studio.AnthropicKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if c.studio.AnthropicKey == "" {
		fatal("Anthropic API key required (-key or ANTHROPIC_API_KEY env)")
	}

	c.studio.EnableLogging = true
	return c
}

func handleInterrupt(cancel context.CancelFunc) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nInterrupted...")
		cancel()
	}()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}