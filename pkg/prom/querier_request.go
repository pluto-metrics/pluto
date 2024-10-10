package prom

import (
	"context"
	"fmt"

	"github.com/pluto-metrics/pluto/pkg/query"
	"go.uber.org/zap"
)

func (q *Querier) request(ctx context.Context, qq string) (*query.Request, error) {
	chRequest, err := query.NewRequest(ctx, q.config.ClickHouse.DSN, query.Opts{
		Database: q.config.ClickHouse.Database,
	})
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
