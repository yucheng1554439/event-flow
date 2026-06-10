package grpcserver

import (
	"encoding/json"
	"time"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/pkg/models"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func topicToProto(t *models.Topic) *eventflowv1.Topic {
	if t == nil {
		return nil
	}
	return &eventflowv1.Topic{
		Id:                t.ID,
		Name:              t.Name,
		Partitions:        int32(t.Partitions),
		ReplicationFactor: int32(t.Replication),
		RetentionHours:    int32(t.RetentionHours),
		CleanupPolicy:     t.CleanupPolicy,
		Compression:       t.Compression,
		CreatedAt:         timestamppb.New(t.CreatedAt),
	}
}

func eventToProto(e *models.Event) *eventflowv1.Event {
	if e == nil {
		return nil
	}
	payload, _ := structpb.NewStruct(map[string]any{})
	if len(e.Payload) > 0 {
		var m map[string]any
		_ = json.Unmarshal(e.Payload, &m)
		payload, _ = structpb.NewStruct(m)
	}
	return &eventflowv1.Event{
		Id:             e.ID,
		Topic:          e.Topic,
		Partition:      int32(e.Partition),
		Offset:         e.Offset,
		EventType:      e.EventType,
		IdempotencyKey: e.IdempotencyKey,
		Payload:        payload,
		PublishedAt:    timestamppb.New(e.PublishedAt),
	}
}

func structToRaw(s *structpb.Struct) json.RawMessage {
	if s == nil {
		return json.RawMessage("{}")
	}
	b, _ := json.Marshal(s.AsMap())
	return b
}

func workflowToProto(w *models.Workflow) *eventflowv1.Workflow {
	if w == nil {
		return nil
	}
	input, _ := structpb.NewStruct(map[string]any{})
	if len(w.Input) > 0 {
		var m map[string]any
		_ = json.Unmarshal(w.Input, &m)
		input, _ = structpb.NewStruct(m)
	}
	return &eventflowv1.Workflow{
		Id:          w.ID,
		Name:        w.Name,
		Status:      string(w.Status),
		Input:       input,
		CurrentStep: w.CurrentStep,
		CreatedAt:   timestamppb.New(w.CreatedAt),
		UpdatedAt:   timestamppb.New(w.UpdatedAt),
	}
}

func stepsToProto(steps []models.WorkflowStep) []*eventflowv1.WorkflowStep {
	out := make([]*eventflowv1.WorkflowStep, 0, len(steps))
	for _, s := range steps {
		out = append(out, &eventflowv1.WorkflowStep{
			Id:     s.ID,
			Name:   s.Name,
			Status: s.Status,
			Error:  s.Error,
		})
	}
	return out
}

func timeFromProto(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
