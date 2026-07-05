package infra

import (
	"embed"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

//go:embed rbac_model.conf rbac_policy.csv
var casbinFiles embed.FS

// stringAdapter is a simple adapter that loads policy from a string
type stringAdapter struct {
	policy string
}

func (a *stringAdapter) LoadPolicy(model model.Model) error {
	lines := strings.Split(a.policy, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := persist.LoadPolicyLine(line, model); err != nil {
			return err
		}
	}
	return nil
}

func (a *stringAdapter) SavePolicy(model model.Model) error {
	return nil // Read-only for embedded policies
}

func (a *stringAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	return nil // Read-only
}

func (a *stringAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return nil // Read-only
}

func (a *stringAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return nil // Read-only
}

// NewCasbinEnforcer creates a Casbin enforcer from embedded config files
func NewCasbinEnforcer() (*casbin.Enforcer, error) {
	modelContent, err := casbinFiles.ReadFile("rbac_model.conf")
	if err != nil {
		return nil, err
	}

	policyContent, err := casbinFiles.ReadFile("rbac_policy.csv")
	if err != nil {
		return nil, err
	}

	m, err := model.NewModelFromString(string(modelContent))
	if err != nil {
		return nil, err
	}

	adapter := &stringAdapter{policy: string(policyContent)}
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}

	return enforcer, nil
}
