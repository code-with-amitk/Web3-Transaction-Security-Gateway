package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/amit/Web3-Transaction-Security-Gateway/internal/policy"
	"github.com/segmentio/kafka-go"
)

// Publisher sends audit events to Kafka (Redpanda-compatible).
type Publisher struct {
	writer *kafka.Writer
	topic  string
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{
		topic: topic,
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, event policy.AuditEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ID),
		Value: body,
		Time:  time.Now().UTC(),
	})
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}

// NoopPublisher logs events when Kafka is disabled (local quick-start).
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, event policy.AuditEvent) error {
	slog.Info("audit event (kafka disabled)", "id", event.ID, "status", event.Status, "action", event.Decision.Action)
	return nil
}

func (NoopPublisher) Close() error { return nil }
