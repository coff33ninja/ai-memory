# Changelog

All notable changes to ai-memory will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.7] - 2026-07-20

### Added
- Evolution engine test suite: 62 tests across tracker, consolidator, adapter, engine, and types
- `TestSimilarMemories`, `TestMergeMemories` — similarity detection and merge verification
- `TestConsolidateMerge`, `TestConsolidateElevate`, `TestConsolidatePrune`, `TestConsolidateCreatesPatterns`, `TestConsolidateDoesNotDuplicatePatterns` — full consolidation pipeline
- `TestAdaptTraitsToneChange`, `TestAdaptTraitsSkillsAdded` — trait adaptation with persona updates
- `TestShouldEvolveInteractionThreshold`, `TestShouldEvolveMemoryThreshold` — evolution trigger logic
- `TestEvolve`, `TestLogInteractionAutoEvolve` — end-to-end evolution cycles
- `TestGetEvolvedRulesWithPatterns`, `TestDiscoverSkills` — rule evolution and skill discovery
- `TestLogToolKnowledge`, `TestLogToolRecipe`, `TestLogToolError`, `TestUpsertAndGetMCPServer` — tool ecosystem tracking
- `push-and-release.ps1` documented as dev tool requiring OpenCode Desktop in README

### Fixed
- `SimilarMemories` scan mismatch — SELECT'd 4 columns but scanned 3, silently skipping all rows; now scans all 4 and compares `experience + lesson` combined
- Pattern dedup LIKE format — `createPatterns` produces `'Pattern observed N times: tag'` but dedup checked for `'Pattern: %(tag)%%'`; fixed to match actual format

## [0.1.6] - 2026-07-20

### Added
- Backup/restore system with 11 providers: local, Google Drive, OneDrive, Dropbox, Box, pCloud, iCloud, MEGA, Nextcloud, Syncthing, GitHub (private repos via `gh` CLI)
- `backup_config`, `backup`, `restore`, `backup_status`, `list_backup_drives` tools
- `backup://status` resource
- Cloud drive detection scans all 26 drive letters for filesystem markers
- Google Drive detected via drive letter `My Drive` subfolder, home folder mount, or registry admin policy mount points
- OneDrive detected via `OneDrive` env var and registry `UserFolder`
- SMB/network share detection via `GetVolumeInformationW` UNC prefix check
- Persona mapping: `map_persona`, `unmap_persona`, `list_persona_mappings` tools
- README.md inside backup archives with persona identities, shared memories, user profile, project contexts, persona mappings, and recent skill usage
- `backups.json` tracks all backups with SHA-256 checksums for restore validation
- Backup pruning: keeps last 3 backups, deletes older from disk or GitHub repo
- Restore validates checksum before extracting
- Auto-backup goroutine runs in background when enabled
- CI test job — `go test` now runs in CI (was only local)
- CI security job — `govulncheck ./...` scans for known vulnerabilities on every push
- Zig download caching in CI/CD workflows (saves ~45s per run)

### Changed
- `backup_config` accepts `provider`, `local_path`, `auto_backup`, `interval_hours` only
- CI pipeline restructured: lint → test + security (parallel) → build
- Release workflow uses `gh release create` with retry (replaced `softprops/action-gh-release`)
- Module Maintenance workflow uses `gh pr create` (replaced raw REST API that failed with 403)

### Fixed
- `pm.Switch()` return value not handled in project handlers
- Module Maintenance workflow failing — `GH_TOKEN` lacked PR creation permission, switched to `gh` CLI
- Release workflow failing on transient GitHub API errors — added 5-attempt retry with backoff
- Dependabot ignore reason documented for `onnxruntime_go` (CGO/Zig CC compatibility)

## [0.1.5] - 2026-07-20

### Added
- Project context system: persistent project tracking across sessions via `project_contexts` table
- `set_project_context` tool — AI calls this at session start with its working directory to tell the memory system which project it's in
- `get_project_context` tool — read the active project
- `list_project_contexts` tool — list all stored projects
- `project://active` resource — always shows the current project
- Startup context now reads active project from DB (not just os.Getwd), with source indicator
- Persona-startup prompt shows project context section with navigation instructions
- AI instructed to navigate to project root if it differs from current working directory

## [0.1.4] - 2026-07-20

### Added
- Expanded design philosophy section in README: core principles, architectural patterns (cascading lookup, SQLite universal store, embedding-on-write, auto-consolidation, graceful degradation, panic recovery, file-based logging, CI/CD enforcement), and development workflow documentation
- Badge labels at top of README (Go version, release, CI, Windows, MCP, last commit, PRs welcome, docs)
- Cross-reference to companion project go-mcp-computer-use
- AI fun comments footer (honest slop edition)

## [0.1.3] - 2026-07-19

### Fixed
- Memories now generate embeddings immediately on store — search finds them right away instead of requiring manual `reindex`

## [0.1.2] - 2026-07-19

### Added
- Persona greeting system: each persona can have a greeting keyword that triggers auto-switching
- `greeting` field on Persona struct (stored in personas.json)
- `FindPersonaByGreeting` method for case-insensitive greeting matching
- `onboard` tool now accepts `greeting` parameter
- `persona://active` and `switch_persona` display greeting keyword
- `persona-startup` prompt lists all personas with their greetings for AI-driven switching
- When user says "hello Akeno", AI detects greeting and calls `switch_persona`

## [0.1.1] - 2026-07-19

### Added
- User profile system: store, get, list, delete user profile fields
- `user_profiles` table with field/value/source/confidence schema
- 4 new MCP tools: `store_user_profile`, `get_user_profile`, `list_user_profile`, `delete_user_profile`
- `user://profile` resource for reading user profile as text
- User profile auto-included in `context://startup` and `persona-startup` prompt
- `persona-startup` prompt added (7th prompt)

## [0.1.0] - 2026-07-19

### Added
- SQLite-based memory storage (experience, lesson, impact, tags, scope)
- Memory review workflow (store → review → apply/dismiss)
- Skills subsystem: git clone, sync, index, search from `coff33ninja/ai-skills`
- RAG vector search with ONNX embeddings (`all-MiniLM-L6-v2`, 384 dims)
- ONNX Runtime auto-download on first run (`%APPDATA%\ai-memory\lib\`)
- Multi-persona architecture with per-persona SQLite DBs
- Shared memory scope across personas
- Persona onboarding and switching
- AI self-evolution engine (interaction tracking, memory consolidation, trait adaptation)
- Tool proficiency learning (knowledge base, recipes, success/fail tracking)
- Tool gap discovery and MCP server registry
- MCP error tracking with `report_issue` capability detection
- 40 MCP tools, 12 resources, 6 prompts
- Smart skill loading with project type detection
- Context resources for project-aware AI sessions
- Skill usage tracking and pattern analysis
- Build scripts (PowerShell) with Zig cc + CGO
- CI/CD pipeline (GitHub Actions): lint, build, release, auto-tag, module maintenance
- Install script with CGO trigger file concept
- Push-and-release script for versioned deployments
- Comprehensive documentation (README, ARCHITECTURE, QUICK-REFERENCE, EVOLUTION)
- gen-tools-doc.go auto-generates tools.md from source
- CGO_TRIGGER file for CGO build gating
- Windows app icon (6 sizes: 16-256px)
- gen-icons.ps1 generates .syso for Windows exe embedding
- Auto-create default persona on first run (no empty state)
- `context://startup` and `persona-startup` prompt detect default persona and guide AI to onboard
- Getting Started section in README: personas, first run, memories, skills, evolution
