package main

const LangSpec = `# SketchLang Reference

## Types
number (float), vec (x,y), sketch (drawable/list), region (closed polygon)

## Syntax
let NAME = expr
trace|draw|scribble expr

## Expressions

### Numbers
Literals, +, -, *, /, parentheses

### Vectors
- (x, y) literal
- origin, x_axis, y_axis (globals)
- centerof sketch
- vec ± vec, vec * num

### Sketches
- dot vec | dot [vec, ...]
- dash vec | dash [vec, ...]
- stroke -> [vec, ...] (straight segments)
- stroke ~> [vec, ...] (Catmull-Rom spline)
- [sketch, ...]
- shade region_or_sketch

### Regions
- regionof sketch (convex hull)

### Transforms (pipe or prefix)
- expr |> translate vec
- expr |> scale number
- expr |> rotate degrees
- expr |> mirror vec
- expr |> at vec
- translate expr vec (prefix form)

## Strokes
stroke -> [...] connects points with straight segments.
stroke ~> [...] interpolates a smooth curve through points.
Both require at least two points.

RIGHT — rectangle:
  let box = [
    stroke -> [(0,0), (10,0), (10,10), (0,10), (0,0)]
  ]

RIGHT — smooth curve:
  let hill = stroke ~> [(0,0), (20,15), (40,20), (60,15), (80,0)]

WRONG:
  stroke (0,0) to (10,0)
  stroke from (0,0) to (10,0) via [...]

## Render Modes
- trace: exact lines
- draw: slight wobble
- scribble: noisy/sketchy

## Rules

=== RULE 1 ===
Transform arguments parse as atoms. No parens needed
unless doing inline math.
  draw s |> scale 2              # fine
  draw s |> scale (2 * 3)        # parens needed

=== RULE 2 ===
Transforms operate relative to sketch center. After mirror
use translate or at to align, otherwise sketches overlap.
  let wing = stroke ~> [(0,0), (5,12), (15,10)]
  let bird = [wing, wing |> mirror y_axis |> translate (15, 0)]

=== RULE 3 ===
When geometry must rotate around a shared point, define it
centered at origin. Use at to place it last.
  let spoke = stroke -> [(-r, 0), (r, 0)]
  draw [spoke, spoke |> rotate 60, spoke |> rotate 120] |> at center

## Example
  let eye = stroke ~> [(0,0), (4,3), (8,0)]
  draw [
    eye |> at (35, 60),
    eye |> at (55, 60)
  ]

  let petal = stroke ~> [(0,0), (-3,4), (3,4), (0,8)]
  let bloom = [
    petal,
    petal |> rotate 45,
    petal |> rotate 90,
    petal |> rotate 135,
    petal |> rotate 180,
    petal |> rotate 225,
    petal |> rotate 270,
    petal |> rotate 315
  ]

  let leaf = stroke ~> [(0,0), (5,8), (3,15), (0,20), (-3,15), (-5,8)]
  let shaded_leaf = [leaf, shade leaf]
  draw shaded_leaf |> scale 0.7 |> rotate 30 |> at (80, 110)

## Notes
- NO dot notation (v.x is invalid)
- NO variable reassignment
- NO type annotations in let bindings
- Transforms are expressions returning new sketches
- dash orientation follows the flow field of nearby strokes
- shade fills a region with random dashes (use scribble for texture)
- Stack multiple shade passes for denser fill
- Coordinates are in mm
- Use short names, reuse with transforms (DRY)
- Minimal comments
`