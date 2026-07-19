package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/db"
	"github.com/coff33ninja/ai-memory/internal/embedding"
	"github.com/coff33ninja/ai-memory/internal/memory"
	"github.com/coff33ninja/ai-memory/internal/rag"
	"github.com/coff33ninja/ai-memory/internal/skills"
)

func handleStore(mem *memory.Store, sharedMem *memory.Store, args map[string]interface{}) (interface{}, error) {
	experience, _ := args["experience"].(string)
	lesson, _ := args["lesson"].(string)
	if experience == "" || lesson == "" {
		return nil, fmt.Errorf("experience and lesson are required")
	}
	var tags []string
	if t, ok := args["tags"].([]interface{}); ok {
		for _, v := range t {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
	}
	scope, _ := args["scope"].(string)

	// If scope is "shared" and shared store is available, store there
	if scope == "shared" && sharedMem != nil {
		m, err := sharedMem.Store(experience, lesson, tags, "shared")
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Memory %d stored [shared]. Shared memories are visible to all personas.", m.ID), nil
	}

	// Default: store in current persona's DB
	m, err := mem.Store(experience, lesson, tags, scope)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Memory %d stored [%s]. %d total, %d pending.", m.ID, m.Scope, countAll(mem), countPending(mem)), nil
}

func handleReview(mem *memory.Store) (interface{}, error) {
	pending, err := mem.ListPending()
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return "No pending memories to review.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d pending memory(ies):\n\n", len(pending)))
	for _, m := range pending {
		sb.WriteString(fmt.Sprintf("#%d — %s\n", m.ID, m.Date))
		sb.WriteString(fmt.Sprintf("  Experience: %s\n", m.Experience))
		sb.WriteString(fmt.Sprintf("  Lesson: %s\n", m.Lesson))
		if len(m.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(m.Tags, ", ")))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Review each, then call apply(id) or dismiss(id).")
	return sb.String(), nil
}

func handleApply(mem *memory.Store, args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	if err := mem.Apply(int64(id)); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Memory %d marked as applied.", int64(id)), nil
}

func handleDismiss(mem *memory.Store, args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	if err := mem.Dismiss(int64(id)); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Memory %d marked as dismissed.", int64(id)), nil
}

func handleStatus(mem *memory.Store, skills *skills.Store) (interface{}, error) {
	ms, _ := mem.Status()
	sc, _ := skills.Status()
	return fmt.Sprintf(
		"Memories: %d total, %d pending review\nSkills: %d indexed\nTotal evolutions: %d",
		ms.MemoryCount, ms.PendingCount, sc, ms.TotalEvolutions,
	), nil
}

func handleSearch(searcher *rag.Searcher, emb *embedding.Embedder, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := 5
	if v, ok := args["topK"].(float64); ok {
		topK = int(v)
	}
	queryEmb, err := emb.Compute(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := searcher.SearchAll(queryEmb, topK)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return fmt.Sprintf("No results for %q.", query), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d result(s) for %q:\n\n", len(results), query))
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("[%s #%d] %s (score: %.3f)\n%s\n\n", r.Type, r.ID, r.Title, r.Score, r.Content))
	}
	return sb.String(), nil
}

func handleSearchMemories(searcher *rag.Searcher, emb *embedding.Embedder, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := 5
	if v, ok := args["topK"].(float64); ok {
		topK = int(v)
	}
	queryEmb, err := emb.Compute(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := searcher.SearchMemories(queryEmb, topK)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return fmt.Sprintf("No memories matching %q.", query), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d memory result(s) for %q:\n\n", len(results), query))
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("#%d %s (score: %.3f)\n%s\n\n", r.ID, r.Title, r.Score, r.Content))
	}
	return sb.String(), nil
}

func handleSearchSkills(searcher *rag.Searcher, emb *embedding.Embedder, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := 5
	if v, ok := args["topK"].(float64); ok {
		topK = int(v)
	}
	queryEmb, err := emb.Compute(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	results, err := searcher.SearchSkills(queryEmb, topK)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return fmt.Sprintf("No skills matching %q.", query), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d skill result(s) for %q:\n\n", len(results), query))
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("#%d %s (score: %.3f)\n%s\n\n", r.ID, r.Title, r.Score, r.Content))
	}
	return sb.String(), nil
}

func handleReindex(searcher *rag.Searcher) (interface{}, error) {
	memCount, skillCount, err := searcher.Reindex()
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Reindexed %d memories and %d skills.", memCount, skillCount), nil
}

func handleSkillsSync(skills *skills.Store) (interface{}, error) {
	msg, err := skills.Sync()
	return msg, err
}

