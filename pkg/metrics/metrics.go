package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events successfully processed",
		},
		[]string{"topic", "consumer_group"},
	)

	EventsFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_failed_total",
			Help: "Total number of events that failed processing",
		},
		[]string{"topic", "consumer_group", "reason"},
	)

	ConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "consumer_lag",
			Help: "Consumer lag per topic partition",
		},
		[]string{"topic", "consumer_group", "partition"},
	)

	QueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Approximate queue depth per topic",
		},
		[]string{"topic"},
	)

	RetryAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total retry attempts for failed events",
		},
		[]string{"topic", "status"},
	)

	WorkflowDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workflow_duration_seconds",
			Help:    "Workflow execution duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"workflow_name", "status"},
	)

	EventsPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total events published to topics",
		},
		[]string{"topic"},
	)

	DLQMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dlq_messages_total",
			Help: "Total messages routed to dead letter queue",
		},
		[]string{"topic"},
	)

	WorkflowCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_completed_total",
			Help: "Total workflows completed successfully",
		},
		[]string{"workflow_name"},
	)

	WorkflowFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_failed_total",
			Help: "Total workflow failures by step",
		},
		[]string{"workflow_name", "step"},
	)
)
