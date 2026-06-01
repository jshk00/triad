package triad

import (
	"context"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	h := New()
	s := &Server{Address: ":8080"}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		<-ctx.Done()
		NoError(t, s.Shutdown(ctx))
	}()
	NoError(t, s.Start(context.Background(), h))
}
