package lg

import "log/slog"

// Error ...
func Error(err error) slog.Attr {
	return slog.String("error", err.Error())

	// @TODO: handle stack
	// slog.GroupAttrs("", attrs...),
	// // Groups with empty keys are inlined.
}
