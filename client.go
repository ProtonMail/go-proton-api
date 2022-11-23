package proton

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bradenaw/juniper/xsync"
	"github.com/go-resty/resty/v2"
)

// clientID is a unique identifier for a client.
var clientID uint64

// AuthHandler is given any new auths that are returned from the API due to an unexpected auth refresh.
type AuthHandler func(Auth)

// Handler is a generic function that can be registered for a certain event (e.g. deauth, API code).
type Handler func()

// Client is the proton client.
type Client struct {
	m *Manager

	// clientID is this client's unique ID.
	clientID uint64

	// attPool is the (lazy-initialized) pool of goroutines that fetch attachments.
	attPool func() *Pool[string, []byte]

	uid      string
	acc      string
	ref      string
	exp      time.Time
	authLock sync.RWMutex

	authHandlers   []AuthHandler
	deauthHandlers []Handler
	hookLock       sync.RWMutex

	deauthOnce sync.Once
}

func newClient(m *Manager, uid string) *Client {
	c := &Client{
		m:        m,
		uid:      uid,
		clientID: atomic.AddUint64(&clientID, 1),
	}

	c.attPool = xsync.Lazy(func() *Pool[string, []byte] {
		return NewPool(m.attPoolSize, c.getAttachment)
	})

	return c
}

func (c *Client) AddAuthHandler(handler AuthHandler) {
	c.hookLock.Lock()
	defer c.hookLock.Unlock()

	c.authHandlers = append(c.authHandlers, handler)
}

func (c *Client) AddDeauthHandler(handler Handler) {
	c.hookLock.Lock()
	defer c.hookLock.Unlock()

	c.deauthHandlers = append(c.deauthHandlers, handler)
}

func (c *Client) AddPreRequestHook(hook resty.RequestMiddleware) {
	c.hookLock.Lock()
	defer c.hookLock.Unlock()

	c.m.rc.OnBeforeRequest(func(rc *resty.Client, r *resty.Request) error {
		if clientID, ok := ClientIDFromContext(r.Context()); !ok || clientID != c.clientID {
			return nil
		}

		return hook(rc, r)
	})
}

func (c *Client) AddPostRequestHook(hook resty.ResponseMiddleware) {
	c.hookLock.Lock()
	defer c.hookLock.Unlock()

	c.m.rc.OnAfterResponse(func(rc *resty.Client, r *resty.Response) error {
		if clientID, ok := ClientIDFromContext(r.Request.Context()); !ok || clientID != c.clientID {
			return nil
		}

		return hook(rc, r)
	})
}

func (c *Client) Close() {
	c.attPool().Done()

	c.authLock.Lock()
	defer c.authLock.Unlock()

	c.uid = ""
	c.acc = ""
	c.ref = ""
	c.exp = time.Time{}

	c.hookLock.Lock()
	defer c.hookLock.Unlock()

	c.authHandlers = nil
	c.deauthHandlers = nil
}

func (c *Client) withAuth(acc, ref string, exp time.Time) *Client {
	c.acc = acc
	c.ref = ref
	c.exp = exp

	return c
}

func (c *Client) do(ctx context.Context, fn func(*resty.Request) (*resty.Response, error)) error {
	if _, err := c.doRes(ctx, fn); err != nil {
		return err
	}

	return nil
}

func (c *Client) doRes(ctx context.Context, fn func(*resty.Request) (*resty.Response, error)) (*resty.Response, error) {
	c.hookLock.RLock()
	defer c.hookLock.RUnlock()

	req, err := c.newReq(ctx)
	if err != nil {
		return nil, err
	}

	// Perform the request.
	res, err := fn(req)

	// If we receive no response, we can't do anything.
	if res.RawResponse == nil {
		return nil, fmt.Errorf("received no response from API: %w", err)
	}

	// If we receive a 401, notify deauth handlers.
	if res.StatusCode() == http.StatusUnauthorized {
		c.deauthOnce.Do(func() {
			for _, handler := range c.deauthHandlers {
				handler()
			}
		})
	}

	return res, err
}

func (c *Client) newReq(ctx context.Context) (*resty.Request, error) {
	c.authLock.Lock()
	defer c.authLock.Unlock()

	r := c.m.r(WithClient(ctx, c.clientID))

	if c.uid != "" {
		r.SetHeader("x-pm-uid", c.uid)
	}

	if time.Now().After(c.exp) {
		auth, err := c.m.authRefresh(ctx, c.uid, c.ref)
		if err != nil {
			return nil, err
		}

		c.acc = auth.AccessToken
		c.ref = auth.RefreshToken
		c.exp = time.Now().Add(time.Duration(auth.ExpiresIn) * time.Second)

		if err := c.handleAuth(auth); err != nil {
			return nil, err
		}
	}

	if c.acc != "" {
		r.SetAuthToken(c.acc)
	}

	return r, nil
}

func (c *Client) handleAuth(auth Auth) error {
	c.hookLock.RLock()
	defer c.hookLock.RUnlock()

	for _, handler := range c.authHandlers {
		handler(auth)
	}

	return nil
}
