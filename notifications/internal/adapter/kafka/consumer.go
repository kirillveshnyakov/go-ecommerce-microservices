package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	"go.uber.org/zap"
)

//go:generate mockgen -source=consumer.go -destination=mocks/consumer_mocks.go -package=mocks
type (
	inboxRepository interface {
		AddMessage(
			ctx context.Context,
			idempotencyKey string,
			message []byte,
			kafkaTopic string,
			kafkaPartition int32,
			kafkaOffset int64,
		) error
		AddDeadMessage(
			ctx context.Context,
			idempotencyKey string,
			message []byte,
			kafkaTopic string,
			kafkaPartition int32,
			kafkaOffset int64,
			messageErr error,
		) error
	}
)

const kafkaSessionTimeoutMs = 10000

func RunConsumer(
	ctx context.Context,
	brokers []string,
	topic string,
	consumerGroup string,
	inboxRepository inboxRepository,
	logger *zap.Logger,
) error {
	if len(brokers) == 0 {
		logger.Error("no kafka brokers", zap.String("topic", topic))
		return fmt.Errorf("no kafka brokers: topic=%s", topic)
	}

	consumer, err := ckafka.NewConsumer(&ckafka.ConfigMap{
		"bootstrap.servers":        joinBrokers(brokers),
		"group.id":                 consumerGroup,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,
		"session.timeout.ms":       kafkaSessionTimeoutMs,
	})
	if err != nil {
		return fmt.Errorf("create kafka consumer: topic=%s, consumer_group=%s: %w", topic, consumerGroup, err)
	}

	defer func() {
		if err = consumer.Close(); err != nil {
			logger.Error("close kafka consumer", zap.Error(err))
		}
	}()

	if err = consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		return fmt.Errorf("kafka consumer - subscribe topic: topic=%s: %w", topic, err)
	}

	logger.Info("kafka consumer started",
		zap.Strings("brokers", brokers),
		zap.String("topic", topic),
		zap.String("group", consumerGroup),
	)

	for {
		if err = ctx.Err(); err != nil {
			logger.Info("kafka consumer stopped", zap.Error(err))
			return nil
		}

		// TODO: batch
		timeoutMs := 1000
		ev := consumer.Poll(timeoutMs)
		if ev == nil {
			continue
		}

		switch e := ev.(type) {
		case *ckafka.Message:
			msgTopic := *e.TopicPartition.Topic
			msgPartition := e.TopicPartition.Partition
			msgOffset := int64(e.TopicPartition.Offset)

			logger.Info("message fetched",
				zap.String("topic", msgTopic),
				zap.Int32("partition", msgPartition),
				zap.Int64("offset", msgOffset),
				zap.ByteString("key", e.Key),
				zap.ByteString("value", e.Value),
			)

			var message port.KafkaMessage
			if err = json.Unmarshal(e.Value, &message); err != nil {
				logger.Error("kafka consumer - message unmarshal",
					zap.String("topic", msgTopic),
					zap.Int32("partition", msgPartition),
					zap.Int64("offset", msgOffset),
					zap.ByteString("raw", e.Value),
					zap.Error(err),
				)

				if errAddDeadMessage := inboxRepository.AddDeadMessage(ctx,
					createIdempotencyKeyForDead(msgTopic, msgPartition, msgOffset),
					e.Value, msgTopic, msgPartition, msgOffset, fmt.Errorf("unmarshal message failed: %w", err),
				); errAddDeadMessage != nil {
					logger.Error(
						"kafka consumer - add dead message",
						zap.String("topic", msgTopic),
						zap.Int32("partition", msgPartition),
						zap.Int64("offset", msgOffset),
						zap.ByteString("raw", e.Value),
						zap.Error(errAddDeadMessage),
					)
					continue
				}

				if err = commitMessage(consumer, e, logger); err != nil {
					time.Sleep(time.Second)
				}
				continue
			}

			if err = inboxRepository.AddMessage(ctx,
				createIdempotencyKey(message.OrderID, message.Status),
				e.Value, msgTopic, msgPartition, msgOffset,
			); err != nil {
				logger.Error(
					"kafka consumer - add message",
					zap.Any("message", message),
					zap.Error(err),
				)
				continue
			}

			if err = commitMessage(consumer, e, logger); err != nil {
				time.Sleep(time.Second)
				continue
			}

		case ckafka.Error:
			logger.Warn("kafka poll error",
				zap.String("code", e.Code().String()),
				zap.Error(e),
			)

			time.Sleep(1 * time.Second)

		case ckafka.AssignedPartitions:
			logger.Info("kafka partitions assigned",
				zap.Any("partitions", e.Partitions),
			)

			if err := consumer.Assign(e.Partitions); err != nil {
				logger.Error("kafka assign partitions", zap.Error(err))
			}

		case ckafka.RevokedPartitions:
			logger.Info("kafka partitions revoked",
				zap.Any("partitions", e.Partitions),
			)

			if err := consumer.Unassign(); err != nil {
				logger.Error("kafka unassign partitions", zap.Error(err))
			}
		default:
		}
	}
}

func commitMessage(consumer *ckafka.Consumer, msg *ckafka.Message, logger *zap.Logger) error {
	msgTopic := *msg.TopicPartition.Topic
	msgPartition := msg.TopicPartition.Partition
	msgOffset := int64(msg.TopicPartition.Offset)

	if _, err := consumer.CommitMessage(msg); err != nil {
		logger.Error("kafka commit",
			zap.Error(err),
			zap.String("topic", msgTopic),
			zap.Int32("partition", msgPartition),
			zap.Int64("offset", msgOffset),
		)
		return fmt.Errorf("commit kafka message: topic=%s partition=%d offset=%d: %w",
			msgTopic, msgPartition, msgOffset, err)
	}

	logger.Info("message committed",
		zap.String("topic", msgTopic),
		zap.Int32("partition", msgPartition),
		zap.Int64("offset", msgOffset),
	)

	return nil
}

func createIdempotencyKey(orderID int64, status string) string {
	return strconv.FormatInt(orderID, 10) + "-" + status
}

func createIdempotencyKeyForDead(topic string, partition int32, offset int64) string {
	return fmt.Sprintf("kafka:%s:%d:%d", topic, partition, offset)
}

func joinBrokers(brokers []string) string {
	return strings.Join(brokers, ",")
}
