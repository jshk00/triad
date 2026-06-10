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
You can also use `triad.NewHTTPError()` without a message, in that case status text is used as an error message. For example, "Unauthorized".

## Default HTTP Error Handler
Triad provide default HTTP error handler which sends response in a JSON format.
```json
{
    "message": "error connecting to redis"
}
```
For a standard error, response is sent as 500 - Internal Server Error; however, if you wraped the error with original error, original error message will be sent. See below examples for returning error with different format.
```go
r := triad.New()
r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
    return errors.New("the trace id is not present") // {"message": "the trace id is not present"}, code 500

    return triad.NewHTTPError(http.StatusUnauthorized, "unathorized") // {"message": "unathorized"}, code: 401

    return triad.NewHTTPError(http.StatusUnauthorized, "unathorized").XML() // <message>unathorized</message> 
    return triad.NewHTTPError(http.StatusUnauthorized, "unathorized").Text() // unauthorized
    return triad.NewHTTPError(http.StatusUnauthorized, "unathorized").Header(map[string]string{"header1": "value1"})  

    return triad.ErrBadRequest // {"message": "bad request"}
    return triad.ErrBadRequest.Header(map[string]string{"header1": "value1"}) 
    return triad.ErrBadRequest.Text()
    return triad.ErrBadRequest.XML()

})
```