func handleSkillsSearch(skills *skills.Store, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	results, err := skills.SearchKeyword(query)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return fmt.Sprintf("No skills matching %q.", query), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d skill(s) matching %q:\n\n", len(results), query))
	for _, sk := range results {
		sb.WriteString(fmt.Sprintf("- %s: %s (%d files)\n", sk.Name, sk.Description, sk.FileCount))
	}
	return sb.String(), nil
}

func handleSkillsIndex(skills *skills.Store) (interface{}, error) {
	n, err := skills.Index()
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Indexed %d skills.", n), nil
}

func handleStoreSkillUsage(mem *memory.Store, args map[string]interface{}) (interface{}, error) {
	skill, _ := args["skill"].(string)
	context, _ := args["context"].(string)
	if skill == "" || context == "" {
		return nil, fmt.Errorf("skill and context are required")
	}
	withSkills, _ := args["with_skills"].(string)
	outcome, _ := args["outcome"].(string)
	u, err := mem.StoreSkillUsage(skill, context, withSkills, outcome)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Skill usage recorded: %s used in %q (id: %d)", u.Skill, u.Context, u.ID), nil
}

func handleListSkillUsage(mem *memory.Store, args map[string]interface{}) (interface{}, error) {
	limit := 20
	if v, ok := args["limit"].(float64); ok {
		limit = int(v)
	}
	usages, err := mem.ListSkillUsage(limit)
	if err != nil {
		return nil, err
	}
	if len(usages) == 0 {
		return "No skill usage recorded yet.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Last %d skill usage(s):\n\n", len(usages)))
	for _, u := range usages {
	 companions := ""
		if u.WithSkills != "" {
			companions = fmt.Sprintf(" [+ %s]", u.WithSkills)
		}
		sb.WriteString(fmt.Sprintf("#%d %s — %s%s\n  Context: %s\n  Outcome: %s\n\n", u.ID, u.Date, u.Skill, companions, u.Context, u.Outcome))
	}
	return sb.String(), nil
}

func handleSummary(mem *memory.Store, skills *skills.Store) (interface{}, error) {
	ms, _ := mem.Status()
	pending, _ := mem.ListPending()
	catalog, _ := skills.Catalog()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Memories: %d total, %d pending\n", ms.MemoryCount, ms.PendingCount))
	sb.WriteString(fmt.Sprintf("Skills: %d available\n\n", len(catalog)))

	if len(pending) > 0 {
		sb.WriteString("Pending reviews:\n")
		for _, m := range pending {
			sb.WriteString(fmt.Sprintf("  #%d: %s\n", m.ID, m.Experience))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Available skills:\n")
	for _, sk := range catalog {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", sk.Name, sk.Description))
	}
	return sb.String(), nil
}

func handleAll(mem *memory.Store, skills *skills.Store, d *db.DB) (interface{}, error) {
	ms, _ := mem.Status()
	allMem, _ := mem.ListAll()
	catalog, _ := skills.Catalog()

	return map[string]interface{}{
		"stats":    ms,
		"memories": allMem,
		"skills":   catalog,
	}, nil
}

func handleMemoryPrompt(mem *memory.Store, skills *skills.Store) (interface{}, error) {
	pending, _ := mem.ListPending()
	allSkills, _ := skills.Catalog()
	p := detectProject()

	skillMap := make(map[string]string)
	for _, sk := range allSkills {
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
	sb.WriteString("You have persistent memory and skills. Here is your context:\n\n")
	sb.WriteString(fmt.Sprintf("Project: %s (%s)\n\n", p.Type, p.Lang))

	if len(pending) > 0 {
		sb.WriteString(fmt.Sprintf("## %d Pending Memories\n", len(pending)))
		for _, m := range pending {
			sb.WriteString(fmt.Sprintf("- %s: %s (lesson: %s)\n", m.Date, m.Experience, m.Lesson))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("## Relevant Skills (%d of %d total)\n", len(relevant), len(allSkills)))
	for _, name := range relevant {
		if desc, ok := skillMap[name]; ok {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
		}
	}

	sb.WriteString("\nIMPORTANT: Before answering, call `search` with the user's question to pull in relevant context.\n")
	sb.WriteString("When you learn something important, call `store` to save it.\n")
	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "assistant",
				"content": map[string]interface{}{"type": "text", "text": sb.String()},
			},
		},
	}, nil
}

func handleReflectPrompt() (interface{}, error) {
	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "assistant",
				"content": map[string]interface{}{
					"type": "text",
					"text": "Reflect on the session:\n1. What happened?\n2. What did you learn?\n3. Should any memory be stored?\n\nIf yes, call store(experience, lesson, tags). If nothing significant, skip.",
				},
			},
		},
	}, nil
}

func countAll(mem *memory.Store) int {
	all, _ := mem.ListAll()
	return len(all)
}

func countPending(mem *memory.Store) int {
	pending, _ := mem.ListPending()
	return len(pending)
}
