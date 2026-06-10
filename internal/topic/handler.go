package topic

import (
	"errors"
	"net/http"

	"github.com/eventflow/eventflow/pkg/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/topics", h.CreateTopic)
	rg.GET("/topics", h.ListTopics)
	rg.GET("/topics/:name", h.GetTopic)
	rg.DELETE("/topics/:name", h.DeleteTopic)
}

func (h *Handler) CreateTopic(c *gin.Context) {
	var req models.CreateTopicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Partitions == 0 {
		req.Partitions = 6
	}
	if req.RetentionHours == 0 {
		req.RetentionHours = 168
	}
	if req.CleanupPolicy == "" {
		req.CleanupPolicy = "delete"
	}

	topic, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		writeTopicError(c, err)
		return
	}
	c.JSON(http.StatusCreated, topic)
}

func (h *Handler) ListTopics(c *gin.Context) {
	topics, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, topics)
}

func (h *Handler) GetTopic(c *gin.Context) {
	topic, err := h.svc.Get(c.Request.Context(), c.Param("name"))
	if err != nil {
		writeTopicError(c, err)
		return
	}
	c.JSON(http.StatusOK, topic)
}

func (h *Handler) DeleteTopic(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("name")); err != nil {
		writeTopicError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": c.Param("name")})
}

func writeTopicError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTopicExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrTopicNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidTopic):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
