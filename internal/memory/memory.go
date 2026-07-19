package memory

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/coff33ninja/ai-memory/internal/db"
)

type Store struct {
	db *db.DB
}

func New(d *db.DB) *Store {
	return &Store{db: d}
}

func (s *Store) Store(experience, lesson string, tags []string, scope string) (*db.Memory, error) {
	m := db.NewMemory(experience, lesson, tags)
	if scope != "" {
		m.Scope = scope
	}
	tagStr := strings.Join(m.Tags, ",")
	result, err := s.db.Conn().Exec(
		"INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		m.Date, m.Experience, m.Lesson, m.Impact, tagStr, m.Scope, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert memory: %w", err)
	}
	id, _ := result.LastInsertId()
	m.ID = id
	return m, nil
}

func (s *Store) Get(id int64) (*db.Memory, error) {
	m := &db.Memory{}
	var tags string
	err := s.db.Conn().QueryRow(
		"SELECT id, date, experience, lesson, impact, tags, scope, created_at, updated_at FROM memories WHERE id = ?", id,
	).Scan(&m.ID, &m.Date, &m.Experience, &m.Lesson, &m.Impact, &tags, &m.Scope, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	if tags != "" {
		m.Tags = strings.Split(tags, ",")
	}
	return m, nil
}

func (s *Store) ListPending() ([]db.Memory, error) {
	return s.listByImpact("under review")
}

func (s *Store) ListAll() ([]db.Memory, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, date, experience, lesson, impact, tags, scope, created_at, updated_at FROM memories ORDER BY date DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	return scanMemories(rows)
}

func (s *Store) Apply(id int64) error {
	return s.setImpact(id, "applied")
}

func (s *Store) Dismiss(id int64) error {
	return s.setImpact(id, "dismissed")
}

func (s *Store) UpdateImpact(id int64, impact string) error {
	return s.setImpact(id, impact)
}

func (s *Store) setImpact(id int64, impact string) error {
	_, err := s.db.Conn().Exec(
		"UPDATE memories SET impact = ?, updated_at = datetime('now') WHERE id = ?", impact, id,
	)
	return err
}

func (s *Store) listByImpact(impact string) ([]db.Memory, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, date, experience, lesson, impact, tags, scope, created_at, updated_at FROM memories WHERE impact = ? ORDER BY date DESC", impact,
	)
	if err != nil {
		return nil, fmt.Errorf("list by impact: %w", err)
	}
	defer rows.Close()
	return scanMemories(rows)
}

func (s *Store) Status() (*db.Status, error) {
	st := &db.Status{}
	s.db.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&st.MemoryCount)
	s.db.Conn().QueryRow("SELECT COUNT(*) FROM memories WHERE impact = 'under review'").Scan(&st.PendingCount)
	s.db.Conn().QueryRow("SELECT COALESCE(SUM(1), 0) FROM memories WHERE impact != 'under review'").Scan(&st.TotalEvolutions)
	return st, nil
}

func (s *Store) StoreSkillUsage(skill, context, withSkills, outcome string) (*db.SkillUsage, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	date := now[:10]
	if outcome == "" {
		outcome = "used"
	}
	result, err := s.db.Conn().Exec(
		"INSERT INTO skill_usage (date, skill, context, with_skills, outcome, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		date, skill, context, withSkills, outcome, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert skill_usage: %w", err)
	}
	id, _ := result.LastInsertId()
	return &db.SkillUsage{ID: id, Date: date, Skill: skill, Context: context, WithSkills: withSkills, Outcome: outcome, CreatedAt: now}, nil
}

