package evolution

import (
	"fmt"

	"github.com/coff33ninja/ai-memory/internal/persona"
)

type Adapter struct {
	tracker *Tracker
	manager *persona.Manager
}

func NewAdapter(tracker *Tracker, manager *persona.Manager) *Adapter {
	return &Adapter{tracker: tracker, manager: manager}
}

func (a *Adapter) AdaptTraits(personaName string) (*TraitAdaptationResult, error) {
	p := a.manager.Get(personaName)
	if p == nil {
		return nil, fmt.Errorf("persona %q not found", personaName)
	}

	result := &TraitAdaptationResult{}

	// Analyze tone performance
	tonePerf, err := a.tracker.TonePerformance(personaName)
	if err == nil && len(tonePerf) > 1 {
		bestTone := ""
		bestScore := 0.0
		for tone, score := range tonePerf {
			if score > bestScore {
				bestTone = tone
				bestScore = score
			}
		}

		// If best tone is significantly better than current, adapt
		if bestTone != "" && bestTone != p.Tone && bestScore >= 4.0 {
			result.ToneBefore = p.Tone
			result.ToneAfter = bestTone
			result.ToneChanged = true
			result.Reason = fmt.Sprintf("Tone '%s' scored %.1f vs current '%s'", bestTone, bestScore, p.Tone)

			a.manager.Update(personaName, "", bestTone, "", "", nil)

			a.tracker.LogEvolution(NewEvolutionEntry(
				personaName,
				"trait_adaptation",
				"tone_changed",
				p.Tone,
				bestTone,
				bestScore/5.0,
			))
		}
	}

	// Analyze skill effectiveness and update skill set
	skillPerf, err := a.tracker.SkillPerformance(personaName)
	if err == nil && len(skillPerf) > 0 {
		currentSkills := make(map[string]bool)
		for _, s := range p.Skills {
			currentSkills[s] = true
		}

		// Add consistently high-performing skills not yet in set
		var skillsAdded []string
		for skill, score := range skillPerf {
			if !currentSkills[skill] && score >= 4.0 {
				skillsAdded = append(skillsAdded, skill)
			}
		}

		if len(skillsAdded) > 0 {
			newSkills := append(p.Skills, skillsAdded...)
			a.manager.Update(personaName, "", "", "", "", newSkills)
			result.SkillsAdded = skillsAdded
			result.DescriptionChanged = true
			result.DescriptionBefore = fmt.Sprintf("Skills: %v", p.Skills)
			result.DescriptionAfter = fmt.Sprintf("Skills: %v", newSkills)

			a.tracker.LogEvolution(NewEvolutionEntry(
				personaName,
				"trait_adaptation",
				"skills_added",
				fmt.Sprintf("%v", p.Skills),
				fmt.Sprintf("%v", newSkills),
				0.8,
			))
		}
	}

	return result, nil
}
