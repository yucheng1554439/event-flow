//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/eventflow/eventflow/internal/replay"
	"github.com/eventflow/eventflow/internal/retry"
	"github.com/eventflow/eventflow/internal/topic"
	"github.com/eventflow/eventflow/internal/workflow"
	"github.com/eventflow/eventflow/pkg/config"
	"github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestTopicAdmin_CreateListDelete(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	repo := topic.NewRepository(env.Store)
	svc := topic.NewService(repo, env.Admin, log)

	name := uniqueTopic("fraud-events")
	created, err := svc.Create(ctx, models.CreateTopicRequest{
		Name: name, Partitions: 3, ReplicationFactor: 1,
		RetentionHours: 168, CleanupPolicy: "delete",
	})
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if created.Partitions != 3 {
		t.Fatalf("expected 3 partitions, got %d", created.Partitions)
	}

	got, err := svc.Get(ctx, name)
	if err != nil {
		t.Fatalf("get topic: %v", err)
	}
	if got.Name != name {
		t.Fatalf("topic name mismatch")
	}

	topics, err := svc.List(ctx)
	if err != nil || len(topics) == 0 {
		t.Fatalf("list topics: %v len=%d", err, len(topics))
	}

	if err := svc.Delete(ctx, name); err != nil {
		t.Fatalf("delete topic: %v", err)
	}
	if _, err := svc.Get(ctx, name); err == nil {
		t.Fatal("expected topic not found after delete")
	}
}

