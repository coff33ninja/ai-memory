package mcp

type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type ResourceDef struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
}

type TemplateDef struct {
	URITemplate  string `json:"uriTemplate"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	MimeType     string `json:"mimeType,omitempty"`
}

type PromptDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var DefaultTools = []ToolDef{
	{
		Name:        "store",
		Description: "Store a new memory entry",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"experience": map[string]interface{}{"type": "string", "description": "What happened"},
				"lesson":     map[string]interface{}{"type": "string", "description": "What was learned"},
				"tags":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Optional tags for categorization"},
				"scope":      map[string]interface{}{"type": "string", "enum": []string{"private", "shared"}, "description": "private = current persona only, shared = visible to all personas (default: private)"},
			},
			"required": []string{"experience", "lesson"},
		},
	},
	{
		Name:        "review",
		Description: "List memory entries with impact 'under review'",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "apply",
		Description: "Mark a memory entry as applied",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "integer", "description": "Memory entry ID"},
			},
			"required": []string{"id"},
		},
	},
	{
		Name:        "dismiss",
		Description: "Mark a memory entry as dismissed",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "integer", "description": "Memory entry ID"},
			},
			"required": []string{"id"},
		},
	},
	{
		Name:        "status",
		Description: "Show memory and skill counts, pending reviews",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "search",
		Description: "Unified semantic search across memories and skills",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query"},
				"topK":  map[string]interface{}{"type": "integer", "description": "Max results (default 5)"},
				"type":  map[string]interface{}{"type": "string", "enum": []string{"all", "memory", "skill"}, "description": "Filter by type"},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "search_memories",
		Description: "Semantic search memories only",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query"},
				"topK":  map[string]interface{}{"type": "integer", "description": "Max results (default 5)"},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "search_skills",
		Description: "Semantic search skills only",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query"},
				"topK":  map[string]interface{}{"type": "integer", "description": "Max results (default 5)"},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "reindex",
		Description: "Rebuild all embeddings from scratch",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "skills_sync",
		Description: "Sync skills repository with remote",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "skills_search",
		Description: "Keyword search skills (fast, no embedding)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search keyword"},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "skills_index",
		Description: "Re-index skills from the cloned repository",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "store_skill_usage",
		Description: "Record that a skill was used in a context, with which other skills were loaded alongside it",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"skill":       map[string]interface{}{"type": "string", "description": "Skill name that was used"},
				"context":     map[string]interface{}{"type": "string", "description": "What task/situation it was used for"},
				"with_skills": map[string]interface{}{"type": "string", "description": "Comma-separated list of other skills loaded alongside it"},
				"outcome":     map[string]interface{}{"type": "string", "description": "Result: used, effective, partial, failed (default: used)"},
			},
			"required": []string{"skill", "context"},
		},
	},
	{
		Name:        "list_skill_usage",
		Description: "Show recent skill usage history — which skills were used in what contexts, with what companions",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer", "description": "Max entries (default 20)"},
			},
		},
	},
	{
		Name:        "onboard",
		Description: "Create a new AI persona with its own memory and skill context",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":        map[string]interface{}{"type": "string", "description": "Persona name (unique, used as directory name)"},
				"identity":    map[string]interface{}{"type": "string", "description": "Who this AI is — name, role, purpose"},
				"tone":        map[string]interface{}{"type": "string", "description": "Communication style (e.g. formal, casual, concise, verbose)"},
				"description": map[string]interface{}{"type": "string", "description": "Brief description of this persona's specialty"},
				"greeting":    map[string]interface{}{"type": "string", "description": "Greeting keyword that triggers switching to this persona (e.g. 'Akeno', 'Hello Akeno')"},
				"skills":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Skill names this persona should use"},
			},
			"required": []string{"name", "identity"},
		},
	},
	{
		Name:        "list_personas",
		Description: "List all configured AI personas",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "switch_persona",
		Description: "Switch to a different AI persona — loads its memories, skills, and context",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Persona name to switch to"},
			},
			"required": []string{"name"},
		},
	},
	{
		Name:        "delete_persona",
		Description: "Delete a persona and its private memories (shared memories are preserved)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Persona name to delete"},
			},
			"required": []string{"name"},
		},
	},
	{
		Name:        "log_interaction",
		Description: "Record an interaction outcome for persona evolution tracking",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"summary":      map[string]interface{}{"type": "string", "description": "Brief summary of the interaction"},
				"outcome_score": map[string]interface{}{"type": "integer", "description": "Outcome score 1-5 (1=poor, 5=excellent)"},
				"skills_used":  map[string]interface{}{"type": "string", "description": "Comma-separated skills used in this interaction"},
				"tone_used":    map[string]interface{}{"type": "string", "description": "Tone used (e.g. formal, casual, concise)"},
			},
			"required": []string{"summary", "outcome_score"},
		},
	},
	{
		Name:        "evolve",
		Description: "Trigger full persona evolution: consolidate memories, adapt traits, discover skills",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "consolidate",
		Description: "Merge similar memories, elevate frequent ones, prune old dismissed ones",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "discover_skills",
		Description: "Find skills from the repo that would help this persona based on interaction patterns",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "evolution_history",
		Description: "Show how this persona has evolved over time",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer", "description": "Max entries (default 20)"},
			},
		},
	},
	{
		Name:        "get_evolved_rules",
		Description: "Get behavioral rules this persona has learned from experience",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "interaction_stats",
		Description: "Show interaction performance stats for this persona",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "log_tool_gap",
		Description: "Record a tool you needed but don't have — helps user discover what tools to expose",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"need":      map[string]interface{}{"type": "string", "description": "What you needed to do (e.g. 'click a button on screen', 'take a screenshot')"},
				"context":   map[string]interface{}{"type": "string", "description": "What you were trying to accomplish"},
				"suggested": map[string]interface{}{"type": "string", "description": "Suggested tool/MCP server name if you know one (e.g. 'go-mcp-computer-use', 'playwright')"},
			},
			"required": []string{"need", "context"},
		},
	},
	{
		Name:        "list_tool_gaps",
		Description: "Show unresolved tool limitations — what the AI needs but doesn't have",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"include_resolved": map[string]interface{}{"type": "boolean", "description": "Include already-resolved gaps (default false)"},
			},
		},
	},
	{
		Name:        "resolve_tool_gap",
		Description: "Mark a tool gap as resolved (user exposed the tool)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "integer", "description": "Tool gap ID to resolve"},
			},
			"required": []string{"id"},
		},
	},
	{
		Name:        "log_tool_knowledge",
		Description: "Record how to use a tool — build your own manual through experience",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool_name":  map[string]interface{}{"type": "string", "description": "Tool name (e.g. 'click', 'screenshot', 'ocr', 'chain')"},
				"how_to_use": map[string]interface{}{"type": "string", "description": "How this tool works, what it does"},
				"what_works": map[string]interface{}{"type": "string", "description": "Patterns that produce good results"},
				"what_fails": map[string]interface{}{"type": "string", "description": "Common mistakes or failure modes"},
				"params":     map[string]interface{}{"type": "string", "description": "Parameter guide — which params to use when"},
				"examples":   map[string]interface{}{"type": "string", "description": "Example invocations that worked"},
			},
			"required": []string{"tool_name", "how_to_use"},
		},
	},
	{
		Name:        "log_tool_recipe",
		Description: "Record a multi-step tool recipe — reusable pattern for complex tasks",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool_name":   map[string]interface{}{"type": "string", "description": "Primary tool used"},
				"recipe_name": map[string]interface{}{"type": "string", "description": "Short name for this recipe (e.g. 'fill_login_form', 'debug_crash')"},
				"steps":       map[string]interface{}{"type": "string", "description": "Step-by-step instructions"},
				"use_case":    map[string]interface{}{"type": "string", "description": "When to use this recipe"},
			},
			"required": []string{"tool_name", "recipe_name", "steps", "use_case"},
		},
	},
	{
		Name:        "get_tool_knowledge",
		Description: "Retrieve what you know about a specific tool before using it",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool_name": map[string]interface{}{"type": "string", "description": "Tool name to look up"},
			},
			"required": []string{"tool_name"},
		},
	},
	{
		Name:        "list_tool_knowledge",
		Description: "Show all tool knowledge you've accumulated",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "get_tool_recipes",
		Description: "Get recipes for a specific tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool_name": map[string]interface{}{"type": "string", "description": "Tool name"},
			},
			"required": []string{"tool_name"},
		},
	},
	{
		Name:        "record_recipe_outcome",
		Description: "Record whether a recipe succeeded or failed",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"recipe_id": map[string]interface{}{"type": "integer", "description": "Recipe ID"},
				"success":   map[string]interface{}{"type": "boolean", "description": "true if succeeded, false if failed"},
			},
			"required": []string{"recipe_id", "success"},
		},
	},
	{
		Name:        "log_tool_error",
		Description: "Record when a tool fails — helps diagnose issues and report to MCP creators",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"tool_name":  map[string]interface{}{"type": "string", "description": "Tool that failed"},
				"error_msg":  map[string]interface{}{"type": "string", "description": "Error message received"},
				"context":    map[string]interface{}{"type": "string", "description": "What you were trying to do"},
				"input_args": map[string]interface{}{"type": "string", "description": "Arguments you passed to the tool"},
				"mcp_server": map[string]interface{}{"type": "string", "description": "MCP server providing this tool (e.g. 'go-mcp-computer-use')"},
			},
			"required": []string{"tool_name", "error_msg", "context"},
		},
	},
	{
		Name:        "list_tool_errors",
		Description: "Show recent tool errors — what's broken and needs fixing",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"include_resolved": map[string]interface{}{"type": "boolean", "description": "Include resolved errors (default false)"},
			},
		},
	},
	{
		Name:        "resolve_tool_error",
		Description: "Mark a tool error as resolved",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "integer", "description": "Error ID to resolve"},
			},
			"required": []string{"id"},
		},
	},
	{
		Name:        "register_mcp_server",
		Description: "Register an MCP server with its capabilities — helps the AI know what tools are available",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":         map[string]interface{}{"type": "string", "description": "Server name (e.g. 'go-mcp-computer-use')"},
				"source":       map[string]interface{}{"type": "string", "description": "Where it came from (e.g. 'github.com/coff33ninja/go-mcp-computer-use')"},
				"has_report":   map[string]interface{}{"type": "boolean", "description": "Has report_issue tool for GitHub issues"},
				"has_screenshot": map[string]interface{}{"type": "boolean", "description": "Has screenshot capture"},
				"has_ocr":      map[string]interface{}{"type": "boolean", "description": "Has OCR text extraction"},
				"has_chain":    map[string]interface{}{"type": "boolean", "description": "Has chain/sequence execution"},
				"tool_count":   map[string]interface{}{"type": "integer", "description": "Number of tools provided"},
				"creator":      map[string]interface{}{"type": "string", "description": "Who made it"},
				"repo_url":     map[string]interface{}{"type": "string", "description": "Repository URL"},
				"description":  map[string]interface{}{"type": "string", "description": "What it does"},
			},
			"required": []string{"name"},
		},
	},
	{
		Name:        "get_mcp_server",
		Description: "Get info about an MCP server — does it have report_issue? What tools does it provide?",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Server name"},
			},
			"required": []string{"name"},
		},
	},
	{
		Name:        "list_mcp_servers",
		Description: "List all known MCP servers and their capabilities",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{},
		},
	},
	{
		Name:        "store_user_profile",
		Description: "Store a user profile field — build user knowledge incrementally from interactions",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field":      map[string]interface{}{"type": "string", "description": "Profile field name (e.g. 'name', 'hobbies', 'interests', 'favorite_color')"},
				"value":      map[string]interface{}{"type": "string", "description": "Value for this field"},
				"source":     map[string]interface{}{"type": "string", "description": "How this was learned (e.g. 'conversation', 'inferred', 'stated') (default: inferred)"},
				"confidence": map[string]interface{}{"type": "number", "description": "Confidence level 0.0-1.0 (default 0.5)"},
			},
			"required": []string{"field", "value"},
		},
	},
	{
		Name:        "get_user_profile",
		Description: "Get a specific user profile field",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field": map[string]interface{}{"type": "string", "description": "Profile field name"},
			},
			"required": []string{"field"},
		},
	},
	{
		Name:        "list_user_profile",
		Description: "List all stored user profile fields",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "delete_user_profile",
		Description: "Delete a user profile field",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field": map[string]interface{}{"type": "string", "description": "Profile field name to delete"},
			},
			"required": []string{"field"},
		},
	},
	{
		Name:        "set_project_context",
		Description: "Set the active project context — tells the AI which project it's working in. Call this at session start with the working directory.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "Project name (e.g. 'ai-memory', 'go-mcp-computer-use')"},
				"root": map[string]interface{}{"type": "string", "description": "Absolute path to project root"},
				"type": map[string]interface{}{"type": "string", "description": "Project type (e.g. 'go', 'node', 'python', 'rust')"},
				"lang": map[string]interface{}{"type": "string", "description": "Primary language (e.g. 'go', 'javascript', 'python')"},
			},
			"required": []string{"name", "root"},
		},
	},
	{
		Name:        "get_project_context",
		Description: "Get the currently active project context",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
	{
		Name:        "list_project_contexts",
		Description: "List all stored project contexts",
		InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
	},
}

var DefaultResources = []ResourceDef{
	{URI: "memory://memories", Name: "Memories", Description: "All memory entries"},
	{URI: "memory://skills", Name: "Skills", Description: "All indexed skills"},
	{URI: "memory://summary", Name: "Summary", Description: "Combined overview of memories and skills", MimeType: "text/plain"},
	{URI: "memory://all", Name: "All", Description: "Everything: memories + skills + stats", MimeType: "application/json"},
	{URI: "skills://catalog", Name: "Skills Catalog", Description: "List of all available skills", MimeType: "application/json"},
	{URI: "context://project", Name: "Project Context", Description: "Auto-detected project type with relevant skills pre-filtered", MimeType: "text/plain"},
	{URI: "context://startup", Name: "Startup Context", Description: "Initial context injected on connect: project type + top relevant skills + memories", MimeType: "text/plain"},
	{URI: "skills://usage", Name: "Skill Usage History", Description: "Recent skill usage patterns — which skills work well together", MimeType: "text/plain"},
	{URI: "persona://active", Name: "Active Persona", Description: "Current persona identity, tone, and context", MimeType: "text/plain"},
	{URI: "persona://all", Name: "All Personas", Description: "List of all personas with their identities", MimeType: "application/json"},
	{URI: "evolution://stats", Name: "Evolution Stats", Description: "Interaction performance and evolution stats for active persona", MimeType: "application/json"},
	{URI: "evolution://rules", Name: "Evolved Rules", Description: "Behavioral rules learned from experience", MimeType: "text/plain"},
	{URI: "user://profile", Name: "User Profile", Description: "What the AI knows about this user — name, interests, preferences", MimeType: "text/plain"},
	{URI: "project://active", Name: "Active Project", Description: "Currently active project context — name, root, type, language", MimeType: "text/plain"},
}

var DefaultTemplates = []TemplateDef{
	{URITemplate: "memory://file/{name}", Name: "Memory or skill by name", Description: "Read a memory entry or skill by name"},
	{URITemplate: "skills://{name}", Name: "Skill by name", Description: "All files for a skill"},
	{URITemplate: "skills://file/{name}/{filename}", Name: "Skill file", Description: "Specific file within a skill"},
}

var DefaultPrompts = []PromptDef{
	{Name: "memory", Description: "Full memory context for session start"},
	{Name: "reflect", Description: "Guide for end-of-session reflection"},
	{Name: "context-inject", Description: "System prompt — instructs AI to proactively search skills+memories before answering"},
	{Name: "skill-usage-recorder", Description: "Instructs AI to record which skills were used and with what companions"},
	{Name: "evolution-loop", Description: "Instructs AI to log interactions and trigger self-evolution"},
	{Name: "mcp-error-handling", Description: "Guide for handling MCP tool errors — log, report, resolve"},
	{Name: "persona-startup", Description: "Persona-specific startup context with user profile"},
}
