# jobs Module

Background jobs and distributed queue contracts.

## Provides

- In-memory queue (`NewInMemoryQueue`)
- SQL distributed queue backend (`NewSQLBackend`)
- Redis queue abstraction (`RedisQueue`)
- Kafka queue abstraction (`KafkaQueue`)
- Interval scheduler (`Scheduler`, `@every <duration>`)

## Optional concrete adapters

- `jobs/redisadapter` (`go-redis`) with build tag `adapters`
- `jobs/kafkaadapter` (`segmentio/kafka-go`) with build tag `adapters`

## Primary API

- `type Queue interface { Enqueue; RunWorker; Close }`
- `func NewInMemoryQueue(buffer int) *InMemoryQueue`
- `func NewSQLBackend(adapter db.Adapter, cfg SQLBackendConfig) *SQLBackend`
