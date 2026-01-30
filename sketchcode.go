package main

/* sketch code can be compiled, viewed, editted */

func NewSketchCode(code string) *SketchCode {
	return &SketchCode { RawCode: code }
}

// attempt to compile code or return info about error
func (sc *SketchCode) Validate(pos, size Vec2, log *Logger) (CompilationResult, error) {
	var (
		err 	error
		serr	[]SketchError
	)
	if _, _, err = Compile(sc.RawCode, "_validate", pos, size, log); err != nil {
		
	}
}
