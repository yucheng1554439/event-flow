package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type MessageHandler func(ctx context.Context, msg *kafka.Message) error

type ConsumerGroup struct {
	consumer *kafka.Consumer
	groupID  string
	topic    string
	handler  MessageHandler
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

func NewConsumerGroup(brokers []string, groupID, topic string, handler MessageHandler) (*ConsumerGroup, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  joinBrokers(brokers),
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
		"session.timeout.ms": 10000,
		"max.poll.interval.ms": 300000,
	})
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}
	if err := c.Subscribe(topic, nil); err != nil {
		c.Close()
		return nil, fmt.Errorf("subscribe topic %s: %w", topic, err)
	}
	return &ConsumerGroup{
		consumer: c,
		groupID:  groupID,
		topic:    topic,
		handler:  handler,
	}, nil
}

func (cg *ConsumerGroup) Start(ctx context.Context) {
	ctx, cg.cancel = context.WithCancel(ctx)
	cg.wg.Add(1)
	go cg.pollLoop(ctx)
}

func (cg *ConsumerGroup) Stop() {
	if cg.cancel != nil {
		cg.cancel()
	}
	cg.wg.Wait()
	cg.consumer.Close()
}

func (cg *ConsumerGroup) pollLoop(ctx context.Context) {
	defer cg.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ev := cg.consumer.Poll(100)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				if err := cg.handler(ctx, e); err != nil {
					continue // at-least-once: do not commit on failure
				}
				_, _ = cg.consumer.CommitMessage(e)
			case kafka.Error:
				if e.IsFatal() {
					return
				}
			}
		}
	}
}

func (cg *ConsumerGroup) GroupID() string { return cg.groupID }
func (cg *ConsumerGroup) Topic() string   { return cg.topic }
