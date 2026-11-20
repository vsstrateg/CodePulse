package kafka

import (
	"codepulse/internal/domain"
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(broker string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &Producer{writer: w}
}

func (p *Producer) Publish(ctx context.Context, event domain.SubmissionEvent) error {
	ev, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.SubmissionID),
		Value: ev,
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func EnsureTopic(broker string, topic string, partitions int) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close kafka conn", "conn", conn, "error", err)
		}
	}()
	numPart, err := conn.ReadPartitions(topic)
	if err == nil && len(numPart) > 0 {
		return nil
	}

	topicConfig := kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     partitions,
		ReplicationFactor: 1,
		ConfigEntries: []kafka.ConfigEntry{
			{
				ConfigName:  "retention.ms",
				ConfigValue: "86400000",
			},
		},
	}
	return conn.CreateTopics(topicConfig)
}
