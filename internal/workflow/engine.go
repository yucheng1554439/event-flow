package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/pkg/metrics"
	"github.com/eventflow/eventflow/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StepFunc func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
type CompensateFunc func(ctx context.Context, input json.RawMessage) error

type StepDefinition struct {
	Name         string
	Execute      StepFunc
	Compensation CompensateFunc
	Timeout      time.Duration
}

type Definition struct {
	Name  string
	Steps []StepDefinition
}

type Engine struct {
	store    *storage.PostgresStore
	cache    *storage.RedisCache
	log      *zap.Logger
	registry map[string]Definition
	failStep string // test/production failure injection
}

func NewEngine(store *storage.PostgresStore, cache *storage.RedisCache, log *zap.Logger) *Engine {
	e := &Engine{
		store:    store,
		cache:    cache,
		log:      log,
		registry: make(map[string]Definition),
	}
	e.registerDefaults()
	return e
}

func (e *Engine) Register(def Definition) {
	e.registry[def.Name] = def
}

func (e *Engine) SetFailStep(name string) { e.failStep = name }
func (e *Engine) ClearFailStep()          { e.failStep = "" }

func (e *Engine) Create(ctx context.Context, req models.CreateWorkflowRequest) (*models.Workflow, error) {
	if _, ok := e.registry[req.Name]; !ok {
		return nil, fmt.Errorf("unknown workflow: %s", req.Name)
	}
	now := time.Now().UTC()
	w := models.Workflow{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Status:      models.WorkflowPending,
		Input:       req.Input,
		CurrentStep: "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := e.store.CreateWorkflow(ctx, w); err != nil {
		return nil, err
	}
	return &w, nil
}

func (e *Engine) Run(ctx context.Context, workflowID string) error {
	w, err := e.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}
	def, ok := e.registry[w.Name]
	if !ok {
		return fmt.Errorf("workflow definition not found: %s", w.Name)
	}

	locked, err := e.cache.LockWorkflow(ctx, workflowID, 5*time.Minute)
	if err != nil || !locked {
		return fmt.Errorf("workflow already running: %s", workflowID)
	}
	defer e.cache.UnlockWorkflow(ctx, workflowID)

	start := time.Now()
	w.Status = models.WorkflowRunning
	w.UpdatedAt = time.Now().UTC()
	_ = e.store.UpdateWorkflow(ctx, *w)

	var stepInput = w.Input
	completedSteps := make([]models.WorkflowStep, 0, len(def.Steps))

	for _, stepDef := range def.Steps {
		w.CurrentStep = stepDef.Name
		w.UpdatedAt = time.Now().UTC()
		_ = e.store.UpdateWorkflow(ctx, *w)
		e.log.Info("workflow step transition",
			zap.String("workflowId", workflowID),
			zap.String("workflow", w.Name),
			zap.String("step", stepDef.Name),
			zap.String("phase", "start"),
		)

		step := e.executeStep(ctx, workflowID, w.Name, stepDef, stepInput, w.Input)
		e.log.Info("workflow step transition",
			zap.String("workflowId", workflowID),
			zap.String("workflow", w.Name),
			zap.String("step", stepDef.Name),
			zap.String("phase", "end"),
			zap.String("status", step.Status),
		)
		if err := e.store.CreateWorkflowStep(ctx, step); err != nil {
			return err
		}
		if step.Status == "failed" {
			return e.compensate(ctx, w, completedSteps)
		}
		completedSteps = append(completedSteps, step)
		stepInput = step.Output
	}

	now := time.Now().UTC()
	w.Status = models.WorkflowCompleted
	w.UpdatedAt = now
	w.CompletedAt = &now
	_ = e.store.UpdateWorkflow(ctx, *w)
	metrics.WorkflowDurationSeconds.WithLabelValues(w.Name, "completed").Observe(time.Since(start).Seconds())
	metrics.WorkflowCompletedTotal.WithLabelValues(w.Name).Inc()
	return nil
}

