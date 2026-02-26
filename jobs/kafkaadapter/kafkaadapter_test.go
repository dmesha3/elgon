//go:build adapters
// +build adapters

package kafkaadapter

import "testing"

func TestProducerConfigValidation(t *testing.T) {
	_, err := NewProducer(ProducerConfig{})
	if err == nil {
		t.Fatal("expected error for missing brokers")
	}
}

func TestConsumerConfigValidation(t *testing.T) {
	_, err := NewConsumer(ConsumerConfig{Brokers: []string{"127.0.0.1:9092"}})
	if err == nil {
		t.Fatal("expected error for missing topic")
	}
}
