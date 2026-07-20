package evolution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coff33ninja/ai-memory/internal/db"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "evolution-test-"+t.Name())
	t.Cleanup(func() { os.RemoveAll(dir) })

	d, err := db.Open(dir)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func newTestTracker(t *testing.T) *Tracker {
	t.Helper()
	return NewTracker(newTestDB(t))
}

func TestLogOutcomeValid(t *testing.T) {
	tr := newTestTracker(t)

	o, err := tr.LogOutcome("alice", "fixed a bug", 4, "debugging", "direct")
	if err != nil {
		t.Fatalf("LogOutcome: %v", err)
	}
	if o.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if o.Persona != "alice" {
		t.Errorf("persona = %q, want %q", o.Persona, "alice")
	}
	if o.Score != 4 {
		t.Errorf("score = %d, want 4", o.Score)
	}
}

func TestLogOutcomeScoreBounds(t *testing.T) {
	tr := newTestTracker(t)

	for _, score := range []int{0, -1, 6, 100} {
		_, err := tr.LogOutcome("alice", "test", score, "", "")
		if err == nil {
			t.Errorf("expected error for score %d, got nil", score)
		}
	}

	for _, score := range []int{1, 3, 5} {
		_, err := tr.LogOutcome("alice", "test", score, "", "")
		if err != nil {
			t.Errorf("unexpected error for score %d: %v", score, err)
		}
	}
}

func TestRecentOutcomes(t *testing.T) {
	tr := newTestTracker(t)

	d := tr.db
	d.Conn().Exec("INSERT INTO interaction_outcomes (persona, summary, outcome_score, skills_used, tone_used, created_at) VALUES ('alice', 'first', 1, '', '', '2025-01-01T00:00:00Z')")
	d.Conn().Exec("INSERT INTO interaction_outcomes (persona, summary, outcome_score, skills_used, tone_used, created_at) VALUES ('alice', 'second', 3, '', '', '2025-01-02T00:00:00Z')")
	d.Conn().Exec("INSERT INTO interaction_outcomes (persona, summary, outcome_score, skills_used, tone_used, created_at) VALUES ('alice', 'third', 5, '', '', '2025-01-03T00:00:00Z')")

	outcomes, err := tr.RecentOutcomes("alice", 2)
	if err != nil {
		t.Fatalf("RecentOutcomes: %v", err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %d", len(outcomes))
	}
	if outcomes[0].Summary != "third" {
		t.Errorf("first result = %q, want %q (most recent)", outcomes[0].Summary, "third")
	}
}

func TestRecentOutcomesDefaultLimit(t *testing.T) {
	tr := newTestTracker(t)

	for i := 0; i < 5; i++ {
		tr.LogOutcome("alice", "test", 3, "", "")
	}

	outcomes, err := tr.RecentOutcomes("alice", 0)
	if err != nil {
		t.Fatalf("RecentOutcomes: %v", err)
	}
	if len(outcomes) != 5 {
		t.Errorf("expected 5 outcomes with default limit, got %d", len(outcomes))
	}
}

func TestRecentOutcomesEmptyPersona(t *testing.T) {
	tr := newTestTracker(t)

	outcomes, err := tr.RecentOutcomes("nonexistent", 10)
	if err != nil {
		t.Fatalf("RecentOutcomes: %v", err)
	}
	if len(outcomes) != 0 {
		t.Errorf("expected 0 outcomes, got %d", len(outcomes))
	}
}

func TestAverageScore(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogOutcome("alice", "a", 2, "", "")
	tr.LogOutcome("alice", "b", 4, "", "")

	avg, err := tr.AverageScore("alice", 10)
	if err != nil {
		t.Fatalf("AverageScore: %v", err)
	}
	if avg != 3.0 {
		t.Errorf("avg = %f, want 3.0", avg)
	}
}

func TestAverageScoreNoData(t *testing.T) {
	tr := newTestTracker(t)

	avg, err := tr.AverageScore("nobody", 10)
	if err != nil {
		t.Fatalf("AverageScore: %v", err)
	}
	if avg != 0.0 {
		t.Errorf("avg = %f, want 0.0", avg)
	}
}

func TestTonePerformance(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogOutcome("alice", "a", 5, "", "direct")
	tr.LogOutcome("alice", "b", 5, "", "direct")
	tr.LogOutcome("alice", "c", 1, "", "casual")

	perf, err := tr.TonePerformance("alice")
	if err != nil {
		t.Fatalf("TonePerformance: %v", err)
	}
	if len(perf) != 2 {
		t.Fatalf("expected 2 tones, got %d", len(perf))
	}
	if perf["direct"] != 5.0 {
		t.Errorf("direct score = %f, want 5.0", perf["direct"])
	}
	if perf["casual"] != 1.0 {
		t.Errorf("casual score = %f, want 1.0", perf["casual"])
	}
}

func TestSkillPerformance(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogOutcome("alice", "a", 4, "debugging", "")
	tr.LogOutcome("alice", "b", 2, "debugging", "")
	tr.LogOutcome("alice", "c", 5, "codegen", "")

	perf, err := tr.SkillPerformance("alice")
	if err != nil {
		t.Fatalf("SkillPerformance: %v", err)
	}
	if len(perf) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(perf))
	}
	// debugging appears twice with scores 4 and 2, avg = 3.0
	if perf["debugging"] != 3.0 {
		t.Errorf("debugging score = %f, want 3.0", perf["debugging"])
	}
	if perf["codegen"] != 5.0 {
		t.Errorf("codegen score = %f, want 5.0", perf["codegen"])
	}
}

