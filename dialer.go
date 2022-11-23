package proton

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
)

func InsecureTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

// NetCtl can be used to control whether a dialer can dial, and whether the resulting
// connection can read or write.
type NetCtl struct {
	canDial   atomicBool
	dialLimit atomicUint64

	canRead   atomicBool
	readLimit atomicUint64

	canWrite   atomicBool
	writeLimit atomicUint64

	onDial  []func(net.Conn)
	onRead  []func([]byte)
	onWrite []func([]byte)

	lock sync.Mutex
}

// NewNetCtl returns a new NetCtl with all fields set to true.
func NewNetCtl() *NetCtl {
	return &NetCtl{
		canDial:  atomicBool{b32(true)},
		canRead:  atomicBool{b32(true)},
		canWrite: atomicBool{b32(true)},
	}
}

// SetCanDial sets whether the dialer can dial.
func (c *NetCtl) SetCanDial(canDial bool) {
	c.canDial.Store(canDial)
}

// SetDialLimit sets the maximum number of times dialers using this controller can dial.
func (c *NetCtl) SetDialLimit(limit uint64) {
	c.dialLimit.Store(limit)
}

// SetCanRead sets whether the connection can read.
func (c *NetCtl) SetCanRead(canRead bool) {
	c.canRead.Store(canRead)
}

// SetReadLimit sets the maximum number of bytes that can be read.
func (c *NetCtl) SetReadLimit(limit uint64) {
	c.readLimit.Store(limit)
}

// SetCanWrite sets whether the connection can write.
func (c *NetCtl) SetCanWrite(canWrite bool) {
	c.canWrite.Store(canWrite)
}

// SetWriteLimit sets the maximum number of bytes that can be written.
func (c *NetCtl) SetWriteLimit(limit uint64) {
	c.writeLimit.Store(limit)
}

// OnDial adds a callback that is called with the created connection when a dial is successful.
func (c *NetCtl) OnDial(f func(net.Conn)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onDial = append(c.onDial, f)
}

// OnRead adds a callback that is called with the read bytes when a read is successful.
func (c *NetCtl) OnRead(f func([]byte)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onRead = append(c.onRead, f)
}

// OnWrite adds a callback that is called with the written bytes when a write is successful.
func (c *NetCtl) OnWrite(f func([]byte)) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.onWrite = append(c.onWrite, f)
}

// Disable is equivalent to disallowing dial, read and write.
func (c *NetCtl) Disable() {
	c.SetCanDial(false)
	c.SetCanRead(false)
	c.SetCanWrite(false)
}

// Enable is equivalent to allowing dial, read and write.
func (c *NetCtl) Enable() {
	c.SetCanDial(true)
	c.SetCanRead(true)
	c.SetCanWrite(true)
}

// Conn is a wrapper around net.Conn that can be used to control whether a connection can read or write.
type Conn struct {
	net.Conn

	ctl *NetCtl

	readLimiter  *readLimiter
	writeLimiter *writeLimiter
}

// Read reads from the wrapped connection, but only if the controller allows it.
func (c *Conn) Read(b []byte) (int, error) {
	if !c.ctl.canRead.Load() {
		return 0, errors.New("cannot read")
	}

	n, err := c.readLimiter.read(c.Conn, b)
	if err != nil {
		return n, err
	}

	for _, f := range c.ctl.onRead {
		f(b[:n])
	}

	return n, err
}

// Write writes to the wrapped connection, but only if the controller allows it.
func (c *Conn) Write(b []byte) (int, error) {
	if !c.ctl.canWrite.Load() {
		return 0, errors.New("cannot write")
	}

	n, err := c.writeLimiter.write(c.Conn, b)
	if err != nil {
		return n, err
	}

	for _, f := range c.ctl.onWrite {
		f(b[:n])
	}

	return n, err
}

// Dialer performs network dialing, but only if the controller allows it.
type Dialer struct {
	ctl *NetCtl

	netDialer *net.Dialer
	tlsDialer *tls.Dialer
	tlsConfig *tls.Config

	readLimiter  *readLimiter
	writeLimiter *writeLimiter

	dialCount atomicUint64
}

