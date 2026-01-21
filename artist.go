package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"sketch-studio/tools/llm"
	"sketch-studio/tools/logger"
)

const (
	maxRetries       = 2
	maxContinuations = 3
	defaultMaxTokens = 16384
)

// Artist handles LLM interactions for sketch generation.
type Artist struct {
	client llm.Client
	log    *logger.Logger
	lang   string
}

// NewArtist creates a new Artist instance.
func NewArtist(client llm.Client, langSpec string, log *logger.Logger) *Artist {
	if log == nil {
		log = logger.Default()
	}
	return &Artist{
		client: client,
		log:    log.WithPrefix("artist"),
		lang:   langSpec,
	}
}

// CreateSketch generates a complete, detailed sketch in a single request.
func (a *Artist) CreateSketch(ctx context.Context, description string) (*SketchResult, *llm.Response, error) {
	return a.createSketch(ctx, description, nil)
}

// CreateSketchWithValidation generates a sketch and validates it compiles.
func (a *Artist) CreateSketchWithValidation(ctx context.Context, description string, validate func(string) (bool, []string)) (*SketchResult, *llm.Response, error) {
	return a.createSketch(ctx, description, validate)
}

func (a *Artist) createSketch(ctx context.Context, description string, validate func(string) (bool, []string)) (*SketchResult, *llm.Response, error) {
	done := a.log.Step("Creating sketch")
	defer done()

	messages := []llm.Message{{Role: "user", Content: description}}
	opts := &llm.RequestOptions{MaxTokens: defaultMaxTokens}

	var lastResp *llm.Response
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Get response, handling continuations if truncated
		content, resp, err := a.completeWithContinuation(ctx, messages, opts)
		if err != nil {
			return nil, nil, err
		}
		lastResp = resp

		result, err := a.parseResponse(content)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				a.log.Warn("Parse error (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
				messages = a.appendRetry(messages, content, fmt.Sprintf("Parse error: %v\n\nPlease fix and include <title>, <summary>, <metadata>, and <code> tags.", err))
				continue
			}
			return nil, lastResp, fmt.Errorf("parse failed after %d attempts: %w", maxRetries+1, err)
		}

		if validate != nil {
			if ok, errors := validate(result.Code); !ok {
				lastErr = fmt.Errorf("compilation errors: %v", errors)
				if attempt < maxRetries {
					a.log.Warn("Compile error (attempt %d/%d): %v", attempt+1, maxRetries+1, errors)
					messages = a.appendRetry(messages, content, fmt.Sprintf("Compilation errors:\n%s\n\nPlease fix and provide corrected code.", strings.Join(errors, "\n")))
					continue
				}
				return nil, lastResp, fmt.Errorf("compilation failed after %d attempts: %w", maxRetries+1, lastErr)
			}
		}

		return result, resp, nil
	}

	return nil, lastResp, lastErr
}

// completeWithContinuation handles responses that hit the token limit
func (a *Artist) completeWithContinuation(ctx context.Context, messages []llm.Message, opts *llm.RequestOptions) (string, *llm.Response, error) {
	resp, err := a.client.CompleteWithRetry(ctx, a.buildSystemPrompt(), messages, 2, opts)
	if err != nil {
		return "", nil, fmt.Errorf("LLM request failed: %w", err)
	}
	a.log.Tokens(resp.InputTokens, resp.OutputTokens)

	content := resp.Content

	// If not truncated, return as-is
	if !resp.WasTruncated() {
		return content, resp, nil
	}

	a.log.Warn("Response truncated, requesting continuation...")

	// Request continuations until complete or max reached
	for i := 0; i < maxContinuations; i++ {
		contMessages := append(messages,
			llm.Message{Role: "assistant", Content: content},
			llm.Message{Role: "user", Content: "Continue exactly where you left off. Do not repeat any code."},
		)

		contResp, err := a.client.CompleteWithRetry(ctx, a.buildSystemPrompt(), contMessages, 2, opts)
		if err != nil {
			return "", nil, fmt.Errorf("continuation request failed: %w", err)
		}
		a.log.Tokens(contResp.InputTokens, contResp.OutputTokens)

		content += contResp.Content
		resp = contResp

		if !contResp.WasTruncated() {
			a.log.Info("Continuation complete")
			break
		}

		if i == maxContinuations-1 {
			a.log.Warn("Max continuations reached, response may be incomplete")
		}
	}

	return content, resp, nil
}

func (a *Artist) buildSystemPrompt() string {
	return fmt.Sprintf(`You are an expert sketch artist using SketchLang.

%s

Create a COMPLETE, EXTREMELY DETAILED sketch. Include all details, shading, and textures.

FORMAT:
<title>SKETCH TITLE</title>
<summary>Description of the sketch and subject placement.</summary>
<metadata>
<subject>Main subject</subject>
<perspective>View angle</perspective>
<style>Art style</style>
</metadata>
<code>
# Complete SketchLang code with ALL details
</code>

REQUIREMENTS:
- Complete sketch with full detail in one response
- Meaningful anchor point names throughout
- Vector math: let pos : vec = (center of shape) + (offset_x, offset_y)
- Use "center of" for derived positions
- NO dot notation (vec.x is invalid)
- NO variable reassignment
- trace = precise lines, draw = organic, scribble = textured
- Use dashes for shading
- Types: number, vec, sketch`, a.lang)
}

func (a *Artist) appendRetry(messages []llm.Message, assistantContent, userFeedback string) []llm.Message {
	return append(messages,
		llm.Message{Role: "assistant", Content: assistantContent},
		llm.Message{Role: "user", Content: userFeedback},
	)
}

func (a *Artist) parseResponse(content string) (*SketchResult, error) {
	code := extractCode(content)
	if code == "" {
		return nil, fmt.Errorf("no <code> block found")
	}

	title := extractTag(content, "title")
	if title == "" {
		title = extractCommentTag(code, "title")
	}
	if title == "" {
		return nil, fmt.Errorf("no <title> found")
	}

	result := &SketchResult{
		Code:     code,
		Title:    title,
		Summary:  firstNonEmpty(extractTag(content, "summary"), extractCommentTag(code, "summary")),
		Metadata: make(map[string]string),
	}

	if meta := extractTag(content, "metadata"); meta != "" {
		result.Metadata["subject"] = extractTag(meta, "subject")
		result.Metadata["perspective"] = extractTag(meta, "perspective")
		result.Metadata["style"] = extractTag(meta, "style")
	}

	return result, nil
}

// --- Parsing helpers ---

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

func extractCommentTag(content, tag string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?i)#\s*<%s>(.+?)</%s>`, tag, tag))
	if m := re.FindStringSubmatch(content); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}

	var result []string
	var inTag bool
	openRe := regexp.MustCompile(fmt.Sprintf(`(?i)#\s*<%s>`, tag))
	closeRe := regexp.MustCompile(fmt.Sprintf(`(?i)#?\s*</%s>`, tag))

	for _, line := range strings.Split(content, "\n") {
		if !inTag && openRe.MatchString(line) {
			inTag = true
			if after := openRe.Split(line, 2); len(after) > 1 && strings.TrimSpace(after[1]) != "" {
				result = append(result, strings.TrimSpace(after[1]))
			}
			continue
		}
		if inTag {
			if closeRe.MatchString(line) {
				break
			}
			cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
			if cleaned != "" {
				result = append(result, cleaned)
			}
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}