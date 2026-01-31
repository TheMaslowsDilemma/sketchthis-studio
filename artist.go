package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const (
	maxRetries      = 1
	maxRetriesLocal = 6
)

func Generate(client LLMClient, desc, imageURL string, log *Logger) (*SketchResult, error) {
	prompt := desc
	if imageURL != "" {
		prompt = fmt.Sprintf("Create an extremely detailed sketch of the image at: %s", imageURL)
	}
	return generate(client, prompt, imageURL, nil, nil, 0, log)
}

func generate(client LLMClient, prompt, imageURL string, prevCode *string, prevErr *SketchError, attempt int, log *Logger) (*SketchResult, error) {
	max := maxRetries
	if client.IsLocal() {
		max = maxRetriesLocal
	}
	if attempt > max {
		return nil, fmt.Errorf("failed after %d attempts", attempt)
	}

	var msg string
	var sys string
	if prevErr != nil && prevCode != nil {
		msg = retryPrompt(*prevCode, *prevErr, imageURL)
		sys = retrySystemPrompt()
	} else if prevErr != nil {
		msg = fmt.Sprintf("Error: %s\n\nYou MUST include <title> and <code> tags.", prevErr.Message)
		sys = initialSystemPrompt()
	} else {
		msg = prompt
		sys = initialSystemPrompt()
	}

	messages := []Message{{Role: "user", Content: msg}}
	log.Debug("attempt %d\n", attempt+1)
	resp, err := client.Complete(sys, messages)
	if err != nil {
		return nil, err
	}

	resp = sanitize(resp)

	result, parseErr := parseResponse(resp, prevCode)
	if parseErr != nil {
		log.Warn("parse error (attempt %d): %v\n", attempt+1, parseErr)
		return generate(client, prompt, imageURL, nil, &SketchError{Message: parseErr.Error()}, attempt+1, log)
	}

	skerr, err := ValidateSketch(result.Code, log)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if skerr != nil {
		log.Warn("compile error (attempt %d): line %d col %d: %s\n", attempt+1, skerr.Line, skerr.Column, skerr.Message)
		return generate(client, prompt, imageURL, &result.Code, skerr, attempt+1, log)
	}

	return result, nil
}

func initialSystemPrompt() string {
	return fmt.Sprintf(`You are an expert sketch artist using SketchLang.

%s

FORMAT:
<title>SKETCH TITLE</title>
<code>
# Complete SketchLang code
</code>
`, LangSpec)
}

func retrySystemPrompt() string {
	return fmt.Sprintf(`You are an expert sketch artist using SketchLang.

%s

You are fixing a compile error. Respond with EITHER:

1. A single line fix:
<edit line="N">corrected line here</edit>

2. A multi-line fix:
<edit line="N-M">
corrected
lines
here
</edit>

3. Or full replacement:
<title>SAME TITLE</title>
<code>
# Complete corrected code
</code>

DO NOT simplify or reduce the sketch. Fix ONLY the error.
`, LangSpec)
}

func retryPrompt(code string, skerr SketchError, imageURL string) string {
	lines := strings.Split(code, "\n")
	start := max(0, skerr.Line-4)
	end := min(len(lines), skerr.Line+3)

	var ctx strings.Builder
	for i := start; i < end; i++ {
		marker := "  "
		if i+1 == skerr.Line {
			marker = "> "
		}
		ctx.WriteString(fmt.Sprintf("%s%3d: %s\n", marker, i+1, lines[i]))
	}

	msg := fmt.Sprintf(`Compile error at line %d, column %d: %s

%s
Fix ONLY this error. Do NOT simplify the sketch.`, skerr.Line, skerr.Column, skerr.Message, ctx.String())

	if imageURL != "" {
		msg += fmt.Sprintf("\n\nRemember: you are sketching the image at %s", imageURL)
	}

	return msg
}

func parseResponse(content string, prevCode *string) (*SketchResult, error) {
	if prevCode != nil {
		if edited := applyEdits(*prevCode, content); edited != *prevCode {
			title := extractTag(content, "title")
			if title == "" {
				title = "Untitled"
			}
			return &SketchResult{Code: edited, Title: title}, nil
		}
	}

	title := extractTag(content, "title")
	if title == "" {
		return nil, fmt.Errorf("no <title> found")
	}

	code := extractCode(content)
	if code == "" {
		return nil, fmt.Errorf("no <code> block found")
	}

	return &SketchResult{Code: code, Title: title}, nil
}

func applyEdits(code, response string) string {
	re := regexp.MustCompile(`(?s)<edit\s+line="(\d+)(?:-(\d+))?">\s*(.*?)\s*</edit>`)
	matches := re.FindAllStringSubmatch(response, -1)
	if len(matches) == 0 {
		return code
	}

	lines := strings.Split(code, "\n")
	for _, m := range matches {
		var start, end int
		fmt.Sscanf(m[1], "%d", &start)
		if m[2] != "" {
			fmt.Sscanf(m[2], "%d", &end)
		} else {
			end = start
		}
		start--
		replacement := strings.Split(m[3], "\n")

		if start >= 0 && end <= len(lines) {
			lines = append(lines[:start], append(replacement, lines[end:]...)...)
		}
	}
	return strings.Join(lines, "\n")
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