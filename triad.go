package triad

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"
)

// Triad is registry of all registered route
type Triad struct {
	mux         *http.ServeMux
	middlewares []MiddlewareFunc
	prefix      string
	// Determines if router is inline
	inline bool
	// locks the middleware after route is registered
	mwlock bool
	// routes contains information about route like
	// method type, path, middleware names, handler name.
	routes *Routes
	// DisableRouteInfo if true stops the information collection
	// of routes. make false if debugging is required.
	DisableRouteInfo bool
	Logger           *slog.Logger
	HTTPErrorHandler
}

func New() *Triad {
	t := &Triad{
		mux:    http.NewServeMux(),
		routes: NewRoutes(),
		Logger: slog.Default(),
	}
	t.HTTPErrorHandler = DefaultErrHandler(false, t.Logger)
	return t
}

// Use appends a middleware handler to the Mux middleware stack.
//
// The middleware stack for any Router will execute before searching for a
// matching route to a specific handler, which provides opportunity to respond
// early, change the course of the request execution, or set request-scoped
// values for the next [HandlerFunc].
//
// NOTE: middleware will execute in order they are passed. all middlewares must
// be registered before routes. The additional middlewar can be registered using
// With and Group handlers.
func (t *Triad) Use(mws ...MiddlewareFunc) {
	if t.mwlock {
		panic("triad: middlewares must be added before any routes are registered")
	}
	t.middlewares = append(t.middlewares, mws...)
}

// Group creates a new inline router with a copy of all parent middlewares.
// It's useful for a group of handlers with the same routing path that use
// additional middleware. This can also be used to mount a subrouter in
// larger projects.
func (t *Triad) Group(pattern string, fn func(g *Triad)) *Triad {
	ir := t.With()
	ir.prefix = t.prefix + pattern
	if fn != nil {
		fn(ir)
	}
	return ir
}

// With adds inline middlewares for an endpoint handler.
// With can be used as group without route so this means
func (t *Triad) With(middlewares ...MiddlewareFunc) *Triad {
	if !t.inline && !t.mwlock {
		t.mwlock = true
	}
	mws := make([]MiddlewareFunc, len(t.middlewares))
	copy(mws, t.middlewares)
	mws = append(mws, middlewares...)
	return &Triad{
		mux:              t.mux,
		middlewares:      mws,
		HTTPErrorHandler: t.HTTPErrorHandler,
		DisableRouteInfo: t.DisableRouteInfo,
		routes:           t.routes,
		prefix:           t.prefix,
		inline:           true,
	}
}

func (t *Triad) Get(pattern string, h HandlerFunc) {
	t.handle(http.MethodGet, pattern, h)
}

func (t *Triad) Head(pattern string, h HandlerFunc) {
	t.handle(http.MethodHead, pattern, h)
}

func (t *Triad) Post(pattern string, h HandlerFunc) {
	t.handle(http.MethodPost, pattern, h)
}

func (t *Triad) Put(pattern string, h HandlerFunc) {
	t.handle(http.MethodPut, pattern, h)
}

func (t *Triad) Patch(pattern string, h HandlerFunc) {
	t.handle(http.MethodPatch, pattern, h)
}

func (t *Triad) Delete(pattern string, h HandlerFunc) {
	t.handle(http.MethodDelete, pattern, h)
}

func (t *Triad) Connect(pattern string, h HandlerFunc) {
	t.handle(http.MethodConnect, pattern, h)
}

func (t *Triad) Options(pattern string, h HandlerFunc) {
	t.handle(http.MethodOptions, pattern, h)
}

func (t *Triad) Trace(pattern string, h HandlerFunc) {
	t.handle(http.MethodTrace, pattern, h)
}

func (t *Triad) Method(method, pattern string, h HandlerFunc) {
	if method == "" {
		panic("triad: method must not be empty")
	}
	t.handle(method, pattern, h)
}

// handle creates route path add it in global routes.
// And handle error handler with HandlerFunc.
func (t *Triad) handle(method, pattern string, h HandlerFunc) {
	t.mwlock = true
	if !t.DisableRouteInfo {
		mws := make([]string, 0, len(t.middlewares))
		for _, mw := range t.middlewares {
			mws = append(mws, runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name())
		}
		t.routes.add(RouteInfo{
			Method:     method,
			Pattern:    t.prefix + pattern,
			Handler:    runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name(),
			Middleware: mws,
		})
	}
	pattern = method + " " + t.prefix + pattern
	t.mux.Handle(pattern, Handler{
		eh: t.HTTPErrorHandler,
		fn: chain(t.middlewares, h),
	})
}

// RouteInfo return routes information such as middleware
// call chain, handler name, http method, pattern
func (t *Triad) RouteInfo() *Routes {
	return t.routes
}

// ServeHTTP implements [http.Handler]
func (t *Triad) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.mux.ServeHTTP(w, r)
}

func (t *Triad) Start(addr string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	srv := &Server{Address: addr}
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error(err.Error())
		}
	}()
	return srv.Start(ctx, t)
}

