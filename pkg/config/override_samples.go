package config

type EnvSamples EnvSeries

func NewEnvSamples() *EnvSamples {
	return &EnvSamples{}
}

func (cfg *Config) GetSamples(values *EnvSamples) (ConfigSamples, error) {
	ret := ConfigSamples{
		Table:                  cfg.Select.TableSamples,
		ClickHouse:             &cfg.ClickHouse,
		SamplesTimestampUInt32: cfg.Select.SamplesTimestampUInt32,
	}

	for _, o := range cfg.OverrideSamples {
		result, err := o.When(values)
		if err != nil {
			return ret, err
		}

		if result {
			ret.Table = mergeZero(ret.Table, o.Table)
			ret.ClickHouse = mergeClickHouse(ret.ClickHouse, o.ClickHouse)
			ret.SamplesTimestampUInt32 = mergeZero(ret.SamplesTimestampUInt32, o.SamplesTimestampUInt32)
			return ret, nil
		}
	}

	return ret, nil
}
