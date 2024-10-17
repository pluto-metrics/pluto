package config

type EnvQueryLog struct {
	Kind      string `expr:"kind"`
	Query     string `expr:"query"`
	ElapsedMs int64  `expr:"elapsed_ms"`
}

func (ch *ClickHouse) compile() error {
	if ch == nil {
		return nil
	}
	return ch.QueryLog.compileWhen(EnvQueryLog{})
}
