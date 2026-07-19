# ai-memory

MCP server that gives AI assistants persistent memory, semantic search, skill recall, and self-evolution across sessions.

## What It Does

ai-memory runs as a child process (stdio transport) and exposes four systems to the AI client:

**Memory** — The AI stores experiences and lessons in SQLite. Each entry goes through a review cycle: store → review → apply or dismiss. Over time, this builds a knowledge base of what worked, what failed, and why.

**Skills** — 51+ AI skills cloned from a curated repository. Indexed in SQLite with ONNX embeddings so the AI can semantic-search for "how to debug a race condition" and get the right skill even if the exact words don't match.

**Multi-Persona** — Separate memory databases per persona. A "debugger" persona learns about crash analysis. A "writer" persona learns about documentation patterns. They share a common skills index and can share memories via scope. Each persona can have a greeting keyword (e.g. "Akeno") — when the user says it, the AI switches to that persona automatically.

**Self-Evolution** — The AI tracks interaction outcomes (scored 1-5), consolidates similar memories, adapts its own tone and skill set, discovers new skills from usage patterns, and builds a tool manual from experience. After every 10 interactions, it evolves.

**User Profiles** — The AI builds a profile on the user (name, hobbies, interests, preferences) incrementally from conversations. Each field has a source and confidence score. Profiles are included in startup context so every session knows who it's talking to.

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
# Data at: %USERPROFILE%\.ai-memory\
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

## Getting Started

### First Run

When the server starts for the first time:

1. **ONNX Runtime** is downloaded to `%APPDATA%\ai-memory\lib\onnxruntime.dll` if not present
2. **Embedding model** (`all-MiniLM-L6-v2.onnx`) is downloaded to `%APPDATA%\ai-memory\lib\`
3. **Personas directory** is created at `%USERPROFILE%\.ai-memory\`
4. **Default persona** is auto-created (name: `default`, identity: "General-purpose assistant", tone: `direct`)
5. **14 common MCP servers** are seeded into the registry (go-mcp-computer-use, playwright, filesystem, github, brave-search, postgres, sqlite, slack, memory, puppeteer, google-drive, notion, fetch, sentry)
6. **Skills repository** is cloned from `https://github.com/coff33ninja/ai-skills` on first `skills_sync` call

The `context://startup` resource and `persona-startup` prompt detect the default persona and include onboarding instructions. When the AI sees these, it should call `onboard` to create a proper persona before doing anything else.

### Setting Up a Persona

Personas are the entry point. Each persona gets its own memory database, but shares the skills index and can share memories across personas via `scope: "shared"`.

On first run, a `default` persona is auto-created. The AI detects this and should call `onboard` to create a real persona. If you're talking to the AI for the first time, say something like "let's get you set up" — it should see the onboarding instructions and call `onboard` itself.

**Create your first persona:**

```
onboard(
  name: "assistant",
  identity: "General-purpose coding assistant",
  tone: "direct",
  description: "Helps with all kinds of software engineering tasks",
  greeting: "Akeno",
  skills: ["debugging-and-error-recovery", "code-review"]
)
```

The `greeting` field is optional — when set, the AI will switch to this persona when the user says that keyword (e.g. "hello Akeno").

This:
- Creates `%USERPROFILE%\.ai-memory\assistant\memory.db` with the memory schema
- Adds the persona to `personas.json`
- Sets it as the active persona (first persona created is always active)
- Stores a welcome memory in the persona's database
- Stores a shared memory about the persona being created

**List personas:**

```
list_personas()
```

Shows all personas with `*` marking the active one.

**Switch persona:**

```
switch_persona(name: "writer")
```

Switches the active persona. All subsequent memory operations go to the new persona's database. Skills are shared across all personas.

**Delete a persona:**

```
delete_persona(name: "old-persona")
```

Renamed to `.old-persona.deleted` (not truly deleted, just hidden).

### User Profiles

The AI learns about you over time and stores it in your profile. This data persists across sessions and is included in startup context.

**Store a profile field:**

```
store_user_profile(
  field: "name",
  value: "Dragohn",
  source: "stated",
  confidence: 1.0
)
```

