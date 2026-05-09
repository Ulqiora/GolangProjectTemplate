CREATE TABLE IF NOT EXISTS users_created_kafka
(
    user_id String,
    provider String,
    occurred_at DateTime64(3, 'UTC')
)
ENGINE = Kafka()
SETTINGS
    kafka_broker_list = 'kafka:29092',
    kafka_topic_list = 'users.created',
    kafka_group_name = 'clickhouse-users-created',
    kafka_format = 'AvroConfluent',
    format_avro_schema_registry_url = 'http://schema-registry:8081',
    kafka_num_consumers = 1;

CREATE TABLE IF NOT EXISTS users_created
(
    user_id String,
    provider LowCardinality(String),
    occurred_at DateTime64(3, 'UTC'),
    inserted_at DateTime DEFAULT now()
)
ENGINE = MergeTree()
ORDER BY (occurred_at, user_id);

CREATE MATERIALIZED VIEW IF NOT EXISTS users_created_mv
TO users_created
AS
SELECT
    user_id,
    provider,
    occurred_at
FROM users_created_kafka;
