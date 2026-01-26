package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	desc := flag.String("d", "", "image description")
	url := flag.String("url", "", "image URL")
	pos := flag.String("pos", "0,0", "position x,y in mm")
	size := flag.String("size", "80,80", "size w,h in mm")
	local := flag.Bool("local", false, "use local LMStudio")
	debug := flag.Bool("debug", false, "emit debug logs")
	output := flag.String("o", "", "output name (default: derived from input)")
	flag.Parse()

	if *desc == "" && *url == "" {
		fatal("provide -d or -url")
	}

	log := &Logger{enabled: *debug}

	var client LLMClient
	if *local {
		client = NewLocalClient(log)
	} else {
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			fatal("ANTHROPIC_API_KEY not set")
		}
		client = NewAnthropicClient(key, log)
	}

	posVec := parseVec(*pos)
	sizeVec := parseVec(*size)

	prompt := *desc
	if *url != "" {
		prompt = fmt.Sprintf("Create an extremely detailed sketch of the image at this URL: %s", *url)
	}

	log.Info("generating sketch...")
	result, err := Generate(client, prompt, log)
	if err != nil {
		fatal("generation failed: %v", err)
	}

	outName := *output
	if outName == "" {
		outName = sanitize(result.Title)
	}

	log.Info("compiling to SVG...")
	svg, err := Compile(result.Code, outName, posVec, sizeVec, log)
	if err != nil {
		fatal("compile failed: %v", err)
	}

	sketchPath := outName + ".sketch"
	svgPath := outName + ".svg"

	must(os.WriteFile(sketchPath, []byte(result.Code), 0644))
	must(os.WriteFile(svgPath, []byte(svg), 0644))

	abs1, _ := filepath.Abs(sketchPath)
	abs2, _ := filepath.Abs(svgPath)
	fmt.Printf("%s\n%s\n", abs1, abs2)
}

func parseVec(s string) Vec2 {
	var x, y float64
	fmt.Sscanf(s, "%f,%f", &x, &y)
	return Vec2{x, y}
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, s)
	if len(s) > 40 {
		s = s[:40]
	}
	return strings.Trim(s, "_")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func must(err error) {
	if err != nil {
		fatal("%v", err)
	}
}