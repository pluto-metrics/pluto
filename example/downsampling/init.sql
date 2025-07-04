-- table for inserts
CREATE TABLE samples_null (
	`id` String,
	`name` String,
	`labels` Map(String, String),
	`timestamp` Int64,
	`value` Float64
)
ENGINE = Null();

CREATE TABLE samples (
	`id` String CODEC(ZSTD(3)),
	`timestamp` Int64 CODEC(Delta(), ZSTD(3)),
	`value` SimpleAggregateFunction(max, Float64) CODEC(Gorilla, ZSTD(3))
)
ENGINE = AggregatingMergeTree()
ORDER BY (id, timestamp)
PARTITION BY intDiv(timestamp,86400000)*86400000 -- 1 day in ms
SETTINGS min_age_to_force_merge_seconds = 3600, min_age_to_force_merge_on_partition_only = 1;

CREATE MATERIALIZED VIEW samples_mv TO samples AS
SELECT id, timestamp, value
FROM samples_null;  

CREATE TABLE series (
	`name` String,
	`labels` Map(String, String),
	`id` String,
	`timestamp` Int64 EPHEMERAL,
	`timestamp_min` SimpleAggregateFunction(min, Int64) DEFAULT `timestamp`,
	`timestamp_max` SimpleAggregateFunction(max, Int64) DEFAULT `timestamp`
)
ENGINE = AggregatingMergeTree()
ORDER BY (name, id)
PARTITION BY intDiv(timestamp_min,86400000)*86400000 -- 1 day in ms
SETTINGS min_age_to_force_merge_seconds = 3600, min_age_to_force_merge_on_partition_only = 1;

CREATE MATERIALIZED VIEW series_mv TO series AS
SELECT name, labels, id, timestamp
FROM samples_null;

-- DOWNSAMPLING TABLE

CREATE TABLE samples_1h (
    `id` String CODEC(ZSTD(3)),
    `timestamp` Int64 CODEC(Delta(), ZSTD(3)),
    `value` Float64,
    `timestamp_key` Int64 
)
ENGINE = ReplacingMergeTree(timestamp)
ORDER BY (id, timestamp_key)
PARTITION BY intDiv(timestamp,864000000)*864000000; -- 10 days

-- downsample to one sample per hour
CREATE MATERIALIZED VIEW samples_1h_mv TO samples_1h AS
SELECT id, timestamp, value,  intDiv(timestamp,3600000)*3600000 as timestamp_key
FROM samples_null;