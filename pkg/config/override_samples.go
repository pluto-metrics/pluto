package config

type EnvSamples EnvSeries

func NewEnvSamples() *EnvSamples {
	return &EnvSamples{}
}

func (cfg *Config) GetSamples(values *EnvSamples) (ConfigSamples, error) {
	ret := ConfigSamples{
		Table:      cfg.Select.TableSamples,
		ClickHouse: &cfg.ClickHouse,
	}

	for _, o := range cfg.OverrideSamples {
		result, err := o.When(values)
		if err != nil {
			return ret, err
		}

		if result {
			ret.Table = mergeZero(ret.Table, o.Table)
			ret.ClickHouse = mergeClickHouse(ret.ClickHouse, o.ClickHouse)
			return ret, nil
		}
	}

	return ret, nil
}
