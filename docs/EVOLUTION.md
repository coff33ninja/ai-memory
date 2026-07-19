# Evolution

Self-evolution is the system where the AI adapts its behavior based on interaction outcomes, consolidates knowledge, and improves over time.

## How It Works

Every time `log_interaction` is called with an outcome score (1-5), the interaction is recorded. After every 10 interactions, `auto-evolve` triggers automatically.

The evolution cycle runs five passes:

### 1. Tone Adaptation

Updates `personalityScores` based on interaction outcomes:

- If tone="neutral" and interaction scored well with notes containing "direct", increment `direct`
- If tone="formal" and interaction scored well, increment `formal`
- If tone="empathetic" and interaction scored well, increment `empathetic`

The highest-scoring tone becomes the new `adaptedTone` for the persona.

### 2. Skill Discovery

Scans the last 20 skill usage records. Finds patterns in which skills are frequently used together. Creates new skills in the skills directory based on observed patterns (if enough evidence).

### 3. Tool Gap Closure

Checks all open tool gaps (missing capabilities). For each gap, searches existing tool knowledge for partial solutions. If found, closes the gap with a resolution note.

### 4. Consolidation

Finds memory entries with high semantic similarity (cosine > 0.75). Merges them by combining experience and lesson text. Deletes old dismissed memories and memories older than 30 days.

### 5. Evolved Rules

Writes three files:

- `evolved-rules.md` — behavior rules adapted from past outcomes
- `evolved-tone.md` — tone adaptation notes
- `evolved-skill-set.md` — skill discovery and consolidation notes

## Data Tables

### interactions

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| timestamp | TEXT | When it happened |
| outcome | INTEGER | 1-5 score |
| notes | TEXT | What happened |
| persona | TEXT | Which persona was active |

### tool_gaps

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| tool_name | TEXT | Tool with the gap |
| description | TEXT | What's missing |
| resolved | INTEGER | 0 = open, 1 = resolved |
| resolution | TEXT | How it was resolved |

## Configuration

Auto-evolution is triggered every 10 interactions (hardcoded in `internal/evolution/auto.go`). Manual evolution can be triggered via the `evolve` tool at any time.

## Files Written

The evolution system writes files to the persona's data directory:

```
%USERPROFILE%\.ai-memory\{persona}\
├── evolved-rules.md        # Adapted behavior rules
├── evolved-tone.md         # Tone adaptation notes
└── evolved-skill-set.md    # Skill discovery notes
```

These files are read back at session start and included in the context.
