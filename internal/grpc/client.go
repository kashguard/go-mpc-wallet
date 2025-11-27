package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pb "github.com/kashguard/go-mpc-wallet/internal/pb/mpc/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client MPC gRPC客户端
type Client struct {
	config         *Config
	conn           *grpc.ClientConn
	nodeClient     pb.MPCNodeClient
	coordClient    pb.MPCCoordinatorClient
	registryClient pb.MPCRegistryClient
}

// Config 客户端配置
type Config struct {
	Target      string
	TLS         bool
	TLSCertFile string
	TLSKeyFile  string
	Timeout     time.Duration
}

// NewClient 创建新的gRPC客户端
func NewClient(config *Config) (*Client, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	var opts []grpc.DialOption

	// TLS配置
	if config.TLS {
		creds, err := loadTLSCredentials(config)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// 连接超时
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Target, err)
	}

	client := &Client{
		config:         config,
		conn:           conn,
	nodeClient:     pb.NewMPCNodeClient(conn),
	coordClient:    pb.NewMPCCoordinatorClient(conn),
	registryClient: pb.NewMPCRegistryClient(conn),
	}

	log.Info().
		Str("target", config.Target).
		Bool("tls", config.TLS).
		Msg("MPC gRPC client connected")

	return client, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// NodeClient 获取节点客户端
func (c *Client) NodeClient() pb.MPCNodeClient {
	return c.nodeClient
}

// CoordinatorClient 获取协调器客户端
func (c *Client) CoordinatorClient() pb.MPCCoordinatorClient {
	return c.coordClient
}

// RegistryClient 获取注册客户端
func (c *Client) RegistryClient() pb.MPCRegistryClient {
	return c.registryClient
}

// Heartbeat 发送心跳
func (c *Client) Heartbeat(ctx context.Context, nodeID string) error {
	req := &pb.HeartbeatRequest{
		NodeId:    nodeID,
		SentAt:    time.Now().Format(time.RFC3339),
		StatusInfo: map[string]string{
			"status": "healthy",
		},
	}

	resp, err := c.nodeClient.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	log.Debug().
		Str("node_id", nodeID).
		Str("coordinator", resp.CoordinatorId).
		Msg("Heartbeat successful")

	return nil
}

// loadTLSCredentials 加载TLS证书
func loadTLSCredentials(config *Config) (credentials.TransportCredentials, error) {
	if config.TLSCertFile == "" || config.TLSKeyFile == "" {
		return credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true, // 开发环境
		}), nil
	}

	// TODO: 实现证书文件加载
	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13,
	}), nil
}
