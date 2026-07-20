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
├── user_profile_handlers.go     store, get, list, delete user profile fields
├── project_handlers.go          set_project_context, map_persona, list_persona_mappings
├── backup_handlers.go           backup, restore, backup_config, list_backup_drives
├── backup_config_file.go        BackupRecord, BackupConfigFile structs
├── drives_windows.go            Windows drive/cloud folder detection

internal/
├── mcp/
│   ├── server.go                JSON-RPC 2.0 stdio server, dispatch, auth
│   └── defs.go                  55 tools, 15 resources, 3 templates, 7 prompts (canonical definitions)
├── db/
│   ├── db.go                    SQLite schema migrations, all CREATE TABLE statements
│   └── types.go                 Shared database types
├── embedding/
│   └── embedding.go             ONNX Runtime session, tokenizer, cosine similarity
├── memory/
│   └── memory.go                SQLite operations for memories, skills, skill_usage
├── rag/
│   └── rag.go                   Unified search across persona + shared DBs
├── skills/
│   └── skills.go                Git clone/pull, SKILL.md parsing, SQLite indexing
├── skills/ai-skills/            Cloned skill repository (51+ skills)
├── evolution/
│   ├── engine.go                Full evolution cycle, auto-evolve every 10 interactions
│   ├── tracker.go               Interaction outcomes, tool gaps, tool knowledge, errors
│   ├── consolidator.go          Merge similar memories, elevate, prune
│   ├── adapter.go               Tone and trait adaptation from performance data
│   └── types.go                 InteractionOutcome, EvolutionEntry, ToolGap, ToolKnowledge, etc.
├── persona/
│   └── persona.go               Persona registry, scoped memory store, greeting matching
└── version/
    └── version.go               Build version (injected via ldflags)
