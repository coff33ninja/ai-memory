package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/evolution"
	"github.com/coff33ninja/ai-memory/internal/persona"
)

func handleLogInteraction(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	summary, _ := args["summary"].(string)
	scoreF, _ := args["outcome_score"].(float64)
	if summary == "" || scoreF == 0 {
		return nil, fmt.Errorf("summary and outcome_score (1-5) are required")
	}
	score := int(scoreF)
	if score < 1 || score > 5 {
		return nil, fmt.Errorf("outcome_score must be 1-5")
	}

	skillsUsed, _ := args["skills_used"].(string)
	toneUsed, _ := args["tone_used"].(string)
	personaName := pm.Active()

	if err := eng.LogInteraction(personaName, summary, score, skillsUsed, toneUsed); err != nil {
		return nil, err
	}

	count, _ := eng.Tracker().InteractionCount(personaName)
	avg, _ := eng.Tracker().AverageScore(personaName, 10)

	return fmt.Sprintf("Interaction logged for %q. Score: %d/5. Total: %d interactions, avg: %.1f", personaName, score, count, avg), nil
}

func handleEvolve(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	result, err := eng.Evolve(personaName)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Evolution complete for %q:\n\n", personaName))

	if result.MemoryConsolidated != nil {
		mc := result.MemoryConsolidated
		sb.WriteString(fmt.Sprintf("Memory consolidation:\n"))
		sb.WriteString(fmt.Sprintf("  Merged: %d similar memories\n", mc.Merged))
		sb.WriteString(fmt.Sprintf("  Elevated: %d frequent memories\n", mc.Elevated))
		sb.WriteString(fmt.Sprintf("  Pruned: %d old dismissed\n", mc.Pruned))
		sb.WriteString(fmt.Sprintf("  Patterns: %d created\n", mc.PatternsCreated))
		sb.WriteString("\n")
	}

	if result.TraitsAdapted != nil {
		ta := result.TraitsAdapted
		if ta.ToneChanged {
			sb.WriteString(fmt.Sprintf("Tone adapted: %q → %q\n", ta.ToneBefore, ta.ToneAfter))
			sb.WriteString(fmt.Sprintf("  Reason: %s\n", ta.Reason))
		}
		if len(ta.SkillsAdded) > 0 {
			sb.WriteString(fmt.Sprintf("Skills added: %s\n", strings.Join(ta.SkillsAdded, ", ")))
		}
		if !ta.ToneChanged && len(ta.SkillsAdded) == 0 {
			sb.WriteString("No trait changes needed yet.\n")
		}
	}

	return sb.String(), nil
}

func handleConsolidate(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	result, err := eng.Consolidator().Consolidate(personaName)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Consolidation complete:\n  Merged: %d similar memories\n  Elevated: %d frequent memories\n  Pruned: %d old dismissed\n  Patterns: %d created",
		result.Merged, result.Elevated, result.Pruned, result.PatternsCreated), nil
}

func handleDiscoverSkills(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	skills, err := eng.DiscoverSkills(personaName)
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return "No new skills discovered yet. Keep interacting to build usage patterns.", nil
	}

	p := pm.Get(personaName)
	newSkills := append(p.Skills, skills...)
	pm.Update(personaName, "", "", "", newSkills)

	return fmt.Sprintf("Discovered %d new skills for %q:\n%s\n\nSkills updated: %v", len(skills), personaName, strings.Join(skills, "\n"), newSkills), nil
}

func handleEvolutionHistory(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	personaName := pm.Active()
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	entries, err := eng.EvolutionHistory(personaName, limit)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return "No evolution history yet.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Evolution history for %q:\n\n", personaName))
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", e.CreatedAt[:10], e.Trigger, e.WhatChanged))
		if e.Before != "" || e.After != "" {
			sb.WriteString(fmt.Sprintf("  %q → %q\n", e.Before, e.After))
		}
	}

	return sb.String(), nil
}

func handleGetEvolvedRules(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	return eng.GetEvolvedRules(personaName)
}

