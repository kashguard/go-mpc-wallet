package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/kashguard/go-mpc-wallet/internal/config"
	pb "github.com/kashguard/go-mpc-wallet/internal/pb/mpc/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server MPC gRPC服务器
type Server struct {
	config       *config.Server
	grpcServer   *grpc.Server
	nodeService  pb.MPCNodeServer
	coordService pb.MPCCoordinatorServer
	regService   pb.MPCRegistryServer
	healthServer *health.Server
	listener     net.Listener
}

// NewServer 创建新的gRPC服务器
func NewServer(cfg *config.Server) (*Server, error) {
	s := &Server{
		config:        cfg,
		healthServer:  health.NewServer(),
		nodeService:   pb.UnimplementedMPCNodeServer{},
		coordService:  pb.UnimplementedMPCCoordinatorServer{},
		regService:    pb.UnimplementedMPCRegistryServer{},
	}

	// 设置健康状态
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// 创建gRPC服务器
	var opts []grpc.ServerOption

	// TLS配置
	if cfg.MPC.TLSEnabled {
		tlsConfig, err := s.createTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	// 创建拦截器
	opts = append(opts,
		grpc.UnaryInterceptor(s.unaryInterceptor),
		grpc.StreamInterceptor(s.streamInterceptor),
	)

	s.grpcServer = grpc.NewServer(opts...)

	// 注册服务
	pb.RegisterMPCNodeServer(s.grpcServer, s.nodeService)
	pb.RegisterMPCCoordinatorServer(s.grpcServer, s.coordService)
	pb.RegisterMPCRegistryServer(s.grpcServer, s.regService)
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// 开发环境启用反射
	// TODO: 检查环境变量
	// if cfg.Environment == "development" {
	reflection.Register(s.grpcServer)
	// }

	return s, nil
}

// SetNodeService 设置节点服务实现
func (s *Server) SetNodeService(service pb.MPCNodeServer) {
	s.nodeService = service
	if s.grpcServer != nil {
		pb.RegisterMPCNodeServer(s.grpcServer, service)
	}
}

// SetCoordinatorService 设置协调器服务实现
func (s *Server) SetCoordinatorService(service pb.MPCCoordinatorServer) {
	s.coordService = service
	if s.grpcServer != nil {
		pb.RegisterMPCCoordinatorServer(s.grpcServer, service)
	}
}

// SetRegistryService 设置注册服务实现
func (s *Server) SetRegistryService(service pb.MPCRegistryServer) {
	s.regService = service
	if s.grpcServer != nil {
		pb.RegisterMPCRegistryServer(s.grpcServer, service)
	}
}

// Start 启动gRPC服务器
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.config.MPC.GRPCPort)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener

	log.Info().
		Str("address", addr).
		Bool("tls", s.config.MPC.TLSEnabled).
		Msg("Starting MPC gRPC server")

	// 在goroutine中启动服务器
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
		}
	}()

	// 等待上下文取消
	<-ctx.Done()
	return s.Stop()
}

// Stop 停止gRPC服务器
func (s *Server) Stop() error {
	log.Info().Msg("Stopping MPC gRPC server")

	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	return nil
}

// createTLSConfig 创建TLS配置
func (s *Server) createTLSConfig() (*tls.Config, error) {
	// TODO: 实现TLS证书加载
	// 这里应该从配置文件或密钥管理器加载证书
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
	}, nil
}

// unaryInterceptor 一元调用拦截器
func (s *Server) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	// 调用处理函数
	resp, err := handler(ctx, req)

	// 记录请求日志
	duration := time.Since(start)
	log.Debug().
		Str("method", info.FullMethod).
		Dur("duration", duration).
		Bool("error", err != nil).
		Msg("gRPC unary call")

	return resp, err
}

// streamInterceptor 流式调用拦截器
func (s *Server) streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()

	// 调用处理函数
	err := handler(srv, stream)

	// 记录请求日志
	duration := time.Since(start)
	log.Debug().
		Str("method", info.FullMethod).
		Dur("duration", duration).
		Bool("error", err != nil).
		Msg("gRPC stream call")

	return err
}
