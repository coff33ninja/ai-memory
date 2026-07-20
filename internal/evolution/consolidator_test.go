package evolution

import (
	"testing"
)

func newTestConsolidator(t *testing.T) *Consolidator {
	t.Helper()
	tr := newTestTracker(t)
	return NewConsolidator(tr, nil)
}

func TestConsolidateEmpty(t *testing.T) {
	c := newTestConsolidator(t)

	result, err := c.Consolidate("alice")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Merged != 0 {
		t.Errorf("merged = %d, want 0", result.Merged)
	}
	if result.Elevated != 0 {
		t.Errorf("elevated = %d, want 0", result.Elevated)
	}
	if result.Pruned != 0 {
		t.Errorf("pruned = %d, want 0", result.Pruned)
	}
	if result.PatternsCreated != 0 {
		t.Errorf("patterns = %d, want 0", result.PatternsCreated)
	}
}

func TestConsolidateMerge(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert two similar memories with embeddings
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, embedding, created_at, updated_at) VALUES ('2025-01-01', 'the quick brown fox jumps over the lazy dog', 'lesson one', 'applied', 'tag1', 'private', X'00', datetime('now'), datetime('now'))")
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, embedding, created_at, updated_at) VALUES ('2025-01-02', 'the quick brown fox jumps over the sleepy dog', 'lesson two', 'applied', 'tag2', 'private', X'00', datetime('now'), datetime('now'))")

	result, err := c.Consolidate("alice")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if result.Merged != 1 {
		t.Errorf("merged = %d, want 1", result.Merged)
	}

	var count int
	d.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 memory after merge, got %d", count)
	}
}

func TestConsolidatePrune(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert old dismissed memory (more than 30 days ago)
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES ('2024-01-01', 'old dismissed', 'lesson', 'dismissed', '', 'private', datetime('now'), datetime('now', '-60 days'))")

	result, err := c.Consolidate("alice")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if result.Pruned != 1 {
		t.Errorf("pruned = %d, want 1", result.Pruned)
	}
}

func TestConsolidateDoesNotPruneRecent(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert recent dismissed memory (less than 30 days ago)
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), 'recent dismissed', 'lesson', 'dismissed', '', 'private', datetime('now'), datetime('now'))")

	result, _ := c.Consolidate("alice")
	if result.Pruned != 0 {
		t.Errorf("pruned = %d, want 0 (too recent)", result.Pruned)
	}

	var count int
	d.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 memory (not pruned), got %d", count)
	}
}

func TestConsolidateElevate(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert "under review" memory with an embedding (has been searched/accessed)
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, embedding, created_at, updated_at) VALUES (date('now'), 'reviewed exp', 'lesson', 'under review', '', 'private', X'00', datetime('now'), datetime('now'))")

	result, err := c.Consolidate("alice")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if result.Elevated != 1 {
		t.Errorf("elevated = %d, want 1", result.Elevated)
	}

	// Verify impact changed to "applied"
	var impact string
	d.Conn().QueryRow("SELECT impact FROM memories LIMIT 1").Scan(&impact)
	if impact != "applied" {
		t.Errorf("impact = %q, want 'applied'", impact)
	}
}

func TestConsolidateDoesNotElevateWithoutEmbedding(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert "under review" memory WITHOUT embedding
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), 'no embedding', 'lesson', 'under review', '', 'private', datetime('now'), datetime('now'))")

	result, _ := c.Consolidate("alice")
	if result.Elevated != 0 {
		t.Errorf("elevated = %d, want 0 (no embedding)", result.Elevated)
	}
}

func TestConsolidateCreatesPatterns(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert 3 memories with the same tag
	for i := 0; i < 3; i++ {
		d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), ?, 'lesson', 'applied', 'recurring-tag', 'private', datetime('now'), datetime('now'))", "experience about topic")
	}

	result, err := c.Consolidate("alice")
	if err != nil {
		t.Fatalf("Consolidate: %v", err)
	}
	if result.PatternsCreated != 1 {
		t.Errorf("patterns = %d, want 1", result.PatternsCreated)
	}

	// Verify pattern memory was created
	var count int
	d.Conn().QueryRow("SELECT COUNT(*) FROM memories WHERE tags LIKE '%pattern%'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 pattern memory, got %d", count)
	}
}

func TestConsolidateDoesNotDuplicatePatterns(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)
	c := NewConsolidator(tr, nil)

	// Insert existing pattern memory matching the format createPatterns produces
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), 'Pattern observed 3 times: recurring-tag', 'lesson', 'applied', 'pattern,recurring-tag', 'private', datetime('now'), datetime('now'))")

	for i := 0; i < 3; i++ {
		d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), ?, 'lesson', 'applied', 'recurring-tag', 'private', datetime('now'), datetime('now'))", "experience about topic")
	}

	result, _ := c.Consolidate("alice")
	if result.PatternsCreated != 0 {
		t.Errorf("patterns = %d, want 0 (already exists)", result.PatternsCreated)
	}
}
