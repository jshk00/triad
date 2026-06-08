---
title: Getting Started
---

## Installation
`go get -u github.com/jshk00/triad`

## Running a Simple Server
```go
package main

import (
    "net/http"

    "github.com/jshk00/triad"
)

func main() {
    r := triad.New()
    r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
        _, err := w.Write([]byte("Hello World!"))
        return err
    })
    r.Start(":8080")
}
```
