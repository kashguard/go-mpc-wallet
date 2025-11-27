package api

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/dropbox/godropbox/time2"
	"github.com/kashguard/go-mpc-wallet/internal/auth"
	"github.com/kashguard/go-mpc-wallet/internal/config"
	"github.com/kashguard/go-mpc-wallet/internal/i18n"
	"github.com/kashguard/go-mpc-wallet/internal/mailer"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/coordinator"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/key"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/node"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/participant"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/protocol"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/session"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/signing"
	"github.com/kashguard/go-mpc-wallet/internal/grpc"
	"github.com/kashguard/go-mpc-wallet/internal/mpc/storage"
	"github.com/kashguard/go-mpc-wallet/internal/persistence"
	"github.com/kashguard/go-mpc-wallet/internal/push"
	"github.com/kashguard/go-mpc-wallet/internal/push/provider"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// PROVIDERS - define here only providers that for various reasons (e.g. cyclic dependency) can't live in their corresponding packages
// or for wrapping providers that only accept sub-configs to prevent the requirements for defining providers for sub-configs.
// https://github.com/google/wire/blob/main/docs/guide.md#defining-providers

// NewPush creates an instance of the push service and registers the configured push providers.
func NewPush(cfg config.Server, db *sql.DB) (*push.Service, error) {
	pusher := push.New(db)

	if cfg.Push.UseFCMProvider {
		fcmProvider, err := provider.NewFCM(cfg.FCMConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create FCM provider: %w", err)
		}
		pusher.RegisterProvider(fcmProvider)
	}

	if cfg.Push.UseMockProvider {
		log.Warn().Msg("Initializing mock push provider")
		mockProvider := provider.NewMock(push.ProviderTypeFCM)
		pusher.RegisterProvider(mockProvider)
	}

	if pusher.GetProviderCount() < 1 {
		log.Warn().Msg("No providers registered for push service")
	}

	return pusher, nil
}

func NewClock(t ...*testing.T) time2.Clock {
	var clock time2.Clock

	useMock := len(t) > 0 && t[0] != nil

	if useMock {
		clock = time2.NewMockClock(time.Now())
	} else {
		clock = time2.DefaultClock
	}

	return clock
}

func NewAuthService(config config.Server, db *sql.DB, clock time2.Clock) *auth.Service {
	return auth.NewService(config, db, clock)
}

func NewMailer(config config.Server) (*mailer.Mailer, error) {
	return mailer.NewWithConfig(config.Mailer, config.SMTP)
}

func NewDB(config config.Server) (*sql.DB, error) {
	return persistence.NewDB(config.Database)
}

func NewI18N(config config.Server) (*i18n.Service, error) {
	return i18n.New(config.I18n)
}

func NoTest() []*testing.T {
	return nil
}

func NewMetadataStore(db *sql.DB) storage.MetadataStore {
	return storage.NewPostgreSQLStore(db)
}

