## Triad

Triad is a lightweight, idiomatic HTTP router built on top of Go's `net/http` and `http.ServeMux` (Go 1.22+). It embraces the standard library while adding centralized error handling, composable middleware, route groups, and a small set of utilities for building maintainable HTTP services.

## Features

* Built on the Go 1.22 `http.ServeMux`
* Standard `net/http` compatible handlers and middleware
* Centralized HTTP error handling using error-returning handlers
* Composable middleware and nested route groups
* Generic path and query parameter decoding
* No external dependencies
* Helper utilities for JSON, XML, HTML, text, SSE, file serving, and request binding
* Small, fast, and production-ready

## Installation

```bash
go get github.com/jshk00/triad/v2
```

## Quick Start

```go
package main

import (
	"net/http"

	"github.com/jshk00/triad/v2"
)

func main() {
	r := triad.New()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
		return triad.Text(w, "Hello, World!", http.StatusOK)
	})

	if err := r.Start(":8080"); err != nil {
		panic(err)
	}
}
```

## Error-Returning Handlers

Instead of writing responses everywhere, handlers simply return an error. Triad invokes a centralized HTTP error handler, making logging and consistent API responses straightforward.

```go
r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) error {
	id, err := triad.PathValue[int](r, "id")
	if err != nil {
		return triad.NewHTTPError(http.StatusBadRequest).Wrap(err)
	}

	return triad.JSON(w, map[string]any{
		"id": id,
	}, http.StatusOK)
})
```

## Documentation

Full documentation is available at:

https://jshk00.github.io/triad

## Examples
Complete examples are available in the repository under [_example](https://github.com/jshk00/triad/tree/main/_examples) including REST APIs, middleware, routing, and server configuration.

## License
MIT