func (s *Store) ListSkillUsage(limit int) ([]db.SkillUsage, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Conn().Query(
		"SELECT id, date, skill, context, with_skills, outcome, created_at FROM skill_usage ORDER BY date DESC LIMIT ?", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list skill_usage: %w", err)
	}
	defer rows.Close()
	var usages []db.SkillUsage
	for rows.Next() {
		var u db.SkillUsage
		if err := rows.Scan(&u.ID, &u.Date, &u.Skill, &u.Context, &u.WithSkills, &u.Outcome, &u.CreatedAt); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, rows.Err()
}

func (s *Store) GetSkillUsageBySkill(skill string, limit int) ([]db.SkillUsage, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Conn().Query(
		"SELECT id, date, skill, context, with_skills, outcome, created_at FROM skill_usage WHERE skill = ? ORDER BY date DESC LIMIT ?",
		skill, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get skill_usage by skill: %w", err)
	}
	defer rows.Close()
	var usages []db.SkillUsage
	for rows.Next() {
		var u db.SkillUsage
		if err := rows.Scan(&u.ID, &u.Date, &u.Skill, &u.Context, &u.WithSkills, &u.Outcome, &u.CreatedAt); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, rows.Err()
}

func scanMemories(rows *sql.Rows) ([]db.Memory, error) {
	var memories []db.Memory
	for rows.Next() {
		var m db.Memory
		var tags string
		if err := rows.Scan(&m.ID, &m.Date, &m.Experience, &m.Lesson, &m.Impact, &tags, &m.Scope, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if tags != "" {
			m.Tags = strings.Split(tags, ",")
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func (s *Store) SetUserProfile(field, value, source string, confidence float64) (*db.UserProfile, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if source == "" {
		source = "inferred"
	}
	if confidence <= 0 {
		confidence = 0.5
	}
	_, err := s.db.Conn().Exec(
		`INSERT INTO user_profiles (field, value, source, confidence, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(field) DO UPDATE SET
		   value = excluded.value,
		   source = excluded.source,
		   confidence = MAX(user_profiles.confidence, excluded.confidence),
		   updated_at = excluded.updated_at`,
		field, value, source, confidence, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("set user profile: %w", err)
	}
	return &db.UserProfile{Field: field, Value: value, Source: source, Confidence: confidence, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Store) GetUserProfile(field string) (*db.UserProfile, error) {
	p := &db.UserProfile{}
	err := s.db.Conn().QueryRow(
		"SELECT id, field, value, source, confidence, created_at, updated_at FROM user_profiles WHERE field = ?", field,
	).Scan(&p.ID, &p.Field, &p.Value, &p.Source, &p.Confidence, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user profile: %w", err)
	}
	return p, nil
}

func (s *Store) ListUserProfile() ([]db.UserProfile, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, field, value, source, confidence, created_at, updated_at FROM user_profiles ORDER BY field",
	)
	if err != nil {
		return nil, fmt.Errorf("list user profile: %w", err)
	}
	defer rows.Close()
	var profiles []db.UserProfile
	for rows.Next() {
		var p db.UserProfile
		if err := rows.Scan(&p.ID, &p.Field, &p.Value, &p.Source, &p.Confidence, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (s *Store) DeleteUserProfile(field string) error {
	_, err := s.db.Conn().Exec("DELETE FROM user_profiles WHERE field = ?", field)
	return err
}

func (s *Store) SetProjectContext(name, root, typ, lang string) (*db.ProjectContext, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	// Deactivate all others
	s.db.Conn().Exec("UPDATE project_contexts SET is_active = 0")
	_, err := s.db.Conn().Exec(
		`INSERT INTO project_contexts (name, root, type, lang, is_active, last_used, created_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		   root = excluded.root,
		   type = excluded.type,
		   lang = excluded.lang,
		   is_active = 1,
		   last_used = excluded.last_used`,
		name, root, typ, lang, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("set project context: %w", err)
	}
	return &db.ProjectContext{Name: name, Root: root, Type: typ, Lang: lang, IsActive: true, LastUsed: now, CreatedAt: now}, nil
}

func (s *Store) GetActiveProjectContext() (*db.ProjectContext, error) {
	p := &db.ProjectContext{}
	err := s.db.Conn().QueryRow(
		"SELECT id, name, root, type, lang, is_active, last_used, created_at FROM project_contexts WHERE is_active = 1",
	).Scan(&p.ID, &p.Name, &p.Root, &p.Type, &p.Lang, &p.IsActive, &p.LastUsed, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active project: %w", err)
	}
	return p, nil
}

func (s *Store) ListProjectContexts() ([]db.ProjectContext, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, name, root, type, lang, is_active, last_used, created_at FROM project_contexts ORDER BY last_used DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("list project contexts: %w", err)
	}
	defer rows.Close()
	var ctxs []db.ProjectContext
	for rows.Next() {
		var p db.ProjectContext
		if err := rows.Scan(&p.ID, &p.Name, &p.Root, &p.Type, &p.Lang, &p.IsActive, &p.LastUsed, &p.CreatedAt); err != nil {
			return nil, err
		}
		ctxs = append(ctxs, p)
	}
	return ctxs, rows.Err()
}

func (s *Store) DeleteProjectContext(name string) error {
	_, err := s.db.Conn().Exec("DELETE FROM project_contexts WHERE name = ?", name)
	return err
}
