# ai-memory — Project Evolution

## What Was: ai-personality

An MCP server that gave AI assistants a persistent, evolving personality. Two implementations (TypeScript + Python), published to npm and PyPI.

**What it had**:
- 6 personality files (identity, traits, values, rules, memories, relationships) with YAML frontmatter
- Memory reflection lifecycle (experience → lesson → impact)
- Vector search via SQLite + MiniLM embeddings (RAG)
- Cross-reference validation between personality files
- Skills subsystem (git-cloned repo, catalog, search)
- Client setup generator for 15+ AI tools

**Why it's being replaced**:
- Personality as a concept is wrong — identity/traits/values/rules/relationships don't persist across sessions meaningfully
- Two runtimes (Node + Python) to maintain identical behavior
- Heavy embedding deps (~50MB WASM, ~300MB Python)
- `npx`/`uvx` distribution is fragile
- Client setup generator is low-value busywork
- The real value was always memory + skills, not personality

---

## What Will Be: ai-memory

An MCP server that gives AI assistants persistent memory and skill recall. One Go binary. No personality abstraction.

**Core insight**: When an AI starts a session, it needs two things:
1. **Memory** — what happened before, what was learned, what the user cares about
2. **Skills** — what it knows how to do, procedural knowledge, tool usage patterns

These are stored together, recalled together, searched together. That's the whole product.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  AI Client                          │
│  (Kiro / Cursor / Claude / OpenCode / Codex / ...) │
│                                                     │
│  Session start: recall memory + skills via MCP      │
│  During session: store new memories, use skills     │
│  Session end: reflect, store, update                │
└──────────────────────┬──────────────────────────────┘
                       │ stdio (JSON-RPC 2.0)
                       ▼
┌─────────────────────────────────────────────────────┐
│                ai-memory-server                     │
│                  (Go binary)                        │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐                │
│  │   memory     │  │    skills    │                │
│  │ (read/write  │  │ (clone/search│                │
│  │  entries)    │  │  catalog)    │                │
│  └──────┬───────┘  └──────┬───────┘                │
│         │                 │                         │
│  ┌──────┴─────────────────┴───────┐                │
│  │         rag (vector DB)         │                │
│  │  SQLite + ONNX embeddings       │                │
│  │  Searches BOTH memory + skills  │                │
│  └────────────────────────────────┘                │
└─────────────────────────────────────────────────────┘
        │
        ▼
~/.ai-memory/
├── memory.db          (SQLite: entries + embeddings + skill index)
├── skills/            (git clone of ai-skills repo)
│   └── <name>/SKILL.md
└── models/
    └── all-MiniLM-L6-v2.onnx  (auto-downloaded on first run)
```

---

## What's Removed (Personality)

All of it. These files no longer exist:

| Removed | Why |
|---------|-----|
| `identity.md` | AI identity is set by the client, not the server |
| `traits.md` | Communication style is a client concern |
| `values.md` | Values are system-prompt material, not persistent data |
| `rules.md` | Rules are client-specific, not portable |
| `relationships.md` | User profiles belong in memory entries, not a separate file |
| Cross-reference validation | Was between personality files — now between memory entries and skills |
| Client setup generator | Each client has its own config; we just provide the MCP server |

---

## What Stays (Memory)

### Memory Entries

Same format, now stored in SQLite instead of markdown files:

```sql
CREATE TABLE memories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  experience TEXT NOT NULL,
  lesson TEXT NOT NULL,
  impact TEXT NOT NULL DEFAULT 'under review',
  tags TEXT,           -- comma-separated tags for categorization
  embedding BLOB,      -- float32 vector for semantic search
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

**Entry lifecycle** (unchanged):
1. `store(experience, lesson, tags?)` → entry with impact="under review"
2. `review()` → list pending entries
3. AI applies lessons, calls `apply(id)` or `dismiss(id)`
4. Entry moves to resolved state

### Memory States

| State | Meaning |
|-------|---------|
| `under review` | New, not yet acted upon |
| `applied` | Lesson was incorporated into behavior |
| `dismissed` | Decided no change needed |
| custom string | AI-set descriptive impact |

---

## What Stays (Skills)

### Skill Storage

Git clone of `https://github.com/coff33ninja/ai-skills` to `~/.ai-memory/skills/`.

```
~/.ai-memory/skills/
├── playwright/
│   ├── SKILL.md
│   └── *.ps1
├── debugging-and-error-recovery/
│   ├── SKILL.md
│   └── ...
└── ... (50+ skills)
```

### Skill Index (NEW — in SQLite)

Skills are indexed in the same database as memory:

```sql
CREATE TABLE skills (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  body TEXT NOT NULL,
  embedding BLOB,
  file_count INTEGER,
  synced_at TEXT NOT NULL
);

CREATE TABLE skill_files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  skill_id INTEGER NOT NULL REFERENCES skills(id),
  filename TEXT NOT NULL,
  content TEXT NOT NULL
);
```

### Why Skills Are Indexed

The AI doesn't just need to "search skills by keyword." It needs to:
1. **Recall relevant skills for a task** — "I'm debugging a React app" → recall debugging skills
2. **Combine memory + skills** — "Last time I used playwright, I learned X" → search both
3. **Semantic skill matching** — "automate a browser" → find playwright even if the word "playwright" isn't in the query

This is why skills get embeddings too. The RAG pipeline searches across both tables in a single query.

---

## What's New (Unified RAG)

### Single Database

One SQLite file (`memory.db`) holds everything:

```sql
-- Memory entries
CREATE TABLE memories (id, date, experience, lesson, impact, tags, embedding, created_at, updated_at);

-- Skill catalog
CREATE TABLE skills (id, name, description, body, embedding, file_count, synced_at);

-- Skill files (full content for recall)
CREATE TABLE skill_files (id, skill_id, filename, content);

-- Metadata
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT);
```

