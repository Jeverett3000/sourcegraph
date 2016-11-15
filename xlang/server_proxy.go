package xlang

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	basictracer "github.com/opentracing/basictracer-go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/ctxvfs"
	"github.com/sourcegraph/go-langserver/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
	"sourcegraph.com/sourcegraph/sourcegraph/xlang/lspext"
)

// serverID identifies a lang/build server by the minimal state
// necessary to reinitialize it. At most one lang/build server per
// serverID will be used; if two clients issue requests that route to
// the same serverID, their requests will be sent to the same
// lang/build server.
type serverID struct {
	contextID
	pathPrefix string // path to subdirectory, if lang/build server should run in subdirectory (otherwise empty)
}

func (id serverID) String() string {
	if id.pathPrefix == "" {
		return fmt.Sprintf("server(%s)", id.contextID)
	}
	return fmt.Sprintf("server(%s prefix=%q)", id.contextID, id.pathPrefix)
}

type serverProxyConn struct {
	conn *jsonrpc2.Conn // the LSP JSON-RPC 2.0 connection to the server

	id serverID

	// clientBroadcast is used to forward incoming requests from servers
	// to clients.
	clientBroadcast func(context.Context, *jsonrpc2.Request)

	// initOnce ensures we only connect and initialize once, and other
	// goroutines wait until the 1st goroutine completes those tasks.
	initOnce sync.Once
	initErr  error // only safe to write inside initOnce.Do(...), only safe to read after calling initOnce.Do(...)

	mu     sync.Mutex
	rootFS ctxvfs.FileSystem // the workspace's file system
	stats  serverProxyConnStats
}

// serverProxyConnStats contains statistics for a proxied connection to a server.
type serverProxyConnStats struct {
	// Created is the time the proxy connection was created
	Created time.Time

	// Last is max(last request sent, last response received), used to
	// disconnect from unused servers
	Last time.Time

	// TotalCount is the total number of calls proxied to the server.
	TotalCount int

	// Counts is the total number of calls proxied to the server per
	// LSP method.
	Counts map[string]int

	// TotalErrorCount is the total number of calls proxied to the server
	// that failed.
	TotalErrorCount int

	// ErrorCounts is the total number of calls proxied to the server that
	// failed per LSP method.
	ErrorCounts map[string]int
}

var (
	serverConnsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "src",
		Subsystem: "xlang",
		Name:      "open_lsp_server_connections",
		Help:      "Open connections (initialized + uninitialized) to the LSP servers.",
	})
	serverConnsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "src",
		Subsystem: "xlang",
		Name:      "cumu_lsp_server_connections",
		Help:      "Cumulative number of connections (initialized + uninitialized) to the LSP servers (total of open + previously closed since process startup).",
	})
	serverConnsMethodCalls = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "src",
		Subsystem: "xlang",
		Name:      "lsp_server_method_calls",
		Help:      "Total number of calls sent to a server proxy before it is shutdown.",
		Buckets:   []float64{1, 2, 4, 8, 16, 32, 64, 128, 256},
	}, []string{"mode"})
	serverConnsFailedMethodCalls = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "src",
		Subsystem: "xlang",
		Name:      "lsp_server_failed_method_calls",
		Help:      "Total number of failed calls sent to a server proxy before it is shutdown.",
		Buckets:   []float64{1, 2, 4, 8, 16, 32, 64, 128, 256},
	}, []string{"mode"})
	serverConnsAliveDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "src",
		Subsystem: "xlang",
		Name:      "lsp_server_alive_seconds",
		Help:      "The number of seconds a proxied connection is kept alive.",
		Buckets:   []float64{1, 10, 300, 2 * 300, 3 * 300, 4 * 300}, // 300 is the default MaxServerIdle
	}, []string{"mode"})
)

func init() {
	prometheus.MustRegister(serverConnsGauge)
	prometheus.MustRegister(serverConnsCounter)
	prometheus.MustRegister(serverConnsMethodCalls)
	prometheus.MustRegister(serverConnsFailedMethodCalls)
	prometheus.MustRegister(serverConnsAliveDuration)
}

