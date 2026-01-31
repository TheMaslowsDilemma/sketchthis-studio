package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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

	log.Info("generating sketch...\n")
	result, err := Generate(client, prompt, *url, log)
	if err != nil {
		fatal("generation failed: %v", err)
	}

	outName := *output
	if outName == "" {
		outName = sanitize(result.Title)
	}

	outDir := filepath.Join("output", outName)
	must(os.MkdirAll(outDir, 0755))

	log.Info("compiling...\n")
	svg, gcode, err := Compile(result.Code, outName, posVec, sizeVec, log)
	if err != nil {
		fatal("compile failed: %v", err)
	}

	sketchPath := filepath.Join(outDir, outName+".sketch")
	svgPath := filepath.Join(outDir, outName+".svg")
	gcodePath := filepath.Join(outDir, outName+".gcode")

	must(os.WriteFile(sketchPath, []byte(result.Code), 0644))
	must(os.WriteFile(svgPath, []byte(svg), 0644))
	must(os.WriteFile(gcodePath, []byte(gcode), 0644))

	abs1, _ := filepath.Abs(sketchPath)
	abs2, _ := filepath.Abs(svgPath)
	abs3, _ := filepath.Abs(gcodePath)
	fmt.Printf("%s\n%s\n%s\n", abs1, abs2, abs3)
}

func parseVec(s string) Vec2 {
	var x, y float64
	fmt.Sscanf(s, "%f,%f", &x, &y)
	return Vec2{x, y}
}


func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r > 127 {
			if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsPunct(r) {
				return r
			}
			return -1
		}
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
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