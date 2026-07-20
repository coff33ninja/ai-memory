package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/db"
	"github.com/coff33ninja/ai-memory/internal/memory"
	"github.com/coff33ninja/ai-memory/internal/persona"
	"github.com/coff33ninja/ai-memory/internal/skills"
)

type ProjectInfo struct {
	Type        string
	Lang        string
	Root        string
	RelevantAll []string // skill names relevant to any project
	Relevant    []string // skill names relevant to this specific project type
}

func detectProject() ProjectInfo {
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()

	p := ProjectInfo{
		Type: "unknown",
		Lang: "unknown",
		Root: cwd,
	}

	// Walk up to find project root
	dir := cwd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			p.Type = "go"
			p.Lang = "go"
			p.Root = dir
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			p.Type = "node"
			p.Lang = "javascript"
			p.Root = dir
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
			p.Type = "python"
			p.Lang = "python"
			p.Root = dir
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
			p.Type = "python"
			p.Lang = "python"
			p.Root = dir
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
			p.Type = "rust"
			p.Lang = "rust"
			p.Root = dir
			break
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			p.Type = "git"
			p.Root = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Also check AI_MEMORY_DIR for the ai-memory project itself
	if v := os.Getenv("AI_MEMORY_DIR"); v != "" {
		p.Root = v
	}
	_ = home

	return p
}

// Skills universally relevant to any project
var universalSkills = []string{
	"debugging-and-error-recovery",
	"follow-existing-patterns",
	"anti-phantom-symbols",
	"anti-premature-termination",
	"self-validate",
	"safe-code-modifications",
	"dont-kill-tokens",
	"skill-loader",
	"context-engineering",
	"code-review",
	"verify-and-cite",
}

// Skills mapped to project types
var projectSkillMap = map[string][]string{
	"go": {
		"toolchain-fallback",
		"performance-optimization",
		"portable-self-contained",
		"anti-global-install",
	},
	"node": {
		"performance-optimization",
		"anti-global-install",
		"ci-cd-automation",
	},
	"python": {
		"anti-global-install",
		"performance-optimization",
		"ci-cd-automation",
	},
	"rust": {
		"toolchain-fallback",
		"performance-optimization",
	},
	"git": {
		"git-workflow-conventional-commits",
		"code-review",
	},
}

