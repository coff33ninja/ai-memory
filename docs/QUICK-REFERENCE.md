# Quick Reference

## Storage

| What | Path |
|------|------|
| Data directory | `%USERPROFILE%\.ai-memory\` |
| Per-persona DB | `%USERPROFILE%\.ai-memory\{name}\memory.db` |
| Shared DB | `%USERPROFILE%\.ai-memory\shared\memory.db` |
| Persona registry | `%USERPROFILE%\.ai-memory\personas.json` |
| Skills repo | `%USERPROFILE%\.ai-memory\skills\ai-skills\` |
| ONNX Runtime | `%APPDATA%\ai-memory\lib\onnxruntime.dll` |
| Embedding model | `%APPDATA%\ai-memory\lib\all-MiniLM-L6-v2.onnx` |

Override with `AI_MEMORY_DIR` environment variable.

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_MEMORY_DIR` | `~/.ai-memory` | Data directory |

## Data Schema

### memories

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| date | TEXT | Entry date |
| experience | TEXT | What happened |
| lesson | TEXT | What was learned |
| impact | TEXT | `under review` / `applied` / `dismissed` / custom |
| tags | TEXT | Comma-separated |
| embedding | BLOB | 384-dim float32 vector |

### skills

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| name | TEXT | Unique skill name |
| description | TEXT | Short description |
| body | TEXT | Full SKILL.md content |
| embedding | BLOB | 384-dim float32 vector |
| file_count | INTEGER | Number of files |

### tool_knowledge

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| tool_name | TEXT | Tool name |
| knowledge_type | TEXT | `manual` or `recipe` |
| title | TEXT | What you know |
| description | TEXT | Details |
| content | TEXT | Full description |
| outcome | TEXT | `success` / `failure` / `partial` |

### tool_errors

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| timestamp | TEXT | When it happened |
| tool | TEXT | Tool name |
| server | TEXT | MCP server name |
| error_message | TEXT | Error text |
| error_type | TEXT | `permission` / `not_found` / `timeout` / etc. |
| resolved | INTEGER | 0 = open, 1 = resolved |

## Tools

### Memory

| Tool | Parameters | Description |
|------|-----------|-------------|
| `store` | `experience`, `lesson`, `tags?` | Store a memory |
| `review` | — | List pending memories |
| `apply` | `id` | Mark memory as applied |
| `dismiss` | `id` | Mark memory as dismissed |
| `status` | — | Counts and summary |
| `search` | `query`, `topK?`, `type?` | Semantic search (all/memory/skill) |
| `search_memories` | `query`, `topK?` | Search memories only |
| `search_skills` | `query`, `topK?` | Search skills only |
| `reindex` | — | Rebuild all embeddings |

### Skills

| Tool | Parameters | Description |
|------|-----------|-------------|
| `skills_sync` | — | Git pull the skills repo |
| `skills_search` | `query` | Keyword search (no embedding) |
| `skills_index` | — | Re-index skills to SQLite |
| `store_skill_usage` | `skills_used`, `task_description`, `notes?` | Record skill usage |
| `list_skill_usage` | — | View usage patterns |

### Persona

| Tool | Parameters | Description |
|------|-----------|-------------|
| `onboard` | `name`, `identity`, `tone?`, `communication_style?`, `skills?` | Create persona |
| `list_personas` | — | List all personas |
| `switch_persona` | — | Show active persona |
| `delete_persona` | `name` | Delete persona |

### Evolution

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_interaction` | `outcome` (1-5), `notes?` | Record interaction outcome |
| `evolve` | — | Trigger full evolution |
| `consolidate` | — | Merge similar memories, prune old |
| `discover_skills` | — | Find new skills from usage |
| `evolution_history` | — | View evolution log |
| `get_evolved_rules` | — | Get adapted behavior rules |
| `interaction_stats` | — | Tone and skill performance |

### Tool Knowledge

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_tool_knowledge` | `tool_name`, `knowledge_type`, `title`, `description`, `content`, `outcome?` | Build manual |
| `log_tool_recipe` | `tool_name`, `title`, `description`, `steps` | Save recipe |
| `get_tool_knowledge` | `tool_name` | Read manual |
| `list_tool_knowledge` | — | List all entries |
| `get_tool_recipes` | `tool_name` | Get recipes |
| `record_recipe_outcome` | `tool_name`, `recipe_title`, `outcome`, `notes?` | Track success/failure |

### Tool Gaps

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_tool_gap` | `tool_name`, `description` | Record missing capability |
| `list_tool_gaps` | — | View unresolved gaps |
| `resolve_tool_gap` | `id`, `resolution?` | Mark resolved |

### Error Tracking

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_tool_error` | `tool`, `server`, `error_message`, `error_type`, `stack_trace?`, `mcp_params?`, `session_id?` | Log error |
| `list_tool_errors` | — | View errors |
| `resolve_tool_error` | `id`, `resolution?` | Mark resolved |

### MCP Registry

| Tool | Parameters | Description |
|------|-----------|-------------|
| `register_mcp_server` | `name`, `description?`, `capabilities?` | Register server |
| `get_mcp_server` | `name` | Get server info |
| `list_mcp_servers` | — | List servers |

## Resources

| URI | Description |
|-----|-------------|
| `memory://memories` | All memory entries (JSON) |
| `memory://skills` | All indexed skills (JSON) |
| `memory://summary` | Combined stats overview |
| `memory://all` | Everything combined |
| `skills://catalog` | Skill catalog with descriptions |
| `skills://usage` | Skill usage patterns |
| `context://project` | Detected project + relevant skills |
| `context://startup` | Full session init context |
| `persona://active` | Current persona details |
| `persona://all` | All personas |
| `evolution://stats` | Interaction stats |
| `evolution://rules` | Adapted behavior rules |

## Prompts

| Name | Purpose |
|------|---------|
| `memory` | Session init: pending memories + relevant skills |
| `reflect` | End-of-session reflection guide |
| `context-inject` | Search-before-answer behavior |
| `skill-usage-recorder` | Guide for recording skill usage |
| `persona-startup` | Persona-aware session init |
| `evolution-loop` | Full evolution cycle |
| `mcp-error-handling` | Error logging guide |

## Scripts

| Script | Purpose |
|--------|---------|
| `scripts\build.ps1` | Build with Zig CC + CGO |
| `scripts\lint.ps1` | go vet + build check |
| `scripts\test.ps1` | Full test suite |
| `scripts\install.ps1` | Clone, build, install |
| `scripts\push-and-release.ps1` | Version bump + push + release |
| `scripts\gen-icons.ps1` | Generate Windows icon resource |
| `scripts\gen-tools-doc.go` | Auto-generate tools.md from source |

## Supported AI Skills

ai-memory indexes 51+ skills from `https://github.com/coff33ninja/ai-skills`. The full catalog is in the `skills` SQLite table after running `skills_sync`.
