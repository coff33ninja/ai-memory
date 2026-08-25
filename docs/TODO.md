# ai-memory TODO

## Knowledge Graph Feature

> Reference: codebase-memory-mcp (C, SQLite, 158 tree-sitter grammars, Cypher, RaBitQ vectors)
> We use Go + SQLite + all-MiniLM-L6-v2 — cleaner stack, faster iteration, memory-native.

### Phase 1 — Schema + CRUD
- [ ] Add `graph_nodes`, `graph_edges`, `graph_props` tables to SQLite schema
- [ ] Node types: `memory`, `entity`, `code`, `project`, `tool`, `concept`, `person`
- [ ] Edge types: `thematic`, `temporal`, `causal`, `uses`, `created`, `calls`, `defines`, `about`, `relates_to`, `contradicts`, `supports`
- [ ] `internal/graph/graph.go` — `AddNode`, `AddEdge`, `GetNode`, `GetEdges`, `DeleteNode`, `DeleteEdge`
- [ ] `internal/graph/query.go` — `Neighbors`, `FindPath` (BFS), `SearchNodes`
- [ ] FTS5 index on node labels for fast text search (like CBM's `nodes_fts`)
- [ ] JSON properties blob on each node (arbitrary metadata)
- [ ] MCP tools: `graph_add_node`, `graph_add_edge`, `graph_query`, `graph_neighbors`, `graph_find_path`, `graph_delete`
- [ ] Tests for all graph CRUD + query operations

### Phase 2 — Auto-link Memories
- [ ] On `store`, compute embedding similarity against recent N memories
- [ ] If cosine similarity > 0.7 → create `thematic` edge automatically
- [ ] Temporal edges: link memories from same session with `temporal` edge
- [ ] Causal edges: regex match "because of", "after", "fixed by", "caused by" → `causal` edge
- [ ] Contradiction detection: opposite impact values on similar topics → `contradicts` edge
- [ ] MCP tool: `graph_link_memories` (manual linking with custom edge type)
- [ ] MCP tool: `graph_unlink_memories` (remove edge)

### Phase 3 — Entity Extraction
- [ ] Extract entities from memory text: people (capitalized names), projects, tools, concepts
- [ ] Use known tool names from `tool_knowledge` table as seed entities
- [ ] Use persona names from `personas.json` as person entities
- [ ] Auto-create `entity` nodes, link memories with `about` edge
- [ ] Entity dedup: cosine similarity on entity name embeddings → merge
- [ ] MCP tools: `graph_add_entity`, `graph_list_entities`, `graph_entity_graph`, `graph_merge_entities`

### Phase 4 — Code-aware Memory (learned from CBM)
- [ ] On `set_project_context`, optionally index codebase via tree-sitter (Go only first — we're Go)
- [ ] Store code symbols as `code` nodes (function, struct, method, variable)
- [ ] CONTAINS edges: project → package → file → function
- [ ] CALLS edges: function A calls function B (static analysis)
- [ ] IMPORTS edges: file A imports file B
- [ ] Link memories to code symbols with `relates_to` edge
- [ ] Query: "what do I know about function X?" → traverse graph
- [ ] MCP tool: `graph_code_context` (given symbol, return related memories + callers + callees)
- [ ] Future: tree-sitter grammars for C, Python, JS/TS, Rust (like CBM's 158 grammars)

### Phase 5 — Graph Query API
- [ ] `graph_search` — semantic search over node labels + properties (embedding cosine)
- [ ] `graph_path` — BFS shortest path between any two nodes
- [ ] `graph_subgraph` — extract neighborhood of a node (depth-limited BFS)
- [ ] `graph_stats` — node/edge counts by type, per-project
- [ ] `graph_communities` — Leiden community detection on subgraph (like CBM)
- [ ] `graph_similar` — MinHash fingerprint similarity between code nodes
- [ ] Resource: `graph://stats` — always show graph summary
- [ ] Resource: `graph://entity/{name}` — entity detail with all connections

---

## Improvements Over codebase-memory-mcp

### What we do better
- [ ] **Runtime feedback**: Tool errors, recipes, interactions feed back into graph weights
- [ ] **Memory↔Code bridge**: Memories link to code symbols, code links to memories
- [ ] **Persona-aware**: Graph queries scoped by persona (private vs shared)
- [ ] **Evolution integration**: Graph patterns feed into trait adaptation
- [ ] **Single binary**: Go + CGO, no daemon, no subprocess supervision needed
- [ ] **Fuzzy search**: Levenshtein edit distance on node names (CBM only has regex)
- [ ] **Write Cypher**: Full CREATE/DELETE/SET support (CBM is read-only)
- [ ] **Embedding-native**: all-MiniLM-L6-v2 at query time, not just indexed vectors

### What we should add that CBM doesn't have
- [ ] **Temporal graph**: Time-weighted edges (recent connections rank higher)
- [ ] **Confidence decay**: Edge confidence decays over time if not reinforced
- [ ] **Memory→Code impact**: "This bug fix touched functions X, Y, Z" → automatic `relates_to` edges
- [ ] **Tool→Code mapping**: "go-mcp-computer-use click handler" → links to actual code
- [ ] **Cross-persona graph**: Shared entities visible across personas, private memories scoped

---

## ML-Embedded Skill-Aware Memory

> Use existing all-MiniLM-L6-v2 embeddings to make the memory system skill-aware.
> No new models needed — just smarter use of what we have.

### Core concept
When storing a memory, compute cosine similarity between memory text and all skill descriptions.
High similarity = memory is about that skill → auto-tag. This makes memories searchable by skill context.

### Schema changes
- [ ] Add `embedding` column to `skills` table (already exists — verify it's populated)
- [ ] Add `skill_tags` column to `memories` table (comma-separated skill names)
- [ ] Add `skill_scores` column to `memories` table (JSON: `{"debugging": 0.82, "code-review": 0.71}`)

### Auto-tagging on store
- [ ] When `store` is called, load all skill embeddings into memory (cached, refresh on skill sync)
- [ ] Compute cosine similarity: memory embedding × each skill embedding
- [ ] If top score > 0.6 → add skill name to `skill_tags`
- [ ] Store all scores in `skill_scores` for later analysis
- [ ] MCP tool response includes `related_skills` field

### Skill-aware search
- [ ] When `search_memories` is called, optionally filter by skill tag
- [ ] Boost search results that match loaded skills (multiply score by 1.2)
- [ ] New parameter: `skill_filter` — only return memories tagged with this skill
- [ ] New parameter: `boost_skills` — boost memories related to these skills

### Skill suggestions
- [ ] New MCP tool: `suggest_skills` — given a memory or query, return top matching skills
- [ ] New MCP tool: `skill_context` — given a skill name, return all related memories
- [ ] Resource: `memory://skills/{name}/memories` — memories tagged with skill

### Context enrichment
- [ ] When searching, prepend skill description to memory text for better semantic match
- [ ] Example: memory "fixed nil pointer in handler" + skill "debugging-and-error-recovery" →
  search text becomes "[debugging-and-error-recovery] fixed nil pointer in handler"

### Evolution integration
- [ ] Track which skills are most associated with high-impact memories
- [ ] Feed skill→impact data into trait adaptation (skill X correlates with good outcomes)
- [ ] Auto-discover new skills from memory patterns (if many memories about testing → suggest test-driven-development)

### Prompt engineering for AI
- [ ] When AI stores a memory, include skill context in the prompt:
  "This memory relates to skills: debugging-and-error-recovery, code-review"
- [ ] When AI retrieves memories, include skill tags in context:
  "Memory (skill: debugging): I learned that nil checks prevent panics"
- [ ] This makes the AI more likely to apply relevant skills automatically

---

## Future Improvements (DJ to add)

<!-- Add your ideas below this line -->

