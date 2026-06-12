---
title: Utilities
---

Triad provides a collection of helper functions for decoding requests and sending common HTTP responses.

## Bind JSON Request

`Bind` decodes the request body into a Go struct.

```go
func main() {
    type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
        Age  int    `json:"age"`
    }

    r := triad.New()

    r.Post("/users", func(w http.ResponseWriter, r *http.Request) error {
        var user User

        if err := triad.Bind(r, &user); err != nil {
            return triad.NewHTTPError(
                http.StatusBadRequest,
                "unable to decode request body",
            ).Wrap(err)
        }

        fmt.Printf("%+v\n", user)

        return triad.Text(w, "user received", http.StatusOK)
    })
}
```

## JSON Response

`JSON` serializes a value as JSON and writes the response.

```go
func main() {
    type User struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
        Age  int    `json:"age"`
    }

    r := triad.New()

    r.Get("/users/1", func(w http.ResponseWriter, r *http.Request) error {
        return triad.JSON(w, User{
            ID:   1,
            Name: "Jane Doe",
            Age:  32,
        }, http.StatusOK)
    })
}
```

## XML Response

`XML` serializes a value as XML and writes the response.

```go
func main() {
    type User struct {
        XMLName xml.Name `xml:"user"`
        ID      int      `xml:"id"`
        Name    string   `xml:"name"`
        Age     int      `xml:"age"`
    }

    r := triad.New()

    r.Get("/users/1", func(w http.ResponseWriter, r *http.Request) error {
        return triad.XML(w, User{
            ID:   1,
            Name: "Jane Doe",
            Age:  32,
        }, http.StatusOK)
    })
    // Produces:
    // <user>
    //   <id>1</id>
    //   <name>Jane Doe</name>
    //   <age>32</age>
    // </user>
}
```

## Plain Text Response

`Text` writes a plain text response.

```go
func main() {
    r := triad.New()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
        return triad.Text(w, "Hello, World!", http.StatusOK)
    })
}
```


## HTML Response

`HTML` writes an HTML response.

```go
const page = `<!DOCTYPE html>
<html>
<head>
    <title>Triad</title>
</head>
<body>
    <h1>Welcome</h1>
    <p>Hello from Triad.</p>
</body>
</html>`

func main() {
    r := triad.New()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
        return triad.HTML(w, page, http.StatusOK)
    })
}
```

## Server-Sent Events (SSE)

`SSE` streams events to the client until the request context is canceled.

```go
type generator struct {
    ch        chan io.WriterTo
    currentID int
}

func (g *generator) start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        defer close(g.ch)

        for {
            select {
            case <-ctx.Done():
                return

            case <-ticker.C:
                g.currentID++
                id := strconv.Itoa(g.currentID)

                g.ch <- &triad.Event{
                    ID:    id,
                    Event: "userdone",
                    Data:  "userID: " + id,
                }
            }
        }
    }()
}

func main() {
    r := triad.New()

    r.Get("/events", func(w http.ResponseWriter, r *http.Request) error {
        gen := &generator{
            ch: make(chan io.WriterTo, 1),
        }

        gen.start(r.Context())

        return triad.SSE(w, r, gen.ch)
    })
}
```

## Send a File

`File` streams a file to the client without loading the entire file into memory.

`Download` streams a file to the client as an attachment.

```go
func main() {
    r := triad.New()

    r.Get("/pdf", func(w http.ResponseWriter, r *http.Request) error {
        // Relative paths are resolved from the application's
        // current working directory.
        return triad.File(w, r, "sample.pdf")
    })
    r.Get("/download", func(w http.ResponseWriter, r *http.Request) error {
        return triad.Download(w, r, "movie-12.mkv")
    })
}
```

For files located elsewhere, provide an absolute path:

```go
return triad.File(w, r, "/var/www/files/sample.pdf")
```
