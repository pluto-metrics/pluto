package prom

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/lg"
	"github.com/pluto-metrics/pluto/pkg/query"
)

func (q *Querier) request(ctx context.Context, ch *config.ClickHouse, qq string) (*query.Request, error) {
	chRequest, err := query.NewRequest(ctx, *ch, query.Opts{})
	if err != nil {
		slog.ErrorContext(ctx, "can't create request to clickhouse", lg.Error(err))
		return nil, err
	}

	_, err = fmt.Fprint(chRequest, qq)
	if err != nil {
		slog.ErrorContext(ctx, "can't write query to clickhouse", lg.Error(err))
		return nil, err
	}

	return chRequest, nil
}
