# Changelog

All notable changes to ai-memory will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
