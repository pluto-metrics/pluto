-- table for inserts
CREATE DATABASE shard2;

CREATE TABLE shard2.samples_null (
	`id` String,
	`name` String,
	`labels` Map(String, String),
	`timestamp` Int64,
	`value` Float64
)
ENGINE = Null();

CREATE TABLE shard2.samples (
	`id` String CODEC(ZSTD(3)),
	`timestamp` Int64 CODEC(Delta(), ZSTD(3)),
	`value` SimpleAggregateFunction(max, Float64) CODEC(Gorilla, ZSTD(3))
)
ENGINE = AggregatingMergeTree()
ORDER BY (id, timestamp)
PARTITION BY intDiv(timestamp,86400000)*86400000 -- 1 day in ms
SETTINGS min_age_to_force_merge_seconds = 3600, min_age_to_force_merge_on_partition_only = 1;

CREATE MATERIALIZED VIEW shard2.samples_mv TO shard2.samples AS
SELECT id, timestamp, value
FROM shard2.samples_null;  

CREATE TABLE shard2.series (
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

CREATE MATERIALIZED VIEW shard2.series_mv TO shard2.series AS
SELECT name, labels, id, timestamp
FROM shard2.samples_null;
