package trace

import (
	"net/http"
	"sync"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type middlewareManager struct {
	sync.RWMutex
	mp  map[string](func(next http.Handler) http.Handler)
	opf func(r *http.Request) string
}

func Middleware(opf func(r *http.Request) string) func(next http.Handler) http.Handler {
	mm := &middlewareManager{
		mp:  make(map[string]func(next http.Handler) http.Handler),
		opf: opf,
	}

	return mm.middleware
}

func (mm *middlewareManager) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op := mm.opf(r)
		if op == "" {
			next.ServeHTTP(w, r)
			return
		}

		mw := mm.get(op)

		mw(next).ServeHTTP(w, r)
	})
}

func (mm *middlewareManager) get(op string) func(next http.Handler) http.Handler {
	mm.RLock()
	mw, ok := mm.mp[op]
	mm.RUnlock()
	if ok {
		return mw
	}

	mm.Lock()
	defer mm.Unlock()
	// check again
	mw, ok = mm.mp[op]
	if ok {
		return mw
	}

	// create
	otelMw := otelhttp.NewMiddleware(op)
	mw = func(h http.Handler) http.Handler {
		return otelMw(otelhttp.WithRouteTag(op, h))
	}
	mm.mp[op] = mw
	return mw
}
