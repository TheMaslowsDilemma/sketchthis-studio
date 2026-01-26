package main

const LangSpec  = `# SketchLang Quick Reference

## Types
- number: float
- vec: 2D point (x, y)
- sketch: drawable or list of sketches

## Syntax
let NAME : type = expr
trace|draw|scribble sketch_expr

## Expressions

Numbers: literals, +, -, *, /, parentheses

Vectors:
  (x, y)              -- construct
  origin              -- (0, 0)
  center of sketch    -- centroid
  flow at vec         -- flow field direction
  vec + vec, vec - vec, vec * number

Sketches:
  dot at vec
  dash at vec
  stroke from vec to vec [via [vec, ...]]
  [sketch, sketch, ...]   -- list

## Render Commands
- trace: exact, clean lines
- draw: slight wobble, hand-drawn
- scribble: heavy noise, sketchy

## Examples

### Curves with control points
let curve : sketch = stroke from (0, 50) to (100, 50) via [(50, 0)]
trace curve

### Centroid and composition
let triangle : sketch = [
  stroke from (50, 10) to (10, 90),
  stroke from (10, 90) to (90, 90),
  stroke from (90, 90) to (50, 10)
]
let heart : vec = center of triangle
let spokes : sketch = [
  stroke from heart to (50, 10),
  stroke from heart to (10, 90),
  stroke from heart to (90, 90),
  dash at (80,80),
  dash at (60,60)
]
trace [triangle, spokes]

### Nested center reference
scribble stroke from origin to center of stroke from heart to (20, 26)

## Rules
- NO dot notation (vec.x invalid)
- NO reassignment
- dash is a sketch, not a statement: scribble dash at (10,10)
- via points create Catmull-Rom splines
- Flow field affects only dash orientation
- Coordinates in mm, comments with #
`