package proton

import (
	"context"
	"strings"
)

func (m *Manager) Quark(ctx context.Context, command string, args ...string) error {
	if _, err := m.r(ctx).SetQueryParam("strInput", strings.Join(args, " ")).Get("/internal/quark/" + command); err != nil {
		return err
	}

	return nil
}