func handleContextProject(skillsStore *skills.Store) (interface{}, error) {
	p := detectProject()
	all, _ := skillsStore.Catalog()

	skillMap := make(map[string]string)
	for _, sk := range all {
		skillMap[sk.Name] = sk.Description
	}

	// Build relevant list: universal + project-specific
	relevant := make([]string, 0, 20)
	seen := make(map[string]bool)

	for _, name := range universalSkills {
		if !seen[name] {
			relevant = append(relevant, name)
			seen[name] = true
		}
	}
	if projSkills, ok := projectSkillMap[p.Type]; ok {
		for _, name := range projSkills {
			if !seen[name] {
				relevant = append(relevant, name)
				seen[name] = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project type: %s (%s)\n", p.Type, p.Lang))
	sb.WriteString(fmt.Sprintf("Root: %s\n", p.Root))
	sb.WriteString(fmt.Sprintf("Total skills available: %d\n\n", len(all)))
	sb.WriteString(fmt.Sprintf("Relevant skills for this project (%d):\n", len(relevant)))
	for _, name := range relevant {
		if desc, ok := skillMap[name]; ok {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", name, desc))
		}
	}

	sb.WriteString("\nTo load a skill's full content, use search_skills or skills_search with the skill name.\n")
	sb.WriteString("Always search before answering to pull in relevant context.\n")

	return sb.String(), nil
}

func handleContextStartup(pm *persona.Manager, mem *memory.Store, skillsStore *skills.Store) (interface{}, error) {
	all, _ := skillsStore.Catalog()
	pending, _ := mem.ListPending()
	ms, _ := mem.Status()

	// Try to get active project from DB first, fall back to auto-detection
	activeProject, _ := mem.GetActiveProjectContext()
	p := detectProject() // fallback
	projectSource := "auto-detected"
	if activeProject != nil {
		p.Type = activeProject.Type
		p.Lang = activeProject.Lang
		p.Root = activeProject.Root
		projectSource = fmt.Sprintf("set by AI (project: %s)", activeProject.Name)
	}

	// Auto-restore persona if mapped to active project
	if pm != nil && activeProject != nil {
		mapping, _ := mem.GetPersonaMapping(activeProject.Name)
		if mapping != nil {
			personaObj := pm.Get(mapping.Persona)
			if personaObj != nil && pm.Active() != mapping.Persona {
				if _, err := pm.Switch(mapping.Persona); err == nil {
					fmt.Fprintf(os.Stderr, "info: auto-switched to persona %q for project %q\n", mapping.Persona, activeProject.Name)
				}
			}
		}
	}

	// Check if we're using the auto-created default persona
	isDefault := false
	if pm != nil {
		active := pm.Active()
		if p := pm.Get(active); p != nil && p.Identity == "General-purpose assistant" && active == "default" {
			isDefault = true
		}
	}

	skillMap := make(map[string]string)
	for _, sk := range all {
		skillMap[sk.Name] = sk.Description
	}

	relevant := make([]string, 0, 20)
	seen := make(map[string]bool)
	for _, name := range universalSkills {
		if !seen[name] {
			relevant = append(relevant, name)
			seen[name] = true
		}
	}
	if projSkills, ok := projectSkillMap[p.Type]; ok {
		for _, name := range projSkills {
			if !seen[name] {
				relevant = append(relevant, name)
				seen[name] = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("# AI Memory — Startup Context\n\n")

	if isDefault {
		sb.WriteString("## FIRST RUN — ONBOARD YOURSELF\n\n")
		sb.WriteString("This is a fresh ai-memory installation with a placeholder persona.\n")
		sb.WriteString("Before doing anything else, call `onboard` to create your real persona:\n\n")
		sb.WriteString("  onboard(\n")
		sb.WriteString("    name: \"<your-name>\",\n")
		sb.WriteString("    identity: \"<who you are — be specific>\",\n")
		sb.WriteString("    tone: \"<direct|formal|empathetic|casual>\",\n")
		sb.WriteString("    description: \"<what you do>\",\n")
		sb.WriteString("    skills: [\"<skill1>\", \"<skill2>\"]\n")
		sb.WriteString("  )\n\n")
		sb.WriteString("Then call `skills_sync` to clone the skills repository.\n")
		sb.WriteString("Then call `skills_index` to index them for search.\n\n")
		sb.WriteString("After onboarding, your memories, skills, and evolution will persist across sessions.\n\n")
		sb.WriteString("---\n\n")
	}

	sb.WriteString(fmt.Sprintf("Project: %s (%s) @ %s [%s]\n", p.Type, p.Lang, p.Root, projectSource))
	sb.WriteString(fmt.Sprintf("Memory: %d entries, %d pending review\n", ms.MemoryCount, ms.PendingCount))
	sb.WriteString(fmt.Sprintf("Skills: %d indexed, %d relevant to this project\n\n", len(all), len(relevant)))

	sb.WriteString("IMPORTANT: If the project above is wrong, call `set_project_context` with the correct project.\n")
	sb.WriteString("Example: set_project_context(name: \"my-app\", root: \"E:\\\\code\\\\my-app\", type: \"go\", lang: \"go\")\n\n")
	sb.WriteString("If the active project root is different from your current working directory, navigate to it first.\n\n")

	// User profile
	profiles, _ := mem.ListUserProfile()
	if len(profiles) > 0 {
		sb.WriteString("## User Profile\n")
		for _, p := range profiles {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", p.Field, p.Value))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("## User Profile\n  No user profile data yet. Learn about the user from conversations and use `store_user_profile` to remember their name, interests, and preferences.\n\n")
	}

	// Backup status
	backups, _ := mem.ListBackups(1)
	if len(backups) > 0 {
		sb.WriteString(fmt.Sprintf("## Backup\n  Last backup: %s (%s)\n\n", backups[0].Timestamp, backups[0].Provider))
	} else {
		sb.WriteString("## Backup\n  No backups configured. Use `backup_config` to set up backup to GitHub, local, or cloud.\n\n")
	}

	if len(pending) > 0 {
		sb.WriteString("## Pending Reviews\n")
		for _, m := range pending {
			sb.WriteString(fmt.Sprintf("  #%d: %s\n", m.ID, m.Experience))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Relevant Skills\n")
	for _, name := range relevant {
		if desc, ok := skillMap[name]; ok {
			sb.WriteString(fmt.Sprintf("  %s — %s\n", name, desc))
		}
	}

	sb.WriteString("\n---\n")
	sb.WriteString("BEFORE answering any question, call `search` with the user's query to find\n")
	sb.WriteString("relevant skills and memories. This pulls in context you need.\n")
	sb.WriteString("When you learn something important, call `store` to save it.\n")

	return sb.String(), nil
}

func handleContextInjectPrompt() (interface{}, error) {
	text := `# System Prompt: Proactive Context Search

You have access to a persistent memory and skills system via MCP tools.

## MANDATORY BEHAVIOR

1. **Before answering ANY question**, call the ` + "`search`" + ` tool with the user's query (or a condensed version of it) to find relevant skills and memories. This is not optional.

2. **When you learn something important** during a session — a gotcha, a decision, a pattern — call ` + "`store`" + ` to save it for future sessions.

3. **After completing a task**, briefly reflect: did you learn something worth storing?

## SEARCH STRATEGY

- For coding questions: search for the language/framework/error message
- For architecture questions: search for the relevant patterns
- For debugging: search for the error or symptom
- For "how do I" questions: search for the tool or concept

## SKILL LOADING

The ` + "`context://project`" + ` resource is auto-loaded and tells you which skills are relevant to your current project. Use ` + "`search_skills`" + ` to load a specific skill's full content when needed.

## MEMORY DISCIPLINE

- Store gotchas (things that tripped you up)
- Store decisions (why you chose X over Y)
- Store patterns (reusable approaches)
- Don't store trivial one-off facts

You are not just answering questions — you are building a knowledge base that persists across sessions.`

	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": map[string]interface{}{"type": "text", "text": text},
			},
		},
	}, nil
}

// keep unused import check happy
var _ = db.Open

func handleSkillsUsage(mem *memory.Store) (interface{}, error) {
	usages, err := mem.ListSkillUsage(30)
	if err != nil {
		return nil, err
	}
	if len(usages) == 0 {
		return "No skill usage recorded yet. Use store_skill_usage to record when skills are used.", nil
	}

	// Build skill -> companion frequency map
	type combo struct {
		skill    string
		companion string
		count    int
	}
	comboMap := make(map[string]map[string]int)
	for _, u := range usages {
		if comboMap[u.Skill] == nil {
			comboMap[u.Skill] = make(map[string]int)
		}
		if u.WithSkills != "" {
			for _, c := range strings.Split(u.WithSkills, ",") {
				c = strings.TrimSpace(c)
				if c != "" {
					comboMap[u.Skill][c]++
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skill Usage Patterns (last %d sessions):\n\n", len(usages)))

	// Group by skill
	skillOrder := make([]string, 0)
	seen := make(map[string]bool)
	for _, u := range usages {
		if !seen[u.Skill] {
			skillOrder = append(skillOrder, u.Skill)
			seen[u.Skill] = true
		}
	}

	for _, skill := range skillOrder {
		var skillUsages []db.SkillUsage
		for _, u := range usages {
			if u.Skill == skill {
				skillUsages = append(skillUsages, u)
			}
		}
		sb.WriteString(fmt.Sprintf("## %s (used %d times)\n", skill, len(skillUsages)))
		for _, u := range skillUsages[:min(3, len(skillUsages))] {
			sb.WriteString(fmt.Sprintf("  %s — %s [%s]\n", u.Date, u.Context, u.Outcome))
		}
		if companions, ok := comboMap[skill]; ok && len(companions) > 0 {
			sb.WriteString("  Frequently paired with: ")
			first := true
			for c, count := range companions {
				if !first {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s(%d)", c, count))
				first = false
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleSkillUsageRecorderPrompt() (interface{}, error) {
	text := `# Skill Usage Recording

When you load and use a skill, record it so future sessions know which skills work well together.

## WHEN TO RECORD

After using a skill for a task, call ` + "`store_skill_usage`" + ` with:

- **skill**: The skill name (e.g. "debugging-and-error-recovery")
- **context**: What you were doing (e.g. "fixed ORT crash in Go MCP server")
- **with_skills**: Other skills loaded at the same time (e.g. "anti-phantom-symbols, self-validate")
- **outcome**: How it went — "effective", "partial", or "failed"

## WHY

This builds a graph of skill combinations. When similar tasks come up, you can see:
- Which skills were used for similar problems
- Which companion skills worked well together
- What outcomes resulted

## EXAMPLE

If you used debugging-and-error-recovery alongside self-validate and anti-phantom-symbols to fix a crash:

` + "```" + `
store_skill_usage(
  skill: "debugging-and-error-recovery",
  context: "Fixed ORT session.Run crash caused by thread affinity",
  with_skills: "self-validate, anti-phantom-symbols",
  outcome: "effective"
)
` + "```" + `

Don't record trivial usage. Only record when a skill materially contributed to solving a problem.`

	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": map[string]interface{}{"type": "text", "text": text},
			},
		},
	}, nil
}
