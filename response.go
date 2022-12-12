package proton

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
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

// nolint:gosec
func catchRetryAfter(_ *resty.Client, res *resty.Response) (time.Duration, error) {
	// 0 and no error means default behaviour which is exponential backoff with jitter.
	if res.StatusCode() != http.StatusTooManyRequests {
		return 0, nil
	}

	// Parse the Retry-After header, or fallback to 10 seconds.
	after, err := strconv.Atoi(res.Header().Get("Retry-After"))
	if err != nil {
		after = 10
	}

	// Add some jitter to the delay.
	after += rand.Intn(10)

	logrus.WithFields(logrus.Fields{
		"pkg":    "go-proton-api",
		"status": res.StatusCode(),
		"url":    res.Request.URL,
		"method": res.Request.Method,
		"after":  after,
	}).Warn("Too many requests, retrying after delay")

	return time.Duration(after) * time.Second, nil
}

func catchTooManyRequests(res *resty.Response, _ error) bool {
	return res.StatusCode() == http.StatusTooManyRequests
}

func catchDialError(res *resty.Response, err error) bool {
	return res.RawResponse == nil
}
