package prom

import (
	"context"
	"fmt"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/query"
	"go.uber.org/zap"
)

func (q *Querier) request(ctx context.Context, ch *config.ClickHouse, qq string) (*query.Request, error) {
	chRequest, err := query.NewRequest(ctx, *ch, query.Opts{})
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
