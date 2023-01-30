package proton

import (
	"net/http"

	"github.com/go-resty/resty/v2"
)

// Option represents a type that can be used to configure the manager.
type Option interface {
	config(*managerBuilder)
}

func WithHostURL(hostURL string) Option {
	return &withHostURL{
		hostURL: hostURL,
	}
}

type withHostURL struct {
	hostURL string
}

func (opt withHostURL) config(builder *managerBuilder) {
	builder.hostURL = opt.hostURL
}

func WithAppVersion(appVersion string) Option {
	return &withAppVersion{
		appVersion: appVersion,
	}
}

type withAppVersion struct {
	appVersion string
}

func (opt withAppVersion) config(builder *managerBuilder) {
	builder.appVersion = opt.appVersion
}

func WithTransport(transport http.RoundTripper) Option {
	return &withTransport{
		transport: transport,
	}
}

type withTransport struct {
	transport http.RoundTripper
}

func (opt withTransport) config(builder *managerBuilder) {
	builder.transport = opt.transport
}

type withSkipVerifyProofs struct {
	skipVerifyProofs bool
}

func (opt withSkipVerifyProofs) config(builder *managerBuilder) {
	builder.verifyProofs = !opt.skipVerifyProofs
}

func WithSkipVerifyProofs() Option {
	return &withSkipVerifyProofs{
		skipVerifyProofs: true,
	}
}

func WithRetryCount(retryCount int) Option {
	return &withRetryCount{
		retryCount: retryCount,
	}
}

type withRetryCount struct {
	retryCount int
}

func (opt withRetryCount) config(builder *managerBuilder) {
	builder.retryCount = opt.retryCount
}

func WithCookieJar(jar http.CookieJar) Option {
	return &withCookieJar{
		jar: jar,
	}
}

type withCookieJar struct {
	jar http.CookieJar
}

func (opt withCookieJar) config(builder *managerBuilder) {
	builder.cookieJar = opt.jar
}

func WithLogger(logger resty.Logger) Option {
	return &withLogger{
		logger: logger,
	}
}

type withLogger struct {
	logger resty.Logger
}

func (opt withLogger) config(builder *managerBuilder) {
	builder.logger = opt.logger
}

func WithDebug(debug bool) Option {
	return &withDebug{
		debug: debug,
	}
}

type withDebug struct {
	debug bool
}

func (opt withDebug) config(builder *managerBuilder) {
	builder.debug = opt.debug
}
