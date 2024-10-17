package config

type EnvSeries struct {
	Start           int64    `expr:"start"`
	End             int64    `expr:"end"`
	Limit           int      `expr:"limit"`
	Step            int64    `expr:"step"`
	Func            string   `expr:"func"`
	Grouping        []string `expr:"grouping"`
	By              bool     `expr:"by"`
	Range           int64    `expr:"range"`
	ShardCount      uint64   `expr:"shard_count"`
	ShardIndex      uint64   `expr:"shard_index"`
	DisableTrimming bool     `expr:"disable_trimming"`
}

func NewEnvSeries() *EnvSeries {
	return &EnvSeries{}
}

func (cfg *Config) GetSeries(values *EnvSeries) (ConfigSeries, error) {
	ret := ConfigSeries{
		Table:                    cfg.Select.TableSeries,
		AutocompleteLookback:     cfg.Select.AutocompleteLookback,
		SeriesPartitionMs:        cfg.Select.SeriesPartitionMs,
		SeriesMaterializedLabels: cfg.Select.SeriesMaterializedLabels,
		ClickHouse:               &cfg.ClickHouse,
	}

	for _, o := range cfg.OverrideSeries {
		result, err := o.When(values)
		if err != nil {
			return ret, err
		}

		if result {
			ret.Table = mergeZero(ret.Table, o.Table)
			ret.AutocompleteLookback = mergeZero(ret.AutocompleteLookback, o.AutocompleteLookback)
			ret.SeriesPartitionMs = mergeZero(ret.SeriesPartitionMs, o.SeriesPartitionMs)
			ret.SeriesMaterializedLabels = mergeNil(ret.SeriesMaterializedLabels, o.SeriesMaterializedLabels)
			ret.ClickHouse = mergeClickHouse(ret.ClickHouse, o.ClickHouse)
			return ret, nil
		}
	}

	return ret, nil
}
