package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/internal/workflow"
	"github.com/eventflow/eventflow/pkg/config"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("workflow-engine")
	log, _ := zap.NewProduction()
	defer log.Sync()

	ctx := context.Background()
	store, err := storage.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("postgres", zap.Error(err))
	}
	defer store.Close()

	cache, err := storage.NewRedisCache(cfg.RedisURL)
	if err != nil {
		log.Fatal("redis", zap.Error(err))
	}
	defer cache.Close()

	engine := workflow.NewEngine(store, cache, log)

	// Poll for pending workflows and execute them.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// In production: query workflows WHERE status = 'pending'
			log.Debug("workflow engine tick")
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		_ = http.ListenAndServe(":8081", nil)
	}()

	// Demo: auto-run OrderFulfillment when triggered via env
	if demo := os.Getenv("DEMO_WORKFLOW"); demo == "true" {
		input, _ := json.Marshal(map[string]any{"orderId": "ord-001", "amount": 45.99})
		w, _ := engine.Create(ctx, models.CreateWorkflowRequest{Name: "OrderFulfillment", Input: input})
		if w != nil {
			_ = engine.Run(ctx, w.ID)
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("workflow-engine shutting down")
}