func NewRedisClient(cfg config.Server) (*redis.Client, error) {
	if cfg.MPC.RedisEndpoint == "" {
		return nil, fmt.Errorf("MPC RedisEndpoint is not configured")
	}

	client := redis.NewClient(&redis.Options{
		Addr: cfg.MPC.RedisEndpoint,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

func NewSessionStore(client *redis.Client) storage.SessionStore {
	return storage.NewRedisStore(client)
}

func NewKeyShareStorage(cfg config.Server) (storage.KeyShareStorage, error) {
	if cfg.MPC.KeyShareStoragePath == "" {
		return nil, fmt.Errorf("MPC KeyShareStoragePath is not configured")
	}
	if cfg.MPC.KeyShareEncryptionKey == "" {
		return nil, fmt.Errorf("MPC KeyShareEncryptionKey is not configured")
	}
	return storage.NewFileSystemKeyShareStorage(cfg.MPC.KeyShareStoragePath, cfg.MPC.KeyShareEncryptionKey)
}

func NewProtocolEngine(cfg config.Server) protocol.Engine {
	curve := "secp256k1"
	if len(cfg.MPC.SupportedProtocols) > 0 {
		// future: switch based on protocol type
	}
	return protocol.NewGG20Protocol(curve)
}

func NewNodeManager(metadataStore storage.MetadataStore, cfg config.Server) *node.Manager {
	heartbeat := time.Duration(cfg.MPC.SessionTimeout)
	if heartbeat <= 0 {
		heartbeat = 30
	}
	return node.NewManager(metadataStore, heartbeat*time.Second)
}

func NewNodeRegistry(manager *node.Manager) *node.Registry {
	return node.NewRegistry(manager)
}

func NewNodeDiscovery(manager *node.Manager) *node.Discovery {
	return node.NewDiscovery(manager)
}

func NewSessionManager(metadataStore storage.MetadataStore, sessionStore storage.SessionStore, cfg config.Server) *session.Manager {
	timeout := time.Duration(cfg.MPC.SessionTimeout)
	if timeout <= 0 {
		timeout = 300
	}
	return session.NewManager(metadataStore, sessionStore, timeout*time.Second)
}

func NewKeyServiceProvider(metadataStore storage.MetadataStore, keyShareStorage storage.KeyShareStorage, protocolEngine protocol.Engine) *key.Service {
	return key.NewService(metadataStore, keyShareStorage, protocolEngine)
}

func NewSigningServiceProvider(protocolEngine protocol.Engine, sessionManager *session.Manager, nodeDiscovery *node.Discovery) *signing.Service {
	return signing.NewService(protocolEngine, sessionManager, nodeDiscovery)
}

func NewCoordinatorServiceProvider(
	metadataStore storage.MetadataStore,
	keyService *key.Service,
	signingService *signing.Service,
	sessionManager *session.Manager,
	nodeManager *node.Manager,
	nodeDiscovery *node.Discovery,
	protocolEngine protocol.Engine,
) *coordinator.Service {
	return coordinator.NewService(metadataStore, keyService, signingService, sessionManager, nodeManager, nodeDiscovery, protocolEngine)
}

func NewParticipantServiceProvider(cfg config.Server, keyShareStorage storage.KeyShareStorage, protocolEngine protocol.Engine) *participant.Service {
	return participant.NewService(cfg.MPC.NodeID, keyShareStorage, protocolEngine)
}

// gRPC相关Provider

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(cfg config.Server) (*grpc.Server, error) {
	return grpc.NewServer(&cfg)
}

// NewGRPCClient 创建gRPC客户端
func NewGRPCClient(cfg config.Server) (*grpc.Client, error) {
	return grpc.NewClient(&grpc.Config{
		Target: fmt.Sprintf("localhost:%d", cfg.MPC.GRPCPort),
		TLS:    cfg.MPC.TLSEnabled,
		Timeout: 30 * time.Second,
	})
}

// NewNodeService 创建节点gRPC服务
func NewNodeService(cfg config.Server) *grpc.NodeService {
	return grpc.NewNodeService(cfg.MPC.NodeID)
}

// NewCoordinatorService 创建协调器gRPC服务
func NewCoordinatorService(cfg config.Server) *grpc.CoordinatorService {
	return grpc.NewCoordinatorService(cfg.MPC.NodeID)
}

// NewRegistryService 创建注册gRPC服务
func NewRegistryService() *grpc.RegistryService {
	return grpc.NewRegistryService()
}

// NewHeartbeatService 创建心跳服务
func NewHeartbeatService(cfg config.Server, client *grpc.Client) *grpc.HeartbeatService {
	return grpc.NewHeartbeatService(&grpc.HeartbeatConfig{
		NodeID:        cfg.MPC.NodeID,
		CoordinatorID: "coordinator", // TODO: 动态获取
		Interval:      30 * time.Second,
		Timeout:       10 * time.Second,
		Client:        client,
	})
}

// NewHeartbeatManager 创建心跳管理器
func NewHeartbeatManager() *grpc.HeartbeatManager {
	return grpc.NewHeartbeatManager()
}
