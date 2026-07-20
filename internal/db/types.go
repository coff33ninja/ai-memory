package db

import "time"

type Memory struct {
	ID         int64    `json:"id"`
	Date       string   `json:"date"`
	Experience string   `json:"experience"`
	Lesson     string   `json:"lesson"`
	Impact     string   `json:"impact"`
	Tags       []string `json:"tags,omitempty"`
	Scope      string   `json:"scope"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type Skill struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Body        string      `json:"body"`
	FileCount   int         `json:"file_count"`
	SyncedAt    string      `json:"synced_at"`
	Files       []SkillFile `json:"files,omitempty"`
}

type SkillFile struct {
	ID       int64  `json:"id"`
	SkillID  int64  `json:"skill_id"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type SearchResult struct {
	Type    string  `json:"type"`    // "memory" or "skill"
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type Status struct {
	MemoryCount    int   `json:"memory_count"`
	PendingCount   int   `json:"pending_count"`
	SkillCount     int   `json:"skill_count"`
	TotalEvolutions int  `json:"total_evolutions"`
}

type SkillUsage struct {
	ID         int64  `json:"id"`
	Date       string `json:"date"`
	Skill      string `json:"skill"`
	Context    string `json:"context"`
	WithSkills string `json:"with_skills"`
	Outcome    string `json:"outcome"`
	CreatedAt  string `json:"created_at"`
}

func NewMemory(experience, lesson string, tags []string) *Memory {
	now := time.Now().UTC().Format(time.RFC3339)
	date := now[:10]
	return &Memory{
		Date:       date,
		Experience: experience,
		Lesson:     lesson,
		Impact:     "under review",
		Tags:       tags,
		Scope:      "private",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

type UserProfile struct {
	ID         int64   `json:"id"`
	Field      string  `json:"field"`
	Value      string  `json:"value"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type ProjectContext struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Root      string `json:"root"`
	Type      string `json:"type"`
	Lang      string `json:"lang"`
	IsActive  bool   `json:"is_active"`
	LastUsed  string `json:"last_used"`
	CreatedAt string `json:"created_at"`
}

type PersonaMapping struct {
	ID        int64  `json:"id"`
	Project   string `json:"project"`
	Persona   string `json:"persona"`
	CreatedAt string `json:"created_at"`
}

type BackupConfig struct {
	ID            int64  `json:"id"`
	Provider      string `json:"provider"`
	LocalPath     string `json:"local_path"`
	AutoBackup    bool   `json:"auto_backup"`
	IntervalHours int    `json:"interval_hours"`
	LastBackup    string `json:"last_backup"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type Backup struct {
	ID           int64  `json:"id"`
	Timestamp    string `json:"timestamp"`
	Provider     string `json:"provider"`
	Checksum     string `json:"checksum"`
	PersonaCount int    `json:"persona_count"`
	MemoryCount  int    `json:"memory_count"`
	SkillCount   int    `json:"skill_count"`
	ArchivePath  string `json:"archive_path"`
	FileSize     int64  `json:"file_size"`
	Status       string `json:"status"`
	ErrorMsg     string `json:"error_msg"`
	CreatedAt    string `json:"created_at"`
}
