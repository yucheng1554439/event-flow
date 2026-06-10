package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/eventflow/eventflow/internal/retry"
	"github.com/eventflow/eventflow/internal/storage"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/config"
	"github.com/eventflow/eventflow/pkg/metrics"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("consumer-worker")
	log, _ := zap.NewProduction()
	defer log.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := storage.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("postgres", zap.Error(err))
	}
	defer store.Close()

	retryEngine := retry.NewEngine(store, config.DefaultRetryPolicy(), log)

	groupID := env("CONSUMER_GROUP", "eventflow-workers")
	topic := env("CONSUMER_TOPIC", "orders")
	workflowAPI := env("WORKFLOW_API_URL", "http://localhost:8080")

	handler := func(ctx context.Context, msg *kafka.Message) error {
		eventType := headerValue(msg.Headers, "event_type")
		eventID := headerValue(msg.Headers, "event_id")

		if err := processEvent(msg.Value); err != nil {
			metrics.EventsFailedTotal.WithLabelValues(topic, groupID, err.Error()).Inc()
			attempt, _ := store.GetMaxRetryAttempt(ctx, eventID)
			log.Warn("event processing failed",
				zap.String("eventId", eventID),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			if _, schedErr := retryEngine.Schedule(ctx, eventID, topic, err.Error(), attempt); schedErr != nil {
				log.Info("retry/DLQ outcome", zap.String("eventId", eventID), zap.Error(schedErr))
			}
			return err
		}

		_ = store.UpsertOffset(ctx, models.ConsumerOffset{
			GroupID: groupID, Topic: topic,
			Partition: int(msg.TopicPartition.Partition),
			Offset:    int64(msg.TopicPartition.Offset),
		})
		metrics.EventsProcessedTotal.WithLabelValues(topic, groupID).Inc()
		metrics.ConsumerLag.WithLabelValues(topic, groupID, fmt.Sprintf("%d", msg.TopicPartition.Partition)).Set(0)

		if eventType == "ShipPurchased" {
			triggerGalacticWorkflow(ctx, workflowAPI, msg.Value, log)
		}

		log.Info("processed event", zap.String("type", eventType), zap.String("id", eventID))
		return nil
	}

	consumer, err := kafkapkg.NewConsumerGroup(cfg.KafkaBrokers, groupID, topic, handler)
	if err != nil {
		log.Fatal("consumer", zap.Error(err))
	}
	consumer.Start(ctx)
	defer consumer.Stop()

	retryInterval := 10 * time.Second
	if os.Getenv("DEMO_MODE") == "true" {
		retryInterval = 2 * time.Second
	}
	go func() {
		ticker := time.NewTicker(retryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = retryEngine.ProcessPending(ctx, func(ctx context.Context, r models.RetryRecord) error {
					log.Info("retry attempt", zap.String("eventId", r.EventID), zap.Int("attempt", r.Attempt))
					return fmt.Errorf("still failing")
				})
			}
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	http.Handle("/metrics", promhttp.Handler())
	go func() { _ = http.ListenAndServe(":8082", nil) }()

	log.Info("consumer-worker started", zap.String("group", groupID), zap.String("topic", topic))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
}

func processEvent(payload []byte) error {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return err
	}
	if v, ok := data["simulateFailure"]; ok {
		if b, ok := v.(bool); ok && b {
			return fmt.Errorf("simulated galactic commerce failure")
		}
	}
	return nil
}

func triggerGalacticWorkflow(ctx context.Context, apiURL string, payload []byte, log *zap.Logger) {
	var input map[string]any
	if json.Unmarshal(payload, &input) != nil {
		input = map[string]any{}
	}
	body, _ := json.Marshal(map[string]any{
		"name":  "GalacticCommerce",
		"input": input,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/api/v1/workflows", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warn("workflow create failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return
	}
	var w struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&w)
	if w.ID == "" {
		return
	}
	runReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/api/v1/workflows/"+w.ID+"/run", nil)
	runResp, err := http.DefaultClient.Do(runReq)
	if err == nil {
		runResp.Body.Close()
		log.Info("galactic workflow started", zap.String("workflowId", w.ID))
	}
}

func headerValue(headers []kafka.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
