package when

import (
	"context"
	"net/http"
)

var contextKeyRequest contextKey = "contextKeyRequest"

type Request struct {
	GetParams map[string]string `expr:"GET"`
	Headers   map[string]string `expr:"HEADER"`
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		env := &Request{
			GetParams: make(map[string]string),
			Headers:   make(map[string]string),
		}

		for k := range r.URL.Query() {
			env.GetParams[k] = r.URL.Query().Get(k)
		}

		for k := range r.Header {
			env.Headers[k] = r.Header.Get(k)
		}

		ctx := context.WithValue(r.Context(), contextKeyRequest, env)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
