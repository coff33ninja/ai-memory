package evolution

import "time"

type InteractionOutcome struct {
	ID           int64  `json:"id"`
	Persona      string `json:"persona"`
	Summary      string `json:"summary"`
	Score        int    `json:"outcome_score"`
	SkillsUsed   string `json:"skills_used"`
	ToneUsed     string `json:"tone_used"`
	CreatedAt    string `json:"created_at"`
}

type EvolutionEntry struct {
	ID          int64   `json:"id"`
	Persona     string  `json:"persona"`
	Trigger     string  `json:"trigger"`
	WhatChanged string  `json:"what_changed"`
	Before      string  `json:"before_val,omitempty"`
	After       string  `json:"after_val,omitempty"`
	Confidence  float64 `json:"confidence"`
	CreatedAt   string  `json:"created_at"`
}

type ToolGap struct {
	ID        int64  `json:"id"`
	Persona   string `json:"persona"`
	Need      string `json:"need"`
	Context   string `json:"context"`
	Suggested string `json:"suggested"`
	Resolved  int    `json:"resolved"`
	CreatedAt string `json:"created_at"`
}

type ToolKnowledge struct {
	ID        int64  `json:"id"`
	Persona   string `json:"persona"`
	ToolName  string `json:"tool_name"`
	HowToUse  string `json:"how_to_use"`
	WhatWorks string `json:"what_works"`
	WhatFails string `json:"what_fails"`
	Params    string `json:"params"`
	Examples  string `json:"examples"`
	UseCount  int    `json:"use_count"`
	LastUsed  string `json:"last_used,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ToolRecipe struct {
	ID           int64  `json:"id"`
	Persona      string `json:"persona"`
	ToolName     string `json:"tool_name"`
	RecipeName   string `json:"recipe_name"`
	Steps        string `json:"steps"`
	UseCase      string `json:"use_case"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	CreatedAt    string `json:"created_at"`
}

type ToolError struct {
	ID        int64  `json:"id"`
	Persona   string `json:"persona"`
	ToolName  string `json:"tool_name"`
	ErrorMsg  string `json:"error_msg"`
	Context   string `json:"context"`
	InputArgs string `json:"input_args"`
	Resolved  int    `json:"resolved"`
	Reported  int    `json:"reported"`
	CreatedAt string `json:"created_at"`
}

type MCPServer struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Source       string `json:"source"`
	HasReport    int    `json:"has_report"`
	HasScreenshot int   `json:"has_screenshot"`
	HasOCR       int    `json:"has_ocr"`
	HasChain     int    `json:"has_chain"`
	ToolCount    int    `json:"tool_count"`
	Creator      string `json:"creator"`
	RepoURL      string `json:"repo_url"`
	Description  string `json:"description"`
	LastSeen     string `json:"last_seen,omitempty"`
	CreatedAt    string `json:"created_at"`
}

func NewInteractionOutcome(persona, summary string, score int, skillsUsed, toneUsed string) *InteractionOutcome {
	return &InteractionOutcome{
		Persona:    persona,
		Summary:    summary,
		Score:      score,
		SkillsUsed: skillsUsed,
		ToneUsed:   toneUsed,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

func NewEvolutionEntry(persona, trigger, whatChanged, before, after string, confidence float64) *EvolutionEntry {
	return &EvolutionEntry{
		Persona:     persona,
		Trigger:     trigger,
		WhatChanged: whatChanged,
		Before:      before,
		After:       after,
		Confidence:  confidence,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}
