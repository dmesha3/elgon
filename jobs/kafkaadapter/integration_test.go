//go:build adapters && integration
// +build adapters,integration

package kafkaadapter

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestKafkaAdapterIntegration(t *testing.T) {
	broker := os.Getenv("ELGON_KAFKA_BROKER")
	if broker == "" {
		broker = "127.0.0.1:9092"
	}
	topic := fmt.Sprintf("elgon-itest-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("elgon-itest-group-%d", time.Now().UnixNano())

	producer, err := NewProducer(ProducerConfig{Brokers: []string{broker}})
	if err != nil {
		t.Fatalf("new producer: %v", err)
	}
	defer producer.Close()

	consumer, err := NewConsumer(ConsumerConfig{Brokers: []string{broker}, Topic: topic, GroupID: groupID})
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Publish once; many Kafka setups auto-create topic on first write.
	if err := producer.Publish(ctx, topic, []byte("k"), []byte(`{"name":"job","payload":"eA"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for kafka message")
		}
		msg, err := consumer.Poll(ctx, 2*time.Second)
		if err != nil {
			continue
		}
		if len(msg) == 0 {
			continue
		}
		return
	}
}
