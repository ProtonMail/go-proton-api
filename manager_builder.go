package proton

import (
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/ProtonMail/gluon/async"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultHostURL is the default host of the API.
	DefaultHostURL = "https://mail.proton.me/api"

	// DefaultAppVersion is the default app version used to communicate with the API.
	// This must be changed (using the WithAppVersion option) for production use.
	DefaultAppVersion = "go-proton-api"
)

type managerBuilder struct {
	hostURL       string
	appVersion    string
	transport     http.RoundTripper
	verifyProofs  bool
	cookieJar     http.CookieJar
	retryCount    int
	logger        resty.Logger
	debug         bool
	panicHandler  async.PanicHandler
	errorsToRetry []int
}

func newManagerBuilder() *managerBuilder {
	return &managerBuilder{
		hostURL:       DefaultHostURL,
		appVersion:    DefaultAppVersion,
		transport:     http.DefaultTransport,
		verifyProofs:  true,
		cookieJar:     nil,
		retryCount:    3,
		logger:        nil,
		debug:         false,
		panicHandler:  async.NoopPanicHandler{},
		errorsToRetry: []int{http.StatusTooManyRequests, http.StatusServiceUnavailable},
	}
}

func (builder *managerBuilder) build() *Manager {
	m := &Manager{
		rc: resty.New(),

		errHandlers: make(map[Code][]Handler),

		verifyProofs: builder.verifyProofs,

		panicHandler: builder.panicHandler,
	}

	// Set the API host.
	m.rc.SetBaseURL(builder.hostURL)

	// Set the transport.
	m.rc.SetTransport(builder.transport)

	// Set the cookie jar.
	m.rc.SetCookieJar(builder.cookieJar)

	// Set the logger.
	if builder.logger != nil {
		m.rc.SetLogger(builder.logger)
	}

	// Set the debug flag.
	m.rc.SetDebug(builder.debug)

	// Set app version in header.
	m.rc.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		req.SetHeader("x-pm-appversion", builder.appVersion)
		return nil
	})

	// Set middleware.
	m.rc.OnAfterResponse(catchAPIError)
	m.rc.OnAfterResponse(updateTime)
	m.rc.OnAfterResponse(m.checkConnUp)
	m.rc.OnError(m.checkConnDown)
	m.rc.OnError(m.handleError)

	// Configure retry mechanism.
	m.rc.SetRetryCount(builder.retryCount)
	m.rc.SetRetryMaxWaitTime(time.Minute)
	m.rc.AddRetryCondition(builder.catchErrorsToRetry)
	m.rc.AddRetryCondition(catchDialError)
	m.rc.AddRetryCondition(catchDropError)
	m.rc.SetRetryAfter(builder.catchRetryAfter)

	// Set the data type of API errors.
	m.rc.SetError(&APIError{})

	return m
}

func (builder *managerBuilder) catchErrorsToRetry(res *resty.Response, _ error) bool {
	for _, err := range builder.errorsToRetry {
		if err == res.StatusCode() {
			return true
		}
	}
	return false
}

// nolint:gosec
func (builder *managerBuilder) catchRetryAfter(_ *resty.Client, res *resty.Response) (time.Duration, error) {
	// 0 and no error means default behaviour which is exponential backoff with jitter.
	if !builder.catchErrorsToRetry(res, errors.New("")) {
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
