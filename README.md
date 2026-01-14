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
    A[Image Description] --> B[Orchestrator]
    B --> C1[Sub Artist 1]
    B --> C2[Sub-Artist 2]
    B --> C3[Sub-Artist 3]
    B --> Cn[... Sub-Artist n]
    C1 --> D[Neighbor Negotiation]
    C2 --> D
    C3 --> D
    Cn --> D
    D --> E[Compile Sub-Works]
    E --> F[Critique of SVG]
    F --> G{Changes Needed?}
    G -->|Yes| B
    G -->|No| H[Merge Works]
    H --> I[Output: Sketchlang]
    J[Logging] -.-> B & C1 & C2 & C3 & Cn & D & E & F & H
```
