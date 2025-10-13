package prom

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/pluto/pkg/lg"
	"github.com/lomik/pluto/pkg/query"
)

func (q *Querier) request(ctx context.Context, ch *config.ClickHouse, qq string) (*query.Request, error) {
	chRequest, err := query.NewRequest(ctx, *ch, query.Opts{
		Discovery:  q.config.Extension.ClickHouseDiscovery,
		HTTPClient: q.config.Extension.HTTPClient,
	})
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
