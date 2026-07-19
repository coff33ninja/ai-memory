package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/evolution"
	"github.com/coff33ninja/ai-memory/internal/persona"
)

func handleLogToolKnowledge(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	toolName, _ := args["tool_name"].(string)
	howToUse, _ := args["how_to_use"].(string)
	if toolName == "" || howToUse == "" {
		return nil, fmt.Errorf("tool_name and how_to_use are required")
	}

	whatWorks, _ := args["what_works"].(string)
	whatFails, _ := args["what_fails"].(string)
	params, _ := args["params"].(string)
	examples, _ := args["examples"].(string)

	personaName := pm.Active()
	if err := eng.Tracker().LogToolKnowledge(personaName, toolName, howToUse, whatWorks, whatFails, params, examples); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Tool knowledge logged for %q: %s", personaName, toolName), nil
}

func handleLogToolRecipe(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	toolName, _ := args["tool_name"].(string)
	recipeName, _ := args["recipe_name"].(string)
	steps, _ := args["steps"].(string)
	useCase, _ := args["use_case"].(string)
	if toolName == "" || recipeName == "" || steps == "" || useCase == "" {
		return nil, fmt.Errorf("tool_name, recipe_name, steps, and use_case are required")
	}

	personaName := pm.Active()
	if err := eng.Tracker().LogToolRecipe(personaName, toolName, recipeName, steps, useCase); err != nil {
		return nil, err
	}

	return fmt.Sprintf("Recipe %q logged for tool %s", recipeName, toolName), nil
}

func handleGetToolKnowledge(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	toolName, _ := args["tool_name"].(string)
	if toolName == "" {
		return nil, fmt.Errorf("tool_name is required")
	}

	personaName := pm.Active()
	k, err := eng.Tracker().GetToolKnowledge(personaName, toolName)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return fmt.Sprintf("No knowledge about %q yet. Use log_tool_knowledge to build your manual.", toolName), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tool knowledge: %s (used %d times)\n\n", k.ToolName, k.UseCount))
	sb.WriteString(fmt.Sprintf("How to use:\n%s\n\n", k.HowToUse))
	if k.WhatWorks != "" {
		sb.WriteString(fmt.Sprintf("What works:\n%s\n\n", k.WhatWorks))
	}
	if k.WhatFails != "" {
		sb.WriteString(fmt.Sprintf("What fails:\n%s\n\n", k.WhatFails))
	}
	if k.Params != "" {
		sb.WriteString(fmt.Sprintf("Parameters:\n%s\n\n", k.Params))
	}
	if k.Examples != "" {
		sb.WriteString(fmt.Sprintf("Examples:\n%s\n", k.Examples))
	}

	// Also get recipes
	recipes, _ := eng.Tracker().GetToolRecipes(personaName, toolName)
	if len(recipes) > 0 {
		sb.WriteString(fmt.Sprintf("\nRecipes for %s:\n", toolName))
		for _, r := range recipes {
			sb.WriteString(fmt.Sprintf("  [%s] %s (✓%d ✗%d)\n", r.RecipeName, r.UseCase, r.SuccessCount, r.FailCount))
		}
	}

	return sb.String(), nil
}

func handleListToolKnowledge(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	personaName := pm.Active()
	items, err := eng.Tracker().AllToolKnowledge(personaName)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return "No tool knowledge yet. Use log_tool_knowledge after using tools to build your manual.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tool knowledge for %q:\n\n", personaName))
	for _, k := range items {
		sb.WriteString(fmt.Sprintf("🔧 %s (used %d times)\n", k.ToolName, k.UseCount))
		sb.WriteString(fmt.Sprintf("   %s\n\n", truncate(k.HowToUse, 80)))
	}

	return sb.String(), nil
}

func handleGetToolRecipes(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	toolName, _ := args["tool_name"].(string)
	if toolName == "" {
		return nil, fmt.Errorf("tool_name is required")
	}

	personaName := pm.Active()
	recipes, err := eng.Tracker().GetToolRecipes(personaName, toolName)
	if err != nil {
		return nil, err
	}
	if len(recipes) == 0 {
		return fmt.Sprintf("No recipes for %s yet. Use log_tool_recipe to create reusable patterns.", toolName), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Recipes for %s:\n\n", toolName))
	for _, r := range recipes {
		sb.WriteString(fmt.Sprintf("📋 %s (✓%d ✗%d)\n", r.RecipeName, r.SuccessCount, r.FailCount))
		sb.WriteString(fmt.Sprintf("   Use case: %s\n", r.UseCase))
		sb.WriteString(fmt.Sprintf("   Steps:\n%s\n\n", r.Steps))
	}

	return sb.String(), nil
}

func handleRecordRecipeOutcome(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	idF, ok := args["recipe_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("recipe_id is required")
	}
	success, ok := args["success"].(bool)
	if !ok {
		return nil, fmt.Errorf("success (boolean) is required")
	}

	if err := eng.Tracker().RecordRecipeOutcome(int64(idF), success); err != nil {
		return nil, err
	}

	status := "success"
	if !success {
		status = "failure"
	}
	return fmt.Sprintf("Recipe #%d recorded as %s", int64(idF), status), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
