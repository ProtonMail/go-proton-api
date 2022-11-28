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

func catchRetryAfter(_ *resty.Client, res *resty.Response) (time.Duration, error) {
	if res.StatusCode() == http.StatusTooManyRequests {
		if after := res.Header().Get("Retry-After"); after != "" {
			l := logrus.WithFields(logrus.Fields{
				"pkg":        "go-proton-api",
				"statusCode": res.StatusCode(),
				"url":        res.Request.URL,
				"verb":       res.Request.Method,
			})

			seconds, err := strconv.Atoi(after)
			if err != nil {
				l.WithField("after", after).WithError(err).Warning(
					"Cannot convert Retry-After to a number, continue with default 10 second cooldown.",
				)
				seconds = 10
			}

			// To avoid spikes when all clients retry at the same time, we add some random wait.
			seconds += rand.Intn(10) //nolint:gosec // It is OK to use weak random number generator here.
			l = l.WithField("delay", seconds)

			// Maximum retry time in client is is one minute. But
			// here wait times can be longer e.g. high API load
			l.Warn("Delay the retry after http response")
			return time.Duration(seconds) * time.Second, nil
		}
	}

	// 0 and no error means default behaviour which is exponential backoff with jitter.
	return 0, nil
}

func catchTooManyRequests(res *resty.Response, _ error) bool {
	return res.StatusCode() == http.StatusTooManyRequests
}

func catchDialError(res *resty.Response, err error) bool {
	return res.RawResponse == nil
}
