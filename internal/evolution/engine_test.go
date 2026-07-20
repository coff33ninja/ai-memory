package evolution

import (
	"testing"

	"github.com/coff33ninja/ai-memory/internal/embedding"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	d := newTestDB(t)
	m := newTestPersonaManager(t)
	return NewEngine(m, d, nil)
}

func TestShouldEvolveInteractionThreshold(t *testing.T) {
	e := newTestEngine(t)

	// Log 9 interactions — should NOT trigger
	for i := 0; i < 9; i++ {
		e.tracker.LogOutcome("alice", "test", 3, "", "")
	}
	ok, _ := e.ShouldEvolve("alice")
	if ok {
		t.Error("should not evolve at 9 interactions")
	}

	// 10th interaction — SHOULD trigger
	e.tracker.LogOutcome("alice", "test", 3, "", "")
	ok, trigger := e.ShouldEvolve("alice")
	if !ok {
		t.Error("should evolve at 10 interactions")
	}
	if trigger == "" {
		t.Error("expected non-empty trigger description")
	}
}

func TestShouldEvolveMemoryThreshold(t *testing.T) {
	e := newTestEngine(t)

	// Insert 49 non-shared memories directly
	d := e.tracker.db
	for i := 0; i < 49; i++ {
		d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), ?, 'lesson', 'applied', '', 'private', datetime('now'), datetime('now'))", "exp")
	}

	ok, _ := e.ShouldEvolve("alice")
	if ok {
		t.Error("should not evolve at 49 memories")
	}

	// 50th memory — SHOULD trigger
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), '50th', 'lesson', 'applied', '', 'private', datetime('now'), datetime('now'))")
	ok, trigger := e.ShouldEvolve("alice")
	if !ok {
		t.Error("should evolve at 50 memories")
	}
	if trigger == "" {
		t.Error("expected non-empty trigger description")
	}
}

func TestShouldEvolveNoTrigger(t *testing.T) {
	e := newTestEngine(t)

	// 5 interactions — below threshold
	for i := 0; i < 5; i++ {
		e.tracker.LogOutcome("alice", "test", 3, "", "")
	}
	ok, _ := e.ShouldEvolve("alice")
	if ok {
		t.Error("should not evolve at 5 interactions")
	}
}

