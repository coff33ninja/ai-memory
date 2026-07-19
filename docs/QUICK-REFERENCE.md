# ai-personality Quick Reference

## Files

```
~/.ai-personality/personality/{identity,traits,values,rules,memories,relationships}.md
~/.ai-personality/rag/vector.db
~/.ai-personality/skills/<name>/SKILL.md
```

## Frontmatter

```yaml
---
type: identity
version: 1
lastUpdated: 2025-01-15
evolution: 3
crossReferences:
  - traits.md#communication-style -- "description"
---
```

## Memory Entry

```yaml
- date: 2025-01-15
  experience: what happened
  lesson: what was learned
  impact: under review | applied | dismissed
  affectedFiles:
    - traits.md
```

## DB Schema

```sql
chunks(id INTEGER PK, filename TEXT, heading TEXT, content TEXT, embedding BLOB, type TEXT)
```

Embeddings: float32 array stored as raw bytes (384 dims for MiniLM).

## Tools

| Tool | Input | Side Effect |
|------|-------|-------------|
| `reflect` | experience, lesson, affectedFiles? | Appends to memories.md |
| `evolve` | — | Lists pending entries |
| `status` | — | Shows stats |
| `validate` | — | Checks cross-refs |
| `search_personality` | query, topK? | Vector search all files |
| `search_memories` | query, topK? | Vector search memories.md |
| `reindex` | — | Rebuilds vector DB |
| `skills_search` | query | Keyword search skills |
| `skills_sync` | — | Git pull skills repo |
| `validate_cross_repo` | — | Checks personality↔skills refs |
| `setup_client` | client, apply? | Generates client config |

## Resources

```
personality://identity|traits|values|rules|memories|relationships|summary|all
personality://file/{filename}
skills://catalog
skills://{name}
skills://file/{name}/{filename}
```

## Embeddings

- TS: `Xenova/all-MiniLM-L6-v2` (384 dims, WASM)
- Python: `BAAI/bge-small-en-v1.5` (384 dims, native ONNX)
- Similarity: cosine, brute-force scan
- Chunk size: by `##` heading, truncated to 512 chars
