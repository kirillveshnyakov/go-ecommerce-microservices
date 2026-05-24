package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	"github.com/segmentio/kafka-go"
)

type publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string, topic string) *publisher {
	return &publisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *publisher) Close() error {
	return p.writer.Close()
}

func (p *publisher) SendOrderStatusChangedNotification(ctx context.Context, message port.Notification) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("notifications kafka publisher - message json marshal: message=%v: %w", message, err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   createKey(message.OrderID),
		Value: raw,
	})
	if err != nil {
		return fmt.Errorf("notifications kafka publisher - send message: message=%v: %w", message, err)
	}

	return nil
}

func createKey(orderID int64) []byte {
	return []byte(strconv.FormatInt(orderID, 10))
}
