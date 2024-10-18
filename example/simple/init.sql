-- table for inserts
CREATE TABLE samples_null (
	`id` String,
	`name` String,
	`labels` Map(String, String),
	`timestamp` Int64,
	`value` Float64,
)
ENGINE = Null();

CREATE TABLE samples (
	`id` String,
	`timestamp` Int64,
	`value` Float64,
)
ENGINE = ReplacingMergeTree()
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
ORDER BY (name, id, labels)
PARTITION BY intDiv(timestamp_min,86400000)*86400000 -- 1 day in ms
SETTINGS min_age_to_force_merge_seconds = 3600, min_age_to_force_merge_on_partition_only = 1;

CREATE MATERIALIZED VIEW series_mv TO series AS
SELECT name, labels, id, timestamp
FROM samples_null;