func TestLogAndRetrieveEvolution(t *testing.T) {
	tr := newTestTracker(t)

	entry := NewEvolutionEntry("alice", "test_trigger", "tone_changed", "old", "new", 0.8)
	err := tr.LogEvolution(entry)
	if err != nil {
		t.Fatalf("LogEvolution: %v", err)
	}
	if entry.ID == 0 {
		t.Error("expected non-zero ID after logging")
	}

	history, err := tr.EvolutionHistory("alice", 10)
	if err != nil {
		t.Fatalf("EvolutionHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(history))
	}
	if history[0].Trigger != "test_trigger" {
		t.Errorf("trigger = %q, want %q", history[0].Trigger, "test_trigger")
	}
	if history[0].Before != "old" {
		t.Errorf("before = %q, want %q", history[0].Before, "old")
	}
}

func TestInteractionCount(t *testing.T) {
	tr := newTestTracker(t)

	count, _ := tr.InteractionCount("alice")
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	tr.LogOutcome("alice", "a", 3, "", "")
	tr.LogOutcome("alice", "b", 4, "", "")

	count, err := tr.InteractionCount("alice")
	if err != nil {
		t.Fatalf("InteractionCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestLogAndListToolGaps(t *testing.T) {
	tr := newTestTracker(t)

	err := tr.LogToolGap("alice", "need api", "context", "suggested fix")
	if err != nil {
		t.Fatalf("LogToolGap: %v", err)
	}

	gaps, err := tr.ToolGaps("alice", false)
	if err != nil {
		t.Fatalf("ToolGaps: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if gaps[0].Need != "need api" {
		t.Errorf("need = %q, want %q", gaps[0].Need, "need api")
	}
	if gaps[0].Resolved != 0 {
		t.Errorf("expected unresolved gap")
	}
}

func TestResolveToolGap(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolGap("alice", "need api", "", "")

	gaps, _ := tr.ToolGaps("alice", false)
	if len(gaps) != 1 {
		t.Fatal("expected 1 gap")
	}

	err := tr.ResolveToolGap(gaps[0].ID)
	if err != nil {
		t.Fatalf("ResolveToolGap: %v", err)
	}

	gaps, _ = tr.ToolGaps("alice", false)
	if len(gaps) != 0 {
		t.Errorf("expected 0 unresolved gaps after resolve, got %d", len(gaps))
	}

	gaps, _ = tr.ToolGaps("alice", true)
	if len(gaps) != 1 {
		t.Errorf("expected 1 total gap when including resolved, got %d", len(gaps))
	}
}

func TestUnresolvedGapCount(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolGap("alice", "a", "", "")
	tr.LogToolGap("alice", "b", "", "")

	count, err := tr.UnresolvedGapCount("alice")
	if err != nil {
		t.Fatalf("UnresolvedGapCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestLogAndGetToolKnowledge(t *testing.T) {
	tr := newTestTracker(t)

	err := tr.LogToolKnowledge("alice", "click_tool", "click at coords", "single click", "double click not supported", "x,y", "click 100,200")
	if err != nil {
		t.Fatalf("LogToolKnowledge: %v", err)
	}

	k, err := tr.GetToolKnowledge("alice", "click_tool")
	if err != nil {
		t.Fatalf("GetToolKnowledge: %v", err)
	}
	if k == nil {
		t.Fatal("expected non-nil knowledge")
	}
	if k.HowToUse != "click at coords" {
		t.Errorf("how_to_use = %q, want %q", k.HowToUse, "click at coords")
	}
	if k.UseCount != 1 {
		t.Errorf("use_count = %d, want 1", k.UseCount)
	}

	// Update existing
	tr.LogToolKnowledge("alice", "click_tool", "", "single click works great", "", "", "")
	k, _ = tr.GetToolKnowledge("alice", "click_tool")
	if k.WhatWorks != "single click works great" {
		t.Errorf("what_works = %q after update, want %q", k.WhatWorks, "single click works great")
	}
	if k.UseCount != 2 {
		t.Errorf("use_count = %d, want 2", k.UseCount)
	}
}

func TestListToolKnowledge(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolKnowledge("alice", "tool_a", "use a", "", "", "", "")
	tr.LogToolKnowledge("alice", "tool_b", "use b", "", "", "", "")

	items, err := tr.ListToolKnowledge("alice")
	if err != nil {
		t.Fatalf("ListToolKnowledge: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestAllToolKnowledge(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolKnowledge("alice", "tool_b", "", "", "", "", "")
	tr.LogToolKnowledge("alice", "tool_a", "", "", "", "", "")

	items, err := tr.AllToolKnowledge("alice")
	if err != nil {
		t.Fatalf("AllToolKnowledge: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ToolName != "tool_a" {
		t.Errorf("first item = %q, want %q (sorted by name)", items[0].ToolName, "tool_a")
	}
}

func TestLogAndGetToolRecipes(t *testing.T) {
	tr := newTestTracker(t)

	err := tr.LogToolRecipe("alice", "chain_tool", "login sequence", "step1,step2", "web login")
	if err != nil {
		t.Fatalf("LogToolRecipe: %v", err)
	}

	recipes, err := tr.GetToolRecipes("alice", "chain_tool")
	if err != nil {
		t.Fatalf("GetToolRecipes: %v", err)
	}
	if len(recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(recipes))
	}
	if recipes[0].RecipeName != "login sequence" {
		t.Errorf("recipe_name = %q, want %q", recipes[0].RecipeName, "login sequence")
	}
}

func TestRecordRecipeOutcome(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolRecipe("alice", "chain_tool", "login", "steps", "use")
	recipes, _ := tr.GetToolRecipes("alice", "chain_tool")
	recipeID := recipes[0].ID

	tr.RecordRecipeOutcome(recipeID, true)
	tr.RecordRecipeOutcome(recipeID, true)
	tr.RecordRecipeOutcome(recipeID, false)

	recipes, _ = tr.GetToolRecipes("alice", "chain_tool")
	if recipes[0].SuccessCount != 2 {
		t.Errorf("success_count = %d, want 2", recipes[0].SuccessCount)
	}
	if recipes[0].FailCount != 1 {
		t.Errorf("fail_count = %d, want 1", recipes[0].FailCount)
	}
}

func TestLogAndGetToolErrors(t *testing.T) {
	tr := newTestTracker(t)

	err := tr.LogToolError("alice", "click_tool", "click failed", "during login", `{"x":100}`)
	if err != nil {
		t.Fatalf("LogToolError: %v", err)
	}

	errors, err := tr.ToolErrors("alice", false)
	if err != nil {
		t.Fatalf("ToolErrors: %v", err)
	}
	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if errors[0].ErrorMsg != "click failed" {
		t.Errorf("error_msg = %q, want %q", errors[0].ErrorMsg, "click failed")
	}
}

func TestMarkErrorReportedAndResolved(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolError("alice", "tool", "error", "", "")
	errs, _ := tr.ToolErrors("alice", false)
	errID := errs[0].ID

	tr.MarkErrorReported(errID)
	tr.MarkErrorResolved(errID)

	errs, _ = tr.ToolErrors("alice", false)
	if len(errs) != 0 {
		t.Errorf("expected 0 unresolved errors, got %d", len(errs))
	}

	errs, _ = tr.ToolErrors("alice", true)
	if len(errs) != 1 {
		t.Fatalf("expected 1 total error, got %d", len(errs))
	}
	if errs[0].Reported != 1 {
		t.Errorf("reported = %d, want 1", errs[0].Reported)
	}
	if errs[0].Resolved != 1 {
		t.Errorf("resolved = %d, want 1", errs[0].Resolved)
	}
}

func TestUnresolvedErrorCount(t *testing.T) {
	tr := newTestTracker(t)

	tr.LogToolError("alice", "t1", "e1", "", "")
	tr.LogToolError("alice", "t2", "e2", "", "")

	count, err := tr.UnresolvedErrorCount("alice")
	if err != nil {
		t.Fatalf("UnresolvedErrorCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestUpsertAndGetMCPServer(t *testing.T) {
	tr := newTestTracker(t)

	err := tr.UpsertMCPServer("playwright", "github.com/ms/playwright", true, true, false, true, 25, "microsoft", "https://github.com/ms/playwright", "browser automation")
	if err != nil {
		t.Fatalf("UpsertMCPServer: %v", err)
	}

	s, err := tr.GetMCPServer("playwright")
	if err != nil {
		t.Fatalf("GetMCPServer: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.Name != "playwright" {
		t.Errorf("name = %q, want %q", s.Name, "playwright")
	}
	if s.ToolCount != 25 {
		t.Errorf("tool_count = %d, want 25", s.ToolCount)
	}
	if s.HasReport != 1 {
		t.Errorf("has_report = %d, want 1", s.HasReport)
	}
	if s.HasOCR != 0 {
		t.Errorf("has_ocr = %d, want 0", s.HasOCR)
	}

	// Update
	tr.UpsertMCPServer("playwright", "updated-source", false, false, false, false, 30, "", "", "updated desc")
	s, _ = tr.GetMCPServer("playwright")
	if s.ToolCount != 30 {
		t.Errorf("tool_count after update = %d, want 30", s.ToolCount)
	}
	if s.Source != "updated-source" {
		t.Errorf("source = %q, want %q", s.Source, "updated-source")
	}
}

func TestGetMCPServerNotFound(t *testing.T) {
	tr := newTestTracker(t)

	s, err := tr.GetMCPServer("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != nil {
		t.Error("expected nil for non-existent server")
	}
}

func TestListMCPServers(t *testing.T) {
	tr := newTestTracker(t)

	tr.UpsertMCPServer("a", "", false, false, false, false, 0, "", "", "")
	tr.UpsertMCPServer("b", "", false, false, false, false, 0, "", "", "")

	servers, err := tr.ListMCPServers()
	if err != nil {
		t.Fatalf("ListMCPServers: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if servers[0].Name != "a" {
		t.Errorf("first = %q, want %q (sorted by name)", servers[0].Name, "a")
	}
}

func TestMemoryCount(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)

	// Insert test memories
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES ('2025-01-01', 'exp1', 'lesson1', 'applied', 'tag1', 'private', datetime('now'), datetime('now'))")
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES ('2025-01-02', 'exp2', 'lesson2', 'under review', 'tag2', 'private', datetime('now'), datetime('now'))")
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES ('2025-01-03', 'exp3', 'lesson3', 'applied', 'tag3', 'shared', datetime('now'), datetime('now'))")

	count, err := tr.MemoryCount("alice", "")
	if err != nil {
		t.Fatalf("MemoryCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 non-shared memories, got %d", count)
	}

	count, err = tr.MemoryCount("alice", "applied")
	if err != nil {
		t.Fatalf("MemoryCount with impact: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 applied memory, got %d", count)
	}
}

func TestSimilarMemories(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)

	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, embedding, created_at, updated_at) VALUES ('2025-01-01', 'the quick brown fox jumps over the lazy dog', 'lesson1', 'applied', '', 'private', X'00', datetime('now'), datetime('now'))")
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, embedding, created_at, updated_at) VALUES ('2025-01-02', 'the quick brown fox jumps over the sleepy dog', 'lesson2', 'applied', '', 'private', X'00', datetime('now'), datetime('now'))")
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, embedding, created_at, updated_at) VALUES ('2025-01-03', 'completely different topic', 'lesson3', 'applied', '', 'private', X'00', datetime('now'), datetime('now'))")

	ids, err := tr.SimilarMemories(0.6)
	if err != nil {
		t.Fatalf("SimilarMemories: %v", err)
	}
	// "quick brown fox jumps over lazy dog" vs "quick brown fox jumps over sleepy dog"
	// overlap = 7/9 = 0.778 >= 0.6 threshold
	if len(ids) < 2 {
		t.Errorf("expected at least 2 IDs (one pair), got %d", len(ids))
	}
}

func TestMergeMemories(t *testing.T) {
	d := newTestDB(t)
	tr := NewTracker(d)

	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES ('2025-01-01', 'experience one', 'lesson one', 'applied', 'tagA,tagB', 'private', datetime('now'), datetime('now'))")
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES ('2025-02-01', 'experience two', 'lesson two', 'applied', 'tagB,tagC', 'private', datetime('now'), datetime('now'))")

	// Get IDs
	var id1, id2 int64
	d.Conn().QueryRow("SELECT id FROM memories ORDER BY id LIMIT 1").Scan(&id1)
	d.Conn().QueryRow("SELECT id FROM memories ORDER BY id DESC LIMIT 1").Scan(&id2)

	err := tr.MergeMemories([]int64{id1, id2})
	if err != nil {
		t.Fatalf("MergeMemories: %v", err)
	}

	// Should have only 1 memory now
	var count int
	d.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 memory after merge, got %d", count)
	}
}

func TestMergeMemoriesTooFew(t *testing.T) {
	tr := newTestTracker(t)
	err := tr.MergeMemories([]int64{1})
	if err != nil {
		t.Errorf("unexpected error for single ID: %v", err)
	}
}

func TestGetDB(t *testing.T) {
	tr := newTestTracker(t)
	if tr.GetDB() == nil {
		t.Error("expected non-nil sql.DB")
	}
}
