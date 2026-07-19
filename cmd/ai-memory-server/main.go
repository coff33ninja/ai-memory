package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coff33ninja/ai-memory/internal/db"
	"github.com/coff33ninja/ai-memory/internal/embedding"
	"github.com/coff33ninja/ai-memory/internal/evolution"
	"github.com/coff33ninja/ai-memory/internal/mcp"
	"github.com/coff33ninja/ai-memory/internal/memory"
	"github.com/coff33ninja/ai-memory/internal/persona"
	"github.com/coff33ninja/ai-memory/internal/rag"
	"github.com/coff33ninja/ai-memory/internal/skills"
	"github.com/coff33ninja/ai-memory/internal/version"
)

var Version = version.Version

func main() {
	dir := dataDir()

	// Persona manager
	pm, err := persona.NewManager(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: persona manager init: %v\n", err)
	}

	// Auto-create default persona on first run
	if pm != nil && len(pm.List()) == 0 {
		p, err := pm.Create("default", "General-purpose assistant", "direct", "Auto-created on first run. Onboard with your identity, tone, and skills.", "", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: auto-create default persona: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "info: auto-created default persona %q\n", p.Name)
			// Store a welcome memory in the default persona's DB
			pDir := pm.PersonaDir(p.Name)
			pDB, pErr := db.Open(pDir)
			if pErr == nil {
				pMem := memory.New(pDB)
				pMem.Store(
					"Default persona created on first run",
					"User should onboard with identity, tone, and skills using the onboard tool",
					[]string{"onboard", "default"},
					"private",
				)
				pDB.Close()
			}
			// Store shared memory about the default persona
			sDir := pm.SharedDir()
			sDB, sErr := db.Open(sDir)
			if sErr == nil {
				sMem := memory.New(sDB)
				sMem.Store(
					"Default persona auto-created on first run",
					"User should create a proper persona with onboard tool",
					[]string{"persona", "default", "onboard"},
					"shared",
				)
				sDB.Close()
			}
		}
	}

	// Embedder (shared across all personas)
	emb, err := embedding.InitEmbedder()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: init embedder: %v\n", err)
		os.Exit(1)
	}
	defer embedding.CloseEmbedder()

	// Open active persona DB (or default to base dir)
	activePersonaName := ""
	if pm != nil {
		activePersonaName = pm.Active()
	}

	// Determine DB directory for active persona
	dbDir := dir
	if pm != nil && activePersonaName != "" {
		p := pm.Get(activePersonaName)
		if p != nil {
			dbDir = pm.PersonaDir(activePersonaName)
		}
	}

	database, err := db.Open(dbDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Seed common MCP servers on first run
	database.SeedCommonServers([]db.MCPServerSeed{
		{
			Name: "go-mcp-computer-use", Source: "github.com/coff33ninja/go-mcp-computer-use",
			HasReport: 1, HasScreenshot: 1, HasOCR: 1, HasChain: 1, ToolCount: 143,
			Creator: "coff33ninja", RepoURL: "https://github.com/coff33ninja/go-mcp-computer-use",
			Description: "Full desktop automation: click, type, OCR, screenshot, chain sequences, UIA element detection",
		},
		{
			Name: "playwright", Source: "github.com/microsoft/playwright-mcp",
			HasReport: 1, HasScreenshot: 1, ToolCount: 25,
			Creator: "microsoft", RepoURL: "https://github.com/microsoft/playwright-mcp",
			Description: "Browser automation via Playwright: navigation, form filling, screenshots, network interception",
		},
		{
			Name: "filesystem", Source: "github.com/modelcontextprotocol/servers",
			ToolCount: 12, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem",
			Description: "File operations: read, write, list, search, watch for changes",
		},
		{
			Name: "github", Source: "github.com/modelcontextprotocol/servers",
			HasReport: 1, ToolCount: 18, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/github",
			Description: "GitHub API: repos, issues, PRs, search, code review",
		},
		{
			Name: "brave-search", Source: "github.com/modelcontextprotocol/servers",
			ToolCount: 2, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search",
			Description: "Web and local search via Brave Search API",
		},
		{
			Name: "postgres", Source: "github.com/modelcontextprotocol/servers",
			ToolCount: 3, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/postgres",
			Description: "PostgreSQL database: query, list tables, explain plans",
		},
		{
			Name: "sqlite", Source: "github.com/modelcontextprotocol/servers",
			ToolCount: 4, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/sqlite",
			Description: "SQLite database: query, list tables, create/modify schema",
		},
		{
			Name: "slack", Source: "github.com/modelcontextprotocol/servers",
			HasReport: 1, ToolCount: 8, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/slack",
			Description: "Slack workspace: messages, channels, users, search",
		},
		{
			Name: "memory", Source: "github.com/modelcontextprotocol/servers",
			ToolCount: 3, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/memory",
			Description: "Knowledge graph memory: create entities, relations, search",
		},
		{
			Name: "puppeteer", Source: "github.com/modelcontextprotocol/servers",
			HasScreenshot: 1, ToolCount: 10, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/puppeteer",
			Description: "Browser automation via Puppeteer: navigate, click, screenshot, evaluate JS",
		},
		{
			Name: "google-drive", Source: "github.com/modelcontextprotocol/servers",
			HasReport: 1, ToolCount: 6, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/google-drive",
			Description: "Google Drive: list, read, search files and folders",
		},
		{
			Name: "notion", Source: "github.com/modelcontextprotocol/servers",
			HasReport: 1, ToolCount: 5, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/notion",
			Description: "Notion API: pages, databases, blocks, search",
		},
		{
			Name: "fetch", Source: "github.com/modelcontextprotocol/servers",
			ToolCount: 1, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/fetch",
			Description: "HTTP fetch: GET/POST URLs, return content as text",
		},
		{
			Name: "sentry", Source: "github.com/modelcontextprotocol/servers",
			HasReport: 1, ToolCount: 4, Creator: "modelcontextprotocol",
			RepoURL: "https://github.com/modelcontextprotocol/servers/tree/main/src/sentry",
			Description: "Sentry error tracking: list issues, get details, resolve",
		},
	})

	// Open shared memory DB
	sharedDBPath := filepath.Join(dir, "shared")
	sharedDB, err := db.Open(sharedDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: open shared db: %v\n", err)
	}
	defer func() { if sharedDB != nil { sharedDB.Close() } }()

	// RAG searcher
	var searcher *rag.Searcher
	if emb != nil {
		searcher = rag.New(database, emb)
		if sharedDB != nil {
			searcher.SetSharedDB(sharedDB)
		}
	}

	// Skills store
	skillsStore := skills.New(database, dir)

	// Memory store
	store := memory.New(database)

	// Evolution engine
	eng := evolution.NewEngine(pm, database, emb)

	// MCP server
	srv := mcp.NewServer(Version)
	srv.RegisterTools(mcp.DefaultTools...)
	srv.RegisterResources(mcp.DefaultResources...)
	srv.RegisterTemplates(mcp.DefaultTemplates...)
	srv.RegisterPrompts(mcp.DefaultPrompts...)
	srv.SetToolHandler(func(name string, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "store":
			var sharedMem *memory.Store
			if sharedDB != nil {
				sharedMem = memory.New(sharedDB)
			}
			return handleStore(store, sharedMem, args)
		case "review":
			return handleReview(store)
		case "apply":
			return handleApply(store, args)
		case "dismiss":
			return handleDismiss(store, args)
		case "status":
			return handleStatus(store, skillsStore)
		case "search":
			return handleSearch(searcher, emb, args)
		case "search_memories":
			return handleSearchMemories(searcher, emb, args)
		case "search_skills":
			return handleSearchSkills(searcher, emb, args)
		case "reindex":
			return handleReindex(searcher)
		case "skills_sync":
			return handleSkillsSync(skillsStore)
		case "skills_search":
			return handleSkillsSearch(skillsStore, args)
		case "skills_index":
			return handleSkillsIndex(skillsStore)
		case "store_skill_usage":
			return handleStoreSkillUsage(store, args)
		case "list_skill_usage":
			return handleListSkillUsage(store, args)
		case "onboard":
			return handleOnboard(pm, store, args)
		case "list_personas":
			return handleListPersonas(pm)
		case "switch_persona":
			return handleSwitchPersona(pm)
		case "switch_persona_by_name":
			return handleSwitchPersonaByName(pm, args)
		case "delete_persona":
			return handleDeletePersona(pm, args)
		case "log_interaction":
			return handleLogInteraction(eng, pm, args)
		case "evolve":
			return handleEvolve(eng, pm)
		case "consolidate":
			return handleConsolidate(eng, pm)
		case "discover_skills":
			return handleDiscoverSkills(eng, pm)
		case "evolution_history":
			return handleEvolutionHistory(eng, pm, args)
		case "get_evolved_rules":
			return handleGetEvolvedRules(eng, pm)
		case "interaction_stats":
			return handleInteractionStats(eng, pm)
		case "log_tool_gap":
			return handleLogToolGap(eng, pm, args)
		case "list_tool_gaps":
			return handleListToolGaps(eng, pm, args)
		case "resolve_tool_gap":
			return handleResolveToolGap(eng, pm, args)
		case "log_tool_knowledge":
			return handleLogToolKnowledge(eng, pm, args)
		case "log_tool_recipe":
			return handleLogToolRecipe(eng, pm, args)
		case "get_tool_knowledge":
			return handleGetToolKnowledge(eng, pm, args)
		case "list_tool_knowledge":
			return handleListToolKnowledge(eng, pm)
		case "get_tool_recipes":
			return handleGetToolRecipes(eng, pm, args)
		case "record_recipe_outcome":
			return handleRecordRecipeOutcome(eng, pm, args)
		case "log_tool_error":
			return handleLogToolError(eng, pm, args)
		case "list_tool_errors":
			return handleListToolErrors(eng, pm, args)
		case "resolve_tool_error":
			return handleResolveToolError(eng, pm, args)
		case "register_mcp_server":
			return handleRegisterMCPServer(eng, pm, args)
		case "get_mcp_server":
			return handleGetMCPServer(eng, pm, args)
		case "list_mcp_servers":
			return handleListMCPServers(eng, pm)
		case "store_user_profile":
			return handleStoreUserProfile(store, args)
		case "get_user_profile":
			return handleGetUserProfile(store, args)
		case "list_user_profile":
			return handleListUserProfile(store, args)
		case "delete_user_profile":
			return handleDeleteUserProfile(store, args)
		default:
			return nil, fmt.Errorf("unknown tool: %s", name)
		}
	})
	srv.SetResourceHandler(func(uri string) (interface{}, error) {
		switch {
		case uri == "memory://memories":
			return handleAll(store, skillsStore, database)
		case uri == "memory://skills":
			return handleSummary(store, skillsStore)
		case uri == "memory://summary":
			return handleSummary(store, skillsStore)
		case uri == "memory://all":
			return handleAll(store, skillsStore, database)
		case uri == "skills://catalog":
			return handleSummary(store, skillsStore)
		case uri == "context://project":
			return handleContextProject(skillsStore)
		case uri == "context://startup":
			return handleContextStartup(pm, store, skillsStore)
		case uri == "skills://usage":
			return handleSkillsUsage(store)
		case uri == "persona://active":
			return handlePersonaActive(pm)
		case uri == "persona://all":
			return handlePersonaAll(pm)
		case uri == "evolution://stats":
			return handleEvolutionStats(pm, eng)
		case uri == "evolution://rules":
			return handleGetEvolvedRules(eng, pm)
		case uri == "user://profile":
			return handleUserProfileResource(store)
		default:
			return nil, fmt.Errorf("unknown resource: %s", uri)
		}
	})
	srv.SetPromptHandler(func(name string, args map[string]interface{}) (interface{}, error) {
		switch name {
		case "memory":
			return handleMemoryPrompt(store, skillsStore)
		case "reflect":
			return handleReflectPrompt()
		case "context-inject":
			return handleContextInjectPrompt()
		case "skill-usage-recorder":
			return handleSkillUsageRecorderPrompt()
		case "persona-startup":
			return handlePersonaStartupPrompt(pm, store, skillsStore)
		case "evolution-loop":
			return handleEvolutionPrompt(pm)
		case "mcp-error-handling":
			return handleMCPErrorPrompt(pm, eng)
		default:
			return nil, fmt.Errorf("unknown prompt: %s", name)
		}
	})

	// Start serving
	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func dataDir() string {
	if v := os.Getenv("AI_MEMORY_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ai-memory")
}

func openDB(dir string) (*db.DB, error) {
	return db.Open(dir)
}
