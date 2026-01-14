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
%%{init: {
  'theme': 'base',
  'themeVariables': {
    'fontSize': '14px',
    'fontFamily': 'Inter, sans-serif',
    'primaryColor': '#6366f1',
    'primaryTextColor': '#ffffff',
    'primaryBorderColor': '#4f46e5',
    'secondaryColor': '#f1f5f9',
    'secondaryTextColor': '#334155',
    'tertiaryColor': '#ecfdf5',
    'lineColor': '#94a3b8',
    'textColor': '#334155'
  }
}}%%

flowchart TB
    subgraph input [" "]
        A[Image Description]
    end

    subgraph orchestration [" "]
        B[Orchestrator]
    end

    subgraph artists ["  Sub-Artists  "]
        direction LR
        C1[Artist 1]
        C2[Artist 2]
        C3[Artist 3]
        Cn[Artist n]
    end

    subgraph processing ["  Processing Pipeline  "]
        D["  Neighbor Negotiation  "]
        E["  Compile Sub-Works  "]
        F["  Critique SVG   "]
        G{"  Changes Needed?  "}
    end

    subgraph output [" "]
        H["  Merge Works  "]
        I["  Output: SketchLang  "]
    end

    J["\tLogs\t"]

    A --> B
    B --> C1 & C2 & C3 & Cn
    C1 & C2 & C3 & Cn --> D
    D --> E
    E --> F
    F --> G
    G -->|Yes| B
    G -->|No| H
    H --> I

    J -.->|monitors| B
    J -.->|monitors| D
    J -.->|monitors| F

    style A fill:#f8fafc,stroke:#cbd5e1,color:#334155
    style B fill:#6366f1,stroke:#4f46e5,color:#ffffff
    style C1 fill:#8b5cf6,stroke:#7c3aed,color:#ffffff
    style C2 fill:#8b5cf6,stroke:#7c3aed,color:#ffffff
    style C3 fill:#8b5cf6,stroke:#7c3aed,color:#ffffff
    style Cn fill:#8b5cf6,stroke:#7c3aed,color:#ffffff
    style D fill:#06b6d4,stroke:#0891b2,color:#ffffff
    style E fill:#14b8a6,stroke:#0d9488,color:#ffffff
    style F fill:#f59e0b,stroke:#d97706,color:#ffffff
    style G fill:#f1f5f9,stroke:#94a3b8,color:#334155
    style H fill:#22c55e,stroke:#16a34a,color:#ffffff
    style I fill:#10b981,stroke:#059669,color:#ffffff
    style J fill:#f1f5f9,stroke:#cbd5e1,color:#64748b

    style input fill:transparent,stroke:transparent
    style orchestration fill:transparent,stroke:transparent
    style artists fill:#f8fafc,stroke:#e2e8f0,stroke-width:1px,rx:8
    style processing fill:#f8fafc,stroke:#e2e8f0,stroke-width:1px,rx:8
    style output fill:transparent,stroke:transparent
```
