package grpc

import (
	"context"
	"io"

	pb "github.com/kashguard/go-mpc-wallet/internal/pb/mpc/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodeService 节点gRPC服务实现
type NodeService struct {
	pb.UnimplementedMPCNodeServer

	nodeID string
	// TODO: 注入实际的服务
	// sessionManager *session.Manager
	// protocolEngine protocol.Engine
}

// NewNodeService 创建节点服务
func NewNodeService(nodeID string) *NodeService {
	return &NodeService{
		nodeID: nodeID,
	}
}

// JoinSigningSession 双向流：加入签名会话
func (s *NodeService) JoinSigningSession(stream pb.MPCNode_JoinSigningSessionServer) error {
	log.Info().
		Str("node_id", s.nodeID).
		Msg("Node joining signing session")

	// 接收初始加入请求
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive join request: %v", err)
	}

	joinReq := req.GetJoinRequest()
	if joinReq == nil {
		return status.Error(codes.InvalidArgument, "missing join request")
	}

	sessionID := joinReq.SessionId
	log.Info().
		Str("session_id", sessionID).
		Str("node_id", s.nodeID).
		Msg("Processing join request")

	// TODO: 验证会话存在性和权限
	// TODO: 将节点加入会话

	// 发送会话确认
	confirmation := &pb.SessionMessage{
		MessageType: &pb.SessionMessage_Confirmation{
			Confirmation: &pb.SessionConfirmation{
				SessionId:    sessionID,
				Status:       "joined",
				Threshold:    2, // TODO: 从会话获取
				TotalNodes:   3, // TODO: 从会话获取
				Participants: []string{s.nodeID}, // TODO: 获取所有参与者
				CurrentRound: 0,
				ConfirmedAt:  "now", // TODO: 使用实际时间
			},
		},
	}

	if err := stream.Send(confirmation); err != nil {
		return status.Errorf(codes.Internal, "failed to send confirmation: %v", err)
	}

	log.Info().
		Str("session_id", sessionID).
		Str("node_id", s.nodeID).
		Msg("Join confirmation sent")

	// 处理协议消息流
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.Info().
				Str("session_id", sessionID).
				Str("node_id", s.nodeID).
				Msg("Stream closed by client")
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive message: %v", err)
		}

		// 处理不同类型的消息
		switch msg.MessageType.(type) {
		case *pb.SessionMessage_JoinRequest:
			// 重复的加入请求，忽略
			continue

		case *pb.SessionMessage_ShareMessage:
			shareMsg := msg.GetShareMessage()
			if err := s.handleShareMessage(stream, sessionID, shareMsg); err != nil {
				return err
			}

		default:
			log.Warn().
				Str("session_id", sessionID).
				Str("node_id", s.nodeID).
				Msg("Unknown message type received")
		}
	}
}

// SubmitSignatureShare 提交签名分片
func (s *NodeService) SubmitSignatureShare(ctx context.Context, req *pb.ShareRequest) (*pb.ShareResponse, error) {
	log.Info().
		Str("session_id", req.SessionId).
		Str("node_id", req.NodeId).
		Int32("round", req.Round).
		Msg("Submitting signature share")

	// TODO: 验证请求
	// TODO: 存储签名分片
	// TODO: 检查是否可以进行下一轮

	return &pb.ShareResponse{
		Accepted:   true,
		Message:    "Share accepted",
		NextRound:  req.Round + 1,
	}, nil
}

// Heartbeat 心跳检测
func (s *NodeService) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	log.Debug().
		Str("node_id", req.NodeId).
		Str("sent_at", req.SentAt).
		Msg("Heartbeat received")

	// TODO: 更新节点最后心跳时间
	// TODO: 检查节点健康状态

	return &pb.HeartbeatResponse{
		Alive:         true,
		CoordinatorId: "coordinator-1", // TODO: 获取实际协调器ID
		ReceivedAt:    "now",           // TODO: 使用实际时间
		Instructions: map[string]string{
			"status": "active",
		},
	}, nil
}

// handleShareMessage 处理签名分片消息
func (s *NodeService) handleShareMessage(stream pb.MPCNode_JoinSigningSessionServer, sessionID string, shareMsg *pb.ShareMessage) error {
	log.Info().
		Str("session_id", sessionID).
		Str("node_id", s.nodeID).
		Int32("round", shareMsg.Round).
		Msg("Handling share message")

	// TODO: 验证签名分片
	// TODO: 存储到临时存储
	// TODO: 检查是否可以聚合

	// 发送回执
	receipt := &pb.SessionMessage{
		MessageType: &pb.SessionMessage_ShareMessage{
			ShareMessage: &pb.ShareMessage{
				ShareData:   []byte("acknowledged"), // TODO: 实际处理
				Round:       shareMsg.Round,
				SubmittedAt: "now", // TODO: 实际时间
			},
		},
	}

	if err := stream.Send(receipt); err != nil {
		return status.Errorf(codes.Internal, "failed to send receipt: %v", err)
	}

	return nil
}
