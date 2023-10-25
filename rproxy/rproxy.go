package rproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// proxy represents the reverse proxy structure, holding the ReverseProxy
// from the httputil package, a zap logger, and the original Director function.
type proxy struct {
	*httputil.ReverseProxy
	log              *zap.Logger
	originalDirector func(*http.Request)
}

// NewProxy initializes and returns a new reverse proxy pointing to the given host and port.
// It takes a remote host address, port, and a logger as its arguments.
func NewProxy(rhost string, rport int, logger *zap.Logger) (*proxy, error) {
	host := fmt.Sprintf("http://%s:%d", rhost, rport)
	url, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	p := &proxy{
		ReverseProxy:     httputil.NewSingleHostReverseProxy(url),
		log:              logger,
		originalDirector: nil,
	}

	// Attach hooks
	p.originalDirector = p.Director
	p.Director = p.hookRequest()
	p.ModifyResponse = p.hookResponse()
	p.ErrorHandler = p.errorHandler

	return p, nil
}

// ProxyRequestHandler returns a function suitable for use as an http.HandlerFunc.
// The returned function will use the provided ReverseProxy to proxy HTTP requests.
func ProxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(rw, req)
	}
}

// hookRequest returns a function that modifies incoming requests.
// The function logs the received requests.
func (p *proxy) hookRequest() func(req *http.Request) {
	return func(req *http.Request) {
		p.originalDirector(req)
		p.log.Info("Received incoming request",
			zap.String("Method", req.Method),
			zap.String("URL", req.URL.String()),
			zap.String("Client IP", req.RemoteAddr),
			zap.String("User-Agent", req.UserAgent()),
			zap.Time("Timestamp", time.Now()))
	}
}

// hookResponse returns a function that modifies outgoing responses.
// The function logs the outgoing responses.
func (p *proxy) hookResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		p.log.Info("Outgoing response",
			zap.String("Status", resp.Status),
			zap.Time("Timestamp", time.Now()))
		return nil
	}
}

// errorHandler logs any errors encountered by the reverse proxy.
func (p *proxy) errorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	p.log.Error("Error encountered while processing request",
		zap.Error(err),
		zap.String("Method", req.Method),
		zap.String("URL", req.URL.String()),
		zap.String("Client IP", req.RemoteAddr),
		zap.String("User-Agent", req.UserAgent()),
		zap.Time("Timestamp", time.Now()))
}
