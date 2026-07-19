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