**Get a profile field:**

```
get_user_profile(field: "name")
```

**List all profile fields:**

```
list_user_profile()
```

Fields are unique — storing the same field again updates the value and increases confidence. Sources: `stated` (user said it directly), `inferred` (AI figured it out), `conversation` (learned during chat).

### How Memories Work

**Store a memory:**

```
store(
  experience: "ORT session.Run panics with 'wrong thread' if called from a different OS thread than the one that created the session",
  lesson: "Always pin ONNX Runtime sessions to the OS thread that created them using runtime.LockOSThread()",
  tags: ["ort", "threading", "crash"]
)
```

Returns: `Memory 1 stored [under review]. 1 total, 1 pending.`

**Review pending memories:**

```
review()
```

Lists all memories with `impact = "under review"`.

**Apply or dismiss:**

```
apply(id: 1)   # Mark as "applied" — lesson was incorporated
dismiss(id: 1) # Mark as "dismissed" — no change needed
```

**Search memories (semantic):**

```
search(query: "how to fix ORT crash", topK: 3)
```

Returns results with cosine similarity scores across memories + skills.

### How Skills Work

**Sync skills from GitHub:**

```
skills_sync()
```

Clones or pulls `https://github.com/coff33ninja/ai-skills` to `%USERPROFILE%\.ai-memory\skills\ai-skills\`. First call clones, subsequent calls pull.

**Index skills to SQLite:**

```
skills_index()
```

Reads all `SKILL.md` files, generates embeddings, stores in the `skills` table.

**Search skills (keyword, fast):**

```
skills_search(query: "debugging")
```

**Search skills (semantic, via embeddings):**

```
search(query: "automate a browser", type: "skill")
```

Returns skills ranked by cosine similarity to the query embedding.

**Record skill usage:**

```
store_skill_usage(
  skill: "debugging-and-error-recovery",
  context: "Fixed ORT crash caused by thread affinity",
  with_skills: "anti-phantom-symbols, self-validate",
  outcome: "effective"
)
```

Builds a graph of which skills work well together for similar tasks.

### How Self-Evolution Works

**Log an interaction:**

```
log_interaction(
  outcome: 5,  # 1-5 scale: 1=failed, 5=excellent
  notes: "Successfully fixed the crash by pinning ORT to the correct thread"
)
```

**Auto-evolution triggers every 10 interactions.** It runs five passes:

1. **Tone adaptation** — tracks which tone (direct/formal/empathetic) scored best, updates `adaptedTone`
2. **Skill discovery** — finds patterns in skill usage, creates new skills from observed combinations
3. **Tool gap closure** — maps open tool gaps to existing knowledge, closes resolved gaps
4. **Consolidation** — merges memories with cosine > 0.75, deletes old dismissed memories
5. **Evolved rules** — writes `evolved-rules.md`, `evolved-tone.md`, `evolved-skill-set.md` to the persona directory

**Manual evolution:**

```
evolve()
```

**View evolution history:**

```
evolution_history()
```

### Recording Tool Knowledge

When you learn how a tool works, build a manual entry:

```
log_tool_knowledge(
  tool_name: "computer_use_click",
  knowledge_type: "manual",
  title: "Click precision",
  description: "Click at exact coordinates. Double-click is 2 rapid single clicks.",
  content: "Use get_cursor_position to verify target coordinates before clicking. For menus, use click_menu_item instead.",
  outcome: "success"
)
```

**Save multi-step recipes:**

```
log_tool_recipe(
  tool_name: "computer_use_chain",
  title: "Login sequence",
  description: "Automated login to a web app",
  steps: '[{"tool":"click","args":{"x":100,"y":200}},{"tool":"type","args":{"text":"user"}},{"tool":"click","args":{"x":300,"y":400}},{"tool":"type","args":{"text":"pass"}}]'
)
```

### MCP Server Registry

Register MCP servers you use so the error tracker knows their capabilities:

```
register_mcp_server(
  name: "playwright",
  source: "github.com/microsoft/playwright-mcp",
  creator: "microsoft",
  repo_url: "https://github.com/microsoft/playwright-mcp",
  description: "Browser automation via Playwright",
  tool_count: 25,
  has_report: true,
  has_screenshot: true
)
```

14 servers are pre-seeded on first run. When an MCP tool fails, the server checks if `report_issue` is available and guides you to file or report the error.

## MCP Interface

### Tools (44)

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
| **User Profile** | `store_user_profile` | Store a user profile field |
| | `get_user_profile` | Get a profile field |
| | `list_user_profile` | List all profile fields |
| | `delete_user_profile` | Delete a profile field |

### Resources (13)

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
| `user://profile` | User profile data |

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

