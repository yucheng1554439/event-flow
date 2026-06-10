package topic

import (
	"context"

	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
	"time"
)

type Repository struct {
	store *storage.PostgresStore
}

func NewRepository(store *storage.PostgresStore) *Repository {
	return &Repository{store: store}
}

func (r *Repository) Create(ctx context.Context, t models.Topic) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	return r.store.CreateTopic(ctx, t)
}

func (r *Repository) Get(ctx context.Context, name string) (*models.Topic, error) {
	return r.store.GetTopic(ctx, name)
}

func (r *Repository) List(ctx context.Context) ([]models.Topic, error) {
	return r.store.ListTopics(ctx)
}

func (r *Repository) Delete(ctx context.Context, name string) error {
	return r.store.DeleteTopic(ctx, name)
}
