package proton

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/go-resty/resty/v2"
)

type Code int

const (
	SuccessCode               Code = 1000
	MultiCode                 Code = 1001
	InvalidValue              Code = 2001
	AppVersionMissingCode     Code = 5001
	AppVersionBadCode         Code = 5003
	PasswordWrong             Code = 8002
	HumanVerificationRequired Code = 9001
	PaidPlanRequired          Code = 10004
	AuthRefreshTokenInvalid   Code = 10013
)

type Error struct {
	Code    Code
	Message string `json:"Error"`
}

func (err Error) Error() string {
	return err.Message
}

func catchAPIError(_ *resty.Client, res *resty.Response) error {
	if !res.IsError() {
		return nil
	}

	var err error

	if apiErr, ok := res.Error().(*Error); ok {
		err = apiErr
	} else {
		err = fmt.Errorf("%v", res.Status())
	}

	return fmt.Errorf("%v: %w", res.StatusCode(), err)
}

func updateTime(_ *resty.Client, res *resty.Response) error {
	date, err := time.Parse(time.RFC1123, res.Header().Get("Date"))
	if err != nil {
		return err
	}

	crypto.UpdateTime(date.Unix())

	return nil
}

func catchRetryAfter(_ *resty.Client, res *resty.Response) (time.Duration, error) {
	if res.StatusCode() == http.StatusTooManyRequests {
		if after := res.Header().Get("Retry-After"); after != "" {
			seconds, err := strconv.Atoi(after)
			if err != nil {
				return 0, err
			}

			return time.Duration(seconds) * time.Second, nil
		}
	}

	return 0, nil
}

func catchTooManyRequests(res *resty.Response, _ error) bool {
	return res.StatusCode() == http.StatusTooManyRequests
}

func catchDialError(res *resty.Response, err error) bool {
	return res.RawResponse == nil
}
