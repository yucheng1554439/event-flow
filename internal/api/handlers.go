package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/eventflow/eventflow/internal/replay"
	"github.com/eventflow/eventflow/internal/retry"
	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/internal/topic"
	"github.com/eventflow/eventflow/internal/workflow"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"github.com/eventflow/eventflow/pkg/metrics"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	store    *storage.PostgresStore
	cache    *storage.RedisCache
	producer *kafkapkg.Producer
	retry    *retry.Engine
	replay   *replay.Service
	workflow *workflow.Engine
	topic    *topic.Handler
	log      *zap.Logger
}

func NewHandler(
	store *storage.PostgresStore,
	cache *storage.RedisCache,
	producer *kafkapkg.Producer,
	retryEngine *retry.Engine,
	replaySvc *replay.Service,
	workflowEngine *workflow.Engine,
	topicHandler *topic.Handler,
	log *zap.Logger,
) *Handler {
	return &Handler{
		store: store, cache: cache, producer: producer,
		retry: retryEngine, replay: replaySvc, workflow: workflowEngine,
		topic: topicHandler, log: log,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	h.topic.RegisterRoutes(v1)
	{
		v1.POST("/events", h.PublishEvent)
		v1.POST("/events/batch", h.PublishBatch)

		v1.POST("/consumer-groups", h.CreateConsumerGroup)
		v1.GET("/consumer-groups/:id/offsets", h.GetOffsets)

		v1.GET("/dlq/:topic", h.ListDLQ)
		v1.GET("/dlq/:topic/stats", h.DLQStats)
		v1.POST("/dlq/:topic/replay", h.ReplayDLQ)

		v1.GET("/retries", h.ListRetries)

		v1.POST("/replay", h.ReplayEvents)

		v1.POST("/workflows", h.CreateWorkflow)
		v1.POST("/workflows/:id/run", h.RunWorkflow)
		v1.GET("/workflows/:id", h.GetWorkflow)
	}
}

func (h *Handler) PublishEvent(c *gin.Context) {
	var req models.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IdempotencyKey != "" {
		dup, err := h.cache.CheckIdempotency(c.Request.Context(), req.IdempotencyKey, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if dup {
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate idempotency key"})
			return
		}
	}

	event, err := h.producer.Publish(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.store.StoreEvent(c.Request.Context(), *event)
	metrics.EventsPublishedTotal.WithLabelValues(req.Topic).Inc()
	c.JSON(http.StatusAccepted, event)
}

func (h *Handler) PublishBatch(c *gin.Context) {
	var req models.BatchPublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	events, err := h.producer.PublishBatch(c.Request.Context(), req.Events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, e := range events {
		_ = h.store.StoreEvent(c.Request.Context(), *e)
		metrics.EventsPublishedTotal.WithLabelValues(e.Topic).Inc()
	}
	c.JSON(http.StatusAccepted, gin.H{"events": events, "count": len(events)})
}

func (h *Handler) CreateConsumerGroup(c *gin.Context) {
	var req struct {
		Topic string `json:"topic" binding:"required"`
		Name  string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g := models.ConsumerGroup{
		ID: uuid.New().String(), Topic: req.Topic, Name: req.Name,
		Members: 0, CreatedAt: time.Now().UTC(),
	}
	if err := h.store.CreateConsumerGroup(c.Request.Context(), g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, g)
}

func (h *Handler) GetOffsets(c *gin.Context) {
	offsets, err := h.store.ListOffsets(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groupId": c.Param("id"), "offsets": offsets})
}

func (h *Handler) ListDLQ(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	msgs, err := h.store.ListDLQ(c.Request.Context(), c.Param("topic"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

func (h *Handler) DLQStats(c *gin.Context) {
	topic := c.Param("topic")
	total, err := h.store.CountDLQ(c.Request.Context(), topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	unreplayed, err := h.store.CountDLQUnreplayed(c.Request.Context(), topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"topic":      topic,
		"total":      total,
		"unreplayed": unreplayed,
	})
}

func (h *Handler) ListRetries(c *gin.Context) {
	topic := c.Query("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "topic query parameter required"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	records, err := h.store.ListRetries(c.Request.Context(), topic, c.Query("eventId"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *Handler) ReplayDLQ(c *gin.Context) {
	count, err := h.replay.Replay(c.Request.Context(), models.ReplayRequest{
		Topic: c.Param("topic"), DLQOnly: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"replayed": count})
}

func (h *Handler) ReplayEvents(c *gin.Context) {
	var req models.ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	count, err := h.replay.Replay(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"replayed": count})
}

func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req models.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	w, err := h.workflow.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (h *Handler) RunWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	go func() {
		if err := h.workflow.Run(context.Background(), workflowID); err != nil {
			h.log.Error("workflow run failed", zap.String("workflowId", workflowID), zap.Error(err))
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"status": "running", "workflowId": c.Param("id")})
}

func (h *Handler) GetWorkflow(c *gin.Context) {
	w, steps, err := h.workflow.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workflow not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"workflow": w, "steps": steps})
}
