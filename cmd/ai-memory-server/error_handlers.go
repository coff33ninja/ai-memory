package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/evolution"
	"github.com/coff33ninja/ai-memory/internal/persona"
)

func handleLogToolError(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	toolName, _ := args["tool_name"].(string)
	errorMsg, _ := args["error_msg"].(string)
	context, _ := args["context"].(string)
	inputArgs, _ := args["input_args"].(string)
	mcpServer, _ := args["mcp_server"].(string)
	if toolName == "" || errorMsg == "" || context == "" {
		return nil, fmt.Errorf("tool_name, error_msg, and context are required")
	}

	personaName := pm.Active()
	if err := eng.Tracker().LogToolError(personaName, toolName, errorMsg, context, inputArgs); err != nil {
		return nil, err
	}

	// Find the MCP server - use provided name or try to extract
	serverName := mcpServer
	if serverName == "" {
		serverName = extractServerName(toolName)
	}
	server, _ := eng.Tracker().GetMCPServer(serverName)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Error logged for %q: %s\n\n", toolName, errorMsg))

	if server != nil && server.HasReport == 1 {
		sb.WriteString(fmt.Sprintf("✅ %s has report_issue built in.\n", serverName))
		sb.WriteString("Call the report_issue tool with:\n")
		sb.WriteString(fmt.Sprintf("  title: \"Error in %s: %s\"\n", toolName, truncateStr(errorMsg, 50)))
		sb.WriteString(fmt.Sprintf("  body: \"## Error\\n%s\\n\\n## Context\\n%s\\n\\n## Args\\n%s\\n\"", errorMsg, context, inputArgs))
		if server.RepoURL != "" {
			sb.WriteString(fmt.Sprintf("  labels: [\"bug\"]\n"))
		}
	} else {
		sb.WriteString("❌ No automatic error reporting for this server.\n\n")

		// Smart guidance based on server knowledge
		if server != nil {
			sb.WriteString(fmt.Sprintf("Server: %s\n", server.Name))
			if server.RepoURL != "" {
				sb.WriteString(fmt.Sprintf("Issues: %s/issues\n", server.RepoURL))
			}
			if server.Creator != "" {
				sb.WriteString(fmt.Sprintf("Creator: %s\n", server.Creator))
			}
		} else {
			// Unknown server - give universal guidance
			sb.WriteString("Unknown MCP server. To report this error:\n")
			sb.WriteString(fmt.Sprintf("1. Find the server's source (check your MCP config)\n"))
			sb.WriteString(fmt.Sprintf("2. Look for a GitHub/GitLab repo link\n"))
			sb.WriteString(fmt.Sprintf("3. Open an issue with:\n"))
		}

		sb.WriteString("\nError details to include:\n")
		sb.WriteString(fmt.Sprintf("  Tool: %s\n", toolName))
		sb.WriteString(fmt.Sprintf("  Error: %s\n", errorMsg))
		sb.WriteString(fmt.Sprintf("  Context: %s\n", context))
		if inputArgs != "" {
			sb.WriteString(fmt.Sprintf("  Args: %s\n", inputArgs))
		}
		sb.WriteString("\nTip: Run register_mcp_server to save this server's info for future errors.\n")
	}

	return sb.String(), nil
}

func handleListToolErrors(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	personaName := pm.Active()
	includeResolved, _ := args["include_resolved"].(bool)

	errors, err := eng.Tracker().ToolErrors(personaName, includeResolved)
	if err != nil {
		return nil, err
	}
	if len(errors) == 0 {
		return "No tool errors recorded. Everything is working (for now).", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tool errors for %q:\n\n", personaName))
	for _, e := range errors {
		status := "❌"
		if e.Resolved == 1 {
			status = "✅"
		}
		reported := ""
		if e.Reported == 1 {
			reported = " [reported]"
		}
		sb.WriteString(fmt.Sprintf("[%s] #%d — %s%s\n", status, e.ID, e.ToolName, reported))
		sb.WriteString(fmt.Sprintf("    Error: %s\n", e.ErrorMsg))
		sb.WriteString(fmt.Sprintf("    Context: %s\n", e.Context))
		if e.InputArgs != "" {
			sb.WriteString(fmt.Sprintf("    Args: %s\n", e.InputArgs))
		}
		sb.WriteString(fmt.Sprintf("    Date: %s\n\n", e.CreatedAt[:10]))
	}

	return sb.String(), nil
}

func handleResolveToolError(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	idF, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}
	id := int64(idF)

	if err := eng.Tracker().MarkErrorResolved(id); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Tool error #%d marked as resolved.", id), nil
}

