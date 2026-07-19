# ADR-002: Go Rewrite — ai-memory

## Status

Proposed

## Context

ai-personality MCP server has two implementations (TypeScript + Python) that conflate personality with memory. The real value is:
1. **Memory** — persistent experience/lesson logging with semantic search
2. **Skills** — procedural knowledge recall alongside memory

Personality files (identity, traits, values, rules, relationships) are client concerns, not persistent data. The embedding infrastructure (SQLite + ONNX) should unify memory and skills into a single searchable store.

## Decision

Rewrite in Go as `ai-memory-server`. Drop personality. Keep memory + skills + full RAG pipeline. One binary, one SQLite database, one embedding model.

## What Changes

| Component | ai-personality | ai-memory |
|-----------|---------------|-----------|
| Identity/traits/values/rules | Personality files | Removed |
| Memories | markdown + YAML in files | SQLite table + embeddings |
| Skills | git clone + filesystem search | SQLite indexed + embeddings |
| Vector search | personality files only | unified memory + skills |
| Database | `rag/vector.db` (chunks only) | `memory.db` (memories + skills + skill_files + meta) |
| Embeddings | WASM (TS) / fastembed (Python) | ONNX via onnxruntime-go, model on GitHub |
| Distribution | npm / PyPI | GitHub Releases (single binary) |
| Client setup | 15-client generator | Just the MCP server command |

## What's Kept

- MCP protocol (stdio, JSON-RPC 2.0)
- Memory entry format (experience, lesson, impact, tags)
- Reflection lifecycle (store → review → apply/dismiss)
- Skills git repo + SKILL.md format
- ONNX embedding model (all-MiniLM-L6-v2, 384 dims)
- Cross-reference validation (now between memories and skills)
- File watcher for auto-reindex

## What's Dropped

- Personality files (identity, traits, values, rules, relationships)
- Client setup generator
- TypeScript implementation
- Python implementation
- npm/PyPI distribution

## Rationale

1. **Correct abstraction**: Memory + skills are what persist. Personality is a client-side concern.
2. **Single binary**: ~25-30MB, no runtime dependency, cross-compiled via Zig CGO.
3. **Unified RAG**: One search across memories AND skills. The AI recalls what it knows and what it learned.
4. **Simpler mental model**: "Store memories, recall skills" vs "Evolve personality through reflection cycles."
5. **Same SQLite**: Pure Go driver (`modernc.org/sqlite`) or CGO via Zig for performance.
6. **Model on GitHub**: Auto-download on first run, no embedding in binary.

## Consequences

- Users get one binary that does everything
- Existing ai-personality memories can be imported (same entry format)
- Skills work identically (same git repo, same SKILL.md)
- No more personality file editing — AI behavior is the client's job
- Database grows with memories (expected: thousands of entries over time)
