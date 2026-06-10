---
title: Start Server
---
Triad provides following `Triad.Start(addr string)` convenience method to start the server. Which uses the default configuration for graceful shutdown.

## HTTP Server
`Triad.Start` is convenience method that starts the http server with Triad serving the request.
```go
func main() {
    r := triad.New()
    // add middlewares and routes
    if err := r.Start(":8080"); err != nil {
        r.Logger.Error("failed to start the server", slog.String("error", err.Error()))
    }
}
```
same functionality using server configuration `triad.Server`
```go
func main() {
    r := triad.New()
    // add middleware and routes
	// ...

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
    srv := &Server{Address: ":8080"}
	
    go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
		    e.Logger.Error("error shuting down server", "error", err)
		}
	}()
	
    if err := srv.Start(ctx, t); err != nil {
		e.Logger.Error("error starting server", "error", err)
    }
}
```
Following is server using `http.Server`
```go
func main() {
    r := triad.New()
    // add middlewares and routes
    // ...

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
    srv := &http.Server{
        Address: ":8080",
        Handler: r,
    }
	
    go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
		    e.Logger.Error("error shuting down server", "error", err)
		}
	}()
	
    if err := srv.ListenAndServe(); err != nil {
		e.Logger.Error("error starting server", "error", err)
    }
}
```

## HTTPS Server
Triad offer `Server.StartTLS` convenience method for starting server in tls mode.
```go
func main() {
	r := triad.New()
	// add middleware and routes
	// ...

	srv := triad.Server{Address: ":8080"}
	if err := srv.StartTLS(context.Background(), e, "server.crt", "server.key"); err != nil {
		r.Logger.Error("failed to start server", "error", err)
	}
}
```
Following is equivalent to previous example using `http.Server` 
```go
func main() {
    r := triad.New()
    // add middleware and routes
    // ...
    s := http.Server{
      Addr:    ":8080",
      Handler: r, // set Echo as handler
      TLSConfig: &tls.Config{
        //MinVersion: 1, // customize TLS configuration
      },
      //ReadTimeout: 30 * time.Second, // use custom timeouts
    }
    if err := s.ListenAndServeTLS("server.crt", "server.key"); err != http.ErrServerClosed {
      log.Fatal(err)
    }
}
``
