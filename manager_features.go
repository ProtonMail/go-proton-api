package proton

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"
)

type FeatureFlagResult struct {
	Code    int             `json:"Code"`
	Toggles []FeatureToggle `json:"toggles"`
}

type FeatureToggle struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func getFeatureFlagEndpoint(stickyKey uuid.UUID) string {
	params := url.Values{}
	params.Set("bridgeStickyKey", stickyKey.String())
	path := fmt.Sprintf("/feature/v2/frontend?%s", params.Encode())
	return path
}

func (m *Manager) GetFeatures(ctx context.Context, stickyKey uuid.UUID) (FeatureFlagResult, error) {
	responseData := FeatureFlagResult{}

	_, err := m.r(ctx).SetResult(&responseData).Get(getFeatureFlagEndpoint(stickyKey))
	if err != nil {
		return FeatureFlagResult{}, err
	}

	return responseData, nil
}
