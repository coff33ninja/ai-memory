package evolution

import (
	"testing"
)

func TestNewInteractionOutcome(t *testing.T) {
	o := NewInteractionOutcome("test-persona", "fixed a bug", 4, "debugging", "direct")
	if o == nil {
		t.Fatal("expected non-nil outcome")
	}
	if o.Persona != "test-persona" {
		t.Errorf("persona = %q, want %q", o.Persona, "test-persona")
	}
	if o.Summary != "fixed a bug" {
		t.Errorf("summary = %q, want %q", o.Summary, "fixed a bug")
	}
	if o.Score != 4 {
		t.Errorf("score = %d, want 4", o.Score)
	}
	if o.SkillsUsed != "debugging" {
		t.Errorf("skills = %q, want %q", o.SkillsUsed, "debugging")
	}
	if o.ToneUsed != "direct" {
		t.Errorf("tone = %q, want %q", o.ToneUsed, "direct")
	}
	if o.CreatedAt == "" {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewEvolutionEntry(t *testing.T) {
	e := NewEvolutionEntry("test-persona", "interaction_threshold", "tone_changed", "casual", "formal", 0.9)
	if e == nil {
		t.Fatal("expected non-nil entry")
	}
	if e.Persona != "test-persona" {
		t.Errorf("persona = %q, want %q", e.Persona, "test-persona")
	}
	if e.Trigger != "interaction_threshold" {
		t.Errorf("trigger = %q, want %q", e.Trigger, "interaction_threshold")
	}
	if e.WhatChanged != "tone_changed" {
		t.Errorf("what_changed = %q, want %q", e.WhatChanged, "tone_changed")
	}
	if e.Before != "casual" {
		t.Errorf("before = %q, want %q", e.Before, "casual")
	}
	if e.After != "formal" {
		t.Errorf("after = %q, want %q", e.After, "formal")
	}
	if e.Confidence != 0.9 {
		t.Errorf("confidence = %f, want 0.9", e.Confidence)
	}
	if e.CreatedAt == "" {
		t.Error("expected CreatedAt to be set")
	}
}

func TestWordOverlap(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected float64
	}{
		{"identical", "hello world", "hello world", 1.0},
		{"no overlap", "hello", "world", 0.0},
		{"partial", "hello world foo", "hello world bar", 0.5},
		{"empty both", "", "", 0.0},
		{"empty one", "hello", "", 0.0},
		{"case insensitive", "Hello World", "hello world", 1.0},
		{"subset", "hello", "hello world", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordOverlap(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("wordOverlap(%q, %q) = %f, want %f", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
