package proton

import (
	"context"
)

type FeatureFlagResult struct {
	Code    int             `json:"Code"`
	Toggles []FeatureToggle `json:"toggles"`
}

type FeatureToggle struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (m *Manager) GetFeatures(ctx context.Context) (FeatureFlagResult, error) {
	responseData := FeatureFlagResult{}

	_, err := m.r(ctx).SetResult(&responseData).Get("/feature/v2/frontend")
	if err != nil {
		return FeatureFlagResult{}, err
	}

	return responseData, nil
}
