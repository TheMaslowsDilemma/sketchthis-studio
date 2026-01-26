package main

import (
	"fmt"
	"regexp"
	"strings"
)

const maxRetries = 3

func Generate(client LLMClient, description string, log *Logger) (*SketchResult, error) {
	messages := []Message{{Role: "user", Content: description}}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, err := client.Complete(systemPrompt(), messages)
		if err != nil {
			return nil, err
		}

		result, err := parseResponse(content)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Warn("parse error (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
				messages = append(messages,
					Message{Role: "assistant", Content: content},
					Message{Role: "user", Content: fmt.Sprintf("Parse error: %v\n\nPlease fix and include <title>, <summary>, and <code> tags.", err)},
				)
				continue
			}
			return nil, fmt.Errorf("parse failed after %d attempts: %w", maxRetries+1, err)
		}

		return result, nil
	}

	return nil, lastErr
}

func GenerateWithValidation(client LLMClient, description string, validate func(string) (bool, []string), log *Logger) (*SketchResult, error) {
	messages := []Message{{Role: "user", Content: description}}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, err := client.Complete(systemPrompt(), messages)
		if err != nil {
			return nil, err
		}

		result, err := parseResponse(content)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Warn("parse error (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
				messages = append(messages,
					Message{Role: "assistant", Content: content},
					Message{Role: "user", Content: fmt.Sprintf("Parse error: %v\n\nPlease fix.", err)},
				)
				continue
			}
			return nil, fmt.Errorf("parse failed: %w", err)
		}

		if validate != nil {
			if ok, errors := validate(result.Code); !ok {
				lastErr = fmt.Errorf("compile errors: %v", errors)
				if attempt < maxRetries {
					log.Warn("compile error (attempt %d/%d): %v", attempt+1, maxRetries+1, errors)
					messages = append(messages,
						Message{Role: "assistant", Content: content},
						Message{Role: "user", Content: fmt.Sprintf("Compilation errors:\n%s\n\nFix and provide corrected code.", strings.Join(errors, "\n"))},
					)
					continue
				}
				return nil, fmt.Errorf("compilation failed: %w", lastErr)
			}
		}

		return result, nil
	}

	return nil, lastErr
}

func systemPrompt() string {
	return fmt.Sprintf(`You are an expert sketch artist using SketchLang.

%s

Create a COMPLETE, EXTREMELY DETAILED sketch.

FORMAT:
<title>SKETCH TITLE</title>
<summary>Description of the sketch.</summary>
<code>
# Complete SketchLang code
</code>

REQUIREMENTS:
- Complete sketch with full detail
- Meaningful anchor point names
- Vector math: let pos : vec = (center of shape) + (offset_x, offset_y)
- Use "center of" for derived positions
- NO dot notation (vec.x is invalid)
- NO variable reassignment
- NO for loops or while loops
- trace = precise lines, draw = organic, scribble = textured
- Use dashes for shading
- Types: number, vec, sketch`, LangSpec)
}

func parseResponse(content string) (*SketchResult, error) {
	code := extractCode(content)
	if code == "" {
		return nil, fmt.Errorf("no <code> block found")
	}

	title := extractTag(content, "title")
	if title == "" {
		return nil, fmt.Errorf("no <title> found")
	}

	return &SketchResult{
		Code:    code,
		Title:   title,
		Summary: extractTag(content, "summary"),
	}, nil
}

func extractCode(content string) string {
	if m := regexp.MustCompile(`(?s)<code>(.*?)</code>`).FindStringSubmatch(content); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile("(?s)```(?:sketchlang)?\\s*\\n(.*?)\\n```").FindStringSubmatch(content); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractTag(content, tag string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?si)<%s>(.*?)</%s>`, tag, tag))
	if m := re.FindStringSubmatch(content); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}