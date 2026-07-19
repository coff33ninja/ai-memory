# Architecture

ai-memory is a single Go binary that runs as an MCP (Model Context Protocol) server over stdio. It persists memory, indexes skills, tracks tool usage, and evolves its behavior across sessions.

## Package Layout

```
cmd/ai-memory-server/
├── main.go                      Entry point, handler wiring, MCP server
├── handlers.go                  store, review, apply, dismiss, status, search, reindex
├── context.go                   detectProject, context resources, skill usage patterns
├── persona_handlers.go          onboard, list, switch, delete personas
├── evolution_handlers.go        log_interaction, evolve, consolidate, discover_skills
├── tool_knowledge_handlers.go   log_tool_knowledge, recipes, recipe outcomes
├── error_handlers.go            log_tool_error, MCP server registry

internal/
├── mcp/
│   ├── server.go                JSON-RPC 2.0 stdio server, dispatch, auth
│   └── defs.go                  40 tools, 12 resources, 6 prompts (canonical definitions)
├── memory/
│   ├── store.go                 SQLite operations, migrations, FTS5
│   ├── search.go                Semantic + keyword search across tables
│   └── embedding.go             ONNX Runtime session, tokenizer, cosine similarity
├── rag/
│   ├── rag.go                   Unified search across persona + shared DBs
│   ├── store.go                 Store memory entries, skill index, tool knowledge
│   └── embedding.go             Re-exports for convenience
├── skills/
│   ├── skills.go                Git clone/pull, SKILL.md parsing, SQLite indexing
│   ├── catalog.go               Skill listing, description search
│   ├── index.go                 Embedding-based skill search
│   └── usage.go                 Skill usage tracking and pattern detection
├── skills/ai-skills/            Cloned skill repository (51+ skills)
├── project/
│   ├── context.go               detectProject, getRelevantSkillsForProject
│   └── project.go               Supported projects: react, go, rust, opencode, etc.
├── skills.go                    loadSkills, searchSkills
├── evolution/
│   ├── engine.go                Full evolution cycle (tone adaptation, skill discovery, gap closure, consolidation)
│   └── auto.go                  Auto-evolve every 10 interactions
├── persona/
│   └── manager.go               Persona registry, scoped memory store
└── types/
    └── types.go                 Memory, Skill, SearchResult, SkillUsage, MCPError, ToolKnowledge, etc.
```

## Data Flow

### Session Start

```
AI Client calls context://startup resource
→ detectProject() identifies project type
→ searchSkills() finds skills relevant to project
→ memories sorted by date (last 10)
→ skill_usage sorted by date (last 20)
→ Return JSON with: project, skills, memories, skill_usage
```

### Store Memory

```
AI calls store(experience, lesson, tags)
→ Insert into memories table
→ Generate 384-dim ONNX embedding of "experience:lesson"
→ Store embedding as BLOB
→ Return entry with impact="under review"
```

### Search

```
AI calls search(query, topK, type)
→ Generate embedding for query
→ If type="all" or omitted:
    → cosineSim against memories table
    → cosineSim against skills table
    → Sort combined results by score, take topK
→ If type="memory": memories only
→ If type="skill": skills only
→ Return results with type label
```

### Self-Evolution

```
log_interaction(outcome=1-5, notes)
→ Insert into interactions table
→ Every 10 interactions: auto-evolve()
  → Tone adaptation: recalculate personalityScores from outcomes
  → Skill discovery: find used skills, discover patterns
  → Tool gap closure: map gaps to skills, close resolved
  → Consolidation: merge memories with cosineSim > 0.75, delete old dismissed
  → Write evolved rules, tone, skill_set to files
```

## Database Schema

### memories

```sql
CREATE TABLE memories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  experience TEXT NOT NULL,
  lesson TEXT NOT NULL,
  impact TEXT NOT NULL DEFAULT 'under review',
  tags TEXT,
  embedding BLOB,           -- float32[384] as bytes
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_memories_date ON memories(date);
CREATE INDEX idx_memories_impact ON memories(impact);
```

### skills

```sql
CREATE TABLE skills (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  body TEXT NOT NULL,
  embedding BLOB,           -- float32[384] as bytes
  file_count INTEGER,
  synced_at TEXT NOT NULL
);
CREATE INDEX idx_skills_name ON skills(name);
```

### skill_files

```sql
CREATE TABLE skill_files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  skill_id INTEGER NOT NULL REFERENCES skills(id),
  filename TEXT NOT NULL,
  content TEXT NOT NULL
);
CREATE INDEX idx_skill_files_skill_id ON skill_files(skill_id);
```

### skill_usage

```sql
CREATE TABLE skill_usage (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  skills_used TEXT NOT NULL,          -- JSON array of skill names
  task_description TEXT NOT NULL,
  notes TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_skill_usage_date ON skill_usage(date);
```

### tool_knowledge

```sql
CREATE TABLE tool_knowledge (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tool_name TEXT NOT NULL,
  knowledge_type TEXT NOT NULL,       -- 'manual' or 'recipe'
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  content TEXT NOT NULL,
  outcome TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_tool_knowledge_tool ON tool_knowledge(tool_name, knowledge_type);
```

### tool_errors

```sql
CREATE TABLE tool_errors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp TEXT NOT NULL,
  tool TEXT NOT NULL,
  server TEXT NOT NULL,
  error_message TEXT NOT NULL,
  error_type TEXT NOT NULL DEFAULT 'unknown',
  stack_trace TEXT,
  mcp_params TEXT,
  session_id TEXT,
  resolved INTEGER DEFAULT 0,
  resolution TEXT
);
CREATE INDEX idx_tool_errors_tool ON tool_errors(tool, timestamp);
```

### mcp_servers

```sql
CREATE TABLE mcp_servers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  capabilities TEXT,            -- JSON array
  registered_at TEXT NOT NULL,
  last_used TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  use_count INTEGER DEFAULT 0
);
```

### personas.json

```json
[
  {
    "name": "default",
    "identity": "General-purpose assistant",
    "tone": "neutral",
    "communication_style": "concise",
    "skills": [],
    "created_at": "...",
    "last_used": "...",
    "memory_count": 42
  }
]
```

## Embedding Pipeline

1. ONNX Runtime session created with `all-MiniLM-L6-v2` model (384 dimensions)
2. Session is pinned to the OS thread that created it (required by ONT)
3. Tokenizer runs in Go — wordpiece + BERT tokenization
4. ONNX `Run()` produces 384-dim float32 array
5. Stored as bytes (`[]byte`) in SQLite BLOB
6. Cosine similarity computed at search time (vector magnitude cached per entry)

Model auto-downloads on first run:
- URL: `https://github.com/coff33ninja/go-mcp-computer-use/releases/latest/download/all-MiniLM-L6-v2.onnx`
- Location: `%APPDATA%\ai-memory\lib\all-MiniLM-L6-v2.onnx`

## MCP Protocol

JSON-RPC 2.0 over stdio:

```json
{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{}}}
{"jsonrpc":"2.0","result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":false},"resources":{"listChanged":false},"prompts":{"listChanged":false}}},"id":1}
{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}
```

Tool calls follow:
```json
{"jsonrpc":"2.0","method":"tools/call","id":2,"params":{"name":"store","arguments":{"experience":"...","lesson":"..."}}}
```

## Error Handling

- All handlers catch panics via `recover()`
- Failed calls return `isError: true` with descriptive message
- MCP tool errors are logged to `tool_errors` table
- ONNX failures fall back to keyword search
- Session mismatches return `MCPErrorWrongSession` or `MCPErrorWrongThread`
