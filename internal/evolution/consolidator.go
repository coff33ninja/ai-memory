package evolution

import (
	"fmt"
	"strings"
	"time"

	"github.com/coff33ninja/ai-memory/internal/embedding"
)

type Consolidator struct {
	tracker *Tracker
	emb     *embedding.Embedder
}

func NewConsolidator(tracker *Tracker, emb *embedding.Embedder) *Consolidator {
	return &Consolidator{tracker: tracker, emb: emb}
}

type ConsolidationResult struct {
	Merged      int `json:"merged"`
	Elevated    int `json:"elevated"`
	Pruned      int `json:"pruned"`
	PatternsCreated int `json:"patterns_created"`
}

func (c *Consolidator) Consolidate(persona string) (*ConsolidationResult, error) {
	result := &ConsolidationResult{}

	// 1. Merge similar memories
	merged, err := c.mergeSimilar(persona)
	if err == nil {
		result.Merged = merged
	}

	// 2. Elevate frequently accessed memories
	elevated, err := c.elevateFrequent(persona)
	if err == nil {
		result.Elevated = elevated
	}

	// 3. Prune old dismissed memories
	pruned, err := c.pruneOld(persona)
	if err == nil {
		result.Pruned = pruned
	}

	// 4. Create pattern memories from repeated experiences
	patterns, err := c.createPatterns(persona)
	if err == nil {
		result.PatternsCreated = patterns
	}

	// Log the consolidation
	c.tracker.LogEvolution(NewEvolutionEntry(
		persona,
		"consolidation",
		"memory_consolidation",
		"",
		fmt.Sprintf("merged=%d elevated=%d pruned=%d patterns=%d", result.Merged, result.Elevated, result.Pruned, result.PatternsCreated),
		1.0,
	))

	return result, nil
}

func (c *Consolidator) mergeSimilar(persona string) (int, error) {
	// Find similar memory pairs using word overlap (fast pre-filter)
	ids, err := c.tracker.SimilarMemories(0.6)
	if err != nil || len(ids) < 2 {
		return 0, err
	}

	merged := 0
	processed := make(map[int64]bool)
	for i := 0; i < len(ids); i += 2 {
		if i+1 >= len(ids) {
			break
		}
		id1, id2 := ids[i], ids[i+1]
		if processed[id1] || processed[id2] {
			continue
		}

		if err := c.tracker.MergeMemories([]int64{id1, id2}); err == nil {
			merged++
			processed[id1] = true
			processed[id2] = true
		}
	}
	return merged, nil
}

func (c *Consolidator) elevateFrequent(persona string) (int, error) {
	// Memories with impact "applied" or "under review" that appear in multiple search results get elevated
	rows, err := c.tracker.GetDB().Query(
		"SELECT id, impact FROM memories WHERE impact = 'under review' AND scope != 'shared'",
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var candidates []struct {
		ID     int64
		Impact string
	}
	for rows.Next() {
		var id int64
		var impact string
		if err := rows.Scan(&id, &impact); err != nil {
			continue
		}
		candidates = append(candidates, struct {
			ID     int64
			Impact string
		}{id, impact})
	}

	elevated := 0
	for _, m := range candidates {
		// Check if this memory has been accessed (has embedding = was searched)
		var hasEmbedding bool
		err := c.tracker.GetDB().QueryRow(
			"SELECT embedding IS NOT NULL FROM memories WHERE id = ?", m.ID,
		).Scan(&hasEmbedding)
		if err == nil && hasEmbedding {
			// Elevate to applied
			c.tracker.GetDB().Exec(
				"UPDATE memories SET impact = 'applied', updated_at = datetime('now') WHERE id = ?",
				m.ID,
			)
			elevated++
		}
	}
	return elevated, nil
}

func (c *Consolidator) pruneOld(persona string) (int, error) {
	// Prune dismissed memories older than 30 days
	result, err := c.tracker.GetDB().Exec(
		"DELETE FROM memories WHERE impact = 'dismissed' AND updated_at < datetime('now', '-30 days')",
	)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

func (c *Consolidator) createPatterns(persona string) (int, error) {
	// Find tags that appear in multiple memories
	rows, err := c.tracker.GetDB().Query(
		"SELECT tags, COUNT(*) as cnt FROM memories WHERE tags != '' AND scope != 'shared' GROUP BY tags HAVING cnt >= 3",
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type tagPattern struct {
		Tags  string
		Count int
	}
	var patterns []tagPattern
	for rows.Next() {
		var p tagPattern
		if err := rows.Scan(&p.Tags, &p.Count); err != nil {
			continue
		}
		patterns = append(patterns, p)
	}

	created := 0
	for _, p := range patterns {
		// Check if pattern memory already exists
		var exists bool
		err := c.tracker.GetDB().QueryRow(
			"SELECT COUNT(*) > 0 FROM memories WHERE experience LIKE ? AND tags LIKE '%pattern%'",
			fmt.Sprintf("Pattern: %%(%s)%%", p.Tags),
		).Scan(&exists)
		if err != nil || exists {
			continue
		}

		// Get experiences with these tags
		tagList := strings.Split(p.Tags, ",")
		placeholders := make([]string, len(tagList))
		args := make([]interface{}, len(tagList))
		for i, t := range tagList {
			placeholders[i] = "tags LIKE ?"
			args[i] = "%" + strings.TrimSpace(t) + "%"
		}

		query := fmt.Sprintf("SELECT experience FROM memories WHERE %s LIMIT 5", strings.Join(placeholders, " OR "))
		expRows, err := c.tracker.GetDB().Query(query, args...)
		if err != nil {
			continue
		}

		var experiences []string
		for expRows.Next() {
			var exp string
			if err := expRows.Scan(&exp); err == nil {
				experiences = append(experiences, exp)
			}
		}
		expRows.Close()

		if len(experiences) < 3 {
			continue
		}

		// Create pattern memory
		patternExp := fmt.Sprintf("Pattern observed %d times: %s", p.Count, p.Tags)
		patternLesson := fmt.Sprintf("Recurring theme across %d experiences: %s", p.Count, strings.Join(experiences[:3], "; "))

		_, err = c.tracker.GetDB().Exec(
			"INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (?, ?, ?, 'applied', ?, 'private', datetime('now'), datetime('now'))",
			time.Now().UTC().Format("2006-01-02"), patternExp, patternLesson, "pattern,"+p.Tags,
		)
		if err == nil {
			created++
		}
	}
	return created, nil
}
