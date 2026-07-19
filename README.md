# ai-memory

MCP server that gives AI assistants persistent memory, semantic search, skill recall, and self-evolution across sessions.

## What It Does

ai-memory runs as a child process (stdio transport) and exposes four systems to the AI client:

**Memory** — The AI stores experiences and lessons in SQLite. Each entry goes through a review cycle: store → review → apply or dismiss. Over time, this builds a knowledge base of what worked, what failed, and why.

**Skills** — 51+ AI skills cloned from a curated repository. Indexed in SQLite with ONNX embeddings so the AI can semantic-search for "how to debug a race condition" and get the right skill even if the exact words don't match.

**Multi-Persona** — Separate memory databases per persona. A "debugger" persona learns about crash analysis. A "writer" persona learns about documentation patterns. They share a common skills index and can share memories via scope.

**Self-Evolution** — The AI tracks interaction outcomes (scored 1-5), consolidates similar memories, adapts its own tone and skill set, discovers new skills from usage patterns, and builds a tool manual from experience. After every 10 interactions, it evolves.

## Prerequisites

- Go 1.26+
- [Zig](https://ziglang.org/) — used as the C cross-compiler for CGO (sqlite3, onnxruntime)
- Git — for cloning the skills repository

## Build

```powershell
# Clone
git clone https://github.com/coff33ninja/ai-memory
cd ai-memory

# Build
.\scripts\build.ps1

# Output: ai-memory-server.exe (~17 MB)
```

The build script handles Zig CC setup, CGO flags, icon embedding, and version injection. See `scripts/build.ps1` for the exact flags.

## Install

```powershell
# From source (requires Go + Zig)
.\scripts\install.ps1 -UseZig

# Binary goes to: %LOCALAPPDATA%\ai-memory\ai-memory-server.exe
# Config at: %USERPROFILE%\.config\ai-memory\config.json
```

## Configure in opencode.json

```json
{
  "mcpServers": {
    "ai-memory": {
      "command": "C:\\Users\\YOU\\AppData\\Local\\ai-memory\\ai-memory-server.exe"
    }
  }
}
```

On first run, the server downloads ONNX Runtime and the embedding model to `%APPDATA%\ai-memory\lib\`. No manual setup required.

## MCP Interface

### Tools (40)

| Category | Tool | Purpose |
|----------|------|---------|
| **Memory** | `store` | Save an experience + lesson with tags |
| | `review` | List pending memories |
| | `apply` | Mark memory as applied |
| | `dismiss` | Mark memory as dismissed |
| | `status` | Memory and skill counts |
| | `search` | Semantic search across memories + skills |
| | `search_memories` | Semantic search memories only |
| | `search_skills` | Semantic search skills only |
| | `reindex` | Rebuild all embeddings |
| **Skills** | `skills_sync` | Git pull the skills repo |
| | `skills_search` | Keyword search skills |
| | `skills_index` | Re-index skills to SQLite |
| | `store_skill_usage` | Record which skills were used together |
| | `list_skill_usage` | View skill usage patterns |
| **Persona** | `onboard` | Create a new persona |
| | `list_personas` | List all personas |
| | `switch_persona` | Show active persona |
| | `delete_persona` | Delete a persona |
| **Evolution** | `log_interaction` | Record interaction outcome (score 1-5) |
| | `evolve` | Trigger full evolution cycle |
| | `consolidate` | Merge similar memories, prune old |
| | `discover_skills` | Find new skills from usage patterns |
| | `evolution_history` | View evolution log |
| | `get_evolved_rules` | Get adapted behavior rules |
| | `interaction_stats` | Tone and skill performance scores |
| **Tool Knowledge** | `log_tool_knowledge` | Build a manual for a tool |
| | `log_tool_recipe` | Save multi-step tool patterns |
| | `get_tool_knowledge` | Read what you know about a tool |
| | `list_tool_knowledge` | List all tool knowledge |
| | `get_tool_recipes` | Get recipes for a tool |
| | `record_recipe_outcome` | Track recipe success/failure |
| **Tool Gaps** | `log_tool_gap` | Record a missing capability |
| | `list_tool_gaps` | View unresolved gaps |
| | `resolve_tool_gap` | Mark gap as resolved |
| **Error Tracking** | `log_tool_error` | Log an MCP tool error |
| | `list_tool_errors` | View logged errors |
| | `resolve_tool_error` | Mark error as resolved |
| **MCP Registry** | `register_mcp_server` | Register server capabilities |
| | `get_mcp_server` | Get server info |
| | `list_mcp_servers` | List known servers |

### Resources (12)

| URI | Description |
|-----|-------------|
| `memory://memories` | All memory entries |
| `memory://skills` | All indexed skills |
| `memory://summary` | Combined stats overview |
| `memory://all` | Everything: memories + skills + stats |
| `skills://catalog` | Skill catalog with descriptions |
| `skills://usage` | Skill usage patterns and pairings |
| `context://project` | Detected project type + relevant skills |
| `context://startup` | Full startup context for session init |
| `persona://active` | Current persona details |
| `persona://all` | All personas |
| `evolution://stats` | Interaction stats and performance |
| `evolution://rules` | Adapted behavior rules |

### Prompts (7)

| Prompt | Purpose |
|--------|---------|
| `memory` | Session init: pending memories + relevant skills |
| `reflect` | End-of-session reflection guide |
| `context-inject` | Mandatory search-before-answer behavior |
| `skill-usage-recorder` | Guide for recording skill usage |
| `persona-startup` | Persona-aware session init |
| `evolution-loop` | Full evolution cycle instructions |
| `mcp-error-handling` | Error logging and reporting guide |

## Data Storage

```
%USERPROFILE%\.ai-memory\
├── {persona-name}/memory.db    # Per-persona SQLite (memories, embeddings, tool knowledge)
├── shared/memory.db            # Shared memories across personas
├── personas.json               # Persona registry (name, identity, tone, skills)
├── skills/ai-skills/           # Cloned skill repository (51+ skills)
└── lib/                        # Auto-downloaded ONNX Runtime + model
    ├── onnxruntime.dll
    ├── all-MiniLM-L6-v2.onnx
    └── tokenizer/
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_MEMORY_DIR` | `~/.ai-memory` | Override data directory |

## CGO

ai-memory requires CGO for sqlite3 and onnxruntime. The `CGO_TRIGGER` file in the repo root enables CGO builds. Remove it to disable (not recommended — the server won't work without CGO).

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts/build.ps1` | Build with Zig CC + CGO + icon embedding |
| `scripts/lint.ps1` | go vet + build check |
| `scripts/test.ps1` | Full test suite |
| `scripts/install.ps1` | Clone, build, install to AppData |
| `scripts/push-and-release.ps1` | Version bump, commit, tag, push, wait for release |
| `scripts/gen-icons.ps1` | Generate Windows icon resource |
| `scripts/gen-tools-doc.go` | Auto-generate tools.md from source |

## CI/CD

| Workflow | Trigger | What It Does |
|----------|---------|--------------|
| `ci.yml` | Push/PR to main | Lint, build, upload artifact |
| `release.yml` | Tag push (`v*`) | Build release binary, create GitHub release |
| `auto-tag.yml` | VERSION file change | Auto-tag from VERSION file |
| `mod-maintenance.yml` | Weekly (Monday 06:00 UTC) | Update Go dependencies |

## Docs

- [Architecture](docs/ARCHITECTURE.md) — package structure, data flows, database schema
- [Evolution](docs/EVOLUTION.md) — self-evolution system design and history
- [Quick Reference](docs/QUICK-REFERENCE.md) — storage paths, tool cheatsheet
- [ADR-002](docs/adr/ADR-002-go-rewrite.md) — decision record for Go rewrite
- [Changelog](docs/meta/CHANGELOG.md) — version history
- [Tools](docs/reference/tools.md) — auto-generated tool catalog
