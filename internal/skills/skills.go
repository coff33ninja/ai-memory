package skills

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coff33ninja/ai-memory/internal/db"
	"gopkg.in/yaml.v3"
)

const skillsRepo = "https://github.com/coff33ninja/ai-skills"

type Store struct {
	db  *db.DB
	dir string
}

func New(d *db.DB, baseDir string) *Store {
	return &Store{db: d, dir: filepath.Join(baseDir, "skills")}
}

func (s *Store) EnsureCloned() error {
	if _, err := os.Stat(filepath.Join(s.dir, ".git")); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(s.dir), 0o755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}
	cmd := exec.Command("git", "clone", skillsRepo, s.dir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	return nil
}

func (s *Store) Sync() (string, error) {
	if err := s.EnsureCloned(); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "-C", s.dir, "pull", "--ff-only")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("sync failed: %s", string(out)), err
	}
	return "skills synced from remote", nil
}

func (s *Store) Index() (int, error) {
	if err := s.EnsureCloned(); err != nil {
		return 0, err
	}

	skillDirs, err := s.findSkillDirs()
	if err != nil {
		return 0, err
	}

	conn := s.db.Conn()
	now := time.Now().UTC().Format(time.RFC3339)
	total := 0

	for _, dirPath := range skillDirs {
		name := filepath.Base(dirPath)
		skillMd := filepath.Join(dirPath, "SKILL.md")
		raw, err := os.ReadFile(skillMd)
		if err != nil {
			continue
		}

		frontmatter, body := parseSkillFrontmatter(string(raw))
		desc, _ := frontmatter["description"].(string)

		// Upsert skill
		var skillID int64
		err = conn.QueryRow(
			"SELECT id FROM skills WHERE name = ?", name,
		).Scan(&skillID)

		if err == sql.ErrNoRows {
			result, err := conn.Exec(
				"INSERT INTO skills (name, description, body, file_count, synced_at) VALUES (?, ?, ?, ?, ?)",
				name, desc, body, 0, now,
			)
			if err != nil {
				continue
			}
			skillID, _ = result.LastInsertId()
		} else if err != nil {
			continue
		} else {
			conn.Exec("UPDATE skills SET description = ?, body = ?, synced_at = ? WHERE id = ?", desc, body, now, skillID)
		}

		// Clear old files
		conn.Exec("DELETE FROM skill_files WHERE skill_id = ?", skillID)

		// Index all files
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		fileCount := 0
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			fp := filepath.Join(dirPath, e.Name())
			content, err := os.ReadFile(fp)
			if err != nil {
				continue
			}
			conn.Exec(
				"INSERT INTO skill_files (skill_id, filename, content) VALUES (?, ?, ?)",
				skillID, e.Name(), string(content),
			)
			fileCount++
		}
		conn.Exec("UPDATE skills SET file_count = ? WHERE id = ?", fileCount, skillID)
		total++
	}

	return total, nil
}

func (s *Store) Catalog() ([]db.Skill, error) {
	rows, err := s.db.Conn().Query(
		"SELECT id, name, description, body, file_count, synced_at FROM skills ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []db.Skill
	for rows.Next() {
		var sk db.Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Body, &sk.FileCount, &sk.SyncedAt); err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

func (s *Store) GetFiles(name string) ([]db.SkillFile, error) {
	var skillID int64
	err := s.db.Conn().QueryRow("SELECT id FROM skills WHERE name = ?", name).Scan(&skillID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Conn().Query(
		"SELECT id, skill_id, filename, content FROM skill_files WHERE skill_id = ? ORDER BY filename", skillID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []db.SkillFile
	for rows.Next() {
		var f db.SkillFile
		if err := rows.Scan(&f.ID, &f.SkillID, &f.Filename, &f.Content); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *Store) SearchKeyword(query string) ([]db.Skill, error) {
	q := strings.ToLower(query)
	rows, err := s.db.Conn().Query(
		"SELECT id, name, description, body, file_count, synced_at FROM skills WHERE lower(name) LIKE ? OR lower(description) LIKE ? OR lower(body) LIKE ?",
		"%"+q+"%", "%"+q+"%", "%"+q+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []db.Skill
	for rows.Next() {
		var sk db.Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Body, &sk.FileCount, &sk.SyncedAt); err != nil {
			return nil, err
		}
		results = append(results, sk)
	}
	return results, rows.Err()
}

func (s *Store) Status() (int, error) {
	var count int
	err := s.db.Conn().QueryRow("SELECT COUNT(*) FROM skills").Scan(&count)
	return count, err
}

func (s *Store) findSkillDirs() ([]string, error) {
	searchRoots := []string{s.dir}
	nested := filepath.Join(s.dir, "skills")
	if info, err := os.Stat(nested); err == nil && info.IsDir() {
		searchRoots = append(searchRoots, nested)
	}

	var dirs []string
	for _, root := range searchRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			skillMd := filepath.Join(root, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillMd); err == nil {
				dirs = append(dirs, filepath.Join(root, e.Name()))
			}
		}
	}
	return dirs, nil
}

func parseSkillFrontmatter(raw string) (map[string]interface{}, string) {
	parts := strings.SplitN(raw, "---", 3)
	if len(parts) < 3 {
		return nil, raw
	}
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, raw
	}
	return fm, strings.TrimSpace(parts[2])
}
