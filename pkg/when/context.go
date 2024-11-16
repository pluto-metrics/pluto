package when

import "context"

type contextKey string

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
