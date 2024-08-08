package proton

import (
	"context"
)

type FeatureFlagResponse struct {
	Code    int             `json:"Code"`
	Toggles []featureToggle `json:"toggles"`
}

type featureToggle struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (m *Manager) GetFeatures(ctx context.Context) (FeatureFlagResponse, error) {
	responseData := FeatureFlagResponse{}

	_, err := m.r(ctx).SetResult(&responseData).Get("/feature/v2/frontend")
	if err != nil {
		return FeatureFlagResponse{}, err
	}

	return responseData, nil
}
