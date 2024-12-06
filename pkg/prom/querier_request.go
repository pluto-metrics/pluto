package prom

import (
	"context"
	"fmt"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/query"
	"github.com/pluto-metrics/pluto/pkg/trace"
)

func (q *Querier) request(ctx context.Context, ch *config.ClickHouse, qq string) (*query.Request, error) {
	ctx, span := trace.Start(ctx, "Querier.request")
	defer span.End()

	chRequest, err := query.NewRequest(ctx, *ch, query.Opts{})
	if err != nil {
		trace.Log(ctx).Error("can't create request to clickhouse", trace.Error(err))
		return nil, err
	}

	_, err = fmt.Fprint(chRequest, qq)
	if err != nil {
		trace.Log(ctx).Error("can't write query to clickhouse", trace.Error(err))
		return nil, err
	}

	return chRequest, nil
}