func (e *Engine) executeStep(ctx context.Context, workflowID, workflowName string, def StepDefinition, input, workflowInput json.RawMessage) models.WorkflowStep {
	now := time.Now().UTC()
	step := models.WorkflowStep{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		Name:       def.Name,
		Status:     "running",
		Input:      input,
		Attempt:    1,
		StartedAt:  &now,
	}

	if e.shouldFailStep(workflowInput, def.Name) {
		completed := time.Now().UTC()
		step.CompletedAt = &completed
		step.Status = "failed"
		step.Error = "demo step failure: " + def.Name
		metrics.WorkflowDurationSeconds.WithLabelValues(def.Name, "failed").Observe(completed.Sub(now).Seconds())
		metrics.WorkflowFailedTotal.WithLabelValues(workflowName, def.Name).Inc()
		return step
	}

	stepCtx, cancel := context.WithTimeout(ctx, def.Timeout)
	defer cancel()

	output, err := def.Execute(stepCtx, input)
	completed := time.Now().UTC()
	step.CompletedAt = &completed
	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		metrics.WorkflowDurationSeconds.WithLabelValues(def.Name, "failed").Observe(completed.Sub(now).Seconds())
		return step
	}
	step.Status = "completed"
	step.Output = output
	return step
}

func (e *Engine) compensate(ctx context.Context, w *models.Workflow, completed []models.WorkflowStep) error {
	w.Status = models.WorkflowCompensating
	w.UpdatedAt = time.Now().UTC()
	_ = e.store.UpdateWorkflow(ctx, *w)

	def := e.registry[w.Name]
	stepMap := make(map[string]StepDefinition, len(def.Steps))
	for _, s := range def.Steps {
		stepMap[s.Name] = s
	}

	for i := len(completed) - 1; i >= 0; i-- {
		step := completed[i]
		stepDef, ok := stepMap[step.Name]
		if !ok || stepDef.Compensation == nil {
			continue
		}
		e.log.Info("executing compensation", zap.String("workflow", w.Name), zap.String("step", step.Name))
		if err := stepDef.Compensation(ctx, step.Input); err != nil {
			e.log.Error("compensation failed", zap.String("step", step.Name), zap.Error(err))
		}
	}

	w.Status = models.WorkflowFailed
	w.UpdatedAt = time.Now().UTC()
	_ = e.store.UpdateWorkflow(ctx, *w)
	metrics.WorkflowDurationSeconds.WithLabelValues(w.Name, "compensated").Observe(0)
	metrics.WorkflowFailedTotal.WithLabelValues(w.Name, w.CurrentStep).Inc()
	return fmt.Errorf("workflow %s failed at step %s", w.ID, w.CurrentStep)
}

func (e *Engine) Get(ctx context.Context, id string) (*models.Workflow, []models.WorkflowStep, error) {
	w, err := e.store.GetWorkflow(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	steps, err := e.store.ListWorkflowSteps(ctx, id)
	return w, steps, err
}

func (e *Engine) registerDefaults() {
	e.Register(Definition{
		Name: "OrderFulfillment",
		Steps: []StepDefinition{
			{Name: "ProcessPayment", Execute: stubStep("payment_processed"), Compensation: namedCompensate("RefundPayment"), Timeout: 30 * time.Second},
			{Name: "ReserveInventory", Execute: stubStep("inventory_reserved"), Compensation: namedCompensate("ReleaseInventory"), Timeout: 30 * time.Second},
			{Name: "SendEmail", Execute: stubStep("email_sent"), Timeout: 15 * time.Second},
		},
	})
	e.Register(Definition{
		Name: "GalacticCommerce",
		Steps: []StepDefinition{
			{Name: "ProcessPayment", Execute: stubStep("credits_debited"), Compensation: namedCompensate("RefundPayment"), Timeout: 30 * time.Second},
			{Name: "ReserveInventory", Execute: stubStep("ship_reserved"), Compensation: namedCompensate("ReleaseShip"), Timeout: 30 * time.Second},
			{Name: "SendConfirmation", Execute: stubStep("confirmation_sent"), Timeout: 15 * time.Second},
		},
	})
}

func (e *Engine) shouldFailStep(input json.RawMessage, stepName string) bool {
	if e.failStep == stepName {
		return true
	}
	var m map[string]any
	if json.Unmarshal(input, &m) != nil {
		return false
	}
	if fs, ok := m["demoFailStep"].(string); ok && fs == stepName {
		return true
	}
	return false
}

func stubStep(result string) StepFunc {
	return func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(map[string]string{"status": result})
	}
}

func namedCompensate(action string) CompensateFunc {
	return func(ctx context.Context, input json.RawMessage) error {
		return nil
	}
}
