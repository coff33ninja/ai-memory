package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/memory"
	"github.com/coff33ninja/ai-memory/internal/persona"
	"github.com/coff33ninja/ai-memory/internal/skills"
)

func handleOnboard(pm *persona.Manager, mem *memory.Store, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	identity, _ := args["identity"].(string)
	if name == "" || identity == "" {
		return nil, fmt.Errorf("name and identity are required")
	}
	tone, _ := args["tone"].(string)
	description, _ := args["description"].(string)
	greeting, _ := args["greeting"].(string)
	var skillNames []string
	if s, ok := args["skills"].([]interface{}); ok {
		for _, v := range s {
			if s, ok := v.(string); ok {
				skillNames = append(skillNames, s)
			}
		}
	}

	_, err := pm.Create(name, identity, tone, description, greeting, skillNames)
	if err != nil {
		return nil, err
	}

	// Store a welcome memory in the new persona's DB
	pDir := pm.PersonaDir(name)
	pDB, err := openDB(pDir)
	if err != nil {
		return fmt.Sprintf("Persona %q created but could not initialize memory DB: %v", name, err), nil
	}
	defer pDB.Close()

	pMem := memory.New(pDB)
	_, err = pMem.Store(
		fmt.Sprintf("Persona %q created: %s", name, identity),
		"New persona onboarded with identity and purpose",
		[]string{"onboard", name},
		"private",
	)
	if err != nil {
		return fmt.Sprintf("Persona %q created but welcome memory failed: %v", name, err), nil
	}

	// Store shared memory about this persona being created
	sDir := pm.SharedDir()
	sDB, err := openDB(sDir)
	if err == nil {
		sMem := memory.New(sDB)
		sMem.Store(
			fmt.Sprintf("Persona %q onboarded: %s", name, identity),
			description,
			[]string{"persona", name, "onboard"},
			"shared",
		)
		sDB.Close()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Persona %q created!\n\n", name))
	sb.WriteString(fmt.Sprintf("Identity: %s\n", identity))
	if tone != "" {
		sb.WriteString(fmt.Sprintf("Tone: %s\n", tone))
	}
	if description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", description))
	}
	if greeting != "" {
		sb.WriteString(fmt.Sprintf("Greeting keyword: %s\n", greeting))
		sb.WriteString(fmt.Sprintf("  → When user says \"%s\", switch to this persona\n", greeting))
	}
	if len(skillNames) > 0 {
		sb.WriteString(fmt.Sprintf("Skills: %s\n", strings.Join(skillNames, ", ")))
	}
	sb.WriteString(fmt.Sprintf("\nDB: %s/memory.db\n", pDir))
	sb.WriteString(fmt.Sprintf("Active persona: %s\n", pm.Active()))
	return sb.String(), nil
}

func handleListPersonas(pm *persona.Manager) (interface{}, error) {
	personas := pm.List()
	active := pm.Active()
	if len(personas) == 0 {
		return "No personas configured. Use `onboard` to create one.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d persona(s):\n\n", len(personas)))
	for _, p := range personas {
		marker := " "
		if p.Name == active {
			marker = "*"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s — %s\n", marker, p.Name, p.Identity))
		if p.Tone != "" {
			sb.WriteString(fmt.Sprintf("    Tone: %s\n", p.Tone))
		}
		if len(p.Skills) > 0 {
			sb.WriteString(fmt.Sprintf("    Skills: %s\n", strings.Join(p.Skills, ", ")))
		}
	}
	sb.WriteString("\n* = active persona")
	return sb.String(), nil
}

func handleSwitchPersonaByName(pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	p, err := pm.Switch(name)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Switched to persona %q\n\n", p.Name))
	sb.WriteString(fmt.Sprintf("Identity: %s\n", p.Identity))
	if p.Tone != "" {
		sb.WriteString(fmt.Sprintf("Tone: %s\n", p.Tone))
	}
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", p.Description))
	}
	if p.Greeting != "" {
		sb.WriteString(fmt.Sprintf("Greeting keyword: %s\n", p.Greeting))
	}
	if len(p.Skills) > 0 {
		sb.WriteString(fmt.Sprintf("Skills: %s\n", strings.Join(p.Skills, ", ")))
	}
	return sb.String(), nil
}

func handleDeletePersona(pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if err := pm.Delete(name); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Persona %q deleted. Active persona: %s", name, pm.Active()), nil
}