func TestEvolve(t *testing.T) {
	e := newTestEngine(t)

	// Insert some memories for consolidation to work on
	d := e.tracker.db
	d.Conn().Exec("INSERT INTO memories (date, experience, lesson, impact, tags, scope, created_at, updated_at) VALUES (date('now'), 'test exp', 'test lesson', 'applied', '', 'private', datetime('now'), datetime('now'))")

	result, err := e.Evolve("alice")
	if err != nil {
		t.Fatalf("Evolve: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Persona != "alice" {
		t.Errorf("persona = %q, want %q", result.Persona, "alice")
	}
	if result.MemoryConsolidated == nil {
		t.Error("expected non-nil MemoryConsolidated")
	}
	if result.TraitsAdapted == nil {
		t.Error("expected non-nil TraitsAdapted")
	}

	// Evolve logs entries from consolidation + the full evolution cycle
	history, _ := e.tracker.EvolutionHistory("alice", 10)
	if len(history) < 2 {
		t.Fatalf("expected at least 2 evolution entries, got %d", len(history))
	}
	// Verify the evolution_cycle entry is present (order may vary within same second)
	found := false
	for _, h := range history {
		if h.Trigger == "evolution_cycle" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'evolution_cycle' entry in history")
	}
}

func TestLogInteractionAutoEvolve(t *testing.T) {
	e := newTestEngine(t)

	// Log 9 interactions
	for i := 0; i < 9; i++ {
		err := e.LogInteraction("alice", "test", 3, "", "")
		if err != nil {
			t.Fatalf("LogInteraction %d: %v", i+1, err)
		}
	}

	// Verify no evolution yet
	history, _ := e.tracker.EvolutionHistory("alice", 10)
	if len(history) != 0 {
		t.Errorf("expected 0 evolution entries before threshold, got %d", len(history))
	}

	// 10th should auto-trigger
	err := e.LogInteraction("alice", "tenth", 4, "", "")
	if err != nil {
		t.Fatalf("LogInteraction 10: %v", err)
	}

	history, _ = e.tracker.EvolutionHistory("alice", 10)
	if len(history) == 0 {
		t.Error("expected evolution entry after 10th interaction")
	}
}

func TestLogInteractionScoreBounds(t *testing.T) {
	e := newTestEngine(t)

	err := e.LogInteraction("alice", "test", 0, "", "")
	if err == nil {
		t.Error("expected error for score 0")
	}

	err = e.LogInteraction("alice", "test", 6, "", "")
	if err == nil {
		t.Error("expected error for score 6")
	}
}

func TestEvolutionHistory(t *testing.T) {
	e := newTestEngine(t)

	e.tracker.LogEvolution(NewEvolutionEntry("alice", "trigger1", "change1", "", "", 1.0))
	e.tracker.LogEvolution(NewEvolutionEntry("alice", "trigger2", "change2", "", "", 0.8))

	history, err := e.EvolutionHistory("alice", 10)
	if err != nil {
		t.Fatalf("EvolutionHistory: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(history))
	}
}

func TestInteractionStats(t *testing.T) {
	e := newTestEngine(t)

	e.tracker.LogOutcome("alice", "a", 4, "coding", "direct")
	e.tracker.LogOutcome("alice", "b", 2, "debugging", "casual")

	stats, err := e.InteractionStats("alice")
	if err != nil {
		t.Fatalf("InteractionStats: %v", err)
	}
	if stats["total_interactions"] != 2 {
		t.Errorf("total_interactions = %v, want 2", stats["total_interactions"])
	}
	if stats["average_score"] != "3.00" {
		t.Errorf("average_score = %v, want '3.00'", stats["average_score"])
	}
}

func TestDiscoverSkills(t *testing.T) {
	e := newTestEngine(t)

	// Add interactions with a skill not in persona's set
	e.tracker.LogOutcome("alice", "a", 5, "architecture", "")
	e.tracker.LogOutcome("alice", "b", 5, "architecture", "")

	discovered, err := e.DiscoverSkills("alice")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(discovered) != 1 {
		t.Fatalf("expected 1 discovered skill, got %d", len(discovered))
	}
	if discovered[0] != "architecture" {
		t.Errorf("discovered = %q, want %q", discovered[0], "architecture")
	}
}

func TestDiscoverSkillsPersonaNotFound(t *testing.T) {
	e := newTestEngine(t)

	_, err := e.DiscoverSkills("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent persona")
	}
}

func TestDiscoverSkillsLowScore(t *testing.T) {
	e := newTestEngine(t)

	// Score below 4.0 threshold
	e.tracker.LogOutcome("alice", "a", 2, "weak-skill", "")

	discovered, err := e.DiscoverSkills("alice")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(discovered) != 0 {
		t.Errorf("expected 0 discovered skills (low score), got %d", len(discovered))
	}
}

func TestGetEvolvedRules(t *testing.T) {
	e := newTestEngine(t)

	// No data yet
	rules, err := e.GetEvolvedRules("alice")
	if err != nil {
		t.Fatalf("GetEvolvedRules: %v", err)
	}
	if rules != "No interaction data yet for rule evolution." {
		t.Errorf("unexpected message: %q", rules)
	}
}

func TestGetEvolvedRulesWithPatterns(t *testing.T) {
	e := newTestEngine(t)

	// Add 4 high-scoring interactions with same tone
	for i := 0; i < 4; i++ {
		e.tracker.LogOutcome("alice", "good", 5, "coding", "formal")
	}

	// Add 3 low-scoring interactions with bad tone
	for i := 0; i < 3; i++ {
		e.tracker.LogOutcome("alice", "bad", 1, "", "aggressive")
	}

	rules, err := e.GetEvolvedRules("alice")
	if err != nil {
		t.Fatalf("GetEvolvedRules: %v", err)
	}
	if rules == "Not enough interaction data for rule evolution. Keep interacting." {
		t.Error("expected actual rules, got placeholder message")
	}
	// Should contain tone recommendation
	if !containsSubstring(rules, "formal") {
		t.Errorf("expected rules to mention 'formal' tone, got: %q", rules)
	}
}

func TestGetEvolvedRulesPersonaNotFound(t *testing.T) {
	e := newTestEngine(t)

	_, err := e.GetEvolvedRules("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent persona")
	}
}

func TestAccessors(t *testing.T) {
	e := newTestEngine(t)

	if e.Tracker() == nil {
		t.Error("expected non-nil Tracker")
	}
	if e.Consolidator() == nil {
		t.Error("expected non-nil Consolidator")
	}
	if e.Adapter() == nil {
		t.Error("expected non-nil Adapter")
	}
}

func TestNewInteractionOutcomeFromEngine(t *testing.T) {
	// Verify the engine uses NewInteractionOutcome correctly
	e := newTestEngine(t)
	err := e.LogInteraction("alice", "test interaction", 4, "coding", "direct")
	if err != nil {
		t.Fatalf("LogInteraction: %v", err)
	}

	outcomes, _ := e.tracker.RecentOutcomes("alice", 10)
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome, got %d", len(outcomes))
	}
	if outcomes[0].Summary != "test interaction" {
		t.Errorf("summary = %q, want %q", outcomes[0].Summary, "test interaction")
	}
}

func TestEvolveMultiplePersonas(t *testing.T) {
	d := newTestDB(t)
	m := newTestPersonaManager(t)
	// Create second persona
	m.Create("bob", "other assistant", "formal", "another persona", "", []string{"testing"})

	e := NewEngine(m, d, (*embedding.Embedder)(nil))

	// Evolve for alice
	result1, err := e.Evolve("alice")
	if err != nil {
		t.Fatalf("Evolve alice: %v", err)
	}
	if result1.Persona != "alice" {
		t.Errorf("persona = %q, want %q", result1.Persona, "alice")
	}

	// Evolve for bob
	result2, err := e.Evolve("bob")
	if err != nil {
		t.Fatalf("Evolve bob: %v", err)
	}
	if result2.Persona != "bob" {
		t.Errorf("persona = %q, want %q", result2.Persona, "bob")
	}

	// Verify separate histories — each Evolve logs consolidation + evolution_cycle
	history1, _ := e.EvolutionHistory("alice", 10)
	history2, _ := e.EvolutionHistory("bob", 10)
	if len(history1) < 2 || len(history2) < 2 {
		t.Errorf("expected at least 2 entries per persona, got alice=%d bob=%d", len(history1), len(history2))
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