func TestProducer_PublishAndPersist(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	topicName := uniqueTopic("orders")
	repo := topic.NewRepository(env.Store)
	svc := topic.NewService(repo, env.Admin, log)
	_, err := svc.Create(ctx, models.CreateTopicRequest{
		Name: topicName, Partitions: 1, ReplicationFactor: 1,
	})
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}

	key := "idempotency-" + uuid.New().String()
	payload, _ := json.Marshal(map[string]any{"userId": 123, "amount": 45.99})
	event, err := env.Producer.Publish(ctx, models.PublishRequest{
		Topic: topicName, EventType: "OrderCreated",
		IdempotencyKey: key, Payload: payload,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	_ = env.Store.StoreEvent(ctx, *event)

	msg, err := kafka.ConsumeOne(ctx, env.Brokers, topicName, "test-producer-"+uuid.New().String())
	if err != nil {
		t.Fatalf("consume kafka: %v", err)
	}
	if string(msg.Value) == "" {
		t.Fatal("empty kafka message")
	}

	stored, err := env.Store.GetEventByIdempotencyKey(ctx, key)
	if err != nil {
		t.Fatalf("event not in postgres: %v", err)
	}
	if stored.EventType != "OrderCreated" {
		t.Fatalf("wrong event type: %s", stored.EventType)
	}
}

func TestProducer_Idempotency(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	dup, err := env.Cache.CheckIdempotency(ctx, "dup-key-1", time.Hour)
	if err != nil || dup {
		t.Fatalf("first idempotency check should be false: dup=%v err=%v", dup, err)
	}
	dup, err = env.Cache.CheckIdempotency(ctx, "dup-key-1", time.Hour)
	if err != nil || !dup {
		t.Fatalf("second idempotency check should be true: dup=%v err=%v", dup, err)
	}
}

func TestConsumer_OffsetCommit(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	topicName := uniqueTopic("consume")
	repo := topic.NewRepository(env.Store)
	svc := topic.NewService(repo, env.Admin, log)
	_, _ = svc.Create(ctx, models.CreateTopicRequest{Name: topicName, Partitions: 1, ReplicationFactor: 1})

	groupID := "test-group-" + uuid.New().String()
	payload, _ := json.Marshal(map[string]string{"event": "test"})
	ev, err := env.Producer.Publish(ctx, models.PublishRequest{
		Topic: topicName, EventType: "TestEvent", Payload: payload,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	err = env.Store.UpsertOffset(ctx, models.ConsumerOffset{
		GroupID: groupID, Topic: topicName,
		Partition: ev.Partition, Offset: ev.Offset,
	})
	if err != nil {
		t.Fatalf("upsert offset: %v", err)
	}

	offsets, err := env.Store.ListOffsets(ctx, groupID)
	if err != nil || len(offsets) == 0 {
		t.Fatalf("offsets not stored: %v", err)
	}
	if offsets[0].Offset != ev.Offset {
		t.Fatalf("offset mismatch: got %d want %d", offsets[0].Offset, ev.Offset)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	policy := config.RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 10 * time.Millisecond, Multiplier: 2}
	engine := retry.NewEngine(env.Store, policy, log)

	eventID := uuid.New().String()
	rec, err := engine.Schedule(ctx, eventID, "orders", "transient error", 0)
	if err != nil {
		t.Fatalf("schedule retry: %v", err)
	}
	if rec.Attempt != 1 {
		t.Fatalf("expected attempt 1, got %d", rec.Attempt)
	}

	count, err := env.Store.CountRetries(ctx, eventID)
	if err != nil || count != 1 {
		t.Fatalf("retry count: %d err=%v", count, err)
	}
}

func TestDLQ_RouteOnMaxRetries(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	policy := config.RetryPolicy{MaxAttempts: 2, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond, Multiplier: 2}
	engine := retry.NewEngine(env.Store, policy, log)

	eventID := uuid.New().String()
	topicName := uniqueTopic("payments")
	_, err := engine.Schedule(ctx, eventID, topicName, "fail", policy.MaxAttempts)
	if err != nil {
		t.Fatalf("expected DLQ routing, got err=%v", err)
	}

	count, err := env.Store.CountDLQ(ctx, topicName)
	if err != nil || count == 0 {
		t.Fatalf("DLQ empty: count=%d err=%v", count, err)
	}
}

func TestReplay_ByTimeRange(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	topicName := uniqueTopic("replay-src")
	targetTopic := uniqueTopic("replay-dst")
	repo := topic.NewRepository(env.Store)
	svc := topic.NewService(repo, env.Admin, log)
	_, _ = svc.Create(ctx, models.CreateTopicRequest{Name: topicName, Partitions: 1, ReplicationFactor: 1})
	_, _ = svc.Create(ctx, models.CreateTopicRequest{Name: targetTopic, Partitions: 1, ReplicationFactor: 1})

	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]string{"order": "1"})
	ev, _ := env.Producer.Publish(ctx, models.PublishRequest{Topic: topicName, EventType: "OrderCreated", Payload: payload})
	ev.PublishedAt = now
	_ = env.Store.StoreEvent(ctx, *ev)

	replaySvc := replay.NewService(env.Store, env.Producer, log)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	count, err := replaySvc.Replay(ctx, models.ReplayRequest{
		Topic: topicName, TargetTopic: targetTopic, StartTime: &start, EndTime: &end,
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 replayed, got %d", count)
	}

	msg, err := kafka.ConsumeOne(ctx, env.Brokers, targetTopic, "replay-group-"+uuid.New().String())
	if err != nil {
		t.Fatalf("consume replayed: %v", err)
	}
	if len(msg.Headers) == 0 {
		t.Fatal("expected replay header")
	}
}

func TestWorkflow_OrderFulfillmentSuccess(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	engine := workflow.NewEngine(env.Store, env.Cache, log)
	input, _ := json.Marshal(map[string]any{"orderId": "ord-1", "amount": 45.99})
	w, err := engine.Create(ctx, models.CreateWorkflowRequest{Name: "OrderFulfillment", Input: input})
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if err := engine.Run(ctx, w.ID); err != nil {
		t.Fatalf("run workflow: %v", err)
	}

	got, steps, err := engine.Get(ctx, w.ID)
	if err != nil {
		t.Fatalf("get workflow: %v", err)
	}
	if got.Status != models.WorkflowCompleted {
		t.Fatalf("status=%s", got.Status)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	names := []string{"ProcessPayment", "ReserveInventory", "SendEmail"}
	for i, n := range names {
		if steps[i].Name != n || steps[i].Status != "completed" {
			t.Fatalf("step %s status=%s", steps[i].Name, steps[i].Status)
		}
	}
}

func TestWorkflow_CompensationOnFailure(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	engine := workflow.NewEngine(env.Store, env.Cache, log)
	engine.SetFailStep("ReserveInventory")
	defer engine.ClearFailStep()

	input, _ := json.Marshal(map[string]any{"orderId": "ord-2"})
	w, _ := engine.Create(ctx, models.CreateWorkflowRequest{Name: "OrderFulfillment", Input: input})
	err := engine.Run(ctx, w.ID)
	if err == nil {
		t.Fatal("expected workflow failure")
	}

	got, steps, _ := engine.Get(ctx, w.ID)
	if got.Status != models.WorkflowFailed {
		t.Fatalf("status=%s want failed", got.Status)
	}
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 steps recorded")
	}
	if steps[0].Status != "completed" || steps[1].Status != "failed" {
		t.Fatalf("unexpected step statuses")
	}
}

func TestWorkflow_LockPreventsDuplicateRun(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	locked, err := env.Cache.LockWorkflow(ctx, "wf-lock-test", time.Minute)
	if err != nil || !locked {
		t.Fatalf("first lock failed")
	}
	locked, err = env.Cache.LockWorkflow(ctx, "wf-lock-test", time.Minute)
	if err != nil || locked {
		t.Fatalf("second lock should fail")
	}
	_ = env.Cache.UnlockWorkflow(ctx, "wf-lock-test")
}
