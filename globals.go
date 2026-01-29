package main

const LangSpec  = `# SketchLang Reference

## Types
number (float), vec (x,y), sketch (drawable/list)

## Syntax
let NAME : type = expr
trace|draw|scribble sketch_expr

## Expressions
Nums: literals, +,-,*,/, parens
Vecs: (x,y), origin, center of sketch, vec±vec, vec*num
Sketches:
  dot at vec
  dash at vec
  stroke from vec to vec [via [vec,...]]
  mirror sketch about vec
  [sketch, ...]

## Render: trace (exact) | draw (wobble) | scribble (noisy)

## Examples
let c : sketch = stroke from (0,50) to (100,50) via [(50,0)]
trace c

let p1 : vec = (50,10)
let p2 : vec = (10,90)
let p3 : vec = (90,90)
let tri : sketch = [stroke from p1 to p2, stroke from p2 to p3, stroke from p3 to p1]
draw mirror tri about (1,0)

scribble stroke from origin to center of tri

## Rules
- Complete detailed sketches
- NO dot notation (v.x invalid), NO reassignment
- Short variable names, minimal comments
- dash is sketch: scribble dash at (10,10)
- via = Catmull-Rom splines

## Rules
- USE short variable names
- Complete sketch with much detail
- NO dot notation (vec.x invalid), NO reassignment
- Minimal Comments if any.
- dash is a sketch, not a statement: scribble dash at (10,10)
- via points create Catmull-Rom splines
- mirror reflects about axis through sketch centroid
`