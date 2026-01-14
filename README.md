## Sketch Studio

### Overview

This project implements a studio in GoLang where an input image description is processed by an LLM (artist) to iteratively produce a final sketch in SketchLang. The process involves planning, dividing into sections, critiquing, compiling, and merging sub-works with transparency via logging.


### Components

**Artist**: Plans steps, divides work into tiles/sections recursively, delegates details, ensures coherency via negotiation.
**Critic**: Reviews specific sections, provides feedback (sections may align with artist's named divisions).
**Compilation**: Verifies sub-pieces align; checks SketchLang compilation errors before merging.
**Merging**: Negotiates changes between neighboring sub-artists; compiles sub-works first to avoid upstream issues.
**Transparency**: Logs top-level plans, draft SVGs, critic responses, compilation errors, token usage (if available), and step timings.
Prompt Design: Key prompts for top-level artist (detailed description + tiling) and sub-artists; easily accessible.


### Program Flow

```mermaid
flowchart TD
    A[Input: Image Description] --> B[Orchestrator Artist: top level scene description & tile segmentation]
    B --> C1[Sub-Artist Tile 1: Plan & Generate Draft]
    B --> C2[Sub-Artist Tile 2: Plan & Generate Draft]
    B --> C3[Sub-Artist Tile 3: Plan & Generate Draft]
    B --> Cn[... More Sub-Artists]
    C1 --> D[Negotiation: Communicate & Align Neighboring Tiles]
    C2 --> D
    C3 --> D
    Cn --> D
    D --> E[Compile Sub-Works: Verify SketchLang & Fix Errors]
    E --> F[Critic: Review Sections & Provide Feedback]
    F --> G{Changes Needed?}
    G -->|Yes| B
    G -->|No| H[Merge Works: Negotiate Final Changes & Compile]
    H --> I[Output: Final SketchLang Sketch]
    J[Logging: Plans, Drafts SVG, Critiques, Errors, Tokens, Timings] -.-> B & C1 & C2 & C3 & Cn & D & E & F & H
```
