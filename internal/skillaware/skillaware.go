package skillaware

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/coff33ninja/ai-memory/internal/embedding"
)

const (
	TagThreshold   = 0.55
	MaxSkillTags   = 5
	CacheRefreshN  = 50
)

type SkillEmbedding struct {
	ID          int64
	Name        string
	Description string
	Embedding   []float32
}

type Cache struct {
	db            *sql.DB
	skills        []SkillEmbedding
	mu            sync.RWMutex
	lastRefreshAt int
}

func NewCache(dbConn *sql.DB) *Cache {
	return &Cache{db: dbConn}
}

func (c *Cache) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	rows, err := c.db.Query(
		"SELECT id, name, description, embedding FROM skills WHERE embedding IS NOT NULL",
	)
	if err != nil {
		return fmt.Errorf("query skills: %w", err)
	}
	defer rows.Close()

	c.skills = c.skills[:0]
	for rows.Next() {
		var s SkillEmbedding
		var embBlob []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &embBlob); err != nil {
			continue
		}
		s.Embedding = embedding.BytesToFloat32(embBlob)
		c.skills = append(c.skills, s)
	}
	c.lastRefreshAt = 0
	return rows.Err()
}

func (c *Cache) GetSkills() []SkillEmbedding {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.skills
}

func (c *Cache) MaybeRefresh() {
	c.mu.Lock()
	c.lastRefreshAt++
	if c.lastRefreshAt < CacheRefreshN {
		c.mu.Unlock()
		return
	}
	c.lastRefreshAt = 0
	c.mu.Unlock()
	c.Refresh()
}

func TagMemory(memEmb []float32, skills []SkillEmbedding) (tags []string, scores map[string]float64) {
	if len(skills) == 0 || len(memEmb) == 0 {
		return nil, nil
	}

	type scored struct {
		name  string
		score float64
	}

	var results []scored
	for _, s := range skills {
		if len(s.Embedding) == 0 {
			continue
		}
		sim := embedding.CosineSimilarity(memEmb, s.Embedding)
		if sim >= TagThreshold {
			results = append(results, scored{name: s.Name, score: sim})
		}
	}

	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	if len(results) > MaxSkillTags {
		results = results[:MaxSkillTags]
	}

	tags = make([]string, len(results))
	scores = make(map[string]float64, len(results))
	for i, r := range results {
		tags[i] = r.name
		scores[r.name] = r.score
	}
	return tags, scores
}

func SuggestSkills(queryEmb []float32, skills []SkillEmbedding, topK int) []SuggestResult {
	if len(skills) == 0 || len(queryEmb) == 0 {
		return nil
	}

	type scored struct {
		name  string
		desc  string
		score float64
	}

	var results []scored
	for _, s := range skills {
		if len(s.Embedding) == 0 {
			continue
		}
		sim := embedding.CosineSimilarity(queryEmb, s.Embedding)
		if sim > 0.1 {
			results = append(results, scored{name: s.Name, desc: s.Description, score: sim})
		}
	}

	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	if topK > len(results) {
		topK = len(results)
	}

	out := make([]SuggestResult, topK)
	for i, r := range results[:topK] {
		out[i] = SuggestResult{
			Name:        r.name,
			Description: r.desc,
			Score:       r.score,
		}
	}
	return out
}

type SuggestResult struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

func FormatScores(scores map[string]float64) string {
	if len(scores) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(scores)
	return string(b)
}

func ParseScores(raw string) map[string]float64 {
	if raw == "" || raw == "{}" {
		return nil
	}
	var scores map[string]float64
	if err := json.Unmarshal([]byte(raw), &scores); err != nil {
		return nil
	}
	return scores
}

func TagsString(tags []string) string {
	return strings.Join(tags, ",")
}

func ParseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}
