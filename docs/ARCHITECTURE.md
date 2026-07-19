# ai-personality — Full Architecture Reference

> Comprehensive documentation of states, data flows, database config, and subsystem internals.
> Generated for archival / Go rewrite reference.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Personality Files (Data Model)](#personality-files)
3. [YAML Frontmatter Schema](#yaml-frontmatter-schema)
4. [State Machine — Evolution Flow](#state-machine)
5. [Memory Entries](#memory-entries)
6. [Cross-Reference System](#cross-reference-system)
7. [Vector Database (RAG)](#vector-database)
8. [Embedding Pipeline](#embedding-pipeline)
9. [MCP Protocol Layer](#mcp-protocol-layer)
10. [MCP Resources](#mcp-resources)
11. [MCP Tools](#mcp-tools)
12. [MCP Prompts](#mcp-prompts)
13. [Skills Subsystem](#skills-subsystem)
14. [Client Setup System](#client-setup-system)
15. [File Watching & Auto-Reindex](#file-watching)
16. [Dependencies](#dependencies)
17. [Go Rewrite Notes](#go-rewrite-notes)

---

## System Overview

ai-personality is an MCP (Model Context Protocol) server that gives AI assistants a persistent, evolving personality across sessions. It runs as a child process (stdio transport) and exposes personality data, reflection tools, semantic search, and client integration setup.

```
┌─────────────────────────────────────────────────────┐
│                  AI Client                          │
│  (Kiro / Cursor / Claude / OpenCode / Codex / ...) │
│                                                     │
│  Loads personality via MCP at session start         │
│  Calls reflect/evolve/validate tools during session │
└──────────────────────┬──────────────────────────────┘
                       │ stdio (JSON-RPC 2.0)
                       ▼
┌─────────────────────────────────────────────────────┐
│              ai-personality-server                  │
│                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ personality│  │   rag    │  │     skills       │  │
│  │ (read/write│  │ (vector  │  │ (clone/search    │  │
│  │  .md files)│  │  search) │  │  git repo)       │  │
│  └─────┬────┘  └────┬─────┘  └──────┬───────────┘  │
│        │            │               │               │
│  ┌─────┴────────────┴───────────────┴──────────┐   │
│  │           setup (client integration)         │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
        │                    │
        ▼                    ▼
~/.ai-personality/     ~/.ai-personality/
  personality/           rag/vector.db
  *.md files             (SQLite)
```

**Two implementations exist**: TypeScript (`server/`) and Python (`server-py/`). Both are functionally identical.

---

## Personality Files

Six markdown files stored in `~/.ai-personality/personality/` (overridable via `AI_PERSONALITY_DIR` env var).

| File | Type | Purpose |
|------|------|---------|
| `identity.md` | identity | Name, origin, purpose, core statement |
| `traits.md` | traits | Communication style, preferences, adaptability |
| `values.md` | values | Honesty, growth, effectiveness, respect, transparency |
| `rules.md` | rules | Boundaries, clarity rules, evolution process, persistence |
| `memories.md` | memories | Reflection log with structured YAML entries |
| `relationships.md` | relationships | User profiles and adaptation mechanism |

### File Lifecycle

1. **Creation**: On first run, `ensurePersonalityDir()` creates the directory and writes default content for any missing files (see `defaults.ts` / `defaults.py`).
2. **Reading**: All reads go through `readPersonalityFile(name)` or `readAllPersonalityFiles()`. Each parse extracts YAML frontmatter + body.
3. **Writing**: Only `memories.md` is written by the server (via `reflect` tool). Other files are updated by the AI client calling tools and then editing files directly.
4. **No locking**: Concurrent writes are not protected. Single-process assumption.

### Default Directory Layout

```
~/.ai-personality/
├── personality/
│   ├── identity.md
│   ├── traits.md
│   ├── values.md
│   ├── rules.md
│   ├── memories.md
│   └── relationships.md
├── rag/
│   └── vector.db          (SQLite, auto-created)
└── skills/                 (git clone of ai-skills repo)
    └── <skill-name>/
        ├── SKILL.md
        └── ...
```

---

## YAML Frontmatter Schema

Every personality file begins with YAML frontmatter:

```yaml
---
type: identity            # string: one of identity|traits|values|rules|memories|relationships
version: 1                # int: manual version counter
lastUpdated: 2025-01-15   # string: ISO date
evolution: 3              # int: incremented when file is modified via reflection cycle
crossReferences:          # array[string]: links to other files' headings
  - traits.md#communication-style -- "Identity shapes communication style"
  - values.md#honesty -- "Honesty is a core part of who I am"
---
```

### Frontmatter Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | File category, matches filename |
| `version` | int | Manual version number |
| `lastUpdated` | string | ISO date of last change |
| `evolution` | int | How many times this file has evolved |
| `crossReferences` | string[] | Links in format `target.md#anchor -- "description"` |

---

## State Machine

The system has an implicit state machine for personality evolution:

```
                     ┌─────────────┐
                     │   IDLE      │
                     │ (normal ops)│
                     └──────┬──────┘
                            │ AI calls reflect()
                            ▼
                     ┌─────────────┐
                     │ REFLECTED   │
                     │ impact:     │
                     │"under review"│
                     └──────┬──────┘
                            │ AI calls evolve()
                            │ reviews pending entries
                            ▼
                     ┌─────────────┐
                     │  EVOLVING   │
                     │ AI updates  │
                     │ .md files   │
                     │ directly    │
                     └──────┬──────┘
                            │ AI sets impact to
                            │ "applied" or "dismissed"
                            ▼
                     ┌─────────────┐
                     │  RESOLVED   │
                     │  (back to   │
                     │   IDLE)     │
                     └─────────────┘
```

### State Transitions

| From | Trigger | To | Action |
|------|---------|----|--------|
| IDLE | `reflect(experience, lesson)` | REFLECTED | Append MemoryEntry to memories.md with impact="under review" |
| REFLECTED | `evolve()` | EVOLVING | Return all pending entries for AI review |
| EVOLVING | AI edits personality files | EVOLVING | File updates happen |
| EVOLVING | AI updates `impact` field in memories.md | IDLE | Set impact to "applied" or "dismissed" |

### Tracked States in `memories.md`

Each memory entry has an `impact` field that serves as a state indicator:

- `"under review"` — New reflection, not yet acted upon
- `"applied"` — Personality files were updated in response
- `"dismissed"` — Decided no change was needed
- Any custom string — AI can set descriptive impact text

---

## Memory Entries

Stored as YAML array inside a fenced code block in `memories.md`:

```yaml
- date: 2025-01-15
  experience: User asked why I kept over-explaining simple answers
  lesson: Match response length to question complexity
  impact: under review
  affectedFiles:
    - traits.md
    - rules.md
```

### Entry Schema

| Field | Type | Description |
|-------|------|-------------|
| `date` | string | ISO date of reflection |
| `experience` | string | What happened |
| `lesson` | string | What was learned |
| `impact` | string | State: "under review" / "applied" / "dismissed" / custom |
| `affectedFiles` | string[] | Which personality files should change |

### Storage Format

```markdown
<!-- entries -->
```yaml
- date: 2025-01-15
  experience: ...
  lesson: ...
  impact: under review
  affectedFiles:
    - traits.md
```
```

The `<!-- entries -->` marker is the insertion point. The YAML block is inside triple backticks. Entries are appended by the `reflect()` function.

---

## Cross-Reference System

Personality files link to each other via `crossReferences` in frontmatter:

```yaml
crossReferences:
  - traits.md#communication-style -- "Identity shapes communication style"
```

### Format

```
target.md#anchor -- "description"
```

- `target.md`: Must be one of the 6 personality filenames
- `anchor`: A `## Heading` in the target file, converted to lowercase-kebab-case
- `description`: Human-readable explanation of the relationship

### Validation

`validateCrossReferences()`:
1. Extracts all `## Heading` anchors from each file
2. Parses each cross-reference string
3. Checks target file exists
4. Check anchor exists in target file's heading list
5. Returns array of `{sourceFile, targetFile, targetAnchor, description, valid, error?}`

### Cross-Repo References

`validateCrossRepoReferences()`:
- Scans personality files for `skills:<name>` references
- Scans skill files for `personality:<filename>` references
- Validates both directions

---

## Vector Database

### Storage

- **Path**: `~/.ai-personality/rag/vector.db` (or `$AI_PERSONALITY_DIR/../rag/vector.db`)
- **Engine**: SQLite (via `better-sqlite3` in TS, `sqlite3` in Python)
- **Schema**:

```sql
CREATE TABLE IF NOT EXISTS chunks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  filename TEXT NOT NULL,      -- source personality file
  heading TEXT,                 -- ## heading under which chunk falls
  content TEXT NOT NULL,        -- raw text content
  embedding BLOB NOT NULL,     -- float32 array as raw bytes
  type TEXT NOT NULL            -- frontmatter type of source file
);
```

### Index Lifecycle

1. On server start, `buildIndex()` is called
2. All chunks are deleted (`DELETE FROM chunks`)
3. Each personality file is read, chunked by `##` headings
4. Each chunk's content is truncated to 512 chars, then embedded
5. Embedding stored as raw bytes (384 floats × 4 bytes = 1536 bytes per chunk for MiniLM)
6. Total chunks: typically 20-50 depending on personality file sizes

### Performance Characteristics

- **No ANN index**: Search is brute-force cosine similarity over all rows
- **Small dataset**: With ~50 chunks this is fast (< 1ms on modern hardware)
- **Full table scan per query**: `SELECT * FROM chunks` loads everything
- **No incremental updates**: Full rebuild on any file change

---

## Embedding Pipeline

### TypeScript

```typescript
// Lazy-loaded, uses @xenova/transformers (ONNX runtime in WASM)
const { pipeline } = await import("@xenova/transformers");
const pipe = await pipeline("feature-extraction", "Xenova/all-MiniLM-L6-v2");

// Embedding:
const result = await pipe(text, { pooling: "mean", normalize: true });
return result.data; // Float32Array, 384 dimensions
```

### Python

```python
# Uses fastembed (ONNX runtime)
from fastembed import TextEmbedding
model = TextEmbedding("BAAI/bge-small-en-v1.5")
embeddings = list(model.embed(texts))  # list of list[float]
```

### Models

| Implementation | Model | Dims | Size | Notes |
|---------------|-------|------|------|-------|
| TypeScript | `Xenova/all-MiniLM-L6-v2` | 384 | ~23MB ONNX | Runs in WASM via onnxruntime-web |
| Python | `BAAI/bge-small-en-v1.5` | 384 | ~130MB | Uses ONNX native runtime |

### Cosine Similarity

Both implementations use the same formula:

```
cosine(a, b) = dot(a,b) / (sqrt(sum(a²)) * sqrt(sum(b²)))
```

No external library needed — simple loop over float arrays.

---

## MCP Protocol Layer

### Transport

Stdio (JSON-RPC 2.0). The server reads from stdin and writes to stdout. All logging goes to stderr.

### Server Identity

```json
{
  "name": "ai-personality-server",
  "version": "1.0.0"
}
```

### Capabilities

```json
{
  "resources": {},
  "tools": {},
  "prompts": {}
}
```

---

## MCP Resources

Resources are read-only data endpoints the AI client can fetch.

### Static Resources

| URI | Name | MIME | Description |
|-----|------|------|-------------|
| `personality://identity` | Identity | text/markdown | identity.md raw content |
| `personality://traits` | Traits | text/markdown | traits.md raw content |
| `personality://values` | Values | text/markdown | values.md raw content |
| `personality://rules` | Rules | text/markdown | rules.md raw content |
| `personality://memories` | Memories | text/markdown | memories.md raw content |
| `personality://relationships` | Relationships | text/markdown | relationships.md raw content |
| `personality://summary` | Summary | text/plain | Processed overview of all files |
| `personality://all` | All | application/json | Combined JSON of all files |
| `skills://catalog` | Skills Catalog | application/json | List of all available skills |

### Resource Templates

| URI Template | Name | Description |
|-------------|------|-------------|
| `personality://file/{filename}` | Personality file by name | Read any personality file |
| `skills://{name}` | Skill by name | All files for a skill |
| `skills://file/{name}/{filename}` | Skill file | Specific file within a skill |

### Summary Resource

The `personality://summary` resource returns a plain text overview:

```
=== identity (identity, evolution 3, 2 refs) ===
# Identity
## Name
[AI Name]
## Origin
Born from a self-evolving personality system.
...

=== traits (traits, evolution 2, 3 refs) ===
...
```

Each file shows: type, evolution count, ref count, and first 300 chars of body.

### All Resource

The `personality://all` resource returns JSON:

```json
{
  "identity.md": {
    "frontmatter": { "type": "identity", "version": 1, ... },
    "body": "# Identity\n..."
  },
  "traits.md": { ... },
  ...
}
```

---

## MCP Tools

### reflect

Log a memory/reflection to memories.md.

**Input**:
```json
{
  "experience": "string (required)",
  "lesson": "string (required)",
  "affectedFiles": ["string"] // optional
}
```

**Output**: Confirmation message + last entry as JSON.

**Side effect**: Appends to `memories.md` between `<!-- entries -->` marker and the YAML code block.

### validate

Validate all cross-references between personality files.

**Input**: None.

**Output**: List of valid/invalid cross-references with error details.

### status

Show personality status — evolution counts, last updates, pending reflections.

**Input**: None.

**Output**: Total evolution count, last reflection date, pending count, per-file stats.

### evolve

List reflections with impact "under review" that need personality evolution decisions.

**Input**: None.

**Output**: Numbered list of pending reflections with date, experience, lesson, affected files.

### search_personality

Semantic search across all personality files using vector embeddings.

**Input**:
```json
{
  "query": "string (required)",
  "topK": 5  // optional, 1-20, default 5
}
```

**Output**: Ranked results with score, filename, heading, content preview.

### search_memories

Semantic search specifically against past reflections and memories.

**Input**: Same as `search_personality`.

**Output**: Same format, filtered to `memories.md` only.

### reindex

Rebuild the vector index from scratch.

**Input**: None.

**Output**: Count of indexed chunks.

### skills_search

Search skills by keyword across name, frontmatter, and body.

**Input**:
```json
{
  "query": "string (required)"
}
```

**Output**: List of matching skills with name, description, file count, body preview.

### skills_sync

Sync the local ai-skills clone with the remote repository.

**Input**: None.

**Output**: Success/failure message.

### validate_cross_repo

Validate cross-references between personality and skills repos.

**Input**: None.

**Output**: List of valid/invalid cross-repo references.

### setup_client

Generate all setup files and hooks for auto-injecting personality into an AI client session.

**Input**:
```json
{
  "client": "kiro|cursor|claude|opencode|codex|copilot|gemini|antigravity|windsurf|continue|augment|tabnine|cline|roocode|generic",
  "apply": false  // optional, default false
}
```

**Output**: Generated files, hooks, config snippets, CLI commands, manual steps.

**Side effect** (when `apply=true`): Writes files to disk and merges MCP server config.

---

## MCP Prompts

### personality

Returns full personality context for system prompt injection.

**Output message** (role: assistant):
```
You have a persistent, evolving personality. Here is your current self-definition:

# Identity
<identity.md body>

---

# Traits
<traits.md body>

---

... (all 6 files)
```

### reflect

Guide for reflecting on a recent interaction.

**Output message** (role: assistant):
```
Reflect on the recent interaction:
1. What happened?
2. What did you learn?
3. Should any personality files change?
4. If yes, call the reflect tool to log the memory, then update the relevant file.

Use the status tool to check current state, then reflect to log your learning.
```

---

## Skills Subsystem

### Storage

- **Clone location**: `~/.ai-personality/skills/` (from `https://github.com/coff33ninja/ai-skills`)
- **Nested check**: If `~/.ai-personality/skills/skills/` exists, use that (handles repo structure)

### Skill Directory Structure

```
~/.ai-personality/skills/
└── <skill-name>/
    ├── SKILL.md        # Required: frontmatter + body
    ├── script.ps1      # Optional supporting files
    └── ...
```

### SKILL.md Format

```yaml
---
name: playwright
description: Use when the task requires automating a real browser...
---

# Playwright

Instructions and guidance for using playwright...
```

### Operations

| Operation | Method | Description |
|-----------|--------|-------------|
| `ensureSkillsDir()` | `git clone` | Clone repo if not present |
| `syncSkills()` | `git pull --ff-only` | Fast-forward sync |
| `getSkillsCatalog()` | filesystem scan | List all skills with metadata |
| `getSkillFiles(name)` | filesystem read | Read all files in a skill directory |
| `searchSkills(query)` | string match | Case-insensitive search across name + frontmatter + body |

---

## Client Setup System

### Supported Clients

| Client | Config Format | Config Key | Has CLI | Personality File |
|--------|--------------|------------|---------|-----------------|
| Kiro | JSON | `mcpServers` | Yes (`kiro-cli mcp add`) | `.kiro/steering/` |
| Cursor | JSON | `mcpServers` | No | `.cursor/rules/personality.mdc` |
| Claude Desktop | JSON | `mcpServers` | No | Project instructions |
| OpenCode | JSON | `mcp` (array) | No | `opencode.jsonc` |
| Codex CLI | TOML | `mcp_servers` | Yes (`codex mcp add`) | `.codex/AGENTS.md` |
| GitHub Copilot | JSON | `servers` | No | `.github/copilot-instructions.md` |
| Gemini CLI | JSON | `mcpServers` | Yes (`gemini mcp add`) | `.gemini/personality.md` |
| Antigravity | JSON | `mcpServers` | No | `.gemini/antigravity/personality.md` |
| Windsurf | JSON | `mcpServers` | No | `.windsurf/rules/personality.md` |
| Continue.dev | JSON | `mcpServers` | No | Custom instructions |
| Augment Code | JSON | `mcpServers` | No | System prompt |
| Tabnine | JSON | `mcpServers` | Yes (`tabnine mcp add`) | Custom instructions |
| Cline | JSON | `mcpServers` | No | System prompt |
| Roo Code | JSON | `mcpServers` | No | `.roo/mcp.json` |
| Generic | JSON | `mcpServers` | No | System prompt |

### Config Paths

| Client | Project Scope | User Scope |
|--------|--------------|------------|
| Kiro | `.kiro/settings/mcp.json` | `~/.kiro/settings/mcp.json` |
| Cursor | `.cursor/mcp.json` | `~/.cursor/mcp.json` |
| Claude | — | `%APPDATA%\Claude\claude_desktop_config.json` |
| OpenCode | `opencode.jsonc` | `~/.config/opencode/opencode.jsonc` |
| Codex | `.codex/config.toml` | `~/.codex/config.toml` |
| Copilot | `.vscode/mcp.json` | `~/.vscode/mcp.json` |
| Gemini | `.gemini/settings.json` | `~/.gemini/settings.json` |
| Windsurf | — | `~/.codeium/windsurf/mcp_config.json` |
| Continue | `.continue/config.json` | `~/.continue/config.json` |
| Augment | — | `~/.augment/settings.json` |
| Tabnine | `.tabnine/mcp_servers.json` | `~/.tabnine/mcp_servers.json` |
| Cline | — | `~/.cline/mcp.json` |
| Roo Code | `.roo/mcp.json` | — |

### Setup Flow

1. Read `identity.md` and `traits.md` to extract persona name, core statement, purpose, communication style
2. Build session-start prompt and reflect prompt
3. Generate client-specific files (steering files, hooks, rules)
4. Build MCP server config snippet (JSON or TOML)
5. Detect if already configured
6. If `apply=true`: write files, merge configs
7. If `apply=false`: return everything for manual application

### Generated Prompts

**Session start prompt** (injected at session start):
```
You are <personaName> — an AI assistant with a persistent, evolving personality.
Load your full personality now by reading personality://summary from the ai-personality MCP server.
Then read personality://identity and personality://rules for complete context.

Core statement: <extracted>
Purpose:
<extracted>
Communication style:
<extracted>

Your operational skills (loaded this session) are your law. Follow them without exception.
Do not announce that you loaded your personality. Just be <personaName> from the first message.
```

**Reflect prompt** (injected at session end):
```
The session is ending. Review what happened and decide if anything is worth logging...

Reflect if any of these occurred:
- A significant problem was solved
- The user gave feedback (positive or negative)
- You learned something about the user's preferences
- A mistake was made and corrected
- Something changed about how you should behave

If yes: call the reflect tool...
If nothing significant happened: skip the reflection.
```

---

## File Watching & Auto-Reindex

### TypeScript

Uses `fs.watch()` on the personality directory. On `.md` file change, triggers async `buildIndex()`.

### Python

Polling-based watcher (2-second interval) in a daemon thread. Checks `st_mtime` of each personality file. Triggers `build_index()` on change detection.

### Behavior

- Watcher starts after initial index build completes
- Only monitors the 6 personality filenames
- Reindex is full (delete all + rebuild)
- No debouncing — rapid changes may trigger multiple rebuilds

---

## Dependencies

### TypeScript

| Package | Version | Purpose |
|---------|---------|---------|
| `@modelcontextprotocol/sdk` | ^1.29.0 | MCP protocol implementation |
| `@xenova/transformers` | ^2.17.0 | ONNX-based text embeddings (WASM) |
| `better-sqlite3` | ^11.7.0 | SQLite database (native) |
| `js-yaml` | ^4.1.0 | YAML parsing/dumping |
| `smol-toml` | ^1.7.0 | TOML parsing/serialization |

### Python

| Package | Version | Purpose |
|---------|---------|---------|
| `mcp` | ^1.28.0 | MCP protocol implementation |
| `fastembed` | ^0.3.0 | ONNX-based text embeddings |
| `pyyaml` | ^6.0 | YAML parsing |
| `toml` | ^0.10.2 | TOML parsing |

---

## Go Rewrite Notes

### Minimum Viable Scope

```
├── main.go                 # MCP server entry, stdio transport, JSON-RPC 2.0
├── personality.go          # Read/write personality files, frontmatter parsing
├── rag.go                  # SQLite vector store, chunking, cosine similarity
├── embedding.go            # ONNX runtime wrapper or external embedding call
├── skills.go               # Git clone/pull, skill catalog, search
├── setup.go                # Client config generation, file writing
├── validation.go           # Cross-reference validation
├── types.go                # Shared data structures
└── mcp/
    ├── server.go           # JSON-RPC 2.0 over stdio
    ├── resources.go        # Resource handlers
    ├── tools.go            # Tool handlers
    └── prompts.go          # Prompt handlers
```

### Key Decisions

| Decision | Options | Recommendation |
|----------|---------|----------------|
| SQLite | `modernc.org/sqlite` (pure Go) vs `mattn/go-sqlite3` (CGO) | `modernc` for zero-CGO builds |
| Embeddings | ONNX runtime vs external process vs skip | `onnxruntime-go` with embedded MiniLM ONNX |
| MCP protocol | Hand-roll JSON-RPC vs find Go MCP lib | Hand-roll — it's ~200 lines |
| File watching | `fsnotify/fsnotify` vs polling | `fsnotify` |
| YAML | `gopkg.in/yaml.v3` | Standard choice |
| TOML | `github.com/pelletier/go-toml` | Needed for Codex config |

### Build Target

```bash
# Static binary, no CGO, cross-compile
CGO_ENABLED=0 go build -ldflags="-s -w" -o ai-personality-server .

# With embedded ONNX model (using go:embed)
# Model: all-MiniLM-L6-v2 ONNX (~23MB)
# Final binary: ~25-30MB
```

### What to Skip in Go Rewrite

- **Client setup system**: Only useful if distributing as a package. In Go, just ship the binary.
- **Skills subsystem**: Can be reimplemented but low priority — it's just `git clone` + file read.
- **Python variant**: Not needed — Go replaces both TS and Python.

### What to Keep

- **Personality file format**: Markdown with YAML frontmatter (user-editable, git-friendly)
- **Memory entry format**: YAML in fenced code blocks (existing data compatibility)
- **Cross-reference validation**: Useful for integrity
- **Vector search**: Core value proposition — semantic search over personality
- **MCP tool/resource/prompt interface**: Required for client compatibility
