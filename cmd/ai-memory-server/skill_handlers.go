package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/embedding"
	"github.com/coff33ninja/ai-memory/internal/memory"
)

func handleSuggestSkills(store *memory.Store, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	topK := 5
	if k, ok := args["top_k"].(float64); ok {
		topK = int(k)
	}

	emb, err := embedding.InitEmbedder()
	if err != nil {
		return nil, fmt.Errorf("init embedder: %w", err)
	}
	defer embedding.CloseEmbedder()

	queryEmb, err := emb.Compute(query)
	if err != nil {
		return nil, fmt.Errorf("compute query embedding: %w", err)
	}

	results := store.SuggestSkillsForQuery(queryEmb, topK)
	if len(results) == 0 {
		return "No relevant skills found for this query.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Top %d skills for: %s\n\n", len(results), query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s (score: %.2f)\n", i+1, r.Name, r.Score))
		if r.Description != "" {
			desc := r.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", desc))
		}
	}
	return sb.String(), nil
}

func handleSkillContext(store *memory.Store, args map[string]interface{}) (interface{}, error) {
	skill, _ := args["skill"].(string)
	if skill == "" {
		return nil, fmt.Errorf("skill is required")
	}
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	memories, err := store.GetMemoriesBySkill(skill, limit)
	if err != nil {
		return nil, fmt.Errorf("get skill context: %w", err)
	}
	if len(memories) == 0 {
		return fmt.Sprintf("No memories tagged with skill %q.", skill), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Memories related to skill %q:\n\n", skill))
	for i, m := range memories {
		score := m.SkillScores[skill]
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (relevance: %.2f)\n", i+1, m.Date, m.Experience, score))
		sb.WriteString(fmt.Sprintf("   Lesson: %s\n\n", m.Lesson))
	}
	return sb.String(), nil
}

func handleSkillMemoriesResource(store *memory.Store, skillName string) (string, error) {
	memories, err := store.GetMemoriesBySkill(skillName, 20)
	if err != nil {
		return "", err
	}
	if len(memories) == 0 {
		return fmt.Sprintf("No memories tagged with skill %q.", skillName), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Memories for skill %q:\n\n", skillName))
	for _, m := range memories {
		score := m.SkillScores[skillName]
		sb.WriteString(fmt.Sprintf("[%s] %s (relevance: %.2f)\n", m.Date, m.Experience, score))
		sb.WriteString(fmt.Sprintf("  %s\n\n", m.Lesson))
	}
	return sb.String(), nil
}
