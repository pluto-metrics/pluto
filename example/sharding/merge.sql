CREATE TABLE series (
	`name` String,
	`labels` Map(String, String),
	`id` String,
	`timestamp` Int64 EPHEMERAL,
	`timestamp_min` SimpleAggregateFunction(min, Int64) DEFAULT `timestamp`,
	`timestamp_max` SimpleAggregateFunction(max, Int64) DEFAULT `timestamp`
) ENGINE=Merge(REGEXP('shard*'), '^series$');

CREATE TABLE samples (
	`id` String CODEC(ZSTD(3)),
	`timestamp` Int64 CODEC(Delta(), ZSTD(3)),
	`value` SimpleAggregateFunction(max, Float64) CODEC(Gorilla, ZSTD(3))
) ENGINE=Merge(REGEXP('shard*'), '^samples$');