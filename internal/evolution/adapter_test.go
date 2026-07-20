package evolution

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coff33ninja/ai-memory/internal/persona"
)

func newTestPersonaManager(t *testing.T) *persona.Manager {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "evolution-persona-test-"+t.Name())
	t.Cleanup(func() { os.RemoveAll(dir) })

	m, err := persona.NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	_, err = m.Create("alice", "test assistant", "casual", "a test persona", "", []string{"coding", "debugging"})
	if err != nil {
		t.Fatalf("Create persona: %v", err)
	}
	return m
}

func newTestAdapter(t *testing.T) (*Adapter, *persona.Manager) {
	t.Helper()
	tr := newTestTracker(t)
	m := newTestPersonaManager(t)
	return NewAdapter(tr, m), m
}

func TestAdaptTraitsPersonaNotFound(t *testing.T) {
	a, _ := newTestAdapter(t)

	_, err := a.AdaptTraits("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent persona")
	}
}

func TestAdaptTraitsNoData(t *testing.T) {
	a, _ := newTestAdapter(t)

	result, err := a.AdaptTraits("alice")
	if err != nil {
		t.Fatalf("AdaptTraits: %v", err)
	}
	if result.ToneChanged {
		t.Error("expected no tone change with no data")
	}
	if len(result.SkillsAdded) != 0 {
		t.Error("expected no skills added with no data")
	}
}

func TestAdaptTraitsToneChange(t *testing.T) {
	a, m := newTestAdapter(t)

	// Add interactions where "formal" tone consistently scores high
	tr := a.tracker
	tr.LogOutcome("alice", "a", 5, "", "formal")
	tr.LogOutcome("alice", "b", 5, "", "formal")
	tr.LogOutcome("alice", "c", 5, "", "formal")
	tr.LogOutcome("alice", "d", 2, "", "casual")

	result, err := a.AdaptTraits("alice")
	if err != nil {
		t.Fatalf("AdaptTraits: %v", err)
	}
	if !result.ToneChanged {
		t.Error("expected tone change")
	}
	if result.ToneBefore != "casual" {
		t.Errorf("tone_before = %q, want %q", result.ToneBefore, "casual")
	}
	if result.ToneAfter != "formal" {
		t.Errorf("tone_after = %q, want %q", result.ToneAfter, "formal")
	}

	// Verify persona was updated
	p := m.Get("alice")
	if p.Tone != "formal" {
		t.Errorf("persona tone = %q, want %q", p.Tone, "formal")
	}
}

func TestAdaptTraitsToneNotChangedIfSameAsCurrent(t *testing.T) {
	a, m := newTestAdapter(t)

	// Current tone is "casual", but "formal" scores higher
	// However, if best tone is already the current tone, no change
	// The adapter checks: bestTone != p.Tone
	// So let's make "casual" score high and "formal" score low
	tr := a.tracker
	tr.LogOutcome("alice", "a", 5, "", "casual")
	tr.LogOutcome("alice", "b", 5, "", "casual")
	tr.LogOutcome("alice", "c", 5, "", "casual")

	// Need at least 2 different tones for the len(perf) > 1 check
	tr.LogOutcome("alice", "d", 1, "", "formal")

	result, err := a.AdaptTraits("alice")
	if err != nil {
		t.Fatalf("AdaptTraits: %v", err)
	}
	if result.ToneChanged {
		t.Error("expected no tone change since best tone matches current")
	}

	p := m.Get("alice")
	if p.Tone != "casual" {
		t.Errorf("persona tone = %q, want 'casual' (unchanged)", p.Tone)
	}
}

func TestAdaptTraitsSkillsAdded(t *testing.T) {
	a, m := newTestAdapter(t)

	tr := a.tracker
	// "analysis" is not in persona's skills but scores high
	tr.LogOutcome("alice", "a", 5, "analysis", "")
	tr.LogOutcome("alice", "b", 5, "analysis", "")

	result, err := a.AdaptTraits("alice")
	if err != nil {
		t.Fatalf("AdaptTraits: %v", err)
	}
	if len(result.SkillsAdded) != 1 {
		t.Fatalf("expected 1 skill added, got %d", len(result.SkillsAdded))
	}
	if result.SkillsAdded[0] != "analysis" {
		t.Errorf("skill added = %q, want %q", result.SkillsAdded[0], "analysis")
	}

	// Verify persona skills were updated
	p := m.Get("alice")
	found := false
	for _, s := range p.Skills {
		if s == "analysis" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'analysis' in persona skills")
	}
}

func TestAdaptTraitsSkillAlreadyExists(t *testing.T) {
	a, m := newTestAdapter(t)

	tr := a.tracker
	// "debugging" is already in persona's skills
	tr.LogOutcome("alice", "a", 5, "debugging", "")

	result, err := a.AdaptTraits("alice")
	if err != nil {
		t.Fatalf("AdaptTraits: %v", err)
	}
	if len(result.SkillsAdded) != 0 {
		t.Errorf("expected no skills added (already present), got %d", len(result.SkillsAdded))
	}

	p := m.Get("alice")
	count := 0
	for _, s := range p.Skills {
		if s == "debugging" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 'debugging' skill, found %d", count)
	}
}

func TestAdaptTraitsSkillNotAddedIfLowScore(t *testing.T) {
	a, m := newTestAdapter(t)

	tr := a.tracker
	// "weak-skill" scores below 4.0 threshold
	tr.LogOutcome("alice", "a", 2, "weak-skill", "")

	result, err := a.AdaptTraits("alice")
	if err != nil {
		t.Fatalf("AdaptTraits: %v", err)
	}
	if len(result.SkillsAdded) != 0 {
		t.Errorf("expected no skills added (low score), got %d", len(result.SkillsAdded))
	}

	p := m.Get("alice")
	for _, s := range p.Skills {
		if s == "weak-skill" {
			t.Error("expected 'weak-skill' NOT in persona skills")
		}
	}
}
