package main

import (
	"fmt"
	"regexp"
	"strings"
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
	return generate(client, prompt, nil, nil, 0, log)
}

func generate(client LLMClient, prompt string, prevCode *string, prevErr *SketchError, attempt int, log *Logger) (*SketchResult, error) {
	max := maxRetries
	if client.IsLocal() {
		max = maxRetriesLocal
	}
	if attempt > max {
		return nil, fmt.Errorf("failed after %d attempts", attempt)
	}

	var msg string
	if prevErr != nil && prevCode != nil {
		msg = retryPrompt(*prevCode, *prevErr)
	} else if prevErr != nil {
		msg = fmt.Sprintf("Error: %s\n\nYou MUST include <title> and <code> tags.", prevErr.Message)
	} else {
		msg = prompt
	}

	messages := []Message{{Role: "user", Content: msg}}
	log.Debug("generating response")
	resp, err := client.Complete(systemPrompt(), messages)
	if err != nil {
		return nil, err
	}
	log.Debug("resp: %s", resp)
	result, parseErr := parseResponse(resp)
	if parseErr != nil {
		log.Warn("parse error (attempt %d): %v\n", attempt+1, parseErr)
		return generate(client, prompt, nil, &SketchError{Message: parseErr.Error()}, attempt+1, log)
	}

	skerr, err := ValidateSketch(result.Code, log)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if skerr != nil {
		log.Warn("compile error (attempt %d): line %d col %d: %s\n", attempt+1, skerr.Line, skerr.Column, skerr.Message)
		return generate(client, prompt, &result.Code, skerr, attempt+1, log)
	}

	return result, nil
}

func initialPrompt() string {
	return fmt.Sprintf(`You are an expert sketch artist using SketchLang.

%s

FORMAT:
<title>SKETCH TITLE</title>
<code>
# Complete SketchLang code
</code>
`, LangSpec)
}

// pulls in context around the error and gives that context
// with the error formatted, then asks for the fix.
func retryPrompt(code string, skerr SketchError) string {
	lines := strings.Split(code, "\n")
	start := max(0, skerr.Line-3)
	end := min(len(lines), skerr.Line+2)
	context := strings.Join(lines[start:end], "\n")

	return fmt.Sprintf(`Compile error at line %d, column %d: %s

Context:
%s

Provide fix as <edit line="%d">corrected line</edit> or full <code> block.`,
		skerr.Line, skerr.Column, skerr.Message, context, skerr.Line)
}

func parseResponse(content string) (*SketchResult, error) {
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
	re := regexp.MustCompile(`(?s)<edit line="(\d+)(?:-(\d+))?">(.*?)</edit>`)
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
		replacement := strings.Split(strings.TrimSpace(m[3]), "\n")

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