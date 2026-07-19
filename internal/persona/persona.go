package persona

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Persona struct {
	Name        string            `json:"name"`
	Identity    string            `json:"identity,omitempty"`
	Tone        string            `json:"tone,omitempty"`
	Description string            `json:"description,omitempty"`
	Greeting    string            `json:"greeting,omitempty"`
	Skills      []string          `json:"skills,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

type Registry struct {
	mu       sync.RWMutex
	dir      string
	file     string
	Personas []Persona `json:"personas"`
	Active   string    `json:"active"`
}

type Manager struct {
	mu       sync.RWMutex
	dir      string
	registry *Registry
}

func NewManager(baseDir string) (*Manager, error) {
	regPath := filepath.Join(baseDir, "personas.json")
	reg := &Registry{
		dir:  baseDir,
		file: regPath,
	}

	data, err := os.ReadFile(regPath)
	if err == nil {
		json.Unmarshal(data, reg)
	}

	return &Manager{dir: baseDir, registry: reg}, nil
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.registry.file, data, 0o644)
}

func (m *Manager) List() []Persona {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry.Personas
}

func (m *Manager) Active() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry.Active
}

func (m *Manager) Get(name string) *Persona {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.registry.Personas {
		if m.registry.Personas[i].Name == name {
			return &m.registry.Personas[i]
		}
	}
	return nil
}

func (m *Manager) Dir() string {
	return m.dir
}

func (m *Manager) PersonaDir(name string) string {
	return filepath.Join(m.dir, name)
}

func (m *Manager) SharedDir() string {
	return filepath.Join(m.dir, "shared")
}

func (m *Manager) Create(name, identity, tone, description, greeting string, skills []string) (*Persona, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.registry.Personas {
		if p.Name == name {
			return nil, fmt.Errorf("persona %q already exists", name)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	p := Persona{
		Name:        name,
		Identity:    identity,
		Tone:        tone,
		Description: description,
		Greeting:    greeting,
		Skills:      skills,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create persona directory
	pDir := filepath.Join(m.dir, name)
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		return nil, fmt.Errorf("create persona dir: %w", err)
	}

	// Create shared dir if not exists
	sDir := filepath.Join(m.dir, "shared")
	os.MkdirAll(sDir, 0o755)

	m.registry.Personas = append(m.registry.Personas, p)
	if m.registry.Active == "" {
		m.registry.Active = name
	}

	if err := m.save(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (m *Manager) Switch(name string) (*Persona, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.registry.Personas {
		if m.registry.Personas[i].Name == name {
			m.registry.Active = name
			if err := m.save(); err != nil {
				return nil, err
			}
			return &m.registry.Personas[i], nil
		}
	}
	return nil, fmt.Errorf("persona %q not found", name)
}

func (m *Manager) Update(name, identity, tone, description, greeting string, skills []string) (*Persona, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.registry.Personas {
		if m.registry.Personas[i].Name == name {
			if identity != "" {
				m.registry.Personas[i].Identity = identity
			}
			if tone != "" {
				m.registry.Personas[i].Tone = tone
			}
			if description != "" {
				m.registry.Personas[i].Description = description
			}
			if greeting != "" {
				m.registry.Personas[i].Greeting = greeting
			}
			if skills != nil {
				m.registry.Personas[i].Skills = skills
			}
			m.registry.Personas[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := m.save(); err != nil {
				return nil, err
			}
			return &m.registry.Personas[i], nil
		}
	}
	return nil, fmt.Errorf("persona %q not found", name)
}

func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i := range m.registry.Personas {
		if m.registry.Personas[i].Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("persona %q not found", name)
	}

	// Don't allow deleting the active persona if it's the only one
	if m.registry.Active == name && len(m.registry.Personas) == 1 {
		return fmt.Errorf("cannot delete the only persona")
	}

	// Remove from list
	m.registry.Personas = append(m.registry.Personas[:idx], m.registry.Personas[idx+1:]...)

	// If we deleted the active one, switch to the first remaining
	if m.registry.Active == name && len(m.registry.Personas) > 0 {
		m.registry.Active = m.registry.Personas[0].Name
	}

	// Delete persona directory (move to recycle bin equivalent - just rename)
	pDir := filepath.Join(m.dir, name)
	backupDir := filepath.Join(m.dir, "."+name+".deleted")
	os.Rename(pDir, backupDir)

	return m.save()
}

// FindPersonaByGreeting searches all personas for one whose greeting matches the input text.
// Returns the persona name if found, empty string otherwise.
func (m *Manager) FindPersonaByGreeting(text string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.registry.Personas {
		if p.Greeting == "" {
			continue
		}
		if strings.Contains(strings.ToLower(text), strings.ToLower(p.Greeting)) {
			return p.Name
		}
	}
	return ""
}
