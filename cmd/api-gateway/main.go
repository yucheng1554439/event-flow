package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eventflow/eventflow/internal/api"
	grpcserver "github.com/eventflow/eventflow/internal/grpc"
	"github.com/eventflow/eventflow/internal/replay"
	"github.com/eventflow/eventflow/internal/retry"
	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/internal/topic"
	"github.com/eventflow/eventflow/internal/workflow"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load("api-gateway")
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

	producer, err := kafkapkg.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		log.Fatal("kafka producer", zap.Error(err))
	}
	defer producer.Close()

	admin, err := kafkapkg.NewAdmin(cfg.KafkaBrokers)
	if err != nil {
		log.Fatal("kafka admin", zap.Error(err))
	}
	defer admin.Close()

	topicRepo := topic.NewRepository(store)
	topicSvc := topic.NewService(topicRepo, admin, log)
	topicHandler := topic.NewHandler(topicSvc)

	retryEngine := retry.NewEngine(store, config.DefaultRetryPolicy(), log)
	replaySvc := replay.NewService(store, producer, log)
	workflowEngine := workflow.NewEngine(store, cache, log)

	handler := api.NewHandler(store, cache, producer, retryEngine, replaySvc, workflowEngine, topicHandler, log)

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	handler.RegisterRoutes(r)

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	grpcSrv := grpcserver.NewServer(topicSvc, store, cache, producer, replaySvc, workflowEngine, log)

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server", zap.Error(err))
		}
	}()
	go func() {
		if err := grpcSrv.Start(cfg.GRPCPort); err != nil {
			log.Fatal("grpc server", zap.Error(err))
		}
	}()

	log.Info("api-gateway started",
		zap.Int("http", cfg.HTTPPort),
		zap.Int("grpc", cfg.GRPCPort),
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	grpcSrv.Stop()
}
