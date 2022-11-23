package server

import (
	"io"
	"net/http/httptest"
	"os"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/ProtonMail/go-proton-api/server/backend"
	"github.com/gin-gonic/gin"
)

type serverBuilder struct {
	withTLS bool
	logger  io.Writer
	origin  string
	cacher  AuthCacher
}

func newServerBuilder() *serverBuilder {
	var logger io.Writer

	if os.Getenv("GO_PROTON_API_SERVER_LOGGER_ENABLED") != "" {
		logger = gin.DefaultWriter
	} else {
		logger = io.Discard
	}

	return &serverBuilder{
		withTLS: true,
		logger:  logger,
		origin:  proton.DefaultHostURL,
	}
}

func (builder *serverBuilder) build() *Server {
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		r: gin.New(),
		b: backend.New(time.Hour),

		proxyOrigin: builder.origin,
		authCacher:  builder.cacher,
	}

	if builder.withTLS {
		s.s = httptest.NewTLSServer(s.r)
	} else {
		s.s = httptest.NewServer(s.r)
	}

	s.r.Use(
		gin.LoggerWithConfig(gin.LoggerConfig{Output: builder.logger}),
		gin.Recovery(),
		s.logCalls(),
		s.handleOffline(),
	)

	initRouter(s)

	return s
}

// Option represents a type that can be used to configure the server.
type Option interface {
	config(*serverBuilder)
}

// WithTLS controls whether the server should serve over TLS.
func WithTLS(tls bool) Option {
	return &withTLS{
		withTLS: tls,
	}
}

type withTLS struct {
	withTLS bool
}

func (opt withTLS) config(builder *serverBuilder) {
	builder.withTLS = opt.withTLS
}

// WithLogger controls where Gin logs to.
func WithLogger(logger io.Writer) Option {
	return &withLogger{
		logger: logger,
	}
}

type withLogger struct {
	logger io.Writer
}

func (opt withLogger) config(builder *serverBuilder) {
	builder.logger = opt.logger
}

func WithProxyOrigin(origin string) Option {
	return &withProxyOrigin{
		origin: origin,
	}
}

type withProxyOrigin struct {
	origin string
}

func (opt withProxyOrigin) config(builder *serverBuilder) {
	builder.origin = opt.origin
}

func WithAuthCacher(cacher AuthCacher) Option {
	return &withAuthCache{
		cacher: cacher,
	}
}

type withAuthCache struct {
	cacher AuthCacher
}

func (opt withAuthCache) config(builder *serverBuilder) {
	builder.cacher = opt.cacher
}