func handleRegisterMCPServer(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	source, _ := args["source"].(string)
	creator, _ := args["creator"].(string)
	repoURL, _ := args["repo_url"].(string)
	description, _ := args["description"].(string)
	toolCount := 0
	if tc, ok := args["tool_count"].(float64); ok {
		toolCount = int(tc)
	}
	hasReport, _ := args["has_report"].(bool)
	hasScreenshot, _ := args["has_screenshot"].(bool)
	hasOCR, _ := args["has_ocr"].(bool)
	hasChain, _ := args["has_chain"].(bool)

	if err := eng.Tracker().UpsertMCPServer(name, source, hasReport, hasScreenshot, hasOCR, hasChain, toolCount, creator, repoURL, description); err != nil {
		return nil, err
	}

	return fmt.Sprintf("MCP server %q registered.", name), nil
}

func handleGetMCPServer(eng *evolution.Engine, pm *persona.Manager, args map[string]interface{}) (interface{}, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	server, err := eng.Tracker().GetMCPServer(name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return fmt.Sprintf("Unknown MCP server: %q. Use register_mcp_server to add it.", name), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("MCP Server: %s\n", server.Name))
	if server.Creator != "" {
		sb.WriteString(fmt.Sprintf("Creator: %s\n", server.Creator))
	}
	if server.RepoURL != "" {
		sb.WriteString(fmt.Sprintf("Repo: %s\n", server.RepoURL))
	}
	if server.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", server.Description))
	}
	sb.WriteString(fmt.Sprintf("Tools: %d\n", server.ToolCount))
	sb.WriteString("\nCapabilities:\n")
	if server.HasReport == 1 {
		sb.WriteString("  ✅ report_issue (can auto-file GitHub issues)\n")
	}
	if server.HasScreenshot == 1 {
		sb.WriteString("  ✅ screenshot capture\n")
	}
	if server.HasOCR == 1 {
		sb.WriteString("  ✅ OCR text extraction\n")
	}
	if server.HasChain == 1 {
		sb.WriteString("  ✅ chain/sequence execution\n")
	}
	if server.LastSeen != "" {
		sb.WriteString(fmt.Sprintf("\nLast seen: %s\n", server.LastSeen[:10]))
	}

	return sb.String(), nil
}

func handleListMCPServers(eng *evolution.Engine, pm *persona.Manager) (interface{}, error) {
	servers, err := eng.Tracker().ListMCPServers()
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 {
		return "No MCP servers registered. Use register_mcp_server to add one.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Known MCP servers (%d):\n\n", len(servers)))
	for _, s := range servers {
		caps := ""
		if s.HasReport == 1 {
			caps += " 📋report"
		}
		if s.HasScreenshot == 1 {
			caps += " 📷screenshot"
		}
		if s.HasOCR == 1 {
			caps += " 🔍ocr"
		}
		if s.HasChain == 1 {
			caps += " ⛓chain"
		}
		sb.WriteString(fmt.Sprintf("• %s (%d tools)%s\n", s.Name, s.ToolCount, caps))
		if s.Creator != "" {
			sb.WriteString(fmt.Sprintf("  by %s\n", s.Creator))
		}
	}

	return sb.String(), nil
}

func extractServerName(toolName string) string {
	// Tools often have prefix like "computer_use_click" → "computer_use"
	// Or just "click" → need to find which server provides it
	// For now, return a reasonable guess
	parts := strings.Split(toolName, "_")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "_")
	}
	return toolName
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func handleMCPErrorPrompt(pm *persona.Manager, eng *evolution.Engine) (interface{}, error) {
	personaName := pm.Active()

	text := fmt.Sprintf(`# MCP Error Handling — %s

When an MCP tool fails:

1. **Log the error** — call log_tool_error with:
   - tool_name: which tool failed
   - error_msg: the exact error message
   - context: what you were trying to do
   - input_args: what arguments you passed

2. **Check server capabilities** — call get_mcp_server to see if it has report_issue

3. **If report_issue available:**
   - Call report_issue with title and body describing the error
   - Mark error as reported

4. **If no report_issue:**
   - Tell the user what happened
   - Suggest they contact the MCP server creator
   - Provide the error details and reproduction steps

5. **After resolution:**
   - Call resolve_tool_error(id) to mark it fixed
   - Log what fixed it in tool_knowledge

Always log errors — even if you can't fix them, the pattern helps diagnose systemic issues.
`, personaName)

	return map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": map[string]interface{}{"type": "text", "text": text},
			},
		},
	}, nil
}
