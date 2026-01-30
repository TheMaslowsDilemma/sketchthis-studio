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
  dot vec
  dash vec
  stroke vec to vec [via [vec,...]]
  translate sketch by vec
  scale sketch by number
  mirror sketch about vec
  [sketch, ...]

## Render: trace (exact) | draw (wobble) | scribble (noisy)

## Examples
let c : sketch = stroke (0,50) to (100,50) via [(50,0)]
trace c

let p1 : vec = (50,10)
let p2 : vec = (10,90)
let p3 : vec = (90,90)
let tri : sketch = [stroke p1 to p2, stroke p2 to p3, stroke p3 to p1]
let big : sketch = scale tri by 2
draw translate big by (100, 0)

scribble stroke origin to center of tri

## Rules
- Define base primitives once and transform them. Prioritize DRY principles to minimize token count.
- USE short variable names (NOT "by" thats a keyword)
- Complete sketch with much detail
- NO dot notation (vec.x invalid), NO reassignment
- Minimal comments if any
- dash is a sketch, not a statement: scribble dash (10,10)
- via points create Catmull-Rom splines
- mirror reflects about axis through sketch centroid, x_axis and y_axis are global constants
- Maximize use of transformations (translate, scale, mirror) for component. 
- FIRST DEFINE COUNTOURS! and ANCHORPOINTS! Then add detail and reuse compoments
`