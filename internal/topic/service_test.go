package topic

import (
	"testing"

	"github.com/eventflow/eventflow/pkg/models"
)

func TestValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     models.CreateTopicRequest
		wantErr bool
	}{
		{"valid", models.CreateTopicRequest{Name: "orders", Partitions: 6, CleanupPolicy: "delete"}, false},
		{"empty name", models.CreateTopicRequest{Name: ""}, true},
		{"invalid chars", models.CreateTopicRequest{Name: "bad topic!"}, true},
		{"bad cleanup", models.CreateTopicRequest{Name: "ok", CleanupPolicy: "invalid"}, true},
		{"compact policy", models.CreateTopicRequest{Name: "changelog", CleanupPolicy: "compact"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateCreateRequest() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
