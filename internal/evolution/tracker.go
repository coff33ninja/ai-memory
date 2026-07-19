package evolution

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/coff33ninja/ai-memory/internal/db"
)

type Tracker struct {
	db *db.DB
}

func NewTracker(d *db.DB) *Tracker {
	return &Tracker{db: d}
}

func (t *Tracker) LogOutcome(persona, summary string, score int, skillsUsed, toneUsed string) (*InteractionOutcome, error) {
	if score < 1 || score > 5 {
		return nil, fmt.Errorf("score must be 1-5, got %d", score)
	}
	o := NewInteractionOutcome(persona, summary, score, skillsUsed, toneUsed)
	result, err := t.db.Conn().Exec(
		"INSERT INTO interaction_outcomes (persona, summary, outcome_score, skills_used, tone_used, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		o.Persona, o.Summary, o.Score, o.SkillsUsed, o.ToneUsed, o.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("log outcome: %w", err)
	}
	id, _ := result.LastInsertId()
	o.ID = id
	return o, nil
}

func (t *Tracker) RecentOutcomes(persona string, limit int) ([]InteractionOutcome, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := t.db.Conn().Query(
		"SELECT id, persona, summary, outcome_score, skills_used, tone_used, created_at FROM interaction_outcomes WHERE persona = ? ORDER BY created_at DESC LIMIT ?",
		persona, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outcomes []InteractionOutcome
	for rows.Next() {
		var o InteractionOutcome
		if err := rows.Scan(&o.ID, &o.Persona, &o.Summary, &o.Score, &o.SkillsUsed, &o.ToneUsed, &o.CreatedAt); err != nil {
			continue
		}
		outcomes = append(outcomes, o)
	}
	return outcomes, rows.Err()
}

func (t *Tracker) AverageScore(persona string, lastN int) (float64, error) {
	if lastN <= 0 {
		lastN = 10
	}
	var avg float64
	err := t.db.Conn().QueryRow(
		"SELECT COALESCE(AVG(outcome_score), 0) FROM (SELECT outcome_score FROM interaction_outcomes WHERE persona = ? ORDER BY created_at DESC LIMIT ?)",
		persona, lastN,
	).Scan(&avg)
	return avg, err
}

func (t *Tracker) TonePerformance(persona string) (map[string]float64, error) {
	rows, err := t.db.Conn().Query(
		"SELECT tone_used, AVG(outcome_score) as avg_score FROM interaction_outcomes WHERE persona = ? AND tone_used != '' GROUP BY tone_used ORDER BY avg_score DESC",
		persona,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var tone string
		var avg float64
		if err := rows.Scan(&tone, &avg); err != nil {
			continue
		}
		result[tone] = avg
	}
	return result, nil
}

func (t *Tracker) SkillPerformance(persona string) (map[string]float64, error) {
	rows, err := t.db.Conn().Query(
		"SELECT skills_used, AVG(outcome_score) as avg_score FROM interaction_outcomes WHERE persona = ? AND skills_used != '' GROUP BY skills_used ORDER BY avg_score DESC",
		persona,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var skills string
		var avg float64
		if err := rows.Scan(&skills, &avg); err != nil {
			continue
		}
		for _, s := range strings.Split(skills, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				result[s] = avg
			}
		}
	}
	return result, nil
}

func (t *Tracker) LogEvolution(entry *EvolutionEntry) error {
	result, err := t.db.Conn().Exec(
		"INSERT INTO evolution_log (persona, trigger, what_changed, before_val, after_val, confidence, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		entry.Persona, entry.Trigger, entry.WhatChanged, entry.Before, entry.After, entry.Confidence, entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("log evolution: %w", err)
	}
	id, _ := result.LastInsertId()
	entry.ID = id
	return nil
}