// ShutDownIdleServers shuts down servers whose last communication
// with the proxy (either a request or a response) was longer than
// maxIdle ago. The Proxy runs ShutDownIdleServers periodically based
// on p.MaxServerIdle.
func (p *Proxy) ShutDownIdleServers(ctx context.Context, maxIdle time.Duration) error {
	cutoff := time.Now().Add(-1 * maxIdle)
	errs := &errorList{}
	var wg sync.WaitGroup
	p.mu.Lock()
	for s := range p.servers {
		s.mu.Lock()
		idle := s.stats.Last.Before(cutoff)
		s.mu.Unlock()
		if idle {
			wg.Add(1)
			go func(s *serverProxyConn) {
				defer wg.Done()
				// removeServerConn attempts to acquire p.mu,
				// take care not to deadlock with our for loop
				// which also holds p.mu
				p.removeServerConn(s)
				shutdownOK := true
				if err := s.shutdownAndExit(ctx); err != nil {
					errs.add(err)
					shutdownOK = false
				}
				if err := s.conn.Close(); err != nil && shutdownOK {
					errs.add(err)
				}
			}(s)
		}
	}
	// Only hold lock during fast loop iter, not while waiting to
	// shutdown and exit each idle connection (otherwise we could
	// block everything for a long time).
	p.mu.Unlock()

	wg.Wait()
	return errs.error()
}

// shutDownServer will terminate the server matching ID. If no such server
// exists, no action is taken.
func (p *Proxy) shutDownServer(ctx context.Context, id serverID) error {
	var c *serverProxyConn

	p.mu.Lock()
	for cc := range p.servers {
		if cc.id == id {
			c = cc
			break
		}
	}
	p.mu.Unlock()

	if c == nil {
		return nil
	}

	// We have found a server. Remove it from the list and do a
	// best-effort shutdown.
	errs := &errorList{}
	p.removeServerConn(c)
	if err := c.shutdownAndExit(ctx); err != nil {
		errs.add(err)
	}
	if err := c.conn.Close(); err != nil {
		errs.add(err)
	}
	return errs.error()
}

// LogServerStats, if true, will log the statistics of a serverProxyConn when
// it is removed.
var LogServerStats = true

func (p *Proxy) removeServerConn(c *serverProxyConn) {
	p.mu.Lock()
	_, ok := p.servers[c]
	if ok {
		delete(p.servers, c)
	}
	serverConnsGauge.Set(float64(len(p.servers)))
	p.mu.Unlock()
	if ok && LogServerStats {
		stats := c.Stats()
		// Machine parseable to assist post processing
		msg, _ := json.Marshal(struct {
			RootPath   string
			Mode       string
			PathPrefix string
			Stats      serverProxyConnStats
		}{
			RootPath:   c.id.rootPath.String(),
			Mode:       c.id.mode,
			PathPrefix: c.id.pathPrefix,
			Stats:      stats,
		})
		log.Printf("tracked removed serverProxyConn: %s", msg)
		serverConnsMethodCalls.WithLabelValues(c.id.mode).Observe(float64(stats.TotalCount))
		serverConnsFailedMethodCalls.WithLabelValues(c.id.mode).Observe(float64(stats.TotalErrorCount))
		serverConnsAliveDuration.WithLabelValues(c.id.mode).Observe(stats.Last.Sub(stats.Created).Seconds())
	}
}