---

## Design Philosophy

Built iteratively across AI-assisted development sessions. The companion project [`go-mcp-computer-use`](https://github.com/coff33ninja/go-mcp-computer-use) provides desktop automation (mouse, keyboard, OCR, window management) — ai-memory gives that agent persistent recall, skill growth, and self-evolution.

The project is guided by a curated set of quality-enforcement skills from [coff33ninja/ai-skills](https://github.com/coff33ninja/ai-skills) — anti-hallucination, anti-slop, safe-code-modifications, anti-sycophancy, code-simplification, context-engineering, don't-kill-tokens, os-awareness, anti-tool-sprawl, follow-existing-patterns, no-dead-code-removal, universal-format-lint, self-validate, verify-and-cite, and others.

### Core Principles

- **Single binary, zero config** — CGO+Zig for self-contained builds, auto-download ONNX models at first run
- **Semantic memory** — embeddings for recall, not just keyword matching. Memories are embedded on store so they're immediately searchable.
- **Evolution over configuration** — the system improves its own behavioral rules from experience, not from hand-tuned configs
- **Scope-aware isolation** — persona DBs prevent cross-contamination, shared scope enables collaboration
- **Composable with desktop automation** — designed to pair with go-mcp-computer-use for agents that can both act and remember
- **Security by design** — the server holds API keys and behavioral rules. Never log secrets, never commit keys, scope-sensitive data to `user:` profiles. See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full trust boundary model.
- **Accessibility first** — multi-persona support enables different interaction styles for different users and use cases, from formal to casual to specialized domains

### Architectural Patterns

- **Cascading lookup** — memories, tool knowledge, skills, and user profiles all follow the same pattern: check local scope first, fall back to shared/global, then search embeddings. This is the same pattern go-mcp-computer-use uses for UI element detection (memory → ONNX → OCR).
- **SQLite as universal store** — both projects use SQLite for persistence. ai-memory uses it for memories, embeddings, personas, tool knowledge, recipes, and user profiles. go-mcp-computer-use uses it for training data, UI element caches, and data logging. FTS5 full-text search is available across all text columns.
- **Embedding-on-write** — embeddings are generated at store time, not on search. This ensures every memory is immediately findable without requiring manual reindex operations.
- **Auto-consolidation** — memories are automatically deduplicated, elevated, and pruned during evolution cycles. Old low-impact entries are removed to keep the database clean.
- **Graceful degradation** — ONNX models are auto-downloaded at first use, but the system works without them (keyword search fallback). Skills are cloned from a remote repo but cached locally. If the network is unavailable, the last-known state is used.
- **Panic recovery** — tool panics log stack traces and return errors instead of crashing the server. The MCP server stays alive even if individual tool calls fail.
- **File-based logging** — rotating JSON logs at `%APPDATA%/ai-memory/logs/`, configurable retention. Error logs are available for AI diagnostics via tool calls.
- **CI/CD enforcement** — lint, build, tools.md generation check, and release pipeline all run on every push. Drift between source and generated docs is caught automatically.

### Development Workflow

The project follows an atomic commit + conventional commits workflow:
1. Make changes
2. Run `scripts/lint.ps1` to verify
3. Stage and commit with conventional message (`feat:`, `fix:`, `docs:`, etc.)
4. Bump VERSION file
5. Update CHANGELOG.md
6. Run `scripts/push-and-release.ps1` to handle commit, tag, push, and release creation

Never push directly — always use the push-and-release script. This ensures proper tagging, release workflow, and binary distribution.