### Unified Search

When the AI searches, it searches everything:

```sql
-- Pseudo-query: find relevant memories AND skills for a query
SELECT 'memory' as type, id, experience as title, lesson as content, score
FROM memories WHERE embedding MATCH ?
UNION ALL
SELECT 'skill' as type, id, name as title, description as content, score
FROM skills WHERE embedding MATCH ?
ORDER BY score DESC
LIMIT ?;
```

The AI gets a combined result set: "here's what you remember, and here's what you know how to do."

### Embedding Model

Same model as ai-personality, now embedded in the Go binary:

| Model | Dims | Size | Source |
|-------|------|------|--------|
| `all-MiniLM-L6-v2` | 384 | ~23MB ONNX | Hosted on GitHub Releases |

Model is auto-downloaded on first launch if not present at `~/.ai-memory/models/`.

### Chunking Strategy

**Memory entries**: Whole entry is one chunk (they're short).

**Skills**: Chunked by `##` headings, same as ai-personality. Each chunk gets:
- `filename` = skill name
- `heading` = section heading
- `content` = text under that heading (truncated to 512 chars)
- `embedding` = vector

---

## MCP Interface

### Resources

| URI | Description |
|-----|-------------|
| `memory://memories` | All memory entries (JSON) |
| `memory://skills` | All indexed skills (JSON) |
| `memory://summary` | Combined overview: recent memories + available skills |
| `memory://all` | Everything: memories + skills + stats |
| `memory://file/{name}` | Specific memory or skill by name |
| `skills://catalog` | Skill list with metadata |
| `skills://{name}` | All files for a skill |
| `skills://file/{name}/{filename}` | Specific skill file |

### Tools

| Tool | Input | What It Does |
|------|-------|--------------|
| `store` | experience, lesson, tags? | Store a new memory entry |
| `review` | — | List entries with impact="under review" |
| `apply` | id | Mark entry as applied |
| `dismiss` | id | Mark entry as dismissed |
| `status` | — | Stats: memory count, skill count, pending reviews |
| `search` | query, topK?, type? | Unified semantic search (memories + skills) |
| `search_memories` | query, topK? | Search memories only |
| `search_skills` | query, topK? | Search skills only |
| `reindex` | — | Rebuild all embeddings |
| `skills_sync` | — | Git pull skills repo |
| `skills_search` | query | Keyword search skills (fast, no embedding) |

### Prompts

| Prompt | Purpose |
|--------|---------|
| `memory` | Full context: recent memories + relevant skills for session start |
| `reflect` | Guide for end-of-session reflection |

---

## Build & Distribution

### Single Binary

```bash
# Build with Zig CGO cross-compiler
CC="zig cc" CXX="zig c++" CGO_ENABLED=1 go build -ldflags="-s -w" -o ai-memory-server .

# Or pure Go (no CGO)
CGO_ENABLED=0 go build -ldflags="-s -w" -o ai-memory-server .
```

### Model Distribution

ONNX model hosted on GitHub Releases, not embedded in binary:

```
github.com/coff33ninja/ai-memory/releases/latest/
├── ai-memory-server-linux-amd64
├── ai-memory-server-windows-amd64.exe
├── ai-memory-server-darwin-arm64
└── all-MiniLM-L6-v2.onnx          (~23MB)
```

On first run:
1. Check `~/.ai-memory/models/all-MiniLM-L6-v2.onnx`
2. If missing, download from GitHub Releases
3. Cache locally

### CI/CD

From go-mcp-computer-use:
- `ci.yml` — lint, test, build on push/PR
- `release.yml` — cross-compile + GitHub Release on tag
- `auto-tag.yml` — auto-tag on merge to main
- `mod-maintenance.yml` — Go module dependency updates

---

## Migration from ai-personality

### Data

No migration needed. ai-personality's `memories.md` format is compatible — the `store` tool writes the same YAML structure. Existing memories can be imported by reading the markdown file and inserting into SQLite.

### Skills

Same git repo, same directory structure. Just change the clone path from `~/.ai-personality/skills/` to `~/.ai-memory/skills/`.

### Config

Each AI client just needs the MCP server command updated:

```json
{
  "mcpServers": {
    "ai-memory": {
      "command": "ai-memory-server"
    }
  }
}
```

No personality files, no steering files, no hooks. Just the server.

---

## Timeline

| Phase | Status | Description |
|-------|--------|-------------|
| Architecture docs | ✅ Done | ARCHITECTURE.md, QUICK-REFERENCE.md, ADR-002 |
| Project evolution doc | ✅ Done | This document |
| CI/CD scaffolding | ✅ Done | Copied from go-mcp-computer-use |
| Go project scaffold | Pending | go.mod, directory structure, types |
| MCP server core | Pending | JSON-RPC 2.0 over stdio |
| SQLite schema | Pending | memories + skills + skill_files + meta tables |
| Memory CRUD | Pending | store/review/apply/dismiss/status |
| Skill indexing | Pending | git clone + parse + embed + store |
| Unified RAG search | Pending | Combined memory + skill vector search |
| ONNX integration | Pending | Model download + embedding pipeline |
| File watcher | Pending | Auto-reindex on skill repo changes |
| CI/CD finalization | Pending | Edit workflows for Go build matrix |
| GitHub release | Pending | Push, tag, binary distribution |

---

## Related

- [ARCHITECTURE.md](ARCHITECTURE.md) — Full reference of the ai-personality system (archived)
- [QUICK-REFERENCE.md](QUICK-REFERENCE.md) — One-page cheat sheet (archived)
- [ADR-002-go-rewrite.md](adr/ADR-002-go-rewrite.md) — Decision record for Go rewrite
