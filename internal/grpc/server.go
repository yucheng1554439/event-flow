package grpcserver

import (
	"fmt"
	"net"

	eventflowv1 "github.com/eventflow/eventflow/api/gen/go/eventflow/v1"
	"github.com/eventflow/eventflow/internal/replay"
	"github.com/eventflow/eventflow/internal/storage"
	"github.com/eventflow/eventflow/internal/topic"
	"github.com/eventflow/eventflow/internal/workflow"
	kafkapkg "github.com/eventflow/eventflow/pkg/kafka"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	log     *zap.Logger
	grpcSrv *grpc.Server
}

func NewServer(
	topicSvc *topic.Service,
	store *storage.PostgresStore,
	cache *storage.RedisCache,
	producer *kafkapkg.Producer,
	replaySvc *replay.Service,
	workflowEngine *workflow.Engine,
	log *zap.Logger,
) *Server {
	grpcSrv := grpc.NewServer()
	eventflowv1.RegisterTopicServiceServer(grpcSrv, NewTopicServer(topicSvc))
	eventflowv1.RegisterEventServiceServer(grpcSrv, NewEventServer(store, cache, producer))
	eventflowv1.RegisterReplayServiceServer(grpcSrv, NewReplayServer(replaySvc))
	eventflowv1.RegisterWorkflowServiceServer(grpcSrv, NewWorkflowServer(workflowEngine))
	reflection.Register(grpcSrv)
	return &Server{log: log, grpcSrv: grpcSrv}
}

func (s *Server) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	s.log.Info("gRPC server listening", zap.Int("port", port))
	return s.grpcSrv.Serve(lis)
}

func (s *Server) Stop() { s.grpcSrv.GracefulStop() }
