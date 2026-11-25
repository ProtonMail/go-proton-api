package proton

import (
	"context"
	"io"
)

func (m *Manager) Download(ctx context.Context, url string) ([]byte, error) {
	res, err := m.r(ctx).SetDoNotParseResponse(true).Get(url)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(res.RawBody())
	if err != nil {
		return nil, err
	}

	if err := res.RawBody().Close(); err != nil {
		return nil, err
	}

	return b, nil
}
