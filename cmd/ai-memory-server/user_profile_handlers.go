package main

import (
	"fmt"
	"strings"

	"github.com/coff33ninja/ai-memory/internal/memory"
)

func handleStoreUserProfile(m *memory.Store, args map[string]interface{}) (interface{}, error) {
	field, _ := args["field"].(string)
	value, _ := args["value"].(string)
	source, _ := args["source"].(string)
	confidence := 0.5
	if c, ok := args["confidence"].(float64); ok && c > 0 {
		confidence = c
	}

	profile, err := m.SetUserProfile(field, value, source, confidence)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ok":    true,
		"field": profile.Field,
		"value": profile.Value,
		"source": profile.Source,
		"confidence": profile.Confidence,
	}, nil
}

func handleGetUserProfile(m *memory.Store, args map[string]interface{}) (interface{}, error) {
	field, _ := args["field"].(string)
	profile, err := m.GetUserProfile(field)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return map[string]interface{}{"found": false, "field": field}, nil
	}
	return map[string]interface{}{
		"found":      true,
		"id":         profile.ID,
		"field":      profile.Field,
		"value":      profile.Value,
		"source":     profile.Source,
		"confidence": profile.Confidence,
		"created_at": profile.CreatedAt,
		"updated_at": profile.UpdatedAt,
	}, nil
}

func handleListUserProfile(m *memory.Store, args map[string]interface{}) (interface{}, error) {
	profiles, err := m.ListUserProfile()
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return map[string]interface{}{"profiles": []interface{}{}, "count": 0}, nil
	}
	var result []map[string]interface{}
	for _, p := range profiles {
		result = append(result, map[string]interface{}{
			"field":      p.Field,
			"value":      p.Value,
			"source":     p.Source,
			"confidence": p.Confidence,
		})
	}
	return map[string]interface{}{"profiles": result, "count": len(result)}, nil
}

func handleDeleteUserProfile(m *memory.Store, args map[string]interface{}) (interface{}, error) {
	field, _ := args["field"].(string)
	if err := m.DeleteUserProfile(field); err != nil {
		return nil, err
	}
	return map[string]interface{}{"ok": true, "deleted": field}, nil
}

func handleUserProfileResource(m *memory.Store) (string, error) {
	profiles, err := m.ListUserProfile()
	if err != nil {
		return "", err
	}
	if len(profiles) == 0 {
		return "No user profile data stored yet. Use store_user_profile to build knowledge about the user from conversations.", nil
	}
	var sb strings.Builder
	sb.WriteString("# User Profile\n\n")
	for _, p := range profiles {
		sb.WriteString(fmt.Sprintf("**%s**: %s (source: %s, confidence: %.0f%%)\n", p.Field, p.Value, p.Source, p.Confidence*100))
	}
	return sb.String(), nil
}
