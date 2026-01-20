package artist

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"sketch-studio/tools/llm"
	"sketch-studio/tools/logger"
)

// Artist handles the creative process of generating sketches via LLM
type Artist struct {
	client llm.Client
	log    *logger.Logger
	lang   string // the SketchLang specification
}

// New creates a new Artist
func New(client llm.Client, langSpec string, log *logger.Logger) *Artist {
	if log == nil {
		log = logger.Default()
	}
	return &Artist{
		client: client,
		log:    log.WithPrefix("artist"),
		lang:   langSpec,
	}
}

// SketchPlan is the initial plan from the artist
type SketchPlan struct {
	Title       string
	Summary     string
	Subject     string
	Perspective string
	Style       string
	Metadata    map[string]string
	Sections    []SectionPlan
	ContourCode string
}

// SectionPlan describes a section of the sketch
type SectionPlan struct {
	Title       string
	Description string
	Neighbors   []string
}

// Plan creates an initial sketch plan from a description
func (a *Artist) Plan(ctx context.Context, description string) (*SketchPlan, *llm.Response, error) {
	done := a.log.Step("Creating sketch plan")
	defer done()

	systemPrompt := a.buildPlanSystemPrompt()
	userPrompt := fmt.Sprintf(`Create a sketch plan for the following request:

%s

Remember to:
1. Provide a detailed summary and metadata
2. Break the sketch into logical sections 
3. Create initial contour SketchLang code that outlines the main shapes
4. Use comments in your SketchLang code to label sections`, description)

	resp, err := a.client.CompleteWithRetry(ctx, systemPrompt, []llm.Message{
		{Role: "user", Content: userPrompt},
	}, 3)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get plan from LLM: %w", err)
	}

	a.log.Tokens(resp.InputTokens, resp.OutputTokens)

	plan, err := a.parsePlanResponse(resp.Content)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to parse plan response: %w", err)
	}

	return plan, resp, nil
}

// ExpandSection has a sub-artist expand a section with more detail
func (a *Artist) ExpandSection(ctx context.Context, plan *SketchPlan, section SectionPlan, existingCode string) (string, *llm.Response, error) {
	done := a.log.Step(fmt.Sprintf("Expanding section: %s", section.Title))
	defer done()

	systemPrompt := a.buildExpandSystemPrompt()

	// Build context about neighbors
	neighborContext := ""
	if len(section.Neighbors) > 0 {
		neighborContext = fmt.Sprintf("\nThis section connects to: %s. Ensure your strokes align at boundaries.", strings.Join(section.Neighbors, ", "))
	}

	userPrompt := fmt.Sprintf(`Expand this section of the sketch with detailed SketchLang code.

SKETCH OVERVIEW:
Title: %s
Summary: %s
Style: %s
Perspective: %s

SECTION TO EXPAND:
Title: %s
Description: %s%s

EXISTING CONTOUR CODE (for reference - do NOT repeat this, only add new code):
%s

Write NEW SketchLang code for this section only. Add strokes for details, shading with dashes, and fine details. Your code will be APPENDED to the existing code, so:
- Do NOT redeclare existing variables
- Use unique variable names (prefix with section name, e.g., %s_point1)
- Reference existing variables if needed for alignment`,
		plan.Title, plan.Summary, plan.Style, plan.Perspective,
		section.Title, section.Description, neighborContext,
		existingCode,
		strings.ReplaceAll(strings.ToLower(section.Title), " ", "_"))

	resp, err := a.client.CompleteWithRetry(ctx, systemPrompt, []llm.Message{
		{Role: "user", Content: userPrompt},
	}, 3)
	if err != nil {
		return "", nil, fmt.Errorf("failed to expand section: %w", err)
	}

	a.log.Tokens(resp.InputTokens, resp.OutputTokens)

	code := extractSketchCode(resp.Content)
	if code == "" {
		return "", resp, fmt.Errorf("no SketchLang code found in response")
	}

	return code, resp, nil
}

