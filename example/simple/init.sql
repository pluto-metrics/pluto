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
PARTITION BY timestamp-timestamp%86400000; -- 1 day in ms

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
PARTITION BY timestamp_min-timestamp_min%86400000; -- 1 day in ms

CREATE MATERIALIZED VIEW series_mv TO series AS
SELECT name, labels, id, timestamp
FROM samples_null;