// getServerConn returns an existing connection to the specified
// server or creates one if none exists.
func (p *Proxy) getServerConn(ctx context.Context, id serverID) (*serverProxyConn, error) {
	var c *serverProxyConn

	// Check for an already established connection.
	p.mu.Lock()
	for cc := range p.servers {
		if cc.id == id {
			p.mu.Unlock()
			c = cc
			break
		}
	}

	// No connection found, so we need to create one.
	if c == nil {
		// We're still holding p.mu. Do the minimum work necessary
		// here to be able to safely unlock it, so we don't block the entire
		// proxy.
		c = &serverProxyConn{
			id:              id,
			clientBroadcast: p.clientBroadcastFunc(id.contextID),
			stats: serverProxyConnStats{
				Created: time.Now(),
				Last:    time.Now(),
			},
		}
		p.servers[c] = struct{}{}
		serverConnsGauge.Set(float64(len(p.servers)))
		serverConnsCounter.Inc()
		p.mu.Unlock()
	}

	// No longer holding p.mu.

	// Connect and initialize.
	didWeInit := false // whether WE (not another goroutine) actually executed the c.initOnce.Do(...) func
	c.initOnce.Do(func() {
		didWeInit = true

		rwc, err := connectToServer(ctx, id.mode)
		if err != nil {
			c.initErr = err
			return
		}
		c.updateLastTime()

		var connOpt []jsonrpc2.ConnOpt
		if p.Trace {
			connOpt = append(connOpt, jsonrpc2.LogMessages(log.New(os.Stderr, "", 0)))
		}
		c.conn = jsonrpc2.NewConn(ctx, rwc, jsonrpc2.HandlerWithError(c.handle), connOpt...)

		// SECURITY NOTE: We assume that the caller to the LSP client
		// proxy has already checked the user's permissions to read
		// this repo, so we don't need to check permissions again
		// here.
		c.rootFS, err = NewRemoteRepoVFS(id.rootPath.CloneURL(), id.rootPath.Rev())
		if err != nil {
			_ = c.conn.Close()
			_ = rwc.Close()
			c.initErr = err
			return
		}

		if err := c.lspInitialize(ctx); err != nil {
			// Ignore cleanup errors (best effort).
			if fs, ok := c.rootFS.(io.Closer); ok && fs != nil {
				_ = fs.Close()
			}
			_ = c.conn.Close()
			_ = rwc.Close()
			c.initErr = err
			return
		}
		c.updateLastTime()

		go func() {
			select {
			case <-c.conn.DisconnectNotify():
			}
			p.removeServerConn(c)
		}()
	})

	err := c.initErr
	if err != nil {
		if didWeInit {
			// If we encounter an error during initialization, fail every
			// other goroutine that was waiting at the time (with the same
			// error), but don't prevent future goroutines from retrying (in
			// case of ephemeral errors).
			p.removeServerConn(c)
		} else {
			// Make it clear that we're just passing along an error
			// that another goroutine received, so it doesn't seem
			// (from the error logs) that we performed the same
			// network/etc. operation many times.
			err = fmt.Errorf("other goroutine failed to connect and initialize LSP server: %s", err)
		}
		c = nil
	}

	return c, err
}

// clientBroadcastFunc returns a function which will broadcast a request to
// all active clients for id.
func (p *Proxy) clientBroadcastFunc(id contextID) func(context.Context, *jsonrpc2.Request) {
	return func(ctx context.Context, req *jsonrpc2.Request) {
		// TODO(sqs): some clients will have already received these
		p.mu.Lock()
		for cc := range p.clients {
			// TODO(sqs): equality match omits pathPrefix
			if cc.context == id {
				// Ignore errors for forwarding diagnostics.
				go cc.handleFromServer(ctx, cc.conn, req)
			}
		}
		p.mu.Unlock()
	}
}

func (c *serverProxyConn) lspInitialize(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "LSP server proxy: initialize",
		opentracing.Tags{"mode": c.id.mode, "rootPath": c.id.rootPath.String()},
	)
	defer span.Finish()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.conn.Call(ctx, "initialize", lspext.InitializeParams{
		InitializeParams: lsp.InitializeParams{RootPath: "file:///"},
		OriginalRootPath: c.id.rootPath.String(),
		Mode:             c.id.mode,
	}, nil, addTraceMeta(ctx))
}

// callServer sends an LSP request to the specified server
// (establishing the connection first if necessary).
func (p *Proxy) callServer(ctx context.Context, id serverID, method string, params, result interface{}) (err error) {
	var c *serverProxyConn

	span, ctx := opentracing.StartSpanFromContext(ctx, "LSP server proxy: "+method,
		opentracing.Tags{"mode": id.mode, "rootPath": id.rootPath.String(), "method": method, "params": params},
	)
	defer func() {
		if err != nil {
			ext.Error.Set(span, true)
			span.LogEvent(fmt.Sprintf("error: %v", err))
			if c != nil {
				c.incMethodErrorStat(method)
			}
		}
		span.Finish()
	}()

	c, err = p.getServerConn(ctx, id)
	if err != nil {
		return err
	}
	c.updateLastTime()
	c.incMethodStat(method)

	return c.conn.Call(ctx, method, params, result, addTraceMeta(ctx))
}

// traceFSRequests is whether to trace the LSP proxy's incoming
// requests for fs/readFile, fs/readDir, fs/stat, fs/lstat, etc. It is
// off by default because there are a lot of these and the traces can
// get quite noisy if it's enabled, but it is configurable because
// it's useful when you are debugging certain perf issues.
var traceFSRequests, _ = strconv.ParseBool(os.Getenv("LSP_PROXY_TRACE_FS_REQUESTS"))

