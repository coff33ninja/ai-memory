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
	srv.SetHandlers(
		database, searcher, skillsStore, skillsStore, store,
		emb, pm, eng,
	)

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