func (t *Tracker) EvolutionHistory(persona string, limit int) ([]EvolutionEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := t.db.Conn().Query(
		"SELECT id, persona, trigger, what_changed, before_val, after_val, confidence, created_at FROM evolution_log WHERE persona = ? ORDER BY created_at DESC LIMIT ?",
		persona, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EvolutionEntry
	for rows.Next() {
		var e EvolutionEntry
		if err := rows.Scan(&e.ID, &e.Persona, &e.Trigger, &e.WhatChanged, &e.Before, &e.After, &e.Confidence, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (t *Tracker) InteractionCount(persona string) (int, error) {
	var count int
	err := t.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM interaction_outcomes WHERE persona = ?", persona,
	).Scan(&count)
	return count, err
}

func (t *Tracker) LogToolGap(persona, need, context, suggested string) error {
	_, err := t.db.Conn().Exec(
		"INSERT INTO tool_gaps (persona, need, context, suggested, created_at) VALUES (?, ?, ?, ?, datetime('now'))",
		persona, need, context, suggested,
	)
	return err
}

func (t *Tracker) ToolGaps(persona string, includeResolved bool) ([]ToolGap, error) {
	query := "SELECT id, persona, need, context, suggested, resolved, created_at FROM tool_gaps WHERE persona = ?"
	if !includeResolved {
		query += " AND resolved = 0"
	}
	query += " ORDER BY created_at DESC"

	rows, err := t.db.Conn().Query(query, persona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gaps []ToolGap
	for rows.Next() {
		var g ToolGap
		if err := rows.Scan(&g.ID, &g.Persona, &g.Need, &g.Context, &g.Suggested, &g.Resolved, &g.CreatedAt); err != nil {
			continue
		}
		gaps = append(gaps, g)
	}
	return gaps, rows.Err()
}

func (t *Tracker) ResolveToolGap(id int64) error {
	_, err := t.db.Conn().Exec("UPDATE tool_gaps SET resolved = 1 WHERE id = ?", id)
	return err
}

func (t *Tracker) UnresolvedGapCount(persona string) (int, error) {
	var count int
	err := t.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM tool_gaps WHERE persona = ? AND resolved = 0", persona,
	).Scan(&count)
	return count, err
}

func (t *Tracker) LogToolKnowledge(persona, toolName, howToUse, whatWorks, whatFails, params, examples string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Check if exists
	var exists bool
	err := t.db.Conn().QueryRow(
		"SELECT COUNT(*) > 0 FROM tool_knowledge WHERE persona = ? AND tool_name = ?",
		persona, toolName,
	).Scan(&exists)

	if exists {
		// Update existing
		_, err = t.db.Conn().Exec(
			`UPDATE tool_knowledge SET
			   how_to_use = CASE WHEN ? != '' THEN ? ELSE how_to_use END,
			   what_works = CASE WHEN ? != '' THEN ? ELSE what_works END,
			   what_fails = CASE WHEN ? != '' THEN ? ELSE what_fails END,
			   params = CASE WHEN ? != '' THEN ? ELSE params END,
			   examples = CASE WHEN ? != '' THEN ? ELSE examples END,
			   use_count = use_count + 1,
			   last_used = ?,
			   updated_at = ?
			 WHERE persona = ? AND tool_name = ?`,
			howToUse, howToUse, whatWorks, whatWorks, whatFails, whatFails, params, params, examples, examples, now, now, persona, toolName,
		)
	} else {
		// Insert new
		_, err = t.db.Conn().Exec(
			`INSERT INTO tool_knowledge (persona, tool_name, how_to_use, what_works, what_fails, params, examples, use_count, last_used, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)`,
			persona, toolName, howToUse, whatWorks, whatFails, params, examples, now, now, now,
		)
	}
	return err
}

func (t *Tracker) GetToolKnowledge(persona, toolName string) (*ToolKnowledge, error) {
	var k ToolKnowledge
	var lastUsed sql.NullString
	err := t.db.Conn().QueryRow(
		"SELECT id, persona, tool_name, how_to_use, what_works, what_fails, params, examples, use_count, last_used, created_at, updated_at FROM tool_knowledge WHERE persona = ? AND tool_name = ?",
		persona, toolName,
	).Scan(&k.ID, &k.Persona, &k.ToolName, &k.HowToUse, &k.WhatWorks, &k.WhatFails, &k.Params, &k.Examples, &k.UseCount, &lastUsed, &k.CreatedAt, &k.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if lastUsed.Valid {
		k.LastUsed = lastUsed.String
	}
	return &k, err
}

func (t *Tracker) ListToolKnowledge(persona string) ([]ToolKnowledge, error) {
	rows, err := t.db.Conn().Query(
		"SELECT id, persona, tool_name, how_to_use, what_works, what_fails, params, examples, use_count, last_used, created_at, updated_at FROM tool_knowledge WHERE persona = ? ORDER BY use_count DESC",
		persona,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ToolKnowledge
	for rows.Next() {
		var k ToolKnowledge
		var lastUsed sql.NullString
		if err := rows.Scan(&k.ID, &k.Persona, &k.ToolName, &k.HowToUse, &k.WhatWorks, &k.WhatFails, &k.Params, &k.Examples, &k.UseCount, &lastUsed, &k.CreatedAt, &k.UpdatedAt); err != nil {
			continue
		}
		if lastUsed.Valid {
			k.LastUsed = lastUsed.String
		}
		items = append(items, k)
	}
	return items, rows.Err()
}

func (t *Tracker) LogToolRecipe(persona, toolName, recipeName, steps, useCase string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := t.db.Conn().Exec(
		"INSERT INTO tool_recipes (persona, tool_name, recipe_name, steps, use_case, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		persona, toolName, recipeName, steps, useCase, now,
	)
	return err
}

func (t *Tracker) GetToolRecipes(persona, toolName string) ([]ToolRecipe, error) {
	rows, err := t.db.Conn().Query(
		"SELECT id, persona, tool_name, recipe_name, steps, use_case, success_count, fail_count, created_at FROM tool_recipes WHERE persona = ? AND tool_name = ? ORDER BY success_count DESC",
		persona, toolName,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []ToolRecipe
	for rows.Next() {
		var r ToolRecipe
		if err := rows.Scan(&r.ID, &r.Persona, &r.ToolName, &r.RecipeName, &r.Steps, &r.UseCase, &r.SuccessCount, &r.FailCount, &r.CreatedAt); err != nil {
			continue
		}
		recipes = append(recipes, r)
	}
	return recipes, rows.Err()
}

func (t *Tracker) RecordRecipeOutcome(recipeID int64, success bool) error {
	if success {
		_, err := t.db.Conn().Exec("UPDATE tool_recipes SET success_count = success_count + 1 WHERE id = ?", recipeID)
		return err
	}
	_, err := t.db.Conn().Exec("UPDATE tool_recipes SET fail_count = fail_count + 1 WHERE id = ?", recipeID)
	return err
}

func (t *Tracker) AllToolKnowledge(persona string) ([]ToolKnowledge, error) {
	rows, err := t.db.Conn().Query(
		"SELECT id, persona, tool_name, how_to_use, what_works, what_fails, params, examples, use_count, last_used, created_at, updated_at FROM tool_knowledge WHERE persona = ? ORDER BY tool_name",
		persona,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ToolKnowledge
	for rows.Next() {
		var k ToolKnowledge
		var lastUsed sql.NullString
		if err := rows.Scan(&k.ID, &k.Persona, &k.ToolName, &k.HowToUse, &k.WhatWorks, &k.WhatFails, &k.Params, &k.Examples, &k.UseCount, &lastUsed, &k.CreatedAt, &k.UpdatedAt); err != nil {
			continue
		}
		if lastUsed.Valid {
			k.LastUsed = lastUsed.String
		}
		items = append(items, k)
	}
	return items, rows.Err()
}

func (t *Tracker) LogToolError(persona, toolName, errorMsg, context, inputArgs string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := t.db.Conn().Exec(
		"INSERT INTO tool_errors (persona, tool_name, error_msg, context, input_args, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		persona, toolName, errorMsg, context, inputArgs, now,
	)
	return err
}

func (t *Tracker) ToolErrors(persona string, includeResolved bool) ([]ToolError, error) {
	query := "SELECT id, persona, tool_name, error_msg, context, input_args, resolved, reported, created_at FROM tool_errors WHERE persona = ?"
	if !includeResolved {
		query += " AND resolved = 0"
	}
	query += " ORDER BY created_at DESC LIMIT 20"

	rows, err := t.db.Conn().Query(query, persona)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var toolErrors []ToolError
	for rows.Next() {
		var e ToolError
		if err := rows.Scan(&e.ID, &e.Persona, &e.ToolName, &e.ErrorMsg, &e.Context, &e.InputArgs, &e.Resolved, &e.Reported, &e.CreatedAt); err != nil {
			continue
		}
		toolErrors = append(toolErrors, e)
	}
	return toolErrors, rows.Err()
}

func (t *Tracker) MarkErrorReported(id int64) error {
	_, err := t.db.Conn().Exec("UPDATE tool_errors SET reported = 1 WHERE id = ?", id)
	return err
}

func (t *Tracker) MarkErrorResolved(id int64) error {
	_, err := t.db.Conn().Exec("UPDATE tool_errors SET resolved = 1 WHERE id = ?", id)
	return err
}

func (t *Tracker) UpsertMCPServer(name, source string, hasReport, hasScreenshot, hasOCR, hasChain bool, toolCount int, creator, repoURL, description string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	reportInt := 0
	if hasReport {
		reportInt = 1
	}
	screenshotInt := 0
	if hasScreenshot {
		screenshotInt = 1
	}
	ocrInt := 0
	if hasOCR {
		ocrInt = 1
	}
	chainInt := 0
	if hasChain {
		chainInt = 1
	}

	var exists bool
	err := t.db.Conn().QueryRow("SELECT COUNT(*) > 0 FROM mcp_servers WHERE name = ?", name).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		_, err = t.db.Conn().Exec(
			`UPDATE mcp_servers SET source = ?, has_report = ?, has_screenshot = ?, has_ocr = ?, has_chain = ?, tool_count = ?, creator = ?, repo_url = ?, description = ?, last_seen = ? WHERE name = ?`,
			source, reportInt, screenshotInt, ocrInt, chainInt, toolCount, creator, repoURL, description, now, name,
		)
	} else {
		_, err = t.db.Conn().Exec(
			`INSERT INTO mcp_servers (name, source, has_report, has_screenshot, has_ocr, has_chain, tool_count, creator, repo_url, description, last_seen, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			name, source, reportInt, screenshotInt, ocrInt, chainInt, toolCount, creator, repoURL, description, now, now,
		)
	}
	return err
}

func (t *Tracker) GetMCPServer(name string) (*MCPServer, error) {
	var s MCPServer
	var lastSeen sql.NullString
	err := t.db.Conn().QueryRow(
		"SELECT id, name, source, has_report, has_screenshot, has_ocr, has_chain, tool_count, creator, repo_url, description, last_seen, created_at FROM mcp_servers WHERE name = ?",
		name,
	).Scan(&s.ID, &s.Name, &s.Source, &s.HasReport, &s.HasScreenshot, &s.HasOCR, &s.HasChain, &s.ToolCount, &s.Creator, &s.RepoURL, &s.Description, &lastSeen, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if lastSeen.Valid {
		s.LastSeen = lastSeen.String
	}
	return &s, err
}

func (t *Tracker) ListMCPServers() ([]MCPServer, error) {
	rows, err := t.db.Conn().Query(
		"SELECT id, name, source, has_report, has_screenshot, has_ocr, has_chain, tool_count, creator, repo_url, description, last_seen, created_at FROM mcp_servers ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []MCPServer
	for rows.Next() {
		var s MCPServer
		var lastSeen sql.NullString
		if err := rows.Scan(&s.ID, &s.Name, &s.Source, &s.HasReport, &s.HasScreenshot, &s.HasOCR, &s.HasChain, &s.ToolCount, &s.Creator, &s.RepoURL, &s.Description, &lastSeen, &s.CreatedAt); err != nil {
			continue
		}
		if lastSeen.Valid {
			s.LastSeen = lastSeen.String
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

func (t *Tracker) UnresolvedErrorCount(persona string) (int, error) {
	var count int
	err := t.db.Conn().QueryRow(
		"SELECT COUNT(*) FROM tool_errors WHERE persona = ? AND resolved = 0", persona,
	).Scan(&count)
	return count, err
}

func (t *Tracker) MemoryCount(persona, impact string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM memories WHERE scope != 'shared'"
	if impact != "" {
		query += " AND impact = ?"
		err := t.db.Conn().QueryRow(query, impact).Scan(&count)
		return count, err
	}
	err := t.db.Conn().QueryRow(query).Scan(&count)
	return count, err
}

func (t *Tracker) SimilarMemories(threshold float64) ([]int64, error) {
	rows, err := t.db.Conn().Query(
		"SELECT id, experience, lesson, embedding FROM memories WHERE embedding IS NOT NULL",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type memEntry struct {
		id     int64
		text   string
		embBin []byte
	}
	var entries []memEntry
	for rows.Next() {
		var e memEntry
		if err := rows.Scan(&e.id, &e.text, &e.embBin); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	// Find pairs with high similarity
	var toMerge []int64
	seen := make(map[int64]bool)
	for i := 0; i < len(entries); i++ {
		if seen[entries[i].id] {
			continue
		}
		for j := i + 1; j < len(entries); j++ {
			if seen[entries[j].id] {
				continue
			}
			// Simple Jaccard on words as fast pre-filter
			sim := wordOverlap(entries[i].text, entries[j].text)
			if sim >= threshold {
				toMerge = append(toMerge, entries[i].id, entries[j].id)
				seen[entries[i].id] = true
				seen[entries[j].id] = true
				break
			}
		}
	}
	return toMerge, nil
}

func wordOverlap(a, b string) float64 {
	setA := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(a)) {
		setA[w] = true
	}
	setB := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(b)) {
		setB[w] = true
	}
	inter := 0
	for w := range setA {
		if setB[w] {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func (t *Tracker) MergeMemories(ids []int64) error {
	if len(ids) < 2 {
		return nil
	}

	// Read all memories
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	var memories []struct {
		ID         int64
		Date       string
		Experience string
		Lesson     string
		Tags       string
		Scope      string
	}

	query := fmt.Sprintf("SELECT id, date, experience, lesson, tags, scope FROM memories WHERE id IN (%s)", strings.Join(placeholders, ","))
	rows, err := t.db.Conn().Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var m struct {
			ID         int64
			Date       string
			Experience string
			Lesson     string
			Tags       string
			Scope      string
		}
		if err := rows.Scan(&m.ID, &m.Date, &m.Experience, &m.Lesson, &m.Tags, &m.Scope); err != nil {
			continue
		}
		memories = append(memories, m)
	}

	if len(memories) < 2 {
		return nil
	}

	// Merge: combine experiences and lessons, keep earliest date, union tags
	mergedExp := make([]string, 0, len(memories))
	mergedLesson := make([]string, 0, len(memories))
	tagSet := make(map[string]bool)
	earliest := memories[0].Date
	keepID := memories[0].ID
	scope := memories[0].Scope

	for _, m := range memories {
		mergedExp = append(mergedExp, m.Experience)
		mergedLesson = append(mergedLesson, m.Lesson)
		if m.Date < earliest {
			earliest = m.Date
		}
		if m.Scope == "shared" {
			scope = "shared"
		}
		for _, t := range strings.Split(m.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagSet[t] = true
			}
		}
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}

	// Update kept memory
	_, err = t.db.Conn().Exec(
		"UPDATE memories SET experience = ?, lesson = ?, date = ?, tags = ?, scope = ?, impact = 'applied', updated_at = datetime('now') WHERE id = ?",
		strings.Join(mergedExp, " | "), strings.Join(mergedLesson, " | "), earliest, strings.Join(tags, ","), scope, keepID,
	)
	if err != nil {
		return err
	}

	// Delete merged memories (keep the first one)
	deleteIDs := ids[1:]
	deletePlaceholders := make([]string, len(deleteIDs))
	deleteArgs := make([]interface{}, len(deleteIDs))
	for i, id := range deleteIDs {
		deletePlaceholders[i] = "?"
		deleteArgs[i] = id
	}
	deleteQuery := fmt.Sprintf("DELETE FROM memories WHERE id IN (%s)", strings.Join(deletePlaceholders, ","))
	_, err = t.db.Conn().Exec(deleteQuery, deleteArgs...)

	return err
}

// GetDB returns the underlying database for direct access by other evolution components
func (t *Tracker) GetDB() *sql.DB {
	return t.db.Conn()
}
