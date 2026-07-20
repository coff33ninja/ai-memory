package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS memories (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	date        TEXT    NOT NULL,
	experience  TEXT    NOT NULL,
	lesson      TEXT    NOT NULL,
	impact      TEXT    NOT NULL DEFAULT 'under review',
	tags        TEXT    DEFAULT '',
	scope       TEXT    NOT NULL DEFAULT 'private',
	embedding   BLOB,
	created_at  TEXT    NOT NULL,
	updated_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS skills (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT    NOT NULL UNIQUE,
	description TEXT    DEFAULT '',
	body        TEXT    NOT NULL,
	embedding   BLOB,
	file_count  INTEGER DEFAULT 0,
	synced_at   TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS skill_files (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
	filename TEXT    NOT NULL,
	content  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS skill_usage (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	date        TEXT    NOT NULL,
	skill       TEXT    NOT NULL,
	context     TEXT    NOT NULL,
	with_skills TEXT    DEFAULT '',
	outcome     TEXT    NOT NULL DEFAULT 'used',
	embedding   BLOB,
	created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memories_impact ON memories(impact);
CREATE INDEX IF NOT EXISTS idx_memories_date ON memories(date);
CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
CREATE INDEX IF NOT EXISTS idx_skill_files_skill ON skill_files(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_usage_skill ON skill_usage(skill);
CREATE INDEX IF NOT EXISTS idx_skill_usage_date ON skill_usage(date);

CREATE TABLE IF NOT EXISTS interaction_outcomes (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	persona       TEXT    NOT NULL,
	summary       TEXT    NOT NULL,
	outcome_score INTEGER NOT NULL,
	skills_used   TEXT    DEFAULT '',
	tone_used     TEXT    DEFAULT '',
	created_at    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS evolution_log (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	persona     TEXT    NOT NULL,
	trigger     TEXT    NOT NULL,
	what_changed TEXT   NOT NULL,
	before_val  TEXT,
	after_val   TEXT,
	confidence  REAL    DEFAULT 1.0,
	created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_interaction_outcomes_persona ON interaction_outcomes(persona);
CREATE INDEX IF NOT EXISTS idx_evolution_log_persona ON evolution_log(persona);

CREATE TABLE IF NOT EXISTS tool_gaps (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	persona     TEXT    NOT NULL,
	need        TEXT    NOT NULL,
	context     TEXT    NOT NULL DEFAULT '',
	suggested   TEXT    NOT NULL DEFAULT '',
	resolved    INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tool_gaps_persona ON tool_gaps(persona);
CREATE INDEX IF NOT EXISTS idx_tool_gaps_resolved ON tool_gaps(resolved);

CREATE TABLE IF NOT EXISTS tool_knowledge (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	persona     TEXT    NOT NULL,
	tool_name   TEXT    NOT NULL,
	how_to_use  TEXT    NOT NULL,
	what_works  TEXT    NOT NULL DEFAULT '',
	what_fails  TEXT    NOT NULL DEFAULT '',
	params      TEXT    NOT NULL DEFAULT '',
	examples    TEXT    NOT NULL DEFAULT '',
	use_count   INTEGER NOT NULL DEFAULT 0,
	last_used   TEXT,
	created_at  TEXT    NOT NULL,
	updated_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS tool_recipes (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	persona     TEXT    NOT NULL,
	tool_name   TEXT    NOT NULL,
	recipe_name TEXT    NOT NULL,
	steps       TEXT    NOT NULL,
	use_case    TEXT    NOT NULL,
	success_count INTEGER NOT NULL DEFAULT 0,
	fail_count  INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tool_knowledge_persona ON tool_knowledge(persona);
CREATE INDEX IF NOT EXISTS idx_tool_knowledge_tool ON tool_knowledge(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_recipes_persona ON tool_recipes(persona);
CREATE INDEX IF NOT EXISTS idx_tool_recipes_tool ON tool_recipes(tool_name);

CREATE TABLE IF NOT EXISTS tool_errors (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	persona     TEXT    NOT NULL,
	tool_name   TEXT    NOT NULL,
	error_msg   TEXT    NOT NULL,
	context     TEXT    NOT NULL DEFAULT '',
	input_args  TEXT    NOT NULL DEFAULT '',
	resolved    INTEGER NOT NULL DEFAULT 0,
	reported    INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS mcp_servers (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	name          TEXT    NOT NULL UNIQUE,
	source        TEXT    NOT NULL DEFAULT '',
	has_report    INTEGER NOT NULL DEFAULT 0,
	has_screenshot INTEGER NOT NULL DEFAULT 0,
	has_ocr       INTEGER NOT NULL DEFAULT 0,
	has_chain     INTEGER NOT NULL DEFAULT 0,
	tool_count    INTEGER NOT NULL DEFAULT 0,
	creator       TEXT    NOT NULL DEFAULT '',
	repo_url      TEXT    NOT NULL DEFAULT '',
	description   TEXT    NOT NULL DEFAULT '',
	last_seen     TEXT,
	created_at    TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tool_errors_persona ON tool_errors(persona);
CREATE INDEX IF NOT EXISTS idx_tool_errors_tool ON tool_errors(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_errors_resolved ON tool_errors(resolved);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_name ON mcp_servers(name);

CREATE TABLE IF NOT EXISTS user_profiles (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	field       TEXT    NOT NULL,
	value       TEXT    NOT NULL,
	source      TEXT    NOT NULL DEFAULT 'inferred',
	confidence  REAL    NOT NULL DEFAULT 0.5,
	created_at  TEXT    NOT NULL,
	updated_at  TEXT    NOT NULL,
	UNIQUE(field)
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_field ON user_profiles(field);

CREATE TABLE IF NOT EXISTS project_contexts (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT    NOT NULL,
	root        TEXT    NOT NULL,
	type        TEXT    NOT NULL DEFAULT 'unknown',
	lang        TEXT    NOT NULL DEFAULT 'unknown',
	is_active   INTEGER NOT NULL DEFAULT 0,
	last_used   TEXT    NOT NULL,
	created_at  TEXT    NOT NULL,
	UNIQUE(name)
);

CREATE INDEX IF NOT EXISTS idx_project_contexts_name ON project_contexts(name);
CREATE INDEX IF NOT EXISTS idx_project_contexts_active ON project_contexts(is_active);

CREATE TABLE IF NOT EXISTS persona_mappings (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	project     TEXT    NOT NULL,
	persona     TEXT    NOT NULL,
	created_at  TEXT    NOT NULL,
	UNIQUE(project)
);

CREATE INDEX IF NOT EXISTS idx_persona_mappings_project ON persona_mappings(project);

CREATE TABLE IF NOT EXISTS backup_config (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	provider        TEXT    NOT NULL DEFAULT 'local',
	local_path      TEXT    NOT NULL DEFAULT '',
	auto_backup     INTEGER NOT NULL DEFAULT 0,
	interval_hours  INTEGER NOT NULL DEFAULT 24,
	last_backup     TEXT,
	created_at      TEXT    NOT NULL,
	updated_at      TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS backups (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp       TEXT    NOT NULL,
	provider        TEXT    NOT NULL,
	checksum        TEXT    NOT NULL DEFAULT '',
	persona_count   INTEGER NOT NULL DEFAULT 0,
	memory_count    INTEGER NOT NULL DEFAULT 0,
	skill_count     INTEGER NOT NULL DEFAULT 0,
	archive_path    TEXT    NOT NULL DEFAULT '',
	file_size       INTEGER NOT NULL DEFAULT 0,
	status          TEXT    NOT NULL DEFAULT 'pending',
	error_msg       TEXT    NOT NULL DEFAULT '',
	created_at      TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_backups_timestamp ON backups(timestamp);
CREATE INDEX IF NOT EXISTS idx_backups_provider ON backups(provider);
`

type DB struct {
	conn *sql.DB
}

func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}
	dbPath := filepath.Join(dir, "memory.db")
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &DB{conn: conn}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

func (d *DB) GetMeta(key string) (string, error) {
	var val string
	err := d.conn.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (d *DB) SetMeta(key, value string) error {
	_, err := d.conn.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

type MCPServerSeed struct {
	Name         string
	Source       string
	HasReport    int
	HasScreenshot int
	HasOCR       int
	HasChain     int
	ToolCount    int
	Creator      string
	RepoURL      string
	Description  string
}

func (d *DB) SeedCommonServers(servers []MCPServerSeed) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for _, s := range servers {
		_, err := d.conn.Exec(`
			INSERT OR IGNORE INTO mcp_servers (name, source, has_report, has_screenshot, has_ocr, has_chain, tool_count, creator, repo_url, description, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.Name, s.Source, s.HasReport, s.HasScreenshot, s.HasOCR, s.HasChain, s.ToolCount, s.Creator, s.RepoURL, s.Description, now)
		if err != nil {
			return fmt.Errorf("seed server %s: %w", s.Name, err)
		}
	}
	return nil
}
