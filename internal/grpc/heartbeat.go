package grpc

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// HeartbeatService 心跳服务
type HeartbeatService struct {
	nodeID         string
	coordinatorID  string
	interval       time.Duration
	timeout        time.Duration
	client         *Client
	stopChan       chan struct{}
	running        bool
	mu             sync.RWMutex
	lastHeartbeat  time.Time
}

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	NodeID        string
	CoordinatorID string
	Interval      time.Duration
	Timeout       time.Duration
	Client        *Client
}

// NewHeartbeatService 创建心跳服务
func NewHeartbeatService(config *HeartbeatConfig) *HeartbeatService {
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &HeartbeatService{
		nodeID:        config.NodeID,
		coordinatorID: config.CoordinatorID,
		interval:      config.Interval,
		timeout:       config.Timeout,
		client:        config.Client,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动心跳服务
func (h *HeartbeatService) Start(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil
	}

	h.running = true

	log.Info().
		Str("node_id", h.nodeID).
		Dur("interval", h.interval).
		Msg("Starting heartbeat service")

	go h.run(ctx)

	return nil
}

// Stop 停止心跳服务
func (h *HeartbeatService) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return nil
	}

	h.running = false
	close(h.stopChan)

	log.Info().
		Str("node_id", h.nodeID).
		Msg("Heartbeat service stopped")

	return nil
}

// IsHealthy 检查心跳服务是否健康
func (h *HeartbeatService) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.running {
		return false
	}

	// 检查最后心跳时间是否在合理范围内
	return time.Since(h.lastHeartbeat) < h.interval*2
}

// GetLastHeartbeat 获取最后心跳时间
func (h *HeartbeatService) GetLastHeartbeat() time.Time {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastHeartbeat
}

// run 运行心跳循环
func (h *HeartbeatService) run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat 发送心跳
func (h *HeartbeatService) sendHeartbeat(ctx context.Context) {
	heartbeatCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	err := h.client.Heartbeat(heartbeatCtx, h.nodeID)
	if err != nil {
		log.Error().
			Err(err).
			Str("node_id", h.nodeID).
			Msg("Heartbeat failed")
		return
	}

	h.mu.Lock()
	h.lastHeartbeat = time.Now()
	h.mu.Unlock()

	log.Debug().
		Str("node_id", h.nodeID).
		Time("last_heartbeat", h.lastHeartbeat).
		Msg("Heartbeat successful")
}

// HeartbeatManager 心跳管理器
type HeartbeatManager struct {
	services map[string]*HeartbeatService
	mu       sync.RWMutex
}

// NewHeartbeatManager 创建心跳管理器
func NewHeartbeatManager() *HeartbeatManager {
	return &HeartbeatManager{
		services: make(map[string]*HeartbeatService),
	}
}

// AddService 添加心跳服务
func (m *HeartbeatManager) AddService(nodeID string, service *HeartbeatService) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[nodeID] = service
}

// RemoveService 移除心跳服务
func (m *HeartbeatManager) RemoveService(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if service, exists := m.services[nodeID]; exists {
		service.Stop()
		delete(m.services, nodeID)
	}
}

// GetService 获取心跳服务
func (m *HeartbeatManager) GetService(nodeID string) (*HeartbeatService, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	service, exists := m.services[nodeID]
	return service, exists
}

// StartAll 启动所有心跳服务
func (m *HeartbeatManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for nodeID, service := range m.services {
		if err := service.Start(ctx); err != nil {
			log.Error().
				Err(err).
				Str("node_id", nodeID).
				Msg("Failed to start heartbeat service")
			return err
		}
	}

	return nil
}

// StopAll 停止所有心跳服务
func (m *HeartbeatManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for nodeID, service := range m.services {
		if err := service.Stop(); err != nil {
			log.Error().
				Err(err).
				Str("node_id", nodeID).
				Msg("Failed to stop heartbeat service")
		}
	}

	// 清空服务列表
	m.services = make(map[string]*HeartbeatService)

	return nil
}

// GetHealthStatus 获取所有服务的健康状态
func (m *HeartbeatManager) GetHealthStatus() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]bool)
	for nodeID, service := range m.services {
		status[nodeID] = service.IsHealthy()
	}

	return status
}
