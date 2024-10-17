package scope

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type queryLogger struct {
	sync.Mutex
	sync.Once
	fields   []zap.Field
	cfg      *config.ClickHouse
	finished bool
}

func queryFormat(q string) string {
	a := strings.Split(q, "\n")
	for i := 0; i < len(a); i++ {
		a[i] = strings.TrimSpace(a[i])
	}
	return strings.TrimSpace(strings.Join(a, " "))
}

func QueryBegin(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyQueryLogger, &queryLogger{})
}

func QuerySetClickhouseConfig(ctx context.Context, cfg *config.ClickHouse) {
	q := contextGet[*queryLogger](ctx, contextKeyQueryLogger)
	if q == nil {
		return
	}
	q.Lock()
	q.cfg = cfg
	q.Unlock()
}

func QuerySetClickhouseSummary(ctx context.Context, summary string) {
	if summary == "" {
		return
	}
	q := contextGet[*queryLogger](ctx, contextKeyQueryLogger)
	if q == nil {
		return
	}
	s := make(map[string]interface{})
	err := json.Unmarshal([]byte(summary), &s)
	if err != nil {
		return
	}
	var fields []zap.Field
	for k, v := range s {
		vInt := cast.ToInt(v)
		if vInt != 0 {
			if k == "elapsed_ns" && vInt > 1_000_000 {
				fields = append(fields, zap.Int("elapsed_ms", vInt/1_000_000))
			} else {
				fields = append(fields, zap.Int(k, vInt))
			}
		}
	}
	if len(fields) == 0 {
		return
	}
	q.Lock()
	q.fields = append(q.fields, fields...)
	q.Unlock()
}

func QueryWith(ctx context.Context, fields ...zap.Field) {
	q := contextGet[*queryLogger](ctx, contextKeyQueryLogger)
	if q == nil {
		return
	}
	q.Lock()
	q.fields = append(q.fields, fields...)
	q.Unlock()
}

func QueryFinish(ctx context.Context) {
	q := contextGet[*queryLogger](ctx, contextKeyQueryLogger)
	if q == nil {
		return
	}

	q.Lock()
	defer q.Unlock()

	if q.finished {
		return
	}

	q.finished = true

	if q.cfg == nil {
		return
	}
	if q.cfg.QueryLog.Pass() {
		return
	}
	vars := config.EnvQueryLog{}

	for i := 0; i < len(q.fields); i++ {
		switch q.fields[i].Key {
		case "query":
			q.fields[i].String = queryFormat(q.fields[i].String)
			vars.Query = q.fields[i].String
			qn := strings.ToLower(q.fields[i].String)
			if strings.HasPrefix(qn, "insert") {
				vars.Kind = "insert"
			} else {
				vars.Kind = "select"
			}
		case "elapsed_ms":
			vars.ElapsedMs = q.fields[i].Integer
		}
	}

	result, err := q.cfg.QueryLog.When(vars)
	if err != nil {
		return
	}
	if cast.ToBool(result) {
		Logger(ctx).Info("query", q.fields...)
	}
}