func (a *Artist) buildPlanSystemPrompt() string {
	return fmt.Sprintf(`You are a expert artist creating sketches using SketchLang, a domain-specific language for pen plotter artwork.

Here is the SketchLang specification:

%s

When given a sketch request, you will:
1. Create a detailed plan with title, summary, subject, perspective, and style
2. Define logical sections of the sketch with titles, descriptions, and neighbor relationships
3. Write initial contour SketchLang code that outlines the major shapes

Format your response as follows:

<plan>
<title>Your Sketch Title</title>
<summary>A detailed description of what the sketch depicts</summary>
<subject>The main subject matter</subject>
<perspective>The viewing angle/perspective</perspective>
<style>The artistic style (minimalist, detailed, expressive, etc.)</style>
<metadata>
key1: value1
key2: value2
</metadata>
<sections>
<section>
<title>Section Name</title>
<description>What this section contains</description>
<neighbors>Neighbor1, Neighbor2</neighbors>
</section>
</sections>
</plan>

<contours>
# Your SketchLang code here
# Use comments to mark section boundaries
</contours>

Important notes:
- Coordinates are in mm, typical canvas is 200x200mm
- Use comments liberally to label sections
- Keep contours simple but well defined - details will be added later
- Think about how sections connect at boundaries

CRITICAL SketchLang constraints (violations will cause compilation errors):
- NO dot notation (vec.x, vec.y) - this does NOT exist
- NO variable reassignment - each variable can only be assigned once
- NO functions or loops - only let bindings and render commands
- NO Duplicate Strokes please.
- Variables must be declared with type: let name : type = value
- Valid types are: number, vec, sketch
- Vectors are created with parentheses: (x, y)
- Use unique variable names (e.g., prefix with section name)`, a.lang)
}

func (a *Artist) buildExpandSystemPrompt() string {
	return fmt.Sprintf(`You are a detail-focused artist adding depth to sketch sections using SketchLang.

Here is the SketchLang specification:

%s

Your task is to expand a section with detailed strokes. You should:
1. Add detail strokes for textures and features
2. Use dashes for shading and tone
3. Maintain consistency with the overall style
4. Ensure strokes align with neighboring sections at boundaries

Provide your SketchLang code inside <code> tags:

<code>
# Your detailed SketchLang code
</code>

Important:
- Do NOT repeat the existing contour code - only write NEW code for this section
- Use trace for clean lines, draw for hand-drawn feel, scribble for sketchy areas
- Dashes orient based on nearby strokes (flow field)
- Use descriptive comments
- Prefix variable names with section name to avoid conflicts (e.g., arm_base, arm_stroke1)

CRITICAL SketchLang constraints (violations will cause compilation errors):
- NO dot notation (vec.x, vec.y) - this does NOT exist
- NO variable reassignment - each variable can only be assigned once
- NO functions or loops - only let bindings and render commands
- Variables must be declared with type: let name : type = value
- Valid types are: number, vec, sketch`, a.lang)
}

func (a *Artist) parsePlanResponse(content string) (*SketchPlan, error) {
	plan := &SketchPlan{
		Metadata: make(map[string]string),
	}

	// Extract plan section
	planMatch := regexp.MustCompile(`(?s)<plan>(.*?)</plan>`).FindStringSubmatch(content)
	if len(planMatch) < 2 {
		return nil, fmt.Errorf("no <plan> section found")
	}
	planContent := planMatch[1]

	// Extract fields
	plan.Title = extractTag(planContent, "title")
	plan.Summary = extractTag(planContent, "summary")
	plan.Subject = extractTag(planContent, "subject")
	plan.Perspective = extractTag(planContent, "perspective")
	plan.Style = extractTag(planContent, "style")

	// Parse metadata
	metaContent := extractTag(planContent, "metadata")
	for _, line := range strings.Split(metaContent, "\n") {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key != "" && val != "" {
				plan.Metadata[key] = val
			}
		}
	}

	// Parse sections
	sectionsContent := extractTag(planContent, "sections")
	sectionMatches := regexp.MustCompile(`(?s)<section>(.*?)</section>`).FindAllStringSubmatch(sectionsContent, -1)
	for _, match := range sectionMatches {
		if len(match) < 2 {
			continue
		}
		sec := SectionPlan{
			Title:       extractTag(match[1], "title"),
			Description: extractTag(match[1], "description"),
		}
		neighborsStr := extractTag(match[1], "neighbors")
		if neighborsStr != "" {
			for _, n := range strings.Split(neighborsStr, ",") {
				n = strings.TrimSpace(n)
				if n != "" {
					sec.Neighbors = append(sec.Neighbors, n)
				}
			}
		}
		plan.Sections = append(plan.Sections, sec)
	}

	// Extract contours
	plan.ContourCode = extractSketchCode(content)

	if plan.Title == "" {
		return nil, fmt.Errorf("no title found in plan")
	}

	return plan, nil
}

func extractTag(content, tag string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?s)<%s>(.*?)</%s>`, tag, tag))
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func extractSketchCode(content string) string {
	// Try <contours> tag first
	if code := extractTag(content, "contours"); code != "" {
		return code
	}
	// Then try <code> tag
	if code := extractTag(content, "code"); code != "" {
		return code
	}
	// Finally try code blocks
	re := regexp.MustCompile("(?s)```(?:sketchlang)?\\s*\\n(.*?)\\n```")
	match := re.FindStringSubmatch(content)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}