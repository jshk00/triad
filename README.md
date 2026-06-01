## Triad
Triad is lightweight wrapper around exisiting standard library mux which was improved in Go1.22.
Simply put it offers error returning handler `func(http.ResponseWriter, *http.Request) error`.
So we can avoid common mistake like forgetting to do naked return after writing error or response in standard `http.HandlerFunc`.

**Features &rarr;**
- [x] Middleware chaining
- [x] Centralize error handling for handler
- [x] Handler Grouping by middleware, route
- [ ] Quality of life helper functions such as Renderer, JSON, Text, Stream, SSE
- [ ] Optional Support as starting TLS server using lets encrypt
- [ ] Optional Support HTTP2 Server using h2c
- [x] No dependencies other than standard library
- [ ] All middlewares ported from Chi
- [x] Optional Support for routing mounted Handler
- [ ] Above 90% code coverage
