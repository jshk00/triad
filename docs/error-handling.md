---
title: Error Handling
---

Triad advocates for centralized HTTP error handling by returning error from middleware and handlers. Centralized error handler allows us to log errors to external services from a unified location and send a customized HTTP response to the client.

You can return a standard error or triad.*HTTPError.

For example, when basic auth middleware finds invalid credentials it returns 401 - Unauthorized error, aborting the current HTTP request.
```go
r := triad.New()
r.Use(func(next triad.HandlerFunc) triad.HandlerFunc {
  return func(w http.ResponseWrite, r *http.Request) error {
    // Extract the credentials from HTTP request header and perform a security
    // check

    // For invalid credentials
    return triad.NewHTTPError(http.StatusUnauthorized, "Please provide valid credentials")

    // For valid credentials call next
    // return next(c)
  }
})
```

You can also use `StatusErrors` which implements `triad.StatusCoder` interface.
```go
r := triad.New()
r.Get("/notfound", func(w http.ResponseWrite, r *http.Request) error {
    return triad.ErrNotFound // send 404 with messahe Not Found
})

// if exposedError true the wrapped error is also get displayed
r.Get("/notfound", func(w http.ResponseWrite, r *http.Request) error {
    return triad.ErrNotFound.Wrap(errors.New("db: user id not found"))
})
```

Following are available `StatusError` by default.
```go
ErrBadRequest                  // 400
ErrUnauthorized                // 401
ErrForbidden                   // 403
ErrNotFound                    // 404
ErrMethodNotAllowed            // 405
ErrRequestTimeout              // 408
ErrStatusRequestEntityTooLarge // 413
ErrUnsupportedMediaType        // 415
ErrTooManyRequests             // 429
ErrInternalServerError         // 500
ErrBadGateway                  // 502
ErrServiceUnavailable          // 503
```

## Default HTTP Error Handler
Triad provide default HTTP error handler which sends response in a JSON format.
```json
{
    "message": "error connecting to redis"
}
```
For a standard error, response is sent as 500 - Internal Server Error; however, if you the error with original error and  `exposedError` parameter is set to true in `DefaultErrHandler`, original error will be sent along with message.
```json
{
    "message": "file provided is not found",
    "error": "os: error opening file"
}
```

## Custom HTTP Error Handler 
Custom HTTP error handler can be set via `r.HTTPErrorHandle`.

For most cases default error HTTP handler should be sufficient; however, a custom HTTP error handler can come handy if you want to capture different type of errors and take action accordingly e.g. send notification email or log error to a centralized system. You can also send customized response to the client e.g. error page or just a JSON response.

To find error in an error chain that implements `triad.StatusCoder`,
```go
code := http.StatusInternalServerError
var sc triad.StatusCoder
if errors.As(err, &sc) { // find error in an error chain that implements StatusCoder
    if tmp := sc.StatusCode(); tmp != 0 {
        code = tmp
    }
}
```

**Example &rarr;**
```go
r := triad.New()
func customErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError

	var sc triad.StatusCoder
	if errors.As(err, &sc) {
		if c := sc.StatusCode(); c != 0 {
			code = c
		}
	}

	// Log the error.
	slog.Error(
		"request failed",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", code),
		slog.String("error", err.Error()),
	)

	// Write a custom response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"message": http.StatusText(code),
		"status":  code,
	})
}
r.HTTPErrorHandler = customErrorHandler
```
