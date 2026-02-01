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
  mirror sketch about vec
  translate sketch by vec
  scale sketch by number
  rotate sketch by number
  [sketch, ...]

## Render: trace (exact) | draw (wobble) | scribble (noisy)

## Examples
let arrow : sketch = [
  stroke (0, 0) to (20, 0),
  stroke (20, 0) to (15, 5),
  stroke (20, 0) to (15, -5)
]

let arrows : sketch = [
  arrow,
  rotate arrow by 45,
  rotate arrow by 90
]

let arrw1 : sketch = scale arrows by 0.4
let arrw2 : sketch = translate arrw1 by (20, 20)
let arrw3 : sketch = translate arrw2 by (20, 0)

draw arrw1
trace arrw2
scribble rotate arrw3 by 90

## Tips
- Prioritize DRY principles to minimize token count. Modular components can be re-used.
- use consice, short variable names (NOT "by" thats a keyword)
- NO dot notation (vec.x invalid), NO reassignment
- Minimal comments if any
- transformations are not render calls.
- dash is a sketch, not a statement: scribble dash (10,10)
- via points create Catmull-Rom splines
- x_axis, y_axis, origin are globally defined vecs
- Maximize use of transformations (translate, scale, mirror) for component. 
- Mirroring happens along the center of a sketch, MAKE SURE to translate mirrors if needed.
- Comments are made with #
`