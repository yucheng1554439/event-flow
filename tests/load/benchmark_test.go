//go:build load

package load

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/eventflow/eventflow/pkg/models"
)

func BenchmarkPublishRequestMarshal(b *testing.B) {
	payload, _ := json.Marshal(map[string]any{"userId": 123, "amount": 45.99})
	req := models.PublishRequest{
		Topic: "orders", EventType: "OrderCreated",
		IdempotencyKey: "bench-key", Payload: payload,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
	_ = context.Background()
}
