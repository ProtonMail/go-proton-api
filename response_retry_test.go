package proton

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

func TestRetryRefreshFailed(t *testing.T) {
	refreshURL := "https://api.proton.me/auth/v4/refresh"
	otherURL := "https://api.proton.me/core/v4/users"

	tests := []struct {
		name string
		res  *resty.Response
		err  error
		want bool
	}{
		{name: "no error", res: refreshRes(refreshURL, http.StatusInternalServerError), err: nil, want: false},
		{name: "refresh 500", res: refreshRes(refreshURL, http.StatusInternalServerError), err: errors.New("boom"), want: false},
		{name: "refresh 400", res: refreshRes(refreshURL, http.StatusBadRequest), err: errors.New("deauth"), want: true},
		{name: "refresh 422", res: refreshRes(refreshURL, http.StatusUnprocessableEntity), err: errors.New("error"), want: true},
		{name: "refresh 200 hook error", res: refreshRes(refreshURL, http.StatusOK), err: errors.New("update time"), want: false},
		{name: "other 500", res: refreshRes(otherURL, http.StatusInternalServerError), err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, retryRefreshFailed(tt.res, tt.err))
		})
	}
}

func refreshRes(url string, status int) *resty.Response {
	return &resty.Response{
		Request:     &resty.Request{URL: url},
		RawResponse: &http.Response{StatusCode: status},
	}
}
