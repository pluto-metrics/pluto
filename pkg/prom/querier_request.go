package prom

import (
	"context"
	"fmt"

	"github.com/pluto-metrics/pluto/pkg/query"
	"github.com/pluto-metrics/pluto/pkg/scope"
	"go.uber.org/zap"
)

func (q *Querier) request(ctx context.Context, qq string) (*query.Request, error) {
	ctx = scope.QueryBegin(ctx)
	scope.QueryWith(ctx, zap.String("query", qq))

	chRequest, err := query.NewRequest(ctx, q.config.ClickHouse, query.Opts{})
	if err != nil {
		zap.L().Error("can't create request to clickhouse", zap.Error(err))
		return nil, err
	}

	_, err = fmt.Fprint(chRequest, qq)
	if err != nil {
		zap.L().Error("can't write query to clickhouse", zap.Error(err))
		return nil, err
	}

	return chRequest, nil
}
