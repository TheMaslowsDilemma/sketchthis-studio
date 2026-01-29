# sketchgen

Minimal CLI tool for generating SketchLang sketches from descriptions or images using LLMs.

## Installation

```bash
go build -o sketchgen .
```

Requires `sketchlang` compiler in PATH.

## Usage

### From Description

```bash
sketchgen -d "an extremely detailed image of the Notre Dame Cathedral" -pos 0,0 -size 80,80
```

### From Image URL

```bash
sketchgen -url "https://example.com/image.jpg" -pos 0,0 -size 80,80
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-d` | | Image description |
| `-url` | | Image URL to sketch |
| `-pos` | `0,0` | Position (x,y) in mm |
| `-size` | `80,80` | Size (w,h) in mm |
| `-o` | auto | Output filename (without extension) |
| `-local` | false | Use local LMStudio instead of Anthropic |
| `-debug` | false | Enable debug logging |


## Outputs

Files are written to `output/<n>/`:
- `<n>.sketch` - SketchLang source code
- `<n>.svg` - SVG preview
- `<n>.gcode` - G-code for plotter

Output paths are printed to stdout (one per line).

## Configuration

### Anthropic (Default)

Set `ANTHROPIC_API_KEY` environment variable:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### Local LMStudio

Start LMStudio with a model loaded, then use `-local`:

```bash
sketchgen -d "a cat" -local
```

Expects OpenAI-compatible API at `http://localhost:1234`.

## Examples

```bash
# Simple sketch
sketchgen -d "a vintage bicycle"

# Detailed with positioning
sketchgen -d "the Eiffel Tower with intricate ironwork details" -pos 10,10 -size 60,100

# From URL with debug output
sketchgen -url "https://example.com/photo.jpg" -debug

# Using local model
sketchgen -d "a Japanese temple" -local -debug
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (message on stderr) |

## Language Spec

Edit `lang.go` to customize the SketchLang specification provided to the LLM.