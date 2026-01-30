package main

import (
	"fmt"
	"regexp"
	"strings"
)

const maxRetries = 3

func Generate(client LLMClient, description, imageURL string, pos, size Vec2, log *Logger) (*SketchResult, error) {
	messages := []Message{{Role: "user", Content: description}}
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, err := client.Complete(systemPrompt(), messages)
		if err != nil {
			return nil, err
		}

		result, err := parseResponse(content)
		if err != nil {
			log.Info("\n---------------------\ncontent\n%s\n---------------------\n", content)
			lastErr = err
			if attempt < maxRetries {
				log.Warn("parse error (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
				retry := fmt.Sprintf("Parse error: %v\n\nYOU MUST INCLUDE <title> and <code> tags.", err)
				if imageURL != "" {
					retry += fmt.Sprintf("\n\nRemember your goal: sketch the image at this URL: %s", imageURL)
				}
				messages = append(messages,
					Message{Role: "assistant", Content: content},
					Message{Role: "user", Content: retry},
				)
				continue
			}
			return nil, fmt.Errorf("parse failed after %d attempts: %w", maxRetries+1, err)
		}

		if _, _, err := Compile(result.Code, "_validate", pos, size, log); err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Warn("compile error (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
				retry := fmt.Sprintf("Compilation failed:\n%s\n\nYour code:\n```\n%s\n```\n\nFix the code.", err, result.Code)
				if imageURL != "" {
					retry += fmt.Sprintf("\n\nRemember your goal: sketch the image at this URL: %s", imageURL)
				}
				messages = append(messages,
					Message{Role: "assistant", Content: content},
					Message{Role: "user", Content: retry},
				)
				continue
			}
			return nil, fmt.Errorf("compilation failed after %d attempts: %w", maxRetries+1, lastErr)
		}

		return result, nil
	}

	return nil, lastErr
}

func Generate(client LLMClient, description, imageURL string, pos, size Vec2, log *Logger) (*SketchResult, error) {
	/*
		----
		fallback is configurable. local models will be more liberal with attempts to self-right.
		----

		1. you have an image description and a langauge for which to express that image in detail.
		2. the language is compilable and can point out where things went wrong.


	*/
	var (
		ms []Message /* messages btw artist and llm */
	)
}

func systemPrompt() string {
	return fmt.Sprintf(`You are an expert sketch artist using SketchLang.

%s

Create a COMPLETE, EXTREMELY DETAILED sketch.

FORMAT:
<title>SKETCH TITLE</title>
<code>
# Complete SketchLang code
</code>
`, LangSpec)
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