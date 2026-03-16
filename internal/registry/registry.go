package registry

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ProviderConfig holds vendor-level settings: API Key reference and base URL
type ProviderConfig struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	APIKeyEnv string `yaml:"api_key_env"`
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"-"` // Populated at load time from env
}

// ModelConfig holds per-model lifecycle information
type ModelConfig struct {
	ID         string `yaml:"id"`
	ProviderID string `yaml:"provider"`
	EOLDate    string `yaml:"eol_date"`
	IsActive   bool   `yaml:"-"`
}

// ScenarioConfig dictates the routing template for a named use case
type ScenarioConfig struct {
	Primary  string `yaml:"primary"`
	Backup   string `yaml:"backup"`
	Strategy string `yaml:"strategy"`
}

// Config represents the raw yaml structure
type Config struct {
	Providers []ProviderConfig          `yaml:"providers"`
	Scenarios map[string]ScenarioConfig `yaml:"scenarios"`
	Models    []ModelConfig             `yaml:"models"`
}

// ModelRegistry holds the active, parsed state
type ModelRegistry struct {
	scenarios map[string]ScenarioConfig
	models    map[string]*ModelConfig
	providers map[string]*ProviderConfig
}

// NewModelRegistry loads the YAML and initializes all models and providers
func NewModelRegistry(configPath string) (*ModelRegistry, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read models config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	reg := &ModelRegistry{
		scenarios: cfg.Scenarios,
		models:    make(map[string]*ModelConfig),
		providers: make(map[string]*ProviderConfig),
	}

	// Load providers and resolve API keys from environment
	for i, p := range cfg.Providers {
		apiKey := os.Getenv(p.APIKeyEnv)
		if apiKey == "" {
			log.Printf("[Registry] ⚠️  Provider '%s' (%s): %s not set, this provider will be unavailable.\n", p.Name, p.ID, p.APIKeyEnv)
		} else {
			log.Printf("[Registry] ✅  Provider '%s' (%s): API key loaded.\n", p.Name, p.ID)
		}
		cfg.Providers[i].APIKey = apiKey
		pCopy := cfg.Providers[i]
		reg.providers[p.ID] = &pCopy
	}

	// Load models and check lifecycle
	for i, m := range cfg.Models {
		eol, _ := time.Parse("2006-01-02", m.EOLDate)
		active := time.Now().Before(eol)
		cfg.Models[i].IsActive = active

		if !active {
			log.Printf("[Registry] ❌  Model '%s' passed its EOL date (%s). Marked Inactive.\n", m.ID, m.EOLDate)
		} else if time.Until(eol) < 30*24*time.Hour {
			log.Printf("[Registry] ⚠️  Warning: Model '%s' approaching EOL on %s.\n", m.ID, m.EOLDate)
		}

		mCopy := cfg.Models[i]
		reg.models[m.ID] = &mCopy
	}

	return reg, nil
}

func (r *ModelRegistry) GetScenario(name string) (*ScenarioConfig, error) {
	s, ok := r.scenarios[name]
	if !ok {
		return nil, fmt.Errorf("scenario '%s' not found in registry", name)
	}
	return &s, nil
}

func (r *ModelRegistry) GetModel(id string) (*ModelConfig, error) {
	m, ok := r.models[id]
	if !ok {
		return nil, fmt.Errorf("model '%s' not found in registry", id)
	}
	return m, nil
}

func (r *ModelRegistry) GetProvider(id string) (*ProviderConfig, error) {
	p, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found in registry", id)
	}
	return p, nil
}