func (c *serverProxyConn) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	c.updateLastTime()
	defer c.updateLastTime()

	// Trace the handling of this request. Only create child spans for
	// significant operations, not when we're just receiving traces or
	// performing simple VFS ops.
	var span opentracing.Span
	if shouldCreateChildSpan := req.Method != "telemetry/event" && (traceFSRequests || !strings.HasPrefix(req.Method, "fs/")); shouldCreateChildSpan {
		op := "LSP server proxy: handle " + req.Method

		// Try to get our parent span context from this JSON-RPC request's metadata.
		if req.Meta != nil {
			var carrier opentracing.TextMapCarrier
			if err := json.Unmarshal(*req.Meta, &carrier); err != nil {
				return nil, err
			}
			if clientCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap, carrier); err == nil {
				span = opentracing.StartSpan(op, ext.RPCServerOption(clientCtx))
				ctx = opentracing.ContextWithSpan(ctx, span)
			} else if err != opentracing.ErrSpanContextNotFound {
				return nil, err
			}
		}

		// Otherwise derive the span from our own context.
		if span == nil {
			span, ctx = opentracing.StartSpanFromContext(ctx, op)
		}

		span.SetTag("method", req.Method)
		defer func() {
			if err != nil {
				ext.Error.Set(span, true)
				span.LogEvent(fmt.Sprintf("error: %v", err))
			}
			span.Finish()
		}()
	}

	switch req.Method {
	case "telemetry/event":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var rawSpan basictracer.RawSpan
		if err := json.Unmarshal(*req.Params, &rawSpan); err != nil {
			return nil, err
		}
		// Recording the raw span as-is requires the lower-level impl
		// types.
		if o, ok := opentracing.GlobalTracer().(basictracer.Tracer); ok {
			if r, ok := o.Options().Recorder.(*lightstep.Recorder); ok {
				r.RecordSpan(rawSpan)
			}
		}
		return nil, nil

	case "fs/readFile", "fs/readDirFiles", "fs/readDir", "fs/stat", "fs/lstat":
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		var path string
		if err := json.Unmarshal(*req.Params, &path); err != nil {
			return nil, err
		}
		if span != nil {
			span.SetTag("path", path)
		}
		return c.handleFS(ctx, req.Method, path)

	case "textDocument/publishDiagnostics":
		// Forward to all clients.
		c.clientBroadcast(ctx, req)

		return nil, nil
	}

	return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: fmt.Sprintf("server proxy handler: method not found: %q", req.Method)}
}

func (c *serverProxyConn) updateLastTime() {
	c.mu.Lock()
	c.stats.Last = time.Now()
	c.mu.Unlock()
}

func (c *serverProxyConn) incMethodStat(method string) {
	c.mu.Lock()
	c.stats.TotalCount++
	if c.stats.Counts == nil {
		c.stats.Counts = make(map[string]int)
	}
	c.stats.Counts[method] = c.stats.Counts[method] + 1
	c.mu.Unlock()
}

func (c *serverProxyConn) incMethodErrorStat(method string) {
	c.mu.Lock()
	c.stats.TotalErrorCount++
	if c.stats.ErrorCounts == nil {
		c.stats.ErrorCounts = make(map[string]int)
	}
	c.stats.ErrorCounts[method] = c.stats.ErrorCounts[method] + 1
	c.mu.Unlock()
}

func (c *serverProxyConn) Stats() serverProxyConnStats {
	c.mu.Lock()
	s := c.stats
	c.mu.Unlock()
	return s
}

// shutdownAndExit sends LSP "shutdown" and "exit" to the LSP server
// and closes this connection. The caller must ensure c is removed
// from proxy.servers.
func (c *serverProxyConn) shutdownAndExit(ctx context.Context) error {
	var errs errorList
	done := make(chan struct{})
	go func() {
		if err := c.conn.Call(ctx, "shutdown", nil, nil); err != nil {
			errs.add(err)
		}
		// Even if "shutdown" failed, still call "exit" to (hopefully)
		// tell the server to REALLY exit.
		if err := c.conn.Notify(ctx, "exit", nil); err != nil {
			errs.add(err)
		}
		close(done)
	}()

	// Respect the ctx deadline so we don't block for too long on an
	// unresponsive server or work.
	select {
	case <-done:
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			errs.add(err)
		}
	}

	// Close file system to free up resources (e.g., if the VFS is
	// backed by a file on disk, this will close the file).
	if fs, ok := c.rootFS.(io.Closer); ok && fs != nil {
		if err := fs.Close(); err != nil {
			errs.add(err)
		}
	}

	return errs.error()
}
