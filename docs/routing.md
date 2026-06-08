---
title: Routing
---
!!! info
    Routing refers to how application's endpoints (URI) responds to requests.

Triad supports all standard HTTP methods, including `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`, `TRACE`, and `CONNECT`.

## Handling HTTP Request Methods
These methods are defined on the `*Triad` as:
```go
// HTTP-method routing along `pattern`
Connect(pattern string, h triad.HandlerFunc)
Delete(pattern string, h triad.HandlerFunc)
Get(pattern string, h triad.HandlerFunc)
Head(pattern string, h triad.HandlerFunc)
Options(pattern string, h triad.HandlerFunc)
Patch(pattern string, h triad.HandlerFunc)
Post(pattern string, h triad.HandlerFunc)
Put(pattern string, h triad.HandlerFunc)
Trace(pattern string, h triad.HandlerFunc)
```

Each method has correponding helper on triad Example:
```go
r := triad.New()
r.Get("/path", listUsers)
r.Post("/path", createUser)
```

for custom or non-standard method use `(*triad).Method`, ie `r.Method("HELLO", "/path", HelloHandler)`

## Routing patterns & url parameters
Each routing method accepts a URL `pattern` and handler function.

Triad uses the same pattern syntax introduced in [Go 1.22's `net/http.ServeMux`](https://go.dev/blog/routing-enhancements).

Named path parameters can be retrieved using `(*http.Request).PathValue`:
```go
r := triad.New()
r.Get("/user/{userID}", )
func getUser(w http.ResponseWriter, r *http.Request) error {
    uid := r.PathValue("userID")
    if uid == "" {
        return fmt.Errorf("userID is empty")
    }
    userID, err := strconv.Atoi(uid)
    if err != nil {
        return fmt.Errorf("invalid userID: %w", err)
    }
    fmt.Println(userID)
}
```

## Type safe param parsing and Custom Decoders
For type-safe `query` and `path` parameter parsing, Triad provide generic utility function `QueryValue` and `PathValue`.
```go
r := triad.New()
r.Get("/user/{userID}", getUser)
func getUser(w http.ResponseWriter, r *http.Request) error {
    userID, err := triad.PathValue[int](r, "userID")
    // generally any parsing error or if decoder not found for given type
    if err != nil {
        return err
    }
    age, err := triad.QueryValue[int](r, "age")
    if err != nil {
        return err
    }
    return triad.Text(w, fmt.Sprintf("userID: %d, age: %d", userID, age), http.StatusOK)
}
```

Triad includes built-in decoders for common types:

- string
- int
- int64
- uint64
- float64

You can register your own decoder for custom or unsupported types &rarr;
```go
triad.RegisterDecoder(reflect.TypeFor[uint32], func(s string) (a any, err error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint32(v), err
})
```

!!! warning
    `RegisterDecoder` is not concurrency-safe. Register all custom decoders during application initialization before using `PathValue` or `QueryValue`. 

## Sub Routers and Routing Groups
Groups allow related routes to share middleware and a common path prefix.
```go
func main() {
    r := triad.New()
    r.With(paginate).Get("/", listArticles)                           // GET /articles

    r.Post("/", createArticle)                                        // POST /articles
    r.Get("/search", searchArticles)                                  // GET /articles/search

    // Subrouters:
    r.Group("/{articleID}", func(g  *triad.Triad) {
      g.Use(ArticleCtx)
      g.Get("/", getArticle)                                          // GET /articles/123
      g.Put("/", updateArticle)                                       // PUT /articles/123
      g.Delete("/", deleteArticle) 
    }
}
```
Groups can also be defined in separate packages.
```go
func main() {
    r := triad.New()
    r.Group("/admin", adminRouter)
}

func adminRouter(g *triad.Triad) {
    g.Use(Auth)
    g.Get("/dashboard", dashboardHandler)
    g.Get("/users", adminUsersHandler)
}
```
