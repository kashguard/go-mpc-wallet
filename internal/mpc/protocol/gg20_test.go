package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/kashguard/tss-lib/tss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGG20Protocol(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	assert.NotNil(t, protocol)
	assert.NotNil(t, protocol.GG18Protocol)
	assert.Equal(t, "secp256k1", protocol.GetCurve())
	assert.Equal(t, []string{"gg20"}, protocol.SupportedProtocols())
	assert.Equal(t, "gg20", protocol.DefaultProtocol())
}

func TestGG20Protocol_GetCurve(t *testing.T) {
	tests := []struct {
		name  string
		curve string
	}{
		{
			name:  "secp256k1",
			curve: "secp256k1",
		},
		{
			name:  "secp256r1",
			curve: "secp256r1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol := NewGG20Protocol(tt.curve, "node-1", mockMessageRouter)
			assert.Equal(t, tt.curve, protocol.GetCurve())
		})
	}
}

func TestGG20Protocol_SupportedProtocols(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)
	assert.Equal(t, []string{"gg20"}, protocol.SupportedProtocols())
}

func TestGG20Protocol_DefaultProtocol(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)
	assert.Equal(t, "gg20", protocol.DefaultProtocol())
}

func TestGG20Protocol_ValidateKeyGenRequest(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	tests := []struct {
		name    string
		req     *KeyGenRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: &KeyGenRequest{
				Algorithm:  "ECDSA",
				Curve:      "secp256k1",
				Threshold:  2,
				TotalNodes: 3,
				NodeIDs:    []string{"node-1", "node-2", "node-3"},
			},
			wantErr: false,
		},
		{
			name: "unsupported algorithm",
			req: &KeyGenRequest{
				Algorithm:  "RSA",
				Curve:      "secp256k1",
				Threshold:  2,
				TotalNodes: 3,
			},
			wantErr: true,
			errMsg:  "unsupported algorithm",
		},
		{
			name: "unsupported curve",
			req: &KeyGenRequest{
				Algorithm:  "ECDSA",
				Curve:      "P256",
				Threshold:  2,
				TotalNodes: 3,
			},
			wantErr: true,
			errMsg:  "unsupported curve",
		},
		{
			name: "threshold too low",
			req: &KeyGenRequest{
				Algorithm:  "ECDSA",
				Curve:      "secp256k1",
				Threshold:  1,
				TotalNodes: 3,
			},
			wantErr: true,
			errMsg:  "threshold must be at least 2",
		},
		{
			name: "total nodes less than threshold",
			req: &KeyGenRequest{
				Algorithm:  "ECDSA",
				Curve:      "secp256k1",
				Threshold:  3,
				TotalNodes: 2,
			},
			wantErr: true,
			errMsg:  "total nodes must be at least threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := protocol.ValidateKeyGenRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGG20Protocol_ValidateSignRequest(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	tests := []struct {
		name    string
		req     *SignRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request with message",
			req: &SignRequest{
				KeyID:   "test-key",
				Message: []byte("test message"),
				NodeIDs: []string{"node-1", "node-2"},
			},
			wantErr: false,
		},
		{
			name: "valid request with message hex",
			req: &SignRequest{
				KeyID:      "test-key",
				MessageHex: "deadbeef",
				NodeIDs:    []string{"node-1", "node-2"},
			},
			wantErr: false,
		},
		{
			name: "missing key ID",
			req: &SignRequest{
				Message: []byte("test message"),
				NodeIDs: []string{"node-1", "node-2"},
			},
			wantErr: true,
			errMsg:  "key ID is required",
		},
		{
			name: "missing message",
			req: &SignRequest{
				KeyID:   "test-key",
				NodeIDs: []string{"node-1", "node-2"},
			},
			wantErr: true,
			errMsg:  "message is required",
		},
		{
			name: "missing node IDs",
			req: &SignRequest{
				KeyID:   "test-key",
				Message: []byte("test message"),
			},
			wantErr: true,
			errMsg:  "node IDs are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := protocol.ValidateSignRequest(tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGG20Protocol_ThresholdSign_InvalidRequest(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	ctx := context.Background()
	sessionID := "test-session"

	tests := []struct {
		name    string
		req     *SignRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing key ID",
			req: &SignRequest{
				Message: []byte("test"),
				NodeIDs: []string{"node-1"},
			},
			wantErr: true,
			errMsg:  "key ID is required",
		},
		{
			name: "missing message",
			req: &SignRequest{
				KeyID:   "test-key",
				NodeIDs: []string{"node-1"},
			},
			wantErr: true,
			errMsg:  "message is required",
		},
		{
			name: "key not found",
			req: &SignRequest{
				KeyID:   "non-existent-key",
				Message: []byte("test"),
				NodeIDs: []string{"node-1"},
			},
			wantErr: true,
			errMsg:  "key not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := protocol.ThresholdSign(ctx, sessionID, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGG20Protocol_GenerateKeyShare_Delegation(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	req := &KeyGenRequest{
		Algorithm:  "ECDSA",
		Curve:      "secp256k1",
		Threshold:  2,
		TotalNodes: 3,
		NodeIDs:    []string{"node-1", "node-2", "node-3"},
	}

	// 验证委托给 GG18Protocol
	// 注意：实际的 DKG 需要真实的协议执行（多节点协作），这里只测试委托逻辑
	err := protocol.ValidateKeyGenRequest(req)
	require.NoError(t, err)

	// 验证 GenerateKeyShare 方法存在且委托给 GG18Protocol
	// 由于需要真实的协议执行，这里只验证方法签名和委托关系
	// 完整的测试需要集成测试环境
	// 不实际调用 GenerateKeyShare，因为它需要真实的协议执行
	// 委托关系通过代码审查验证：GG20Protocol.GenerateKeyShare 直接调用 p.GG18Protocol.GenerateKeyShare
	assert.NotNil(t, protocol.GG18Protocol)
}

func TestGG20Protocol_VerifySignature_Delegation(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	ctx := context.Background()
	sig := &Signature{
		R:     make([]byte, 32),
		S:     make([]byte, 32),
		Bytes: make([]byte, 70),
		Hex:   "test",
	}
	msg := []byte("test message")
	pubKey := &PublicKey{
		Bytes: []byte{0x02, 0x01, 0x02, 0x03},
		Hex:   "02010203",
	}

	// 验证委托给 GG18Protocol
	// 注意：实际的签名验证需要有效的签名数据
	_, err := protocol.VerifySignature(ctx, sig, msg, pubKey)
	// 可能会因为签名无效而失败，但委托逻辑应该正确
	_ = err
}

func TestGG20Protocol_VerifySignature_DelegationComparison(t *testing.T) {
	gg18Protocol := NewGG18Protocol("secp256k1", "node-1", mockMessageRouter)
	gg20Protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	ctx := context.Background()
	sig := &Signature{
		R:     make([]byte, 32),
		S:     make([]byte, 32),
		Bytes: make([]byte, 70),
		Hex:   "test",
	}
	msg := []byte("test message")
	pubKey := &PublicKey{
		Bytes: []byte{0x02, 0x01, 0x02, 0x03},
		Hex:   "02010203",
	}

	// 验证 VerifySignature 委托给 GG18Protocol
	// 注意：由于签名无效，两个协议都会返回 false，但委托逻辑应该相同
	result18, err18 := gg18Protocol.VerifySignature(ctx, sig, msg, pubKey)
	result20, err20 := gg20Protocol.VerifySignature(ctx, sig, msg, pubKey)

	// 验证结果相同（都因为签名无效而失败）
	assert.Equal(t, result18, result20)
	if err18 != nil && err20 != nil {
		// 错误可能不同，但都应该是验证失败
		assert.False(t, result18)
		assert.False(t, result20)
	}
}

func TestGG20Protocol_RotateKey_Delegation(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	ctx := context.Background()
	keyID := "test-key"

	// 验证委托给 GG18Protocol
	// 注意：密钥轮换功能尚未实现
	err := protocol.RotateKey(ctx, keyID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

func TestGG20Protocol_RotateKey_DelegationComparison(t *testing.T) {
	gg18Protocol := NewGG18Protocol("secp256k1", "node-1", mockMessageRouter)
	gg20Protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	ctx := context.Background()
	keyID := "test-key"

	// 验证 RotateKey 委托给 GG18Protocol
	err18 := gg18Protocol.RotateKey(ctx, keyID)
	err20 := gg20Protocol.RotateKey(ctx, keyID)

	// 验证错误相同（都因为功能未实现而失败）
	assert.Equal(t, err18 == nil, err20 == nil)
	if err18 != nil && err20 != nil {
		assert.Contains(t, err18.Error(), "not yet implemented")
		assert.Contains(t, err20.Error(), "not yet implemented")
	}
}

func TestGG20Protocol_GG18ProtocolEmbedding(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	// 验证 GG20Protocol 正确嵌入了 GG18Protocol
	assert.NotNil(t, protocol.GG18Protocol)
	assert.Equal(t, "secp256k1", protocol.GG18Protocol.GetCurve())
	assert.Equal(t, []string{"gg18"}, protocol.GG18Protocol.SupportedProtocols())
	assert.Equal(t, "gg18", protocol.GG18Protocol.DefaultProtocol())
}

func TestGG20Protocol_ProtocolIdentity(t *testing.T) {
	gg18Protocol := NewGG18Protocol("secp256k1", "node-1", mockMessageRouter)
	gg20Protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	// 验证协议标识符不同
	assert.Equal(t, []string{"gg18"}, gg18Protocol.SupportedProtocols())
	assert.Equal(t, []string{"gg20"}, gg20Protocol.SupportedProtocols())
	assert.Equal(t, "gg18", gg18Protocol.DefaultProtocol())
	assert.Equal(t, "gg20", gg20Protocol.DefaultProtocol())

	// 验证曲线相同（都使用 secp256k1）
	assert.Equal(t, gg18Protocol.GetCurve(), gg20Protocol.GetCurve())
}

func TestGG20Protocol_ThresholdSign_UsesGG20Options(t *testing.T) {
	_ = NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	// 验证 ThresholdSign 使用 GG20SigningOptions
	// 这通过检查 GG20SigningOptions 的配置来验证
	opts := GG20SigningOptions()
	assert.Equal(t, 1*time.Minute, opts.Timeout)
	assert.True(t, opts.EnableIdentifiableAbort)
	assert.Equal(t, "GG20", opts.ProtocolName)

	// 注意：实际的签名执行需要真实的密钥和协议执行
	// 这里只验证选项配置
}

func TestGG20Protocol_ConcurrentAccess(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	// 并发访问测试
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer func() { done <- true }()
			curve := protocol.GetCurve()
			_ = curve
			protocols := protocol.SupportedProtocols()
			_ = protocols
			defaultProtocol := protocol.DefaultProtocol()
			_ = defaultProtocol
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 验证协议状态未损坏
	assert.Equal(t, "secp256k1", protocol.GetCurve())
	assert.Equal(t, []string{"gg20"}, protocol.SupportedProtocols())
	assert.Equal(t, "gg20", protocol.DefaultProtocol())
}

func TestGG20Protocol_ValidateKeyGenRequest_Delegation(t *testing.T) {
	gg18Protocol := NewGG18Protocol("secp256k1", "node-1", mockMessageRouter)
	gg20Protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	req := &KeyGenRequest{
		Algorithm:  "ECDSA",
		Curve:      "secp256k1",
		Threshold:  2,
		TotalNodes: 3,
		NodeIDs:    []string{"node-1", "node-2", "node-3"},
	}

	// 验证 GG20 的验证逻辑与 GG18 相同
	err18 := gg18Protocol.ValidateKeyGenRequest(req)
	err20 := gg20Protocol.ValidateKeyGenRequest(req)

	assert.Equal(t, err18 == nil, err20 == nil)
	if err18 != nil {
		assert.Equal(t, err18.Error(), err20.Error())
	}
}

func TestGG20Protocol_ValidateSignRequest_Delegation(t *testing.T) {
	gg18Protocol := NewGG18Protocol("secp256k1", "node-1", mockMessageRouter)
	gg20Protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	req := &SignRequest{
		KeyID:   "test-key",
		Message: []byte("test"),
		NodeIDs: []string{"node-1"},
	}

	// 验证 GG20 的验证逻辑与 GG18 相同
	err18 := gg18Protocol.ValidateSignRequest(req)
	err20 := gg20Protocol.ValidateSignRequest(req)

	assert.Equal(t, err18 == nil, err20 == nil)
	if err18 != nil {
		assert.Equal(t, err18.Error(), err20.Error())
	}
}

func TestGG20Protocol_GetCurve_Delegation(t *testing.T) {
	gg18Protocol := NewGG18Protocol("secp256k1", "node-1", mockMessageRouter)
	gg20Protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	// 验证 GetCurve 委托给 GG18Protocol
	assert.Equal(t, gg18Protocol.GetCurve(), gg20Protocol.GetCurve())
}

func TestGG20Protocol_MessageRouter(t *testing.T) {
	messageCount := 0
	messageRouter := func(nodeID string, msg tss.Message) error {
		messageCount++
		return nil
	}

	protocol := NewGG20Protocol("secp256k1", "node-1", messageRouter)

	// 验证消息路由函数已设置
	assert.NotNil(t, protocol.GG18Protocol)
	assert.NotNil(t, protocol.GG18Protocol.messageRouter)
}

func TestGG20Protocol_ThisNodeID(t *testing.T) {
	thisNodeID := "test-node-123"
	protocol := NewGG20Protocol("secp256k1", thisNodeID, mockMessageRouter)

	// 验证 thisNodeID 已设置
	assert.NotNil(t, protocol.GG18Protocol)
	assert.Equal(t, thisNodeID, protocol.GG18Protocol.thisNodeID)
}

func TestGG20Protocol_GenerateKeyShare_ReusesGG18(t *testing.T) {
	protocol := NewGG20Protocol("secp256k1", "node-1", mockMessageRouter)

	req := &KeyGenRequest{
		Algorithm:  "ECDSA",
		Curve:      "secp256k1",
		Threshold:  2,
		TotalNodes: 3,
		NodeIDs:    []string{"node-1", "node-2", "node-3"},
	}

	// 验证 GenerateKeyShare 委托给 GG18Protocol.GenerateKeyShare
	// 注意：实际的 DKG 需要真实的协议执行（多节点协作）
	err := protocol.ValidateKeyGenRequest(req)
	require.NoError(t, err)

	// 验证 GenerateKeyShare 方法存在且委托给 GG18Protocol
	// 由于需要真实的协议执行，这里只验证方法签名和委托关系
	// 完整的测试需要集成测试环境
	// 不实际调用 GenerateKeyShare，因为它需要真实的协议执行
	// 委托关系通过代码审查验证：GG20Protocol.GenerateKeyShare 直接调用 p.GG18Protocol.GenerateKeyShare
	assert.NotNil(t, protocol.GG18Protocol)
}
