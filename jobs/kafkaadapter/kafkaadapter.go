//go:build adapters
// +build adapters

package kafkaadapter

import (
	"context"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
)

// ProducerConfig configures kafka producer settings.
type ProducerConfig struct {
	Brokers []string
	Dialer  *kafka.Dialer
}

// ConsumerConfig configures kafka consumer settings.
type ConsumerConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	MinBytes       int
	MaxBytes       int
	CommitInterval time.Duration
	Dialer         *kafka.Dialer
}

// Producer adapts kafka-go writer to jobs.KafkaProducer.
type Producer struct {
	writer *kafka.Writer
}

// Consumer adapts kafka-go reader to jobs.KafkaConsumer.
type Consumer struct {
	reader *kafka.Reader
}

func NewProducer(cfg ProducerConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafkaadapter: at least one broker is required")
	}
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Brokers,
		Balancer: &kafka.LeastBytes{},
		Dialer:   cfg.Dialer,
	})
	return &Producer{writer: writer}, nil
}

func (p *Producer) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	if topic == "" {
		return errors.New("kafkaadapter: topic is required")
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: key, Value: value, Time: time.Now()})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func NewConsumer(cfg ConsumerConfig) (*Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafkaadapter: at least one broker is required")
	}
	if cfg.Topic == "" {
		return nil, errors.New("kafkaadapter: topic is required")
	}
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 1
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10e6
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		CommitInterval: cfg.CommitInterval,
		Dialer:         cfg.Dialer,
	})
	return &Consumer{reader: reader}, nil
}

func (c *Consumer) Poll(ctx context.Context, timeout time.Duration) ([]byte, error) {
	readCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		readCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	msg, err := c.reader.ReadMessage(readCtx)
	if err != nil {
		return nil, err
	}
	return msg.Value, nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
