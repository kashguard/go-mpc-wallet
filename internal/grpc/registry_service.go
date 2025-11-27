package grpc

import (
	"context"
	"fmt"

	pb "github.com/kashguard/go-mpc-wallet/internal/pb/mpc/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RegistryService 注册gRPC服务实现
type RegistryService struct {
	pb.UnimplementedMPCRegistryServer

	// TODO: 注入注册中心实现
	// registry *consul.Registry
}

// NewRegistryService 创建注册服务
func NewRegistryService() *RegistryService {
	return &RegistryService{}
}

// RegisterNode 注册节点
func (s *RegistryService) RegisterNode(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Info().
		Str("node_id", req.Node.NodeId).
		Str("node_type", req.Node.NodeType).
		Strs("capabilities", req.Capabilities).
		Msg("Registering node")

	// TODO: 验证节点信息
	// TODO: 存储到注册中心
	// TODO: 设置健康检查

	return &pb.RegisterResponse{
		Success:      true,
		RegistrationId: fmt.Sprintf("reg-%s", req.Node.NodeId),
		Message:      "Node registered successfully",
		RegisteredAt: "now", // TODO: 实际时间
	}, nil
}

// UnregisterNode 注销节点
func (s *RegistryService) UnregisterNode(ctx context.Context, req *pb.UnregisterRequest) (*pb.UnregisterResponse, error) {
	log.Info().
		Str("node_id", req.NodeId).
		Msg("Unregistering node")

	// TODO: 从注册中心移除节点

	return &pb.UnregisterResponse{
		Success: true,
		Message: "Node unregistered successfully",
	}, nil
}

// DiscoverNodes 发现节点
func (s *RegistryService) DiscoverNodes(ctx context.Context, req *pb.DiscoveryRequest) (*pb.DiscoveryResponse, error) {
	log.Info().
		Str("node_type", req.NodeType).
		Strs("capabilities", req.Capabilities).
		Int32("limit", req.Limit).
		Msg("Discovering nodes")

	// TODO: 从注册中心查询节点

	// 模拟返回
	nodes := []*pb.NodeInfo{
		{
			NodeId:   "node1",
			NodeType: "participant",
			Endpoint: "localhost:50051",
			Version:  "v1.0.0",
		},
		{
			NodeId:   "node2",
			NodeType: "participant",
			Endpoint: "localhost:50052",
			Version:  "v1.0.0",
		},
	}

	return &pb.DiscoveryResponse{
		Nodes:     nodes,
		TotalCount: int32(len(nodes)),
	}, nil
}

// GetNodeInfo 获取节点信息
func (s *RegistryService) GetNodeInfo(ctx context.Context, req *pb.NodeInfoRequest) (*pb.NodeInfoResponse, error) {
	log.Info().
		Str("node_id", req.NodeId).
		Msg("Getting node info")

	// TODO: 从注册中心查询节点信息

	return &pb.NodeInfoResponse{
		Node: &pb.NodeInfo{
			NodeId:   req.NodeId,
			NodeType: "participant",
			Endpoint: "localhost:50051",
			Version:  "v1.0.0",
		},
		Exists: true,
	}, nil
}

// HealthCheck 健康检查
func (s *RegistryService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	log.Debug().
		Str("node_id", req.NodeId).
		Str("check_type", req.CheckType).
		Msg("Health check")

	// TODO: 执行实际健康检查

	return &pb.HealthCheckResponse{
		Healthy:  true,
		Status:   "passing",
		Message:  "Node is healthy",
		CheckedAt: "now", // TODO: 实际时间
		Details: map[string]string{
			"cpu":    "normal",
			"memory": "normal",
			"disk":   "normal",
		},
	}, nil
}

// WatchNodes 监听节点变化
func (s *RegistryService) WatchNodes(req *pb.WatchRequest, stream pb.MPCRegistry_WatchNodesServer) error {
	log.Info().
		Str("node_type", req.NodeType).
		Bool("include_health", req.IncludeHealthUpdates).
		Msg("Starting node watch")

	// TODO: 实现真正的监听机制
	// 这里只是模拟

	// 发送一个示例事件
	event := &pb.WatchResponse{
		EventType: "node_registered",
		Node: &pb.NodeInfo{
			NodeId:   "node1",
			NodeType: "participant",
			Endpoint: "localhost:50051",
		},
		EventTime: "now",
	}

	if err := stream.Send(event); err != nil {
		return status.Errorf(codes.Internal, "failed to send watch event: %v", err)
	}

	// 在实际实现中，这里应该保持流打开并监听注册中心的变化
	return nil
}