// CompatHandler converts [http.Handler] to [HandlerFunc].
func CompatHandler(h http.Handler) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h.ServeHTTP(w, r)
		return nil
	}
}

// CompatHandler converts [http.HandlerFunc] to [HandlerFunc].
func Compat(h http.HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h(w, r)
		return nil
	}
}

// CompatMiddleware converts standard net/http middleware into MiddlewareFunc.
//
// Since net/http middleware uses http.Handler and has no error return, this
// adapter does not propagate errors returned by downstream HandlerFunc if it
// is the type of http.Handler or http.HandlerFunc.
func CompatMiddleware(mw func(http.Handler) http.Handler) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			// 1. Wrap next handler to HandlerFunc
			// 2. Wrap it into original middlware
			// 3. Call ServeHTTP on original wrapped handler
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err = next(w, r)
			})).ServeHTTP(w, r)
			return err
		}
	}
}

// chain builds a http.Handler composed of an inline middleware
// stack and endpoint handler in the order they are passed.
func chain(middlewares []MiddlewareFunc, h HandlerFunc) HandlerFunc {
	// Return ahead of time if there aren't any middlewares for the chain
	if len(middlewares) == 0 {
		return h
	}

	// Wrap the end handler with the middleware chain
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

const (
	website = "https://jshk00.github.io/triad"
	version = "1.0.0"
	banner  = `
  _____   ____    ___      _      ____
 |_   _| |  _ \  |_ _|    / \    |  _ \
   | |   | |_) |  | |    / _ \   | | | |
   | |   |  _ <   | |   / ___ \  | |_| |
   |_|   |_| \_\ |___| /_/   \_\ |____/
`
)

// Mime Types
const (
	charsetUTF8 = "charset=UTF-8"
	// MIMEApplicationJSON JavaScript Object Notation (JSON) https://www.rfc-editor.org/rfc/rfc8259
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMETextXML                          = "text/xml"
	MIMETextStream                       = "text/event-stream"
	MIMETextXMLCharsetUTF8               = MIMETextXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

// Headers
const (
	HeaderAccept         = "Accept"
	HeaderAcceptEncoding = "Accept-Encoding"
	// HeaderAllow is the name of the "Allow" header field used to list the set of methods
	// advertised as supported by the target resource. Returning an Allow header is mandatory
	// for status 405 (method not found) and useful for the OPTIONS method in responses.
	// See RFC 7231: https://datatracker.ietf.org/doc/html/rfc7231#section-7.4.1
	HeaderAllow               = "Allow"
	HeaderAuthorization       = "Authorization"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderContentEncoding     = "Content-Encoding"
	HeaderContentLength       = "Content-Length"
	HeaderContentType         = "Content-Type"
	HeaderCookie              = "Cookie"
	HeaderSetCookie           = "Set-Cookie"
	HeaderIfModifiedSince     = "If-Modified-Since"
	HeaderLastModified        = "Last-Modified"
	HeaderLocation            = "Location"
	HeaderRetryAfter          = "Retry-After"
	HeaderUpgrade             = "Upgrade"
	HeaderVary                = "Vary"
	HeaderWWWAuthenticate     = "WWW-Authenticate"
	HeaderXForwardedFor       = "X-Forwarded-For"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSsl       = "X-Forwarded-Ssl"
	HeaderXUrlScheme          = "X-Url-Scheme"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXRealIP             = "X-Real-Ip"
	HeaderXRequestID          = "X-Request-Id"
	HeaderXCorrelationID      = "X-Correlation-Id"
	HeaderXRequestedWith      = "X-Requested-With"
	HeaderServer              = "Server"

	// HeaderOrigin request header indicates the origin (scheme, hostname, and port) that caused the
	// request. See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin
	HeaderOrigin       = "Origin"
	HeaderCacheControl = "Cache-Control"
	HeaderConnection   = "Connection"

	// Access control
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity         = "Strict-Transport-Security"
	HeaderXContentTypeOptions             = "X-Content-Type-Options"
	HeaderXXSSProtection                  = "X-XSS-Protection"
	HeaderXFrameOptions                   = "X-Frame-Options"
	HeaderContentSecurityPolicy           = "Content-Security-Policy"
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only"
	HeaderXCSRFToken                      = "X-CSRF-Token"
	HeaderReferrerPolicy                  = "Referrer-Policy"
)

type (
	// HTTPErrorHandler is a custom function which writes
	// the error returned from handler to [http.Response].
	HTTPErrorHandler func(http.ResponseWriter, *http.Request, error)
	// HandlerFunc is signature type for handler registration.
	HandlerFunc func(w http.ResponseWriter, r *http.Request) error
	// MiddlewareFunc is a middleware function signature.
	MiddlewareFunc func(next HandlerFunc) HandlerFunc
)

// Handler implements [http.Handler] interface with custom
// ErrorHandler and middleware wraping built in.
type Handler struct {
	eh HTTPErrorHandler
	fn HandlerFunc
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.fn(w, r); err != nil {
		h.eh(w, r, err)
	}
}
