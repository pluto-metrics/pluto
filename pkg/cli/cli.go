package cli

import (
	"context"
	"strings"

	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/rowbinary"
)

func New(ctx context.Context, mainConfig *config.Config, queryConfig *config.ClickHouse) rowbinary.Client {
	opts := make([]rowbinary.ClientOption, 0)
	if mainConfig.Extension.ClickHouseDiscovery != nil {
		opts = append(opts, rowbinary.WithDiscovery(mainConfig.Extension.ClickHouseDiscovery))
	}
	if mainConfig.Extension.HTTPClient != nil {
		opts = append(opts, rowbinary.WithHTTPClient(mainConfig.Extension.HTTPClient))
	}
	opts = append(opts, rowbinary.WithDSN(queryConfig.DSN))
	for key, value := range queryConfig.Params {
		if strings.HasPrefix(strings.ToLower(key), "x-clickhouse-") {
			opts = append(opts, rowbinary.WithHeader(key, value))
		} else {
			opts = append(opts, rowbinary.WithParam(key, value))
		}
	}
	return rowbinary.NewClient(ctx, opts...)
}
