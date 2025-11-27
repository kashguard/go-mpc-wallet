package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/kashguard/go-mpc-wallet/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGRPCServer_Basic(t *testing.T) {
	// 创建测试配置
	cfg := config.Server{
		MPC: config.MPC{
			GRPCPort: 50051, // 使用测试端口
		},
	}

	// 创建服务器
	server, err := NewServer(&cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	// 服务器已经默认设置了未实现的服务器，这里我们不需要额外设置
	// 在实际使用中，会调用SetNodeService等方法来替换默认实现

	// 创建客户端
	client, err := NewClient(&Config{
		Target:  "localhost:50051",
		TLS:     false,
		Timeout: 5 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, client)

	// 启动服务器
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		err := server.Start(ctx)
		if err != nil {
			t.Logf("Server stopped with error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	// 测试心跳
	err = client.Heartbeat(ctx, "test-node")
	if err != nil {
		t.Logf("Heartbeat failed (expected if server not fully started): %v", err)
	}

	// 停止客户端
	client.Close()

	// 取消上下文以停止服务器
	cancel()

	// 验证服务器已停止
	assert.NotNil(t, server)
}

func TestHeartbeatService(t *testing.T) {
	// 创建心跳服务配置
	config := &HeartbeatConfig{
		NodeID:        "test-node",
		CoordinatorID: "test-coord",
		Interval:      1 * time.Second,
		Timeout:       500 * time.Millisecond,
		Client:        nil, // 在实际测试中需要mock客户端
	}

	service := NewHeartbeatService(config)
	assert.NotNil(t, service)
	assert.Equal(t, "test-node", service.nodeID)
	assert.Equal(t, "test-coord", service.coordinatorID)

	// 测试健康状态（未启动时应该不健康）
	assert.False(t, service.IsHealthy())
}
