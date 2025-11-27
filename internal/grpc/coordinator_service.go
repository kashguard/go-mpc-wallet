package grpc

import (
	"context"

	pb "github.com/kashguard/go-mpc-wallet/internal/pb/mpc/v1"
	"github.com/rs/zerolog/log"
)

// CoordinatorService 协调器gRPC服务实现
type CoordinatorService struct {
	pb.UnimplementedMPCCoordinatorServer

	nodeID string
	// TODO: 注入实际的服务
	// sessionManager *session.Manager
	// keyService *key.Service
}

// NewCoordinatorService 创建协调器服务
func NewCoordinatorService(nodeID string) *CoordinatorService {
	return &CoordinatorService{
		nodeID: nodeID,
	}
}

// CreateSigningSession 创建签名会话
func (s *CoordinatorService) CreateSigningSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	log.Info().
		Str("key_id", req.KeyId).
		Str("protocol", req.Protocol).
		Msg("Creating signing session")

	// TODO: 验证密钥存在性
	// TODO: 创建签名会话
	// TODO: 选择参与节点

	sessionID := "session-" + req.KeyId // TODO: 生成唯一ID

	return &pb.CreateSessionResponse{
		SessionId:          sessionID,
		Status:             "created",
		Threshold:          2, // TODO: 从密钥配置获取
		TotalNodes:         3, // TODO: 从密钥配置获取
		ParticipatingNodes: []string{"node1", "node2", "node3"}, // TODO: 实际选择的节点
		CreatedAt:          "now", // TODO: 实际时间
		ExpiresAt:          "later", // TODO: 计算过期时间
	}, nil
}

// GetSessionStatus 获取会话状态
func (s *CoordinatorService) GetSessionStatus(ctx context.Context, req *pb.SessionStatusRequest) (*pb.SessionStatusResponse, error) {
	log.Info().
		Str("session_id", req.SessionId).
		Msg("Getting session status")

	// TODO: 从会话管理器获取状态

	return &pb.SessionStatusResponse{
		SessionId:          req.SessionId,
		Status:             "active",
		CurrentRound:       1,
		TotalRounds:        4,
		ParticipatingNodes: []string{"node1", "node2", "node3"},
		Signature:          "",
		CreatedAt:          "now",
		CompletedAt:        "",
		DurationMs:         0,
	}, nil
}

// AggregateSignatures 聚合签名
func (s *CoordinatorService) AggregateSignatures(ctx context.Context, req *pb.AggregateRequest) (*pb.AggregateResponse, error) {
	log.Info().
		Str("session_id", req.SessionId).
		Msg("Aggregating signatures")

	// TODO: 验证会话完成
	// TODO: 收集所有签名分片
	// TODO: 执行签名聚合
	// TODO: 验证最终签名

	return &pb.AggregateResponse{
		Success:    true,
		Signature:  "aggregated-signature-hex", // TODO: 实际签名
		PublicKey:  "public-key-hex",           // TODO: 从密钥获取
		MessageHash: "message-hash-hex",         // TODO: 计算消息哈希
		AggregatedAt: "now",                     // TODO: 实际时间
	}, nil
}
