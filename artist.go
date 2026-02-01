package main

import (
	"fmt"
	"regexp"
	"strings"
	"sort"
)

const (
	maxRetries      = 2
	maxRetriesLocal = 6
)


func Generate(client LLMClient, desc, imageURL string, log *Logger) (*SketchResult, error) {
	prompt := promptFrom(desc, imageURL)
	return generate(client, prompt, desc, imageURL, nil, nil, 0, log)
}

func promptFrom(desc, imageURL string) string {
	p := "Using the custom sketchlang dsl, create a sketch of " + desc + "."
	if imageURL != "" {
		p = fmt.Sprintf("Create an extremely detailed sketch of the image at: %s", imageURL)
	}
	return p
}

func generate(client LLMClient, prompt, desc, imageURL string, prevCode *string, prevErr *SketchError, attempt int, log *Logger) (*SketchResult, error) {
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
		msg = retryPrompt(*prevCode, *prevErr, desc, imageURL)
		sys = retrySystemPrompt()
	} else {
		msg = promptFrom(desc, imageURL) + " Dont forget <title> and <code> tags!"
		sys = initialSystemPrompt()
	}

	if client.IsLocal() {
		msg += " Plan your work out in detail in a <plan> section. Dont overuse transformations if you dont need them."
	}

	messages := []Message{{Role: "user", Content: msg}}
	log.Info("attempt %d", attempt + 1)
	log.Debug("user message: %s", msg)
	resp, err := client.Complete(sys, messages)
	if err != nil {
		return nil, err
	}

	resp = sanitize(resp)
	log.Debug("\n-----------------------------------------------------------\n%s\n-----------------------------------------------------------", resp)
	result, parseErr := parseResponse(resp, prevCode)
	if parseErr != nil {
		log.Warn("parse error (attempt %d): %v\n", attempt+1, parseErr)
		return generate(client, prompt, desc, imageURL, nil, &SketchError{Message: parseErr.Error()}, attempt+1, log)
	}

	skerr, err := ValidateSketch(result.Code, log)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	if skerr != nil {
		log.Warn("compile error (attempt %d): line %d col %d: %s\n", attempt+1, skerr.Line, skerr.Column, skerr.Message)
		return generate(client, prompt, desc, imageURL, &result.Code, skerr, attempt+1, log)
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

func retryPrompt(code string, skerr SketchError, desc, imageURL string) string {
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
	msg += "Remember: " + promptFrom(desc, imageURL)

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
	re := regexp.MustCompile(`(?s)<edit\s+line="(\d+)(?:-(\d+))?">(.*?)</edit>`)
	matches := re.FindAllStringSubmatch(response, -1)
	if len(matches) == 0 {
		return code
	}

	// parse all edits
	type edit struct {
		start, end int
		content    string
	}
	edits := make([]edit, 0, len(matches))
	for _, m := range matches {
		var e edit
		fmt.Sscanf(m[1], "%d", &e.start)
		if m[2] != "" {
			fmt.Sscanf(m[2], "%d", &e.end)
		} else {
			e.end = e.start
		}
		e.start--
		e.content = strings.TrimSpace(m[3])
		edits = append(edits, e)
	}

	// sort by start line descending
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].start > edits[j].start
	})

	lines := strings.Split(code, "\n")
	for _, e := range edits {
		if e.start >= 0 && e.end <= len(lines) {
			replacement := strings.Split(e.content, "\n")
			lines = append(lines[:e.start], append(replacement, lines[e.end:]...)...)
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
