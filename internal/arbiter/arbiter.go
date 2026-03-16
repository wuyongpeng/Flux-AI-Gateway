package arbiter

import (
	"fmt"
	"log"

	"flux-ai-gateway/internal/provider"
	"flux-ai-gateway/internal/registry"
	"flux-ai-gateway/internal/scheduler"
)

// PolicyArbiter sits between the Proxy and the Registry.
// It resolves abstract scenario names into concrete BackendProviders.
type PolicyArbiter struct {
	reg *registry.ModelRegistry
}

func NewPolicyArbiter(reg *registry.ModelRegistry) *PolicyArbiter {
	return &PolicyArbiter{reg: reg}
}

// resolveBackend looks up model + provider and returns an instantiated BackendProvider
func (a *PolicyArbiter) resolveBackend(modelID string) (scheduler.BackendProvider, error) {
	model, err := a.reg.GetModel(modelID)
	if err != nil {
		return nil, err
	}
	if !model.IsActive {
		return nil, fmt.Errorf("model '%s' is inactive (EOL)", modelID)
	}

	prov, err := a.reg.GetProvider(model.ProviderID)
	if err != nil {
		return nil, err
	}

	backend, err := provider.NewProvider(model.ID, prov.ID, prov.APIKey, prov.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend for model '%s': %w", modelID, err)
	}

	return backend, nil
}

// GetBackends resolves a scenario name into its primary/backup BackendProviders + strategy.
func (a *PolicyArbiter) GetBackends(scenarioName string) ([]scheduler.BackendProvider, scheduler.BackendProvider, string, error) {
	scen, err := a.reg.GetScenario(scenarioName)
	if err != nil {
		return nil, nil, "", err
	}

	prim, primErr := a.resolveBackend(scen.Primary)
	if primErr != nil {
		log.Printf("[Arbiter] ⚠️  Primary model '%s' unavailable: %v. Trying backup directly.", scen.Primary, primErr)
	}

	back, backErr := a.resolveBackend(scen.Backup)
	if backErr != nil {
		log.Printf("[Arbiter] ⚠️  Backup model '%s' unavailable: %v.", scen.Backup, backErr)
	}

	// Both down → fatal
	if primErr != nil && backErr != nil {
		return nil, nil, "", fmt.Errorf("CRITICAL: both primary and backup unavailable for scenario '%s'", scenarioName)
	}

	// Primary down → run backup only
	if primErr != nil {
		return []scheduler.BackendProvider{back}, nil, "failover", nil
	}

	// Backup down → run primary only (no safety net)
	if backErr != nil {
		log.Printf("[Arbiter] Running '%s' without a backup safety net.", scen.Primary)
		return []scheduler.BackendProvider{prim}, nil, "failover", nil
	}

	if scen.Strategy == "hedged" || scen.Strategy == "failover" || scen.Strategy == "fallback" {
		return []scheduler.BackendProvider{prim, back}, back, scen.Strategy, nil
	}

	return []scheduler.BackendProvider{prim}, back, scen.Strategy, nil
}
