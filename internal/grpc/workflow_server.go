package grpcserver

import (
	"context"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/internal/workflow"
	"github.com/eventflow/eventflow/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WorkflowServer struct {
	eventflowv1.UnimplementedWorkflowServiceServer
	engine *workflow.Engine
}

func NewWorkflowServer(engine *workflow.Engine) *WorkflowServer {
	return &WorkflowServer{engine: engine}
}

func (s *WorkflowServer) StartWorkflow(ctx context.Context, req *eventflowv1.StartWorkflowRequest) (*eventflowv1.Workflow, error) {
	w, err := s.engine.Create(ctx, models.CreateWorkflowRequest{
		Name:  req.Name,
		Input: structToRaw(req.Input),
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return workflowToProto(w), nil
}

func (s *WorkflowServer) RunWorkflow(ctx context.Context, req *eventflowv1.RunWorkflowRequest) (*eventflowv1.RunWorkflowResponse, error) {
	id := req.WorkflowId
	go func() {
		_ = s.engine.Run(context.Background(), id)
	}()
	return &eventflowv1.RunWorkflowResponse{WorkflowId: req.WorkflowId, Status: "running"}, nil
}

func (s *WorkflowServer) GetWorkflow(ctx context.Context, req *eventflowv1.GetWorkflowRequest) (*eventflowv1.GetWorkflowResponse, error) {
	w, steps, err := s.engine.Get(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "workflow not found")
	}
	return &eventflowv1.GetWorkflowResponse{
		Workflow: workflowToProto(w),
		Steps:    stepsToProto(steps),
	}, nil
}
