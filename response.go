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
	UsernameInvalid           Code = 6003 // Deprecated, but still used.
	PasswordWrong             Code = 8002
	HumanVerificationRequired Code = 9001
	PaidPlanRequired          Code = 10004
	AuthRefreshTokenInvalid   Code = 10013
)

// APIError represents an error returned by the API.
type APIError struct {
	// Status is the HTTP status code of the response that caused the error.
	Status int

	// Code is the error code returned by the API.
	Code Code

	// Message is the error message returned by the API.
	Message string `json:"Error"`
}

func (err APIError) Error() string {
	return err.Message
}

// NetError represents a network error. It is returned when the API is unreachable.
type NetError struct {
	// Cause is the underlying error that caused the network error.
	Cause error

	// Message is an additional message that describes the network error.
	Message string
}

func newNetError(err error, message string) *NetError {
	return &NetError{Cause: err, Message: message}
}

func (err *NetError) Error() string {
	return fmt.Sprintf("%s: %v", err.Message, err.Cause)
}

func (err *NetError) Unwrap() error {
	return err.Cause
}

func (err *NetError) Is(target error) bool {
	_, ok := target.(*NetError)
	return ok
}

func catchAPIError(_ *resty.Client, res *resty.Response) error {
	if !res.IsError() {
		return nil
	}

	var err error

	if apiErr, ok := res.Error().(*APIError); ok {
		apiErr.Status = res.StatusCode()
		err = apiErr
	} else {
		statusCode := res.StatusCode()
		statusText := res.Status()

		// Catch error that may slip through when APIError deserialization routine fails for whichever reason.
		if statusCode >= 400 {
			err = &APIError{
				Status:  statusCode,
				Code:    0,
				Message: statusText,
			}
		} else {
			err = fmt.Errorf("%v", res.Status())
		}
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
