package scope

import "context"

type contextKey string

var contextKeyQueryLogger contextKey = "queryLogger"
var contextKeyAccessLogger contextKey = "accessLogger"
var contextKeyZapLogger contextKey = "zapLogger"

func contextGet[T any](ctx context.Context, key contextKey) T {
	v := ctx.Value(key)
	var empty T
	if v == nil {
		return empty
	}
	if out, ok := v.(T); ok {
		return out
	}
	return empty
}
