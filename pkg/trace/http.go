package trace

import (
	"net/http"
)

/*
func muxHandleFunc(mux *http.ServeMux, pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
	mux.Handle(
		pattern,
		otelhttp.NewHandler(
			otelhttp.WithRouteTag(
				pattern,
				http.HandlerFunc(handlerFunc),
			),
			pattern,
			otelhttp.WithMeterProvider(nil),
			otelhttp.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
				return nil
			}),
		),
	)
}
*/

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
