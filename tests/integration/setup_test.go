//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eventflow/eventflow/internal/storage"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testEnv struct {
	Brokers     []string
	DatabaseURL string
	RedisURL    string
	Store       *storage.PostgresStore
	Cache       *storage.RedisCache
	Producer    *kafkapkg.Producer
	Admin       *kafkapkg.Admin
}

var sharedEnv *testEnv

func TestMain(m *testing.M) {
	ctx := context.Background()
	env, cleanup, err := initSharedEnv(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration setup failed: %v\n", err)
		os.Exit(1)
	}
	sharedEnv = env
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func initSharedEnv(ctx context.Context) (*testEnv, func(), error) {
	pg, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("eventflow"),
		postgres.WithUsername("eventflow"),
		postgres.WithPassword("eventflow"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres container: %w", err)
	}

	rd, err := redis.RunContainer(ctx)
	if err != nil {
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("redis container: %w", err)
	}

	kc, err := kafka.RunContainer(ctx)
	if err != nil {
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("kafka container: %w", err)
	}

	brokers, err := kc.Brokers(ctx)
	if err != nil {
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("kafka brokers: %w", err)
	}

	pgConn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("postgres conn: %w", err)
	}
	redisConn, err := rd.ConnectionString(ctx)
	if err != nil {
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("redis conn: %w", err)
	}

	if err := runMigrations(pgConn); err != nil {
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, err
	}

	store, err := storage.NewPostgresStore(ctx, pgConn)
	if err != nil {
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("postgres store: %w", err)
	}

	cache, err := storage.NewRedisCache(redisConn)
	if err != nil {
		store.Close()
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("redis cache: %w", err)
	}

	producer, err := kafkapkg.NewProducer(brokers)
	if err != nil {
		_ = cache.Close()
		store.Close()
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("kafka producer: %w", err)
	}

	admin, err := kafkapkg.NewAdmin(brokers)
	if err != nil {
		producer.Close()
		_ = cache.Close()
		store.Close()
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
		return nil, nil, fmt.Errorf("kafka admin: %w", err)
	}

	// Allow Kafka broker to stabilize before running tests.
	time.Sleep(5 * time.Second)

	cleanup := func() {
		admin.Close()
		producer.Close()
		_ = cache.Close()
		store.Close()
		_ = kc.Terminate(ctx)
		_ = rd.Terminate(ctx)
		_ = pg.Terminate(ctx)
	}

	return &testEnv{
		Brokers: brokers, DatabaseURL: pgConn, RedisURL: redisConn,
		Store: store, Cache: cache, Producer: producer, Admin: admin,
	}, cleanup, nil
}

func setupEnv(t *testing.T) *testEnv {
	t.Helper()
	if sharedEnv == nil {
		t.Fatal("shared test environment not initialized")
	}
	return sharedEnv
}

func runMigrations(databaseURL string) error {
	ctx := context.Background()
	store, err := storage.NewPostgresStore(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("migration connect: %w", err)
	}
	defer store.Close()

	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	for _, file := range []string{"001_initial_schema.sql", "002_add_cleanup_policy.sql"} {
		path := filepath.Join(root, "migrations", file)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		if _, err := store.Pool().Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("exec migration %s: %w", file, err)
		}
	}
	return nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func uniqueTopic(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
