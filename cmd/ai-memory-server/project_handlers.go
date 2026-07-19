package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/memory"
)

func handleSetProjectContext(store *memory.Store, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	root, _ := args["root"].(string)
	if name == "" || root == "" {
		return nil, fmt.Errorf("name and root are required")
	}
	typ, _ := args["type"].(string)
	lang, _ := args["lang"].(string)
	if typ == "" {
		typ = "unknown"
	}
	if lang == "" {
		lang = "unknown"
	}

	p, err := store.SetProjectContext(name, root, typ, lang)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Active project set to %q (%s/%s) @ %s", p.Name, p.Type, p.Lang, p.Root), nil
}

func handleGetActiveProjectContext(store *memory.Store) (interface{}, error) {
	p, err := store.GetActiveProjectContext()
	if err != nil {
		return nil, err
	}
	if p == nil {
		return "No active project context. Use `set_project_context` to set one.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active project: %s\n", p.Name))
	sb.WriteString(fmt.Sprintf("Root: %s\n", p.Root))
	sb.WriteString(fmt.Sprintf("Type: %s (%s)\n", p.Type, p.Lang))
	sb.WriteString(fmt.Sprintf("Last used: %s\n", p.LastUsed))
	return sb.String(), nil
}

func handleListProjectContexts(store *memory.Store) (interface{}, error) {
	ctxs, err := store.ListProjectContexts()
	if err != nil {
		return nil, err
	}
	if len(ctxs) == 0 {
		return "No project contexts stored. Use `set_project_context` to add one.", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d project context(s):\n\n", len(ctxs)))
	for _, p := range ctxs {
		marker := "  "
		if p.IsActive {
			marker = "* "
		}
		sb.WriteString(fmt.Sprintf("%s%s — %s (%s) @ %s\n", marker, p.Name, p.Type, p.Lang, p.Root))
		sb.WriteString(fmt.Sprintf("   Last used: %s\n", p.LastUsed))
	}
	sb.WriteString("\n* = active project")
	return sb.String(), nil
}
