package evolution

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/db"
	"github.com/coff33ninja/ai-memory/internal/embedding"
	"github.com/coff33ninja/ai-memory/internal/persona"
)

const (
	InteractionThreshold = 10
	MemoryThreshold      = 50
)

type Engine struct {
	manager      *persona.Manager
	tracker      *Tracker
	consolidator *Consolidator
	adapter      *Adapter
	emb          *embedding.Embedder
}

func NewEngine(manager *persona.Manager, database *db.DB, emb *embedding.Embedder) *Engine {
	tracker := NewTracker(database)
	consolidator := NewConsolidator(tracker, emb)
	adapter := NewAdapter(tracker, manager)

	return &Engine{
		manager:      manager,
		tracker:      tracker,
		consolidator: consolidator,
		adapter:      adapter,
		emb:          emb,
	}
}

func (e *Engine) Tracker() *Tracker       { return e.tracker }
func (e *Engine) Consolidator() *Consolidator { return e.consolidator }
func (e *Engine) Adapter() *Adapter       { return e.adapter }

func (e *Engine) ShouldEvolve(personaName string) (bool, string) {
	count, err := e.tracker.InteractionCount(personaName)
	if err != nil {
		return false, ""
	}
	if count > 0 && count%InteractionThreshold == 0 {
		return true, fmt.Sprintf("interaction_threshold (%d interactions)", count)
	}

	memCount, err := e.tracker.MemoryCount(personaName, "")
	if err != nil {
		return false, ""
	}
	if memCount > 0 && memCount%MemoryThreshold == 0 {
		return true, fmt.Sprintf("memory_threshold (%d memories)", memCount)
	}

	return false, ""
}

func (e *Engine) Evolve(personaName string) (*EvolutionResult, error) {
	result := &EvolutionResult{Persona: personaName}

	// 1. Consolidate memories
	consolResult, err := e.consolidator.Consolidate(personaName)
	if err == nil {
		result.MemoryConsolidated = consolResult
	}

	// 2. Adapt traits
	traitResult, err := e.adapter.AdaptTraits(personaName)
	if err == nil {
		result.TraitsAdapted = traitResult
	}

	// 3. Log the full evolution
	e.tracker.LogEvolution(NewEvolutionEntry(
		personaName,
		"evolution_cycle",
		"full_evolution",
		"",
		fmt.Sprintf("consolidated=%+v adapted=%+v", consolResult, traitResult),
		1.0,
	))

	return result, nil
}

func (e *Engine) LogInteraction(personaName, summary string, score int, skillsUsed, toneUsed string) error {
	_, err := e.tracker.LogOutcome(personaName, summary, score, skillsUsed, toneUsed)
	if err != nil {
		return err
	}

	// Check if evolution should trigger
	if ok, trigger := e.ShouldEvolve(personaName); ok {
		fmt.Printf("evolution triggered: %s\n", trigger)
		_, err := e.Evolve(personaName)
		return err
	}
	return nil
}

type EvolutionResult struct {
	Persona            string                `json:"persona"`
	MemoryConsolidated *ConsolidationResult  `json:"memory_consolidated,omitempty"`
	TraitsAdapted      *TraitAdaptationResult `json:"traits_adapted,omitempty"`
}

type TraitAdaptationResult struct {
	ToneChanged    bool   `json:"tone_changed"`
	ToneBefore     string `json:"tone_before,omitempty"`
	ToneAfter      string `json:"tone_after,omitempty"`
	DescriptionChanged bool `json:"description_changed"`
	DescriptionBefore  string `json:"description_before,omitempty"`
	DescriptionAfter   string `json:"description_after,omitempty"`
	SkillsAdded    []string `json:"skills_added,omitempty"`
	SkillsRemoved  []string `json:"skills_removed,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

func (e *Engine) EvolutionHistory(personaName string, limit int) ([]EvolutionEntry, error) {
	return e.tracker.EvolutionHistory(personaName, limit)
}

func (e *Engine) InteractionStats(personaName string) (map[string]interface{}, error) {
	avgScore, err := e.tracker.AverageScore(personaName, 20)
	if err != nil {
		return nil, err
	}

	tonePerf, _ := e.tracker.TonePerformance(personaName)
	skillPerf, _ := e.tracker.SkillPerformance(personaName)
	count, _ := e.tracker.InteractionCount(personaName)

	return map[string]interface{}{
		"total_interactions": count,
		"average_score":      fmt.Sprintf("%.2f", avgScore),
		"tone_performance":   tonePerf,
		"skill_performance":  skillPerf,
	}, nil
}

func (e *Engine) DiscoverSkills(personaName string) ([]string, error) {
	p := e.manager.Get(personaName)
	if p == nil {
		return nil, fmt.Errorf("persona %q not found", personaName)
	}

	skillPerf, err := e.tracker.SkillPerformance(personaName)
	if err != nil {
		return nil, err
	}

	// Find skills not in persona's set that appear in high-scoring interactions
	var discovered []string
	currentSkills := make(map[string]bool)
	for _, s := range p.Skills {
		currentSkills[s] = true
	}

	for skill, score := range skillPerf {
		if !currentSkills[skill] && score >= 4.0 {
			discovered = append(discovered, skill)
		}
	}

	return discovered, nil
}

func (e *Engine) GetEvolvedRules(personaName string) (string, error) {
	p := e.manager.Get(personaName)
	if p == nil {
		return "", fmt.Errorf("persona %q not found", personaName)
	}

	outcomes, err := e.tracker.RecentOutcomes(personaName, 30)
	if err != nil || len(outcomes) == 0 {
		return "No interaction data yet for rule evolution.", nil
	}

	var rules []string

	// Analyze high-scoring interactions for patterns
	highScore := make([]InteractionOutcome, 0)
	lowScore := make([]InteractionOutcome, 0)
	for _, o := range outcomes {
		if o.Score >= 4 {
			highScore = append(highScore, o)
		} else if o.Score <= 2 {
			lowScore = append(lowScore, o)
		}
	}

	if len(highScore) > 3 {
		// Find common tones in high-scoring interactions
		toneCounts := make(map[string]int)
		for _, o := range highScore {
			if o.ToneUsed != "" {
				toneCounts[o.ToneUsed]++
			}
		}
		bestTone := ""
		bestCount := 0
		for tone, count := range toneCounts {
			if count > bestCount {
				bestTone = tone
				bestCount = count
			}
		}
		if bestTone != "" && bestCount >= 3 {
			rules = append(rules, fmt.Sprintf("Use tone '%s' (worked %d/%d times)", bestTone, bestCount, len(highScore)))
		}

		// Find common skill combos
		skillCounts := make(map[string]int)
		for _, o := range highScore {
			for _, s := range strings.Split(o.SkillsUsed, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					skillCounts[s]++
				}
			}
		}
		for skill, count := range skillCounts {
			if count >= 3 {
				rules = append(rules, fmt.Sprintf("Skill '%s' consistently effective (used in %d high-score interactions)", skill, count))
			}
		}
	}

	if len(lowScore) > 2 {
		// Find what NOT to do
		badTones := make(map[string]int)
		for _, o := range lowScore {
			if o.ToneUsed != "" {
				badTones[o.ToneUsed]++
			}
		}
		for tone, count := range badTones {
			if count >= 2 {
				rules = append(rules, fmt.Sprintf("Avoid tone '%s' (failed %d times)", tone, count))
			}
		}
	}

	if len(rules) == 0 {
		return "Not enough interaction data for rule evolution. Keep interacting.", nil
	}

	return fmt.Sprintf("# Evolved Rules for %s\n\n%s", personaName, strings.Join(rules, "\n")), nil
}