```

## Data Flow

### Session Start

```
AI Client calls context://startup resource
→ detectProject() identifies project type
→ searchSkills() finds skills relevant to project
→ List user profile fields
→ memories sorted by date (last 10)
→ skill_usage sorted by date (last 20)
→ Return text with: project, user profile, skills, memories, skill_usage
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
log_interaction(outcome_score=1-5, summary)
→ Insert into interaction_outcomes table
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
  tags TEXT DEFAULT '',
  scope TEXT NOT NULL DEFAULT 'private',
  embedding BLOB,
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
  description TEXT DEFAULT '',
  body TEXT NOT NULL,
  embedding BLOB,
  file_count INTEGER DEFAULT 0,
  synced_at TEXT NOT NULL
);
CREATE INDEX idx_skills_name ON skills(name);
```

### skill_files

```sql
CREATE TABLE skill_files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  filename TEXT NOT NULL,
  content TEXT NOT NULL
);
CREATE INDEX idx_skill_files_skill ON skill_files(skill_id);
```

### skill_usage

```sql
CREATE TABLE skill_usage (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date TEXT NOT NULL,
  skill TEXT NOT NULL,
  context TEXT NOT NULL,
  with_skills TEXT DEFAULT '',
  outcome TEXT NOT NULL DEFAULT 'used',
  embedding BLOB,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_skill_usage_skill ON skill_usage(skill);
CREATE INDEX idx_skill_usage_date ON skill_usage(date);
```

### interaction_outcomes

```sql
CREATE TABLE interaction_outcomes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  persona TEXT NOT NULL,
  summary TEXT NOT NULL,
  outcome_score INTEGER NOT NULL,
  skills_used TEXT DEFAULT '',
  tone_used TEXT DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX idx_interaction_outcomes_persona ON interaction_outcomes(persona);
```

### evolution_log

```sql
CREATE TABLE evolution_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  persona TEXT NOT NULL,
  trigger TEXT NOT NULL,
  what_changed TEXT NOT NULL,
  before_val TEXT,
  after_val TEXT,
  confidence REAL DEFAULT 1.0,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_evolution_log_persona ON evolution_log(persona);
```

### tool_knowledge

```sql
CREATE TABLE tool_knowledge (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  persona TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  how_to_use TEXT NOT NULL,
  what_works TEXT NOT NULL DEFAULT '',
  what_fails TEXT NOT NULL DEFAULT '',
  params TEXT NOT NULL DEFAULT '',
  examples TEXT NOT NULL DEFAULT '',
  use_count INTEGER NOT NULL DEFAULT 0,
  last_used TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX idx_tool_knowledge_persona ON tool_knowledge(persona);
CREATE INDEX idx_tool_knowledge_tool ON tool_knowledge(tool_name);
```

### tool_recipes

```sql
CREATE TABLE tool_recipes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  persona TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  recipe_name TEXT NOT NULL,
  steps TEXT NOT NULL,
  use_case TEXT NOT NULL,
  success_count INTEGER NOT NULL DEFAULT 0,
  fail_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_tool_recipes_persona ON tool_recipes(persona);
CREATE INDEX idx_tool_recipes_tool ON tool_recipes(tool_name);
```

### tool_errors

```sql
CREATE TABLE tool_errors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  persona TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  error_msg TEXT NOT NULL,
  context TEXT NOT NULL DEFAULT '',
  input_args TEXT NOT NULL DEFAULT '',
  resolved INTEGER NOT NULL DEFAULT 0,
  reported INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_tool_errors_persona ON tool_errors(persona);
CREATE INDEX idx_tool_errors_tool ON tool_errors(tool_name);
CREATE INDEX idx_tool_errors_resolved ON tool_errors(resolved);
```

### tool_gaps

```sql
CREATE TABLE tool_gaps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  persona TEXT NOT NULL,
  need TEXT NOT NULL,
  context TEXT NOT NULL DEFAULT '',
  suggested TEXT NOT NULL DEFAULT '',
  resolved INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_tool_gaps_persona ON tool_gaps(persona);
CREATE INDEX idx_tool_gaps_resolved ON tool_gaps(resolved);
```

### mcp_servers

```sql
CREATE TABLE mcp_servers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  source TEXT NOT NULL DEFAULT '',
  has_report INTEGER NOT NULL DEFAULT 0,
  has_screenshot INTEGER NOT NULL DEFAULT 0,
  has_ocr INTEGER NOT NULL DEFAULT 0,
  has_chain INTEGER NOT NULL DEFAULT 0,
  tool_count INTEGER NOT NULL DEFAULT 0,
  creator TEXT NOT NULL DEFAULT '',
  repo_url TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  last_seen TEXT,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_mcp_servers_name ON mcp_servers(name);
```

### user_profiles

```sql
CREATE TABLE user_profiles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  field TEXT NOT NULL,
  value TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'inferred',
  confidence REAL NOT NULL DEFAULT 0.5,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(field)
);
CREATE INDEX idx_user_profiles_field ON user_profiles(field);
```

### project_contexts

```sql
CREATE TABLE project_contexts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_name TEXT NOT NULL UNIQUE,
  project_type TEXT NOT NULL,
  root_path TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### persona_mappings

```sql
CREATE TABLE persona_mappings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project TEXT NOT NULL UNIQUE,
  persona TEXT NOT NULL,
  created_at TEXT NOT NULL
);
```

### backup_config

```sql
CREATE TABLE backup_config (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider TEXT NOT NULL DEFAULT 'local',
  auto_backup INTEGER NOT NULL DEFAULT 0,
  interval_hours INTEGER NOT NULL DEFAULT 24,
  local_path TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### backups

```sql
CREATE TABLE backups (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  backup_path TEXT NOT NULL,
  provider TEXT NOT NULL,
  file_size INTEGER NOT NULL DEFAULT 0,
  encrypted INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
```

### meta

```sql
CREATE TABLE meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

### personas.json

```json
{
  "active": "default",
  "personas": [
    {
      "name": "default",
      "identity": "General-purpose assistant",
      "tone": "direct",
      "description": "Auto-created on first run",
      "greeting": "",
      "skills": [],
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
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
