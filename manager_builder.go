package proton

import (
	"net/http"
	"time"

	"github.com/ProtonMail/gluon/queue"
	"github.com/go-resty/resty/v2"
)

const (
	// DefaultHostURL is the default host of the API.
	DefaultHostURL = "https://mail.proton.me/api"

	// DefaultAppVersion is the default app version used to communicate with the API.
	// This must be changed (using the WithAppVersion option) for production use.
	DefaultAppVersion = "go-proton-api"
)

type managerBuilder struct {
	hostURL      string
	appVersion   string
	transport    http.RoundTripper
	verifyProofs bool
	cookieJar    http.CookieJar
	retryCount   int
	logger       resty.Logger
	debug        bool
	panicHandler queue.PanicHandler
}

func newManagerBuilder() *managerBuilder {
	return &managerBuilder{
		hostURL:      DefaultHostURL,
		appVersion:   DefaultAppVersion,
		transport:    http.DefaultTransport,
		verifyProofs: true,
		cookieJar:    nil,
		retryCount:   3,
		logger:       nil,
		debug:        false,
		panicHandler: queue.NoopPanicHandler{},
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
	m.rc.AddRetryCondition(catchTooManyRequests)
	m.rc.AddRetryCondition(catchDialError)
	m.rc.AddRetryCondition(catchDropError)
	m.rc.SetRetryAfter(catchRetryAfter)

	// Set the data type of API errors.
	m.rc.SetError(&APIError{})

	return m
}
