package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ConsumeOne(ctx context.Context, brokers []string, topic, groupID string) (*kafka.Message, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  joinBrokers(brokers),
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, err
	}
	defer consumer.Close()

	if err := consumer.Subscribe(topic, nil); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			ev := consumer.Poll(500)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				return e, nil
			case kafka.Error:
				if e.IsFatal() {
					return nil, e
				}
			}
		}
	}
	return nil, fmt.Errorf("no message received on topic %s", topic)
}
