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
| scope | TEXT | `private` or `shared` |
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
| persona | TEXT | Persona name |
| tool_name | TEXT | Tool name |
| how_to_use | TEXT | How the tool works |
| what_works | TEXT | Patterns that produce good results |
| what_fails | TEXT | Common mistakes |
| params | TEXT | Parameter guide |
| examples | TEXT | Example invocations |
| use_count | INTEGER | Times used |

### tool_errors

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| persona | TEXT | Persona name |
| tool_name | TEXT | Tool name |
| error_msg | TEXT | Error text |
| context | TEXT | What you were trying to do |
| input_args | TEXT | Arguments passed |
| resolved | INTEGER | 0 = open, 1 = resolved |
| reported | INTEGER | 0 = not reported, 1 = reported |

### user_profiles

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| field | TEXT | Profile field (unique) |
| value | TEXT | Value |
| source | TEXT | `stated` / `inferred` / `conversation` |
| confidence | REAL | 0.0 - 1.0 |

## Tools

### Memory

| Tool | Parameters | Description |
|------|-----------|-------------|
| `store` | `experience`, `lesson`, `tags?`, `scope?` | Store a memory |
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
| `store_skill_usage` | `skill`, `context`, `with_skills?`, `outcome?` | Record skill usage |
| `list_skill_usage` | — | View usage patterns |

### Persona

| Tool | Parameters | Description |
|------|-----------|-------------|
| `onboard` | `name`, `identity`, `tone?`, `description?`, `greeting?`, `skills?` | Create persona |
| `list_personas` | — | List all personas |
| `switch_persona` | — | Show active persona |
| `delete_persona` | `name` | Delete persona |

### Evolution

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_interaction` | `summary`, `outcome_score` (1-5), `skills_used?`, `tone_used?` | Record interaction outcome |
| `evolve` | — | Trigger full evolution |
| `consolidate` | — | Merge similar memories, prune old |
| `discover_skills` | — | Find new skills from usage |
| `evolution_history` | — | View evolution log |
| `get_evolved_rules` | — | Get adapted behavior rules |
| `interaction_stats` | — | Tone and skill performance |

### Tool Knowledge

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_tool_knowledge` | `tool_name`, `how_to_use`, `what_works?`, `what_fails?`, `params?`, `examples?` | Build manual |
| `log_tool_recipe` | `tool_name`, `recipe_name`, `steps`, `use_case` | Save recipe |
| `get_tool_knowledge` | `tool_name` | Read manual |
| `list_tool_knowledge` | — | List all entries |
| `get_tool_recipes` | `tool_name` | Get recipes |
| `record_recipe_outcome` | `recipe_id`, `success` (bool), `notes?` | Track success/failure |

### Tool Gaps

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_tool_gap` | `need`, `context`, `suggested?` | Record missing capability |
| `list_tool_gaps` | `include_resolved?` | View gaps |
| `resolve_tool_gap` | `id` | Mark resolved |

### Error Tracking

| Tool | Parameters | Description |
|------|-----------|-------------|
| `log_tool_error` | `tool_name`, `error_msg`, `context`, `input_args?`, `mcp_server?` | Log error |
| `list_tool_errors` | `include_resolved?` | View errors |
| `resolve_tool_error` | `id` | Mark resolved |

### MCP Registry

| Tool | Parameters | Description |
|------|-----------|-------------|
| `register_mcp_server` | `name`, `source?`, `has_report?`, `has_screenshot?`, `has_ocr?`, `has_chain?`, `tool_count?`, `creator?`, `repo_url?`, `description?` | Register server |
| `get_mcp_server` | `name` | Get server info |
| `list_mcp_servers` | — | List servers |

### User Profile

| Tool | Parameters | Description |
|------|-----------|-------------|
| `store_user_profile` | `field`, `value`, `source?`, `confidence?` | Store a profile field |
| `get_user_profile` | `field` | Get a profile field |
| `list_user_profile` | — | List all fields |
| `delete_user_profile` | `field` | Delete a field |

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
| `user://profile` | User profile data |
| `project://active` | Active project context |
| `backup://status` | Backup status and history |

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
