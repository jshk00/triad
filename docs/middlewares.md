---
title: Middleware
---
## Introduction
!!! info
    Middleware performs some specific function on the HTTP request or response at a specific stage in the HTTP pipeline before or after the user defined controller.
    Middleware is a design pattern to eloquently add cross cutting concerns like logging, handling authentication without having many code contact points.

Typical middleware use cases:
- Authentication
- Metrics Collections
- Rate Limiting
- Request/Response Modification
- Tracing and Observability

In triad middleware has signature `func(next triad.HandlerFunc) triad.HandlerFunc` internally expands to `func(next  func(http.ResponseWriter, *Request) error) func(http.ResponseWriter, *Request) error`. Triad middleware are fully compatible with `net/http` ecosystem. Any middleware implemented as `func(next http.Handler) http.Handler` can be used through compatibility layer. The main difference is `triad's` middleware participates in error propagation model. If any handler or middleware returns an error, Triad automatically invokes the configured error handler.

## Creating middleware
The example below stores a user ID in the request context and makes it available to downstream handlers.

```go
// http middleware setting value on request context
func UserMiddleware(next triad.HandlerFunc) triad.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) error {
        // create a request from exisiting request context pass the required
        // key and value pair.
        r = r.WithContext(context.WithValue(r.Context(), "user", "123"))
        // call the next handler in call chain passing newly created request
        // and response object. value set in request context will be accessible.
        return next(w, r)
    }
}

// Handler for fetching and sending middleware value set in call chain context
func UserHandler(w http.ResponseWriter, r *http.Request) error {
    // fetch the value from request context set in
    // UserMiddleware Above.
    user, ok := r.Context().Value("user").(string)
    if !ok {
        return errors.New("unexpected value type received")
    }
    // respond to client
    return triad.Text(w, "hi "+user, http.StatusOK)
}
```

Middleware execution order follows the registration chain, and any values added to the request context become available to subsequent middleware and handlers.

## Using standard library middleware
!!! warning
    Standard net/http middleware does not support Triad's error-returning handler model. Any errors must be handled and written to the response inside the middleware itself.

To register a standard library middleware, wrap it with `triad.CompatMiddleware`.

```go
// Standard stdlib middleware which logs request data
func Logger(next http.Handler) http.Handler {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &respWriter{ResponseWriter: w}
		next.ServeHTTP(w, r)
		end := time.Since(start)
		slog.Info(
			"request received",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_address", r.RemoteAddr),
			slog.Int("status_code", rw.code),
			slog.String("latency", end.String()),
		)
	}
}

func main() {
    r := triad.New()
    // Registering using Compatibility layer
    r.Use(traid.CompatMiddleware(Logger))
    r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
        return triad.Text(w, "root handler", http.StatusOK)
    })
}
```

## Middleware registration
Middleware must be registered **before any routes are added** to the router scope where the middleware applies.

This rule exists because route middleware chains are finalized when a route is registered.

**Root Router &rarr;**
Register global middleware before any routes:
```go
r := triad.New()
r.Use(AuthMiddleware)
r.Use(LoggerMiddleware)
r.Get("/", HomeHandler)
```
**Invalid**
```go
r := triad.New()
r.Get("/", HomeHandler)
r.Use(AuthMiddleware) // invalid
```

**Inline Group(`With`) &rarr;**
Middleware added through With() is applied only to routes registered through that derived router.
```go
r := triad.New()
r.Use(GlobalMiddleware)
api := r.With(AuthMiddleware)
api.Get("/profile", ProfileHandler)
```
**Invalid**
```go
r := traid.New()
api := r.With(AuthMiddleware)
r.Use(GlobalMiddleware) // invalid
api.Get("/profile", ProfileHandler)
```

`api` will not inherit middleware that is registered on the parent router after `With()` is created.

**Route Group &rarr;**
Middleware for a group must be registered before any routes inside that group.
```go
r := triad.New()
r.Group("/admin", func(g *triad.Triad) {
    g.Use(AdminMiddleware)
    g.Get("/", DashboardHandler)
})
```
**Invalid**
```go
r := triad.New()
r.Group("/admin", func(g *triad.Triad) {
    g.Get("/", DashboardHandler)
    g.Use(AdminMiddleware) // invalid
})
```

## Middleware Hierarchy
Middleware inherited from parent scopes.
```go
r.Use(GlobalMiddleware)

r.Get("/", GlobalHandler)

r.With(AuthMiddleware).Get("/profile", ProfileHandler)

r.Group("/admin", func(g *triad.Triad) {
    g.Use(AdminMiddleware)

    g.Get("/", DashboardHandler)
})
```

Request Flow &rarr;
```sh
# Global Flow
GlobalMiddleware
    ↓
GlobalHandler

# Inline Group Flow
GlobalMiddleware
    ↓
AuthMiddleware
    ↓
ProfileHandler

# Group Flow
GlobalMiddleware
    ↓
AdminMiddleware
    ↓
DashboardHandler
```
