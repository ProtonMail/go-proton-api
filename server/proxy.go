package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/henrybear327/go-proton-api"
)

func (s *proxyServer) newProxy(path string) http.HandlerFunc {
	origin, err := url.Parse(s.origin)
	if err != nil {
		panic(err)
	}

	return (&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = origin.Scheme
			req.URL.Host = origin.Host
			req.URL.Path = origin.Path + strings.TrimPrefix(path, s.base)
			req.Host = origin.Host
		},

		ModifyResponse: s.targetResponseModifier,
		ErrorHandler:   s.targetErrorHandler,

		Transport: s.transport,
	}).ServeHTTP
}

func (s *proxyServer) targetResponseModifier(r *http.Response) error {
	if s.debug {
		fmt.Printf("PROXY RESP %d: %s %s\n", r.StatusCode, r.Request.Method, r.Request.RequestURI)
	}
	if r.StatusCode == http.StatusInternalServerError || r.StatusCode == http.StatusBadGateway {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			body = []byte(fmt.Sprintf(`{"read_error": "%v"}`, err))
		}

		fmt.Printf("PROXY RESP %d CHANGING TO 429: %s %s\n=====BODY=====\n%s\n", r.StatusCode, r.Request.Method, r.Request.RequestURI, string(body))
		r.StatusCode = http.StatusTooManyRequests
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	return nil
}

func (s *proxyServer) targetErrorHandler(rw http.ResponseWriter, r *http.Request, targetError error) {
	if s.debug {
		fmt.Printf("PROXY ERROR: %s %s: %v\n", r.Method, r.RequestURI, targetError)
	}

	rw.WriteHeader(http.StatusTooManyRequests)
}

func (s *Server) handleProxy(base string) gin.HandlerFunc {
	return func(c *gin.Context) {
		proxy := newProxyServer(s.proxyOrigin, base, s.proxyTransport)

		proxy.handle("/", s.handleProxyAll)

		if s.authCacher != nil {
			proxy.handle("/auth/v4", s.handleProxyAuth)
			proxy.handle("/auth/v4/info", s.handleProxyAuthInfo)
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (s *Server) handleProxyAll(proxier func(string) HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := proxier(r.URL.Path)(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
}

func (s *Server) handleProxyAuth(proxier func(string) HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleProxyAuthPost(w, r, proxier(r.URL.Path))

		case http.MethodDelete:
			s.handleProxyAuthDelete(w, r, proxier(r.URL.Path))
		}
	}
}

func (s *Server) handleProxyAuthPost(w http.ResponseWriter, r *http.Request, proxier HandlerFunc) {
	req, err := readFromBody[proton.AuthReq](r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if info, ok := s.authCacher.GetAuth(req.Username); ok {
		if err := writeBody(w, info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		b, err := proxier(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := readFrom[proton.Auth](b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.authCacher.SetAuth(req.Username, res)
	}
}

func (s *Server) handleProxyAuthDelete(w http.ResponseWriter, r *http.Request, proxier HandlerFunc) {
	// When caching, we don't need to do anything here.
}

func (s *Server) handleProxyAuthInfo(proxier func(string) HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := readFromBody[proton.AuthInfoReq](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if info, ok := s.authCacher.GetAuthInfo(req.Username); ok {
			if err := writeBody(w, info); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			b, err := proxier(r.URL.Path)(w, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			res, err := readFrom[proton.AuthInfo](b)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			s.authCacher.SetAuthInfo(req.Username, res)
		}
	}
}

type HandlerFunc func(http.ResponseWriter, *http.Request) ([]byte, error)

type proxyServer struct {
	mux *http.ServeMux

	origin, base string

	transport http.RoundTripper

	debug bool
}

func newProxyServer(origin, base string, transport http.RoundTripper) *proxyServer {
	return &proxyServer{
		mux:       http.NewServeMux(),
		origin:    origin,
		base:      base,
		transport: transport,
		debug:     os.Getenv("GO_PROTON_API_SERVER_LOGGER_ENABLED") != "",
	}
}

func (s *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *proxyServer) handle(path string, h func(func(string) HandlerFunc) http.HandlerFunc) {
	s.mux.Handle(s.base+path, h(func(path string) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
			buf := new(bytes.Buffer)

			// Call the proxy, capturing whatever data it writes.
			s.newProxy(path)(&writerWrapper{w, buf}, r)

			// If there is a gzip header entry, decode it.
			if strings.Contains(w.Header().Get("Content-Encoding"), "gzip") {
				return gzipDecode(buf.Bytes())
			}

			// Otherwise, return the original written data.
			return buf.Bytes(), nil
		}
	}))
}

type writerWrapper struct {
	http.ResponseWriter

	buf *bytes.Buffer
}

func (w *writerWrapper) Write(b []byte) (int, error) {
	if _, err := w.buf.Write(b); err != nil {
		return 0, err
	}

	return w.ResponseWriter.Write(b)
}

func readFrom[T any](b []byte) (T, error) {
	var v T

	if err := json.Unmarshal(b, &v); err != nil {
		return *new(T), err
	}

	return v, nil
}

func readFromBody[T any](r *http.Request) (T, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return *new(T), err
	}
	defer r.Body.Close()

	v, err := readFrom[T](b)
	if err != nil {
		return *new(T), err
	}

	r.Body = io.NopCloser(bytes.NewReader(b))

	return v, nil
}

func writeBody[T any](w http.ResponseWriter, v T) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(b); err != nil {
		return err
	}

	return nil
}

func gzipDecode(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
