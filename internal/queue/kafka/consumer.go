package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(broker string, topic string, groupId string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: groupId,
	})
	return &Consumer{reader: r}
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, err
	}
	return msg, nil
}

func (c *Consumer) CommitMessage(ctx context.Context, msg kafka.Message) error {
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}
	return nil
}
