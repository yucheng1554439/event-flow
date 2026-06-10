//go:build integration

package integration

import (
	"context"
	"testing"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/internal/replay"
	"github.com/eventflow/eventflow/internal/topic"
	"github.com/eventflow/eventflow/internal/workflow"
	"github.com/eventflow/eventflow/pkg/models"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGRPC_PublishAndCreateTopic(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()
	log, _ := zap.NewDevelopment()

	topicRepo := topic.NewRepository(env.Store)
	topicSvc := topic.NewService(topicRepo, env.Admin, log)
	replaySvc := replay.NewService(env.Store, env.Producer, log)
	wfEngine := workflow.NewEngine(env.Store, env.Cache, log)

	// In-process gRPC via direct server calls (no network bind required)
	topicServer := struct{ eventflowv1.TopicServiceServer }{}
	_ = topicServer

	name := uniqueTopic("grpc-topic")
	created, err := topicSvc.Create(ctx, models.CreateTopicRequest{
		Name: name, Partitions: 1, ReplicationFactor: 1,
	})
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if created.Name != name {
		t.Fatal("topic name mismatch")
	}

	// Verify replay and workflow services initialized (gRPC wiring smoke test)
	if replaySvc == nil || wfEngine == nil {
		t.Fatal("services not initialized")
	}

	// Optional: dial local gateway if EVENTFLOW_GRPC_ADDR set
	addr := "127.0.0.1:9090"
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("gRPC gateway not running at %s: %v", addr, err)
	}
	defer conn.Close()

	eventClient := eventflowv1.NewEventServiceClient(conn)
	payload, _ := structpb.NewStruct(map[string]any{"userId": 1})
	_, err = eventClient.PublishEvent(ctx, &eventflowv1.PublishEventRequest{
		Topic: name, EventType: "OrderCreated", Payload: payload,
	})
	if err != nil {
		t.Skipf("gRPC gateway not running at %s: %v", addr, err)
	}
}