// NewDialer returns a new dialer using the given net controller.
// It optionally uses a provided tls config.
func NewDialer(ctl *NetCtl, tlsConfig *tls.Config) *Dialer {
	return &Dialer{
		ctl: ctl,

		netDialer: &net.Dialer{},
		tlsDialer: &tls.Dialer{Config: tlsConfig},
		tlsConfig: tlsConfig,

		readLimiter:  newReadLimiter(ctl),
		writeLimiter: newWriteLimiter(ctl),

		dialCount: atomicUint64{0},
	}
}

// DialContext dials a network connection, but only if the controller allows it.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.dialWithDialer(ctx, network, addr, d.netDialer)
}

// DialTLSContext dials a TLS network connection, but only if the controller allows it.
func (d *Dialer) DialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.dialWithDialer(ctx, network, addr, d.tlsDialer)
}

// dialWithDialer dials a network connection using the given dialer, but only if the controller allows it.
func (d *Dialer) dialWithDialer(ctx context.Context, network, addr string, dialer dialer) (net.Conn, error) {
	if !d.ctl.canDial.Load() {
		return nil, errors.New("cannot dial")
	}

	if limit := d.ctl.dialLimit.Load(); limit > 0 && d.dialCount.Load() >= limit {
		return nil, errors.New("dial limit reached")
	} else {
		d.dialCount.Add(1)
	}

	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	d.ctl.lock.Lock()
	defer d.ctl.lock.Unlock()

	for _, f := range d.ctl.onDial {
		f(conn)
	}

	return &Conn{
		Conn: conn,
		ctl:  d.ctl,

		readLimiter:  d.readLimiter,
		writeLimiter: d.writeLimiter,
	}, nil
}

// GetRoundTripper returns a new http.RoundTripper that uses the dialer.
func (d *Dialer) GetRoundTripper() http.RoundTripper {
	return &http.Transport{
		DialContext:     d.DialContext,
		DialTLSContext:  d.DialTLSContext,
		TLSClientConfig: d.tlsConfig,
	}
}

type dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type readLimiter struct {
	ctl *NetCtl

	count atomicUint64
}

// newReadLimiter returns a new io.Reader that reads from r, but only up to limit bytes.
func newReadLimiter(ctl *NetCtl) *readLimiter {
	return &readLimiter{
		ctl: ctl,
	}
}

func (limiter *readLimiter) read(r io.Reader, b []byte) (int, error) {
	if limit := limiter.ctl.readLimit.Load(); limit > 0 && limiter.count.Load() >= limit {
		return 0, fmt.Errorf("refusing to read: read limit reached")
	}

	n, err := r.Read(b)
	if err != nil {
		return n, err
	}

	if limit := limiter.ctl.readLimit.Load(); limit > 0 {
		if new := limiter.count.Add(uint64(n)); new >= limit {
			return 0, fmt.Errorf("read failed: read limit reached")
		}
	}

	return n, err
}

type writeLimiter struct {
	ctl *NetCtl

	count atomicUint64
}

// newWriteLimiter returns a new io.Writer that writes to w, but only up to limit bytes.
func newWriteLimiter(ctl *NetCtl) *writeLimiter {
	return &writeLimiter{
		ctl: ctl,
	}
}

func (limiter *writeLimiter) write(w io.Writer, b []byte) (int, error) {
	if limit := limiter.ctl.writeLimit.Load(); limit > 0 && limiter.count.Load() >= limit {
		return 0, fmt.Errorf("refusing to write: write limit reached")
	}

	n, err := w.Write(b)
	if err != nil {
		return n, err
	}

	if limit := limiter.ctl.writeLimit.Load(); limit > 0 {
		if new := limiter.count.Add(uint64(n)); new >= limit {
			return 0, fmt.Errorf("write failed: write limit reached")
		}
	}

	return n, err
}