func handlePersonaActive(pm *persona.Manager) (interface{}, error) {
	name := pm.Active()
	p := pm.Get(name)
	if p == nil {
		return "No active persona.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active persona: %s\n\n", p.Name))
	sb.WriteString(fmt.Sprintf("Identity: %s\n", p.Identity))
	if p.Tone != "" {
		sb.WriteString(fmt.Sprintf("Tone: %s\n", p.Tone))
	}
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", p.Description))
	}
	if p.Greeting != "" {
		sb.WriteString(fmt.Sprintf("Greeting keyword: %s\n", p.Greeting))
	}
	if len(p.Skills) > 0 {
		sb.WriteString(fmt.Sprintf("Skills: %s\n", strings.Join(p.Skills, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", p.CreatedAt))
	return sb.String(), nil
}

func handlePersonaAll(pm *persona.Manager) (interface{}, error) {
	personas := pm.List()
	active := pm.Active()
	return map[string]interface{}{
		"active":  active,
		"personas": personas,
	}, nil
}

func handlePersonaStartupPrompt(pm *persona.Manager, mem *memory.Store, skillsStore *skills.Store) (interface{}, error) {
	name := pm.Active()
	p := pm.Get(name)
	if p == nil {
		return handleContextInjectPrompt()
	}

	// Check if this is the auto-created default persona
	isDefault := p.Name == "default" && p.Identity == "General-purpose assistant"

	pending, _ := mem.ListPending()
	allSkills, _ := skillsStore.Catalog()

	skillMap := make(map[string]string)
	for _, sk := range allSkills {
		skillMap[sk.Name] = sk.Description
	}

	// Get active project context from DB
	activeProject, _ := mem.GetActiveProjectContext()

	var sb strings.Builder

	if isDefault {
		sb.WriteString("# FIRST RUN — ONBOARD YOURSELF\n\n")
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

	// Project context section
	if activeProject != nil {
		sb.WriteString(fmt.Sprintf("## Active Project: %s\n", activeProject.Name))
		sb.WriteString(fmt.Sprintf("Root: %s\n", activeProject.Root))
		sb.WriteString(fmt.Sprintf("Type: %s (%s)\n\n", activeProject.Type, activeProject.Lang))
	} else {
		sb.WriteString("## Project Context\n")
		sb.WriteString("No project context set. Call `set_project_context` with your working directory:\n")
		sb.WriteString("  set_project_context(name: \"<project-name>\", root: \"<absolute-path>\", type: \"go|node|python|...\", lang: \"go|javascript|python|...\")\n\n")
	}

	sb.WriteString(fmt.Sprintf("# Persona: %s\n\n", p.Name))
	sb.WriteString(fmt.Sprintf("Identity: %s\n", p.Identity))
	if p.Tone != "" {
		sb.WriteString(fmt.Sprintf("Tone: %s\n", p.Tone))
	}
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", p.Description))
	}
	if p.Greeting != "" {
		sb.WriteString(fmt.Sprintf("Your greeting keyword: %s\n", p.Greeting))
	}
	sb.WriteString("\n")

	// List all personas with their greetings for switching
	allPersonas := pm.List()
	if len(allPersonas) > 1 {
		sb.WriteString("## Available Personas (for greeting-based switching)\n")
		for _, ap := range allPersonas {
			greetInfo := ""
			if ap.Greeting != "" {
				greetInfo = fmt.Sprintf(" — greeting: \"%s\"", ap.Greeting)
			}
			marker := "  "
			if ap.Name == name {
				marker = "* "
			}
			sb.WriteString(fmt.Sprintf("%s%s: %s%s\n", marker, ap.Name, ap.Identity, greetInfo))
		}
		sb.WriteString("\nWhen the user's message contains a greeting keyword (e.g. \"hello Akeno\"), call `switch_persona` to that persona.\n")
		sb.WriteString("* = current persona\n\n")
	}

	if len(pending) > 0 {
		sb.WriteString(fmt.Sprintf("## %d Pending Memories\n", len(pending)))
		for _, m := range pending {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", m.Date, m.Experience))
		}
		sb.WriteString("\n")
	}

	if len(p.Skills) > 0 {
		sb.WriteString("## Persona Skills\n")
		for _, name := range p.Skills {
			if desc, ok := skillMap[name]; ok {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
			}
		}
		sb.WriteString("\n")
	}

	// User profile
	profiles, _ := mem.ListUserProfile()
	if len(profiles) > 0 {
		sb.WriteString("## User Profile\n")
		for _, p := range profiles {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", p.Field, p.Value))
		}
		sb.WriteString("\nUse this to personalize responses. When you learn more about the user, call `store_user_profile`.\n\n")
	} else {
		sb.WriteString("## User Profile\nNo data yet. As you interact with the user, learn their name, interests, and preferences, then call `store_user_profile` to remember them.\n\n")
	}

	sb.WriteString("BEFORE answering, call `search` with the user's query to pull in relevant context.\n")
	sb.WriteString("When you learn something important, call `store` to save it.\n")

	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": map[string]interface{}{"type": "text", "text": sb.String()},
			},
		},
	}, nil
}
