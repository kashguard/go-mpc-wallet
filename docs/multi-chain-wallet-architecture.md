# 多链钱包管理架构设计文档

**版本**: v1.0  
**创建日期**: 2025-12-10  
**状态**: 设计阶段

---

## 目录

- [1. 概述](#1-概述)
- [2. 核心设计原则](#2-核心设计原则)
- [3. 架构设计](#3-架构设计)
- [4. 关键组件设计](#4-关键组件设计)
- [5. 数据库设计](#5-数据库设计)
- [6. API 设计](#6-api-设计)
- [7. 扩展性设计](#7-扩展性设计)
- [8. 实施路线图](#8-实施路线图)
- [9. 关键设计决策](#9-关键设计决策)

---

## 1. 概述

### 1.1 设计目标

设计一个支持大部分主流区块链的钱包管理系统，实现：

- 🌐 **多链支持**：统一管理 Bitcoin、Ethereum 及所有主流区块链
- 🔄 **智能协议选择**：根据链类型自动选择最适合的 TSS 协议
- 🔧 **灵活扩展**：易于添加新链支持
- 🎯 **用户友好**：简化用户操作，同时保持高级配置能力

### 1.2 核心价值

- **统一接口**：一套 API 管理所有链的密钥和签名
- **自动优化**：根据链特性自动选择最优协议
- **向后兼容**：支持现有密钥和配置
- **企业级**：完整的审计、策略和合规支持

---

## 2. 核心设计原则

### 2.1 链类型与协议映射策略

不同区块链需要不同的签名算法和协议。以下是推荐的映射关系：

| 链类型 | 签名算法 | 推荐协议 | 曲线 | 地址格式 | 说明 |
|--------|---------|---------|------|---------|------|
| **Bitcoin (Legacy)** | ECDSA | GG20 | secp256k1 | P2PKH (Base58) | 传统比特币地址 |
| **Bitcoin (Taproot)** | Schnorr | FROST | secp256k1 | Bech32m | Taproot 地址（BIP-340） |
| **Ethereum** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | 以太坊主网 |
| **BSC** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | 币安智能链 |
| **Avalanche C-Chain** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | Avalanche C链 |
| **Polygon** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | Polygon 网络 |
| **Arbitrum** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | Arbitrum L2 |
| **Optimism** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | Optimism L2 |
| **Base** | ECDSA | GG20 | secp256k1 | 0x... (Keccak256) | Base L2 |
| **Solana** | EdDSA | FROST | Ed25519 | Base58 | Solana 网络 |
| **Cosmos** | ECDSA | GG20 | secp256k1 | Bech32 | Cosmos Hub |
| **Osmosis** | ECDSA | GG20 | secp256k1 | Bech32 | Osmosis DEX |
| **Terra** | ECDSA | GG20 | secp256k1 | Bech32 | Terra 网络 |

### 2.2 协议选择策略

#### 自动选择规则

```go
// 链类型到协议的映射规则
func GetProtocolForChain(chainType string) string {
    chainProtocolMap := map[string]string{
        // EVM 兼容链（统一使用 ECDSA）
        "ethereum":     "gg20",
        "bsc":          "gg20",
        "avalanche":    "gg20",
        "polygon":      "gg20",
        "arbitrum":     "gg20",
        "optimism":     "gg20",
        "base":         "gg20",
        
        // Bitcoin 系列
        "bitcoin":      "gg20",  // 传统地址，Taproot 使用 FROST
        
        // Cosmos 生态
        "cosmos":       "gg20",
        "osmosis":      "gg20",
        "terra":        "gg20",
        
        // EdDSA 链
        "solana":       "frost",
    }
    
    if protocol, ok := chainProtocolMap[chainType]; ok {
        return protocol
    }
    return "gg20" // 默认使用 GG20
}
```

#### 协议选择优先级

1. **用户显式指定**：如果请求中指定了 `protocol`，优先使用
2. **链类型映射**：根据 `chain_type` 自动选择推荐协议
3. **全局默认**：使用系统默认协议（GG20）

#### 协议兼容性验证

在创建密钥时，验证协议与链类型的兼容性：

```go
// 协议与链类型的兼容性矩阵
var protocolChainCompatibility = map[string][]string{
    "gg18":  {"bitcoin", "ethereum", "bsc", "avalanche", "polygon", "cosmos"},
    "gg20":  {"bitcoin", "ethereum", "bsc", "avalanche", "polygon", "cosmos"},
    "frost": {"bitcoin", "solana"}, // FROST 支持 Schnorr 和 EdDSA
}
```

---

## 3. 架构设计

### 3.1 分层架构图

```
┌─────────────────────────────────────────────────────────┐
│                    API Layer                            │
│  POST /v1/mpc/keys (chain_type + protocol可选)          │
│  POST /v1/mpc/keys/{keyId}/address                     │
│  POST /v1/mpc/sign (自动选择协议)                       │
│  GET  /v1/mpc/keys (支持按chain_type过滤)              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              Chain Adapter Layer                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Bitcoin  │  │ Ethereum │  │  Solana  │  ...        │
│  │ Adapter │  │ Adapter  │  │ Adapter  │            │
│  └──────────┘  └──────────┘  └──────────┘            │
│                                                         │
│  ChainAdapterFactory (统一管理所有链适配器)              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│           Protocol Selection Layer                      │
│  ┌──────────────────────────────────────┐              │
│  │  Protocol Registry                   │              │
│  │  - GG18 (ECDSA/secp256k1)            │              │
│  │  - GG20 (ECDSA/secp256k1)            │              │
│  │  - FROST (Schnorr/EdDSA)             │              │
│  │                                       │              │
│  │  Chain → Protocol Mapping            │              │
│  └──────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              TSS Protocol Layer                         │
│  (tss-lib: keygen, signing)                            │
└─────────────────────────────────────────────────────────┘
```

### 3.2 数据流设计

#### 创建密钥流程

```
1. API 接收请求 (chain_type, protocol可选)
   ↓
2. Protocol Selection Layer
   - 如果 protocol 未指定，根据 chain_type 自动选择
   - 验证协议与链类型的兼容性
   ↓
3. Protocol Registry
   - 获取对应的协议引擎实例
   ↓
4. TSS Protocol Layer
   - 执行 DKG 生成密钥分片
   ↓
5. Chain Adapter Layer
   - 根据 chain_type 获取适配器
   - 生成区块链地址
   ↓
6. 保存元数据
   - 保存 key_id, public_key, protocol, chain_type, address
```

#### 签名流程

```
1. API 接收签名请求 (key_id, message)
   ↓
2. 从数据库获取密钥元数据
   - 读取 protocol, chain_type
   ↓
3. Protocol Registry
   - 根据 protocol 获取对应的协议引擎
   ↓
4. Chain Adapter
   - 根据 chain_type 构建交易（如果需要）
   ↓
5. TSS Protocol Layer
   - 执行阈值签名
   ↓
6. Chain Adapter
   - 格式化签名（根据链类型）
   ↓
7. 返回签名结果
```

---

## 4. 关键组件设计

### 4.1 Chain Adapter Factory（链适配器工厂）

**位置**: `internal/mpc/chain/factory.go`

**职责**:
- 统一管理所有链适配器
- 根据链类型返回对应的适配器实例
- 支持链配置管理

**设计**:

```go
package chain

import (
    "fmt"
    "math/big"
    "sync"
    
    "github.com/btcsuite/btcd/chaincfg"
)

// ChainAdapterFactory 链适配器工厂
type ChainAdapterFactory struct {
    mu       sync.RWMutex
    adapters map[string]Adapter
    configs  map[string]*ChainConfig
}

// NewChainAdapterFactory 创建链适配器工厂
func NewChainAdapterFactory() *ChainAdapterFactory {
    factory := &ChainAdapterFactory{
        adapters: make(map[string]Adapter),
        configs:  make(map[string]*ChainConfig),
    }
    
    // 初始化所有链适配器
    factory.initAdapters()
    
    return factory
}

// initAdapters 初始化所有链适配器
func (f *ChainAdapterFactory) initAdapters() {
    // Bitcoin
    f.Register("bitcoin", NewBitcoinAdapter(&chaincfg.MainNetParams))
    
    // EVM 兼容链
    f.Register("ethereum", NewEthereumAdapter(big.NewInt(1)))
    f.Register("bsc", NewEthereumAdapter(big.NewInt(56)))
    f.Register("avalanche", NewEthereumAdapter(big.NewInt(43114)))
    f.Register("polygon", NewEthereumAdapter(big.NewInt(137)))
    f.Register("arbitrum", NewEthereumAdapter(big.NewInt(42161)))
    f.Register("optimism", NewEthereumAdapter(big.NewInt(10)))
    f.Register("base", NewEthereumAdapter(big.NewInt(8453)))
    
    // 其他链
    // f.Register("solana", NewSolanaAdapter())
    // f.Register("cosmos", NewCosmosAdapter())
}

// Register 注册链适配器
func (f *ChainAdapterFactory) Register(chainType string, adapter Adapter) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.adapters[chainType] = adapter
}

// GetAdapter 获取链适配器
func (f *ChainAdapterFactory) GetAdapter(chainType string) (Adapter, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()
    
    adapter, ok := f.adapters[chainType]
    if !ok {
        return nil, fmt.Errorf("unsupported chain type: %s", chainType)
    }
    return adapter, nil
}

// ListSupportedChains 列出所有支持的链
func (f *ChainAdapterFactory) ListSupportedChains() []string {
    f.mu.RLock()
    defer f.mu.RUnlock()
    
    chains := make([]string, 0, len(f.adapters))
    for chainType := range f.adapters {
        chains = append(chains, chainType)
    }
    return chains
}
```

### 4.2 Protocol Registry（协议注册表）

**位置**: `internal/mpc/protocol/registry.go`

**职责**:
- 管理所有协议引擎实例
- 提供链类型到协议的映射
- 根据协议名称或链类型返回对应的引擎

**设计**:

```go
package protocol

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/kashguard/go-mpc-wallet/internal/config"
    mpcgrpc "github.com/kashguard/go-mpc-wallet/internal/mpc/grpc"
    "github.com/kashguard/tss-lib/tss"
)

// ProtocolRegistry 协议注册表
type ProtocolRegistry struct {
    mu                sync.RWMutex
    protocols         map[string]Engine
    defaultProtocol   string
    chainProtocolMap  map[string]string // 链类型到协议的映射
    messageRouter     func(sessionID string, nodeID string, msg tss.Message, isBroadcast bool) error
}

// NewProtocolRegistry 创建协议注册表
func NewProtocolRegistry(
    cfg config.Server,
    grpcClient *mpcgrpc.GRPCClient,
) *ProtocolRegistry {
    curve := "secp256k1"
    thisNodeID := cfg.MPC.NodeID
    if thisNodeID == "" {
        thisNodeID = "default-node"
    }
    
    // 创建消息路由器
    messageRouter := func(sessionID string, nodeID string, msg tss.Message, isBroadcast bool) error {
        ctx := context.Background()
        if len(sessionID) > 0 && sessionID[:4] == "key-" {
            return grpcClient.SendKeygenMessage(ctx, nodeID, msg, sessionID, isBroadcast)
        } else {
            return grpcClient.SendSigningMessage(ctx, nodeID, msg, sessionID)
        }
    }
    
    registry := &ProtocolRegistry{
        protocols: make(map[string]Engine),
        chainProtocolMap: map[string]string{
            // EVM 兼容链
            "ethereum":     "gg20",
            "bsc":          "gg20",
            "avalanche":    "gg20",
            "polygon":      "gg20",
            "arbitrum":     "gg20",
            "optimism":     "gg20",
            "base":         "gg20",
            
            // Bitcoin 系列
            "bitcoin":      "gg20", // 传统地址，Taproot 使用 FROST
            
            // Cosmos 生态
            "cosmos":       "gg20",
            "osmosis":      "gg20",
            "terra":        "gg20",
            
            // EdDSA 链
            "solana":       "frost",
        },
        messageRouter: messageRouter,
    }
    
    // 注册所有协议实例
    registry.Register("gg18", NewGG18Protocol(curve, thisNodeID, messageRouter))
    registry.Register("gg20", NewGG20Protocol(curve, thisNodeID, messageRouter))
    registry.Register("frost", NewFROSTProtocol(curve, thisNodeID, messageRouter))
    
    // 设置默认协议
    defaultProtocol := cfg.MPC.DefaultProtocol
    if defaultProtocol == "" {
        defaultProtocol = "gg20"
    }
    registry.SetDefault(defaultProtocol)
    
    return registry
}

// Register 注册协议引擎
func (r *ProtocolRegistry) Register(name string, engine Engine) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.protocols[name] = engine
}

// GetEngine 根据协议名称获取引擎
func (r *ProtocolRegistry) GetEngine(protocol string) (Engine, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    engine, ok := r.protocols[protocol]
    if !ok {
        return nil, fmt.Errorf("protocol %s not found", protocol)
    }
    return engine, nil
}

// GetProtocolForChain 根据链类型获取推荐协议
func (r *ProtocolRegistry) GetProtocolForChain(chainType string) string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    if protocol, ok := r.chainProtocolMap[chainType]; ok {
        return protocol
    }
    return r.defaultProtocol
}

// SetDefault 设置默认协议
func (r *ProtocolRegistry) SetDefault(name string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.defaultProtocol = name
}

// GetDefault 获取默认协议引擎
func (r *ProtocolRegistry) GetDefault() (Engine, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    if r.defaultProtocol == "" {
        return nil, fmt.Errorf("no default protocol set")
    }
    return r.GetEngine(r.defaultProtocol)
}

// ListProtocols 列出所有支持的协议
func (r *ProtocolRegistry) ListProtocols() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    protocols := make([]string, 0, len(r.protocols))
    for name := range r.protocols {
        protocols = append(protocols, name)
    }
    return protocols
}

// ValidateProtocolForChain 验证协议与链类型的兼容性
func (r *ProtocolRegistry) ValidateProtocolForChain(protocol, chainType string) error {
    // 协议与链类型的兼容性矩阵
    compatibility := map[string][]string{
        "gg18":  {"bitcoin", "ethereum", "bsc", "avalanche", "polygon", "cosmos"},
        "gg20":  {"bitcoin", "ethereum", "bsc", "avalanche", "polygon", "cosmos"},
        "frost": {"bitcoin", "solana"}, // FROST 支持 Schnorr 和 EdDSA
    }
    
    supportedChains, ok := compatibility[protocol]
    if !ok {
        return fmt.Errorf("unknown protocol: %s", protocol)
    }
    
    for _, chain := range supportedChains {
        if chain == chainType {
            return nil
        }
    }
    
    return fmt.Errorf("protocol %s is not compatible with chain type %s", protocol, chainType)
}
```

### 4.3 Chain Config（链配置管理）

**位置**: `internal/mpc/chain/config.go`

**设计**:

```go
package chain

import (
    "math/big"
)

// ChainConfig 链配置
type ChainConfig struct {
    ChainType    string   // 链类型标识
    ChainID      *big.Int // 链ID（EVM链需要）
    Protocol     string   // 推荐协议
    Curve        string   // 曲线类型
    Algorithm    string   // 签名算法
    AddressType  string   // 地址格式：P2PKH, Bech32, 0x...
    Network      string   // 网络类型：mainnet, testnet
    Description  string   // 链描述
}

// ChainConfigs 预定义的链配置
var ChainConfigs = map[string]*ChainConfig{
    "ethereum": {
        ChainType:   "ethereum",
        ChainID:     big.NewInt(1),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Ethereum Mainnet",
    },
    "bsc": {
        ChainType:   "bsc",
        ChainID:     big.NewInt(56),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Binance Smart Chain",
    },
    "avalanche": {
        ChainType:   "avalanche",
        ChainID:     big.NewInt(43114),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Avalanche C-Chain",
    },
    "polygon": {
        ChainType:   "polygon",
        ChainID:     big.NewInt(137),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Polygon Network",
    },
    "arbitrum": {
        ChainType:   "arbitrum",
        ChainID:     big.NewInt(42161),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Arbitrum One",
    },
    "optimism": {
        ChainType:   "optimism",
        ChainID:     big.NewInt(10),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Optimism",
    },
    "base": {
        ChainType:   "base",
        ChainID:     big.NewInt(8453),
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "0x",
        Network:     "mainnet",
        Description: "Base",
    },
    "bitcoin": {
        ChainType:   "bitcoin",
        ChainID:     nil,
        Protocol:    "gg20", // 或 "frost" for Taproot
        Curve:       "secp256k1",
        Algorithm:   "ECDSA", // 或 "Schnorr" for Taproot
        AddressType: "P2PKH", // 或 "Bech32m" for Taproot
        Network:     "mainnet",
        Description: "Bitcoin Mainnet",
    },
    "solana": {
        ChainType:   "solana",
        ChainID:     nil,
        Protocol:    "frost",
        Curve:       "Ed25519",
        Algorithm:   "EdDSA",
        AddressType: "Base58",
        Network:     "mainnet",
        Description: "Solana Mainnet",
    },
    "cosmos": {
        ChainType:   "cosmos",
        ChainID:     nil,
        Protocol:    "gg20",
        Curve:       "secp256k1",
        Algorithm:   "ECDSA",
        AddressType: "Bech32",
        Network:     "mainnet",
        Description: "Cosmos Hub",
    },
}

// GetChainConfig 获取链配置
func GetChainConfig(chainType string) (*ChainConfig, error) {
    config, ok := ChainConfigs[chainType]
    if !ok {
        return nil, fmt.Errorf("unsupported chain type: %s", chainType)
    }
    return config, nil
}
```

### 4.4 Key Service 增强

**修改**: `internal/mpc/key/service.go`

**关键改动**:

1. 添加 `ProtocolRegistry` 依赖
2. 在 `CreateKey` 中实现协议自动选择
3. 保存 `protocol` 字段到元数据

**设计要点**:

```go
// Service 密钥服务
type Service struct {
    metadataStore     storage.MetadataStore
    keyShareStorage   storage.KeyShareStorage
    protocolRegistry  *protocol.ProtocolRegistry // 新增
    chainFactory      *chain.ChainAdapterFactory // 新增
    dkgService        *DKGService
}

// CreateKey 创建密钥（执行DKG）
func (s *Service) CreateKey(ctx context.Context, req *CreateKeyRequest) (*KeyMetadata, error) {
    // 1. 如果没有指定协议，根据链类型自动选择
    if req.Protocol == "" {
        req.Protocol = s.protocolRegistry.GetProtocolForChain(req.ChainType)
        log.Info().
            Str("chain_type", req.ChainType).
            Str("auto_selected_protocol", req.Protocol).
            Msg("Protocol auto-selected based on chain type")
    }
    
    // 2. 验证协议与链类型的兼容性
    if err := s.protocolRegistry.ValidateProtocolForChain(req.Protocol, req.ChainType); err != nil {
        return nil, errors.Wrapf(err, "protocol %s is not compatible with chain type %s", req.Protocol, req.ChainType)
    }
    
    // 3. 根据协议选择对应的引擎
    protocolEngine, err := s.protocolRegistry.GetEngine(req.Protocol)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to get protocol engine for %s", req.Protocol)
    }
    
    // 4. 执行DKG（使用选定的协议引擎）
    // ... DKG 执行逻辑 ...
    
    // 5. 生成地址
    chainAdapter, err := s.chainFactory.GetAdapter(req.ChainType)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to get chain adapter for %s", req.ChainType)
    }
    
    pubKeyBytes, err := hex.DecodeString(dkgResp.PublicKey.Hex)
    if err != nil {
        return nil, errors.Wrap(err, "failed to decode public key")
    }
    
    address, err := chainAdapter.GenerateAddress(pubKeyBytes)
    if err != nil {
        return nil, errors.Wrap(err, "failed to generate address")
    }
    
    // 6. 保存密钥元数据（包含protocol字段）
    keyMetadata := &KeyMetadata{
        KeyID:       keyID,
        PublicKey:   dkgResp.PublicKey.Hex,
        Algorithm:   req.Algorithm,
        Curve:       req.Curve,
        Protocol:    req.Protocol, // 保存协议类型
        ChainType:   req.ChainType,
        Address:     address,      // 保存生成的地址
        Threshold:   req.Threshold,
        TotalNodes:  req.TotalNodes,
        Status:      "Active",
        Description: req.Description,
        Tags:        req.Tags,
        CreatedAt:   now,
        UpdatedAt:   now,
    }
    
    // ... 保存逻辑 ...
}
```

### 4.5 Signing Service 增强

**修改**: `internal/mpc/signing/service.go`

**关键改动**:

1. 从密钥元数据中读取 `protocol`
2. 根据 `protocol` 选择对应的协议引擎进行签名

**设计要点**:

```go
// Sign 执行阈值签名
func (s *SigningService) Sign(ctx context.Context, req *SignRequest) (*SignResponse, error) {
    // 1. 获取密钥元数据
    keyMetadata, err := s.keyService.GetKey(ctx, req.KeyID)
    if err != nil {
        return nil, errors.Wrap(err, "failed to get key")
    }
    
    // 2. 根据密钥的protocol选择对应的协议引擎
    protocolEngine, err := s.protocolRegistry.GetEngine(keyMetadata.Protocol)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to get protocol engine for %s", keyMetadata.Protocol)
    }
    
    // 3. 根据链类型获取适配器（用于构建交易）
    chainAdapter, err := s.chainFactory.GetAdapter(keyMetadata.ChainType)
    if err != nil {
        return nil, errors.Wrapf(err, "failed to get chain adapter for %s", keyMetadata.ChainType)
    }
    
    // 4. 构建交易（如果需要）
    var message []byte
    if req.Transaction != nil {
        tx, err := chainAdapter.BuildTransaction(req.Transaction)
        if err != nil {
            return nil, errors.Wrap(err, "failed to build transaction")
        }
        message = []byte(tx.Hash)
    } else {
        message = req.Message
    }
    
    // 5. 执行阈值签名
    signReq := &protocol.SignRequest{
        KeyID:      req.KeyID,
        Message:    message,
        MessageHex: hex.EncodeToString(message),
        NodeIDs:    req.NodeIDs,
    }
    
    signResp, err := protocolEngine.ThresholdSign(ctx, sessionID, signReq)
    if err != nil {
        return nil, errors.Wrap(err, "failed to execute threshold signing")
    }
    
    // 6. 格式化签名（根据链类型）
    // 不同链的签名格式可能不同，需要适配器处理
    
    return &SignResponse{
        Signature: signResp.Signature,
        KeyID:     req.KeyID,
        // ...
    }, nil
}
```

---

## 5. 数据库设计

### 5.1 Keys 表增强

在现有的 `keys` 表中添加 `protocol` 字段：

```sql
-- 添加 protocol 字段
ALTER TABLE keys ADD COLUMN protocol VARCHAR(50) NOT NULL DEFAULT 'gg20';

-- 添加索引
CREATE INDEX idx_keys_protocol ON keys(protocol);
CREATE INDEX idx_keys_chain_type_protocol ON keys(chain_type, protocol);

-- 更新现有记录的 protocol（如果为空）
UPDATE keys SET protocol = 'gg20' WHERE protocol IS NULL OR protocol = '';
```

### 5.2 数据库迁移文件

创建迁移文件：`migrations/YYYYMMDDHHMMSS-add_protocol_to_keys.sql`

```sql
-- +migrate Up
ALTER TABLE keys ADD COLUMN protocol VARCHAR(50) NOT NULL DEFAULT 'gg20';
CREATE INDEX idx_keys_protocol ON keys(protocol);
CREATE INDEX idx_keys_chain_type_protocol ON keys(chain_type, protocol);

-- +migrate Down
DROP INDEX IF EXISTS idx_keys_chain_type_protocol;
DROP INDEX IF EXISTS idx_keys_protocol;
ALTER TABLE keys DROP COLUMN IF EXISTS protocol;
```

### 5.3 Storage 接口更新

**修改**: `internal/mpc/storage/interface.go`

```go
// KeyMetadata 密钥元数据
type KeyMetadata struct {
    KeyID        string
    PublicKey    string
    Algorithm    string
    Curve        string
    Protocol     string    // 新增字段
    Threshold    int
    TotalNodes   int
    ChainType    string
    Address      string
    Status       string
    Description  string
    Tags         map[string]string
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletionDate *time.Time
}
```

---

## 6. API 设计

### 6.1 创建密钥 API

**路径**: `POST /api/v1/mpc/keys`

**请求体**:

```yaml
# api/definitions/mpc.yml
PostCreateKeyPayload:
  type: object
  required: [algorithm, curve, threshold, total_nodes, chain_type]
  properties:
    algorithm:
      type: string
      enum: [ECDSA, EdDSA]
      example: ECDSA
    curve:
      type: string
      enum: [secp256k1, secp256r1, Ed25519]
      example: secp256k1
    threshold:
      type: integer
      minimum: 2
      example: 2
    total_nodes:
      type: integer
      minimum: 2
      example: 3
    chain_type:
      type: string
      enum: [bitcoin, ethereum, bsc, avalanche, polygon, arbitrum, optimism, base, solana, cosmos]
      example: ethereum
      description: "区块链类型"
    protocol:  # 新增字段（可选）
      type: string
      enum: [gg18, gg20, frost]
      example: gg20
      description: "TSS协议类型，如果不指定则根据chain_type自动选择"
    description:
      type: string
      example: "企业多签钱包密钥"
    tags:
      type: object
      additionalProperties:
        type: string
```

**响应体**:

```yaml
CreateKeyResponse:
  type: object
  required: [key_id, public_key, algorithm, curve, protocol, threshold, total_nodes, chain_type, status, address]
  properties:
    key_id:
      type: string
      example: "key-1234567890abcdef"
    public_key:
      type: string
      example: "03f685d21e54d7346a57f4a60f68e013345f04072da2b72164df16d71d8c26b45e"
    algorithm:
      type: string
      example: "ECDSA"
    curve:
      type: string
      example: "secp256k1"
    protocol:  # 新增字段
      type: string
      example: "gg20"
      description: "使用的TSS协议"
    threshold:
      type: integer
      example: 2
    total_nodes:
      type: integer
      example: 3
    chain_type:
      type: string
      example: "ethereum"
    address:  # 新增字段
      type: string
      example: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
      description: "生成的区块链地址"
    status:
      type: string
      enum: [Active, Inactive, PendingDeletion, Deleted]
      example: "Active"
    description:
      type: string
    tags:
      type: object
      additionalProperties:
        type: string
    created_at:
      type: string
      format: date-time
```

### 6.2 使用示例

#### 示例1：创建以太坊密钥（自动选择GG20）

```bash
curl -X POST http://localhost:8080/api/v1/mpc/keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "algorithm": "ECDSA",
    "curve": "secp256k1",
    "threshold": 2,
    "total_nodes": 3,
    "chain_type": "ethereum"
  }'
```

**响应**:
```json
{
  "key_id": "key-abc123...",
  "public_key": "03f685d21e54d7346a57f4a60f68e013345f04072da2b72164df16d71d8c26b45e",
  "algorithm": "ECDSA",
  "curve": "secp256k1",
  "protocol": "gg20",
  "chain_type": "ethereum",
  "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
  "status": "Active"
}
```

#### 示例2：创建比特币Taproot密钥（显式指定FROST）

```bash
curl -X POST http://localhost:8080/api/v1/mpc/keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "algorithm": "ECDSA",
    "curve": "secp256k1",
    "threshold": 2,
    "total_nodes": 3,
    "chain_type": "bitcoin",
    "protocol": "frost"
  }'
```

#### 示例3：创建Solana密钥（自动选择FROST）

```bash
curl -X POST http://localhost:8080/api/v1/mpc/keys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "algorithm": "EdDSA",
    "curve": "Ed25519",
    "threshold": 2,
    "total_nodes": 3,
    "chain_type": "solana"
  }'
```

### 6.3 查询密钥 API（支持按链类型过滤）

**路径**: `GET /api/v1/mpc/keys?chain_type=ethereum`

**查询参数**:
- `chain_type`: 按链类型过滤
- `protocol`: 按协议类型过滤
- `status`: 按状态过滤
- `limit`: 分页大小
- `offset`: 分页偏移

---

## 7. 扩展性设计

### 7.1 添加新链的步骤

#### 步骤1：实现 Chain Adapter

```go
// internal/mpc/chain/polygon.go
package chain

import (
    "math/big"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/rlp"
)

// PolygonAdapter Polygon 网络适配器（复用EVM逻辑）
type PolygonAdapter struct {
    *EthereumAdapter // 复用EVM链逻辑
}

// NewPolygonAdapter 创建Polygon适配器
func NewPolygonAdapter() *PolygonAdapter {
    return &PolygonAdapter{
        EthereumAdapter: NewEthereumAdapter(big.NewInt(137)),
    }
}
```

#### 步骤2：注册到 Factory

```go
// 在 NewChainAdapterFactory 的 initAdapters 方法中
f.Register("polygon", NewPolygonAdapter())
```

#### 步骤3：配置协议映射（如果需要）

```go
// 在 ProtocolRegistry 的 chainProtocolMap 中
"polygon": "gg20",
```

#### 步骤4：更新 API 定义

```yaml
# api/definitions/mpc.yml
chain_type:
  enum: [bitcoin, ethereum, bsc, avalanche, polygon, ...]
```

### 7.2 链分类策略

#### EVM 兼容链（统一处理）

所有 EVM 兼容链可以复用 `EthereumAdapter`，只需配置不同的 `chainID`：

```go
evmChains := map[string]*big.Int{
    "ethereum":  big.NewInt(1),
    "bsc":       big.NewInt(56),
    "avalanche": big.NewInt(43114),
    "polygon":   big.NewInt(137),
    "arbitrum":  big.NewInt(42161),
    "optimism":  big.NewInt(10),
    "base":      big.NewInt(8453),
}

for chainType, chainID := range evmChains {
    f.Register(chainType, NewEthereumAdapter(chainID))
}
```

#### Bitcoin 系列（需要特殊处理）

Bitcoin 系列链需要支持多种地址格式：
- P2PKH (Legacy)
- P2SH (SegWit)
- Bech32 (Native SegWit)
- Bech32m (Taproot)

可以通过 `address_type` 参数指定：

```go
// BitcoinAdapter 支持多种地址格式
func (a *BitcoinAdapter) GenerateAddress(pubKey []byte, addressType string) (string, error) {
    switch addressType {
    case "P2PKH":
        return a.generateP2PKHAddress(pubKey)
    case "P2SH":
        return a.generateP2SHAddress(pubKey)
    case "Bech32":
        return a.generateBech32Address(pubKey)
    case "Bech32m":
        return a.generateBech32mAddress(pubKey)
    default:
        return a.generateP2PKHAddress(pubKey) // 默认
    }
}
```

### 7.3 协议扩展

如果需要添加新的 TSS 协议：

1. 实现协议接口：`internal/mpc/protocol/new_protocol.go`
2. 注册到 ProtocolRegistry
3. 配置链类型映射（如果需要）

---

## 8. 实施路线图

### 阶段1：基础架构（当前状态）

**已完成**:
- ✅ Chain Adapter 接口定义
- ✅ Bitcoin 和 Ethereum 适配器实现
- ✅ 多协议支持（GG18/GG20/FROST）
- ✅ 密钥创建和存储

**待完成**:
- ⏳ Protocol Registry 实现
- ⏳ Chain Adapter Factory 实现
- ⏳ 协议自动选择逻辑

### 阶段2：协议自动选择（优先级：高）

**任务清单**:
1. 实现 `ProtocolRegistry`（`internal/mpc/protocol/registry.go`）
2. 实现 `ChainAdapterFactory`（`internal/mpc/chain/factory.go`）
3. 修改 `Key Service` 支持协议自动选择
4. 添加 `protocol` 字段到数据库
5. 更新 API 定义，添加 `protocol` 字段
6. 实现协议兼容性验证

**预计工作量**: 2-3 天

### 阶段3：扩展链支持（优先级：中）

**任务清单**:
1. 添加更多 EVM 链适配器（BSC, Polygon, Arbitrum, Optimism, Base）
2. 实现 Solana 适配器（EdDSA/Ed25519）
3. 实现 Cosmos 适配器（Bech32地址）
4. 支持 Bitcoin 多种地址格式（P2PKH, Bech32, Taproot）

**预计工作量**: 3-5 天

### 阶段4：高级功能（优先级：低）

**任务清单**:
1. 支持测试网/主网切换
2. 支持自定义链配置
3. 支持多地址格式（同一密钥生成不同格式地址）
4. 链配置管理界面

**预计工作量**: 5-7 天

---

## 9. 关键设计决策

### 9.1 协议选择策略

**决策**: 智能自动选择 + 用户可覆盖

**理由**:
- 简化用户体验：大多数用户不需要了解协议细节
- 保持灵活性：高级用户可以根据需求指定协议
- 降低错误率：自动选择减少配置错误

**实现**:
```go
// 优先级：用户指定 > 链类型映射 > 全局默认
if req.Protocol != "" {
    protocol = req.Protocol
} else {
    protocol = registry.GetProtocolForChain(req.ChainType)
}
```

### 9.2 密钥复用策略

**决策**: 同一密钥可用于同一曲线/算法的不同链

**理由**:
- 减少密钥数量：一个 secp256k1 密钥可用于 Bitcoin 和 Ethereum
- 简化管理：用户不需要为每条链创建独立密钥
- 降低成本：减少 DKG 执行次数

**限制**:
- 不同曲线/算法的链不能复用（如 Solana 需要独立的 Ed25519 密钥）

### 9.3 地址生成时机

**决策**: 创建密钥时自动生成地址

**理由**:
- 用户体验好：创建密钥后立即获得地址
- 减少API调用：不需要额外的地址生成请求
- 数据一致性：地址与密钥一起保存

**备选方案**:
- 按需生成：支持同一密钥生成多种地址格式（Bitcoin P2PKH/Taproot）

### 9.4 链适配器设计

**决策**: 使用适配器模式，统一接口

**理由**:
- 易于扩展：添加新链只需实现 Adapter 接口
- 代码复用：EVM 链可以复用相同逻辑
- 测试友好：可以轻松 mock 适配器

**接口设计**:
```go
type Adapter interface {
    GenerateAddress(pubKey []byte) (string, error)
    BuildTransaction(req *BuildTxRequest) (*Transaction, error)
}
```

### 9.5 协议注册表设计

**决策**: 单例模式，包含所有协议实例

**理由**:
- 性能优化：避免重复创建协议实例
- 状态管理：协议实例可以维护内部状态
- 资源效率：减少内存占用

**实现**:
```go
// 在应用启动时创建一次
protocolRegistry := NewProtocolRegistry(cfg, grpcClient)

// 在需要时获取对应的协议引擎
engine, err := protocolRegistry.GetEngine("gg20")
```

---

## 10. 技术债务和注意事项

### 10.1 当前限制

1. **FROST 协议限制**:
   - 当前实现使用 Ed25519，不支持 secp256k1 上的 Schnorr
   - 需要修改 FROST 实现以支持 Bitcoin Taproot

2. **地址格式支持**:
   - Bitcoin 目前只支持 P2PKH
   - 需要扩展支持 Bech32 和 Taproot

3. **交易构建**:
   - 当前实现较简单，需要完善各链的交易构建逻辑

### 10.2 未来优化方向

1. **性能优化**:
   - 协议引擎实例池化
   - 地址生成缓存

2. **功能增强**:
   - 支持多地址格式
   - 支持测试网/主网切换
   - 支持自定义链配置

3. **监控和可观测性**:
   - 协议选择统计
   - 链类型使用统计
   - 性能指标收集

---

## 11. 参考资源

### 11.1 相关文档

- [MPC 详细设计文档](./mpc-detailed-design.md)
- [TSS 协议对比分析](./tss-protocol-comparison.md)

### 11.2 外部资源

- [Bitcoin BIP-340: Schnorr Signatures](https://github.com/bitcoin/bips/blob/master/bip-0340.mediawiki)
- [Ethereum EIP-191: Signed Data Standard](https://eips.ethereum.org/EIPS/eip-191)
- [FROST Protocol Specification](https://datatracker.ietf.org/doc/draft-irtf-cfrg-frost/)

---

## 12. 版本历史

| 版本 | 日期 | 作者 | 说明 |
|------|------|------|------|
| v1.0 | 2025-12-10 | System | 初始版本，多链钱包管理架构设计 |

---

**文档状态**: 设计阶段  
**下一步**: 等待实施阶段2（协议自动选择）

