package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrKafkaClientNil = errors.New("jobs: kafka client is nil")

// KafkaProducer is the minimal producer contract.
type KafkaProducer interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
	Close() error
}

// KafkaConsumer is the minimal consumer contract.
type KafkaConsumer interface {
	Poll(ctx context.Context, timeout time.Duration) ([]byte, error)
	Close() error
}

// KafkaQueueConfig configures Kafka queue behavior.
type KafkaQueueConfig struct {
	Topic       string
	PollTimeout time.Duration
}

// KafkaQueue is a distributed queue backed by Kafka topic publish/poll.
type KafkaQueue struct {
	producer KafkaProducer
	consumer KafkaConsumer
	cfg      KafkaQueueConfig
}

func NewKafkaQueue(producer KafkaProducer, consumer KafkaConsumer, cfg KafkaQueueConfig) *KafkaQueue {
	if cfg.Topic == "" {
		cfg.Topic = "elgon.jobs"
	}
	if cfg.PollTimeout <= 0 {
		cfg.PollTimeout = time.Second
	}
	return &KafkaQueue{producer: producer, consumer: consumer, cfg: cfg}
}

func (q *KafkaQueue) Enqueue(ctx context.Context, msg Message) error {
	if q.producer == nil {
		return ErrKafkaClientNil
	}
	value, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.producer.Publish(ctx, q.cfg.Topic, []byte(msg.Name), value)
}

func (q *KafkaQueue) RunWorker(ctx context.Context, handler Handler) {
	if q.consumer == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		payload, err := q.consumer.Poll(ctx, q.cfg.PollTimeout)
		if err != nil {
			continue
		}
		var msg Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}
		_ = handler(ctx, msg)
	}
}

func (q *KafkaQueue) Close() {
	if q.producer != nil {
		_ = q.producer.Close()
	}
	if q.consumer != nil {
		_ = q.consumer.Close()
	}
}
