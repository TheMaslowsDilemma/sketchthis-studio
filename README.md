# sketchgen

Minimal CLI tool for generating SketchLang sketches from descriptions or images using LLMs.

## Installation

```bash
go build -o sketchstudio .
```

Requires `sketchlang` compiler in PATH. See https://github.com/TheMaslowsDilemma/sketchthis-dsl
Also requires `ANTHROPIC_API_KEY` in ENV.

## Usage

### From Description

```bash
sketchstudio -d "an extremely detailed image of the Notre Dame Cathedral" -pos 0,0 -size 80,80
```

### From Image URL

```bash
sketchstudio -url "https://example.com/image.jpg" -pos 0,0 -size 80,80
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

The tool generates two files:
- `<name>.sketch` — SketchLang source code
- `<name>.svg` — SVG preview

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
sketchstudio -d "a cat" -local
```

Expects OpenAI-compatible API at `http://localhost:1234`.

## Examples

```bash
# Simple sketch
sketchstudio -d "a vintage bicycle"

# Detailed with positioning
sketchstudio -d "the Eiffel Tower with intricate ironwork details" -pos 10,10 -size 60,100

# From URL with debug output
sketchstudio -url "https://example.com/photo.jpg" -debug

# Using local model
sketchstudio -d "an extremely detailed sketch of the Notre Dame Cathedral" -local -debug
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (message on stderr) |

## Language Spec

Edit `lang.go` to customize the SketchLang specification provided to the LLM.