func handleInteractionStats(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	stats, err := eng.InteractionStats(personaName)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Interaction stats for %q:\n\n", personaName))
	sb.WriteString(fmt.Sprintf("Total interactions: %v\n", stats["total_interactions"]))
	sb.WriteString(fmt.Sprintf("Average score: %v\n", stats["average_score"]))

	if tonePerf, ok := stats["tone_performance"].(map[string]float64); ok && len(tonePerf) > 0 {
		sb.WriteString("\nTone performance:\n")
		for tone, score := range tonePerf {
			sb.WriteString(fmt.Sprintf("  %s: %.1f/5\n", tone, score))
		}
	}

	if skillPerf, ok := stats["skill_performance"].(map[string]float64); ok && len(skillPerf) > 0 {
		sb.WriteString("\nSkill performance:\n")
		for skill, score := range skillPerf {
			sb.WriteString(fmt.Sprintf("  %s: %.1f/5\n", skill, score))
		}
	}

	unresolved, _ := eng.Tracker().UnresolvedGapCount(personaName)
	if unresolved > 0 {
		sb.WriteString(fmt.Sprintf("\n⚠ %d unresolved tool gaps (run list_tool_gaps to see)", unresolved))
	}

	return sb.String(), nil
}

func handleLogToolGap(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	need, _ := args["need"].(string)
	context, _ := args["context"].(string)
	suggested, _ := args["suggested"].(string)
	if need == "" || context == "" {
		return nil, fmt.Errorf("need and context are required")
	}

	personaName := pm.Active()
	if err := eng.Tracker().LogToolGap(personaName, need, context, suggested); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Tool gap logged for %q: %s\nUser can run `list_tool_gaps` to see what tools to expose.", personaName, need), nil
}

func handleListToolGaps(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	personaName := pm.Active()
	includeResolved, _ := args["include_resolved"].(bool)

	gaps, err := eng.Tracker().ToolGaps(personaName, includeResolved)
	if err != nil {
		return nil, err
	}
	if len(gaps) == 0 {
		return "No tool gaps recorded. The AI has everything it needs (for now).", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tool gaps for %q:\n\n", personaName))
	for _, g := range gaps {
		status := "❌"
		if g.Resolved == 1 {
			status = "✅"
		}
		sb.WriteString(fmt.Sprintf("[%s] #%d — %s\n", status, g.ID, g.Need))
		sb.WriteString(fmt.Sprintf("    Context: %s\n", g.Context))
		if g.Suggested != "" {
			sb.WriteString(fmt.Sprintf("    Suggested: %s\n", g.Suggested))
		}
		sb.WriteString(fmt.Sprintf("    Date: %s\n\n", g.CreatedAt[:10]))
	}

	return sb.String(), nil
}

func handleResolveToolGap(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	idF, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	id := int64(idF)

	if err := eng.Tracker().ResolveToolGap(id); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Tool gap #%d marked as resolved.", id), nil
}

func handleEvolutionStats(pm *persona.Manager, eng *evolution.Engine) (interface{}, error) {
	personaName := pm.Active()
	stats, err := eng.InteractionStats(personaName)
	if err != nil {
		return nil, err
	}
	stats["persona"] = personaName
	return stats, nil
}

func handleEvolutionPrompt(pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	p := pm.Get(personaName)

	text := fmt.Sprintf(`# Evolution Loop — %s

After EVERY interaction, you MUST:

1. **Log the interaction** — call log_interaction with:
   - summary: what happened in 1 sentence
   - outcome_score: 1-5 (1=failure, 3=okay, 5=excellent)
   - skills_used: which skills you used (comma-separated)
   - tone_used: your tone (formal, casual, concise, etc.)

2. **Log tool gaps** — if you needed a tool you don't have, call log_tool_gap:
   - need: what you needed to do
   - context: what you were trying to accomplish
   - suggested: MCP server name if you know one

3. **Learn tools** — after using a tool, call log_tool_knowledge:
   - tool_name: the tool you used
   - how_to_use: how it works
   - what_works: patterns that produce good results
   - what_fails: common mistakes
   - params: parameter guide
   - examples: example invocations

4. **Save recipes** — for multi-step patterns, call log_tool_recipe:
   - tool_name, recipe_name, steps, use_case

5. **Check knowledge** — before using a tool, call get_tool_knowledge to see what you know

6. **Periodically evolve** — after every 10 interactions, call evolve:
   - Consolidate similar memories
   - Adapt your traits based on what works
   - Discover new skills

7. **Check evolved rules** — call get_evolved_rules to see what you've learned

Your persona: %s
Tone: %s
Skills: %v

Remember: You evolve through YOUR OWN experience. Every interaction teaches you something. Log it.
Log tool gaps so the user can expose you to new capabilities.
Build your own tool manual — you'll thank yourself later.
`, personaName, p.Identity, p.Tone, p.Skills)

	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": map[string]interface{}{"type": "text", "text": text},
			},
		},
	}, nil
}
