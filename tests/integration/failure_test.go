//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/eventflow/eventflow/internal/retry"
	"github.com/eventflow/eventflow/pkg/config"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestFailureInjection_RetryThenDLQ(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	policy := config.RetryPolicy{
		MaxAttempts: 3, InitialBackoff: time.Millisecond,
		MaxBackoff: time.Millisecond, Multiplier: 2,
	}
	engine := retry.NewEngine(env.Store, policy, log)
	eventID := uuid.New().String()
	topic := uniqueTopic("chaos")

	_, _ = engine.Schedule(ctx, eventID, topic, "chaos injection", policy.MaxAttempts)

	count, err := env.Store.CountDLQ(ctx, topic)
	if err != nil || count == 0 {
		t.Fatalf("expected DLQ message after chaos, count=%d err=%v", count, err)
	}
}

func TestFailureInjection_RedisLockExpiry(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	wfID := "chaos-wf-" + uuid.New().String()
	ok, err := env.Cache.LockWorkflow(ctx, wfID, 100*time.Millisecond)
	if err != nil || !ok {
		t.Fatalf("lock failed")
	}
	time.Sleep(150 * time.Millisecond)
	ok, err = env.Cache.LockWorkflow(ctx, wfID, time.Minute)
	if err != nil || !ok {
		t.Fatalf("expected lock after TTL expiry")
	}
}
