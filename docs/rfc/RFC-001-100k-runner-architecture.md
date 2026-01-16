# RFC-001: 10 万 Runner 规模架构设计

| 属性 | 值 |
|------|-----|
| **状态** | Draft |
| **作者** | AgentsMesh Team |
| **创建日期** | 2026-01-09 |
| **目标** | 支撑 100,000 个并发 Runner 连接 |

---

## 1. 概述

### 1.1 背景

AgentsMesh 平台需要支撑大规模用户自托管的 Runner 连接。Runner 是用户部署在自己机器上的运行节点，承载多个 AgentPod（AI Agent 运行实例）。随着用户规模增长，平台需要能够处理 10 万级别的并发 WebSocket 长连接。

### 1.2 目标

- 支撑 **100,000** 个并发 Runner WebSocket 连接
- 支撑 **300,000** 个活跃 AgentPod（假设每 Runner 平均 3 个）
- 心跳延迟 P99 < 500ms
- 系统可用性 > 99.9%

### 1.3 非目标

- Runner 端的性能优化（用户自托管，不在本 RFC 范围内）
- 前端 Web 应用的扩展（单独 RFC）

---

## 2. 当前架构分析

### 2.1 系统架构

```
┌─────────────┐     WebSocket      ┌─────────────┐
│   Runner    │◄──────────────────►│   Backend   │
│  (用户机器)  │                    │   (Go)      │
└─────────────┘                    └──────┬──────┘
                                          │
                              ┌───────────┼───────────┐
                              │           │           │
                              ▼           ▼           ▼
                        ┌──────────┐ ┌─────────┐ ┌─────────┐
                        │PostgreSQL│ │  Redis  │ │   Hub   │
                        │          │ │         │ │(Frontend│
                        │          │ │         │ │   WS)   │
                        └──────────┘ └─────────┘ └─────────┘
```

### 2.2 核心组件

| 组件 | 文件路径 | 职责 |
|------|----------|------|
| ConnectionManager | `backend/internal/service/runner/connection_manager.go` | 管理 Runner WebSocket 连接 |
| PodCoordinator | `backend/internal/service/runner/pod_coordinator.go` | 协调 AgentPod 生命周期 |
| TerminalRouter | `backend/internal/service/runner/terminal_router.go` | 路由终端数据流 |
| Hub | `backend/internal/websocket/hub.go` | 管理前端 WebSocket 连接 |

### 2.3 瓶颈分析

#### 2.3.1 ConnectionManager - 单一全局锁

**当前实现：**

```go
type ConnectionManager struct {
    connections  map[int64]*RunnerConnection
    mu           sync.RWMutex  // 全局单锁
}
```

**问题：**
- 10 万连接共享一把锁，读写竞争严重
- 高并发下锁等待时间显著增加
- 无法有效利用多核 CPU

**影响评估：**
- 锁竞争导致连接操作延迟增加 10-100x
- 心跳处理吞吐量受限

#### 2.3.2 数据库压力

**心跳写入：**
- 心跳间隔：30 秒
- 10 万 Runner：100,000 ÷ 30 = **3,333 次/秒 UPDATE**

**AgentPod 调和：**
- 每次心跳触发 Pod 状态同步
- 假设每 Runner 3 个 AgentPod：**~10,000 次/秒查询**

**当前连接池：**

```go
// backend/internal/infra/database/database.go
sqlDB.SetMaxOpenConns(100)  // 严重不足
```

#### 2.3.3 TerminalRouter 内存压力

**Scrollback Buffer：**
- 每个 AgentPod 100KB 缓冲区
- 30 万 AgentPod × 100KB = **30GB 内存**

**当前实现：**

```go
type TerminalRouter struct {
    scrollbackBuffers map[string]*ScrollbackBuffer  // 全部常驻内存
}
```

#### 2.3.4 WebSocket Hub 瓶颈

**当前实现：**

```go
func (h *Hub) Run() {
    for {
        select {
        case msg := <-h.podBroadcast:  // 单 goroutine 串行处理
            // ...
        }
    }
}
```

**问题：**
- 单 goroutine 处理所有广播
- 通道 buffer 仅 256，高并发时阻塞

---

## 3. 资源消耗估算

### 3.1 内存需求

| 组件 | 计算 | 消耗 |
|------|------|------|
| ConnectionManager | 100,000 × 145KB | 14.5 GB |
| Goroutines | 200,000 × 8KB | 1.6 GB |
| Scrollback Buffers | 300,000 × 100KB | 30 GB |
| WebSocket Hub | 客户端管理 | 1 GB |
| 应用基础 | 框架、缓存等 | 2 GB |
| **总计** | | **~50 GB** |

### 3.2 数据库 QPS

| 操作 | 频率 |
|------|------|
| Runner 心跳 UPDATE | 3,333/秒 |
| AgentPod SELECT | 3,333/秒 |
| AgentPod UPDATE | ~5,000/秒 |
| 终端活动更新 | ~10,000/秒 |
| **总计** | **~20,000/秒** |

### 3.3 网络带宽

| 流量类型 | 计算 | 带宽 |
|----------|------|------|
| 心跳入站 | 3,333/秒 × 1KB | ~27 Mbps |
| 心跳出站 | 3,333/秒 × 0.5KB | ~13 Mbps |
| 终端数据 | 10% 活跃 × 5KB/s | ~400 Mbps |
| **总计** | | **~500 Mbps - 1 Gbps** |

---

## 4. 架构设计

### 4.1 总体架构

```
                                 ┌─────────────────────────────────┐
                                 │        DNS / Global LB          │
                                 │     (Route53 / CloudFlare)      │
                                 └───────────────┬─────────────────┘
                                                 │
                    ┌────────────────────────────┼────────────────────────────┐
                    │                            │                            │
                    ▼                            ▼                            ▼
         ┌─────────────────┐          ┌─────────────────┐          ┌─────────────────┐
         │   Region A      │          │   Region B      │          │   Region C      │
         │   (us-east)     │          │   (eu-west)     │          │   (ap-east)     │
         └────────┬────────┘          └────────┬────────┘          └────────┬────────┘
                  │                            │                            │
                  ▼                            ▼                            ▼
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                                     │
│                                                                                     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                        Ingress Controller (NGINX)                             │  │
│  │              WebSocket Sticky Sessions / SSL Termination                      │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                            │
│          ┌─────────────────────────────┼─────────────────────────────┐              │
│          │                             │                             │              │
│          ▼                             ▼                             ▼              │
│  ┌───────────────┐            ┌───────────────┐            ┌───────────────┐        │
│  │  Backend #1   │            │  Backend #2   │            │  Backend #N   │        │
│  │  8 CPU, 32GB  │            │  8 CPU, 32GB  │     ...    │  8 CPU, 32GB  │        │
│  └───────┬───────┘            └───────┬───────┘            └───────┬───────┘        │
│          │                             │                             │              │
│          └─────────────────────────────┼─────────────────────────────┘              │
│                                        │                                            │
│  ┌─────────────────────────────────────┼─────────────────────────────────────────┐  │
│  │                                     │                                         │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                         Redis Cluster                                   │  │  │
│  │  │     连接状态 / AgentPod 缓存 / 心跳聚合 / Pub/Sub                        │  │  │
│  │  └─────────────────────────────────────────────────────────────────────────┘  │  │
│  │                                     │                                         │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    PostgreSQL (主从复制)                                │  │  │
│  │  │                  Master + 2 Read Replicas                               │  │  │
│  │  └─────────────────────────────────────────────────────────────────────────┘  │  │
│  │                                                                               │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Backend 水平扩展

**设计原则：**
- 单实例承载 ~10,000 Runner 连接
- 10 万 Runner 需要 10-15 个 Backend 实例
- Runner WebSocket 连接需要固定到同一 Backend（Sticky Session）

**Ingress 配置：**

```yaml
# nginx ingress annotation
nginx.ingress.kubernetes.io/upstream-hash-by: "$remote_addr"
nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
```

### 4.3 Redis 缓存层设计

#### 4.3.1 Key 设计

| 用途 | Key 模式 | TTL | 说明 |
|------|----------|-----|------|
| Runner 状态 | `runner:{id}:status` | 60s | 减少 DB 读取 |
| AgentPod 信息 | `agentpod:{id}:info` | 300s | 快速查询 |
| 心跳批次 | `heartbeat:batch:{ts}` | 10s | 聚合后批量写 DB |
| Scrollback | `scrollback:{pod_key}` | 3600s | 终端回滚缓冲（LRU evict 时持久化） |

**注意**：由于采用 Org 级别路由隔离（见 4.4 节），以下不再需要：
- ~~跨实例 Pub/Sub 广播~~
- ~~Pod 位置发现~~
- ~~终端数据跨实例转发~~

### 4.4 Org 级别路由隔离（核心策略）

#### 4.4.1 设计原则

AgentsMesh 的资源模型以 Organization 为边界：

```
Organization
    ├── Runners (属于 Org)
    ├── AgentPods (属于 Org)
    ├── Tickets (属于 Org)
    └── Channels (Org 内 Pod 之间)
```

**核心洞察**：只要同一 Org 的所有请求路由到同一 Backend 实例，就**完全不需要跨实例通信**。

#### 4.4.2 架构设计

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   所有请求携带 X-Organization-ID Header                                      │
│                         │                                                   │
│                         ▼                                                   │
│                 NGINX (hash $org_id consistent)                             │
│                         │                                                   │
│          ┌──────────────┼──────────────┐                                    │
│          ▼              ▼              ▼                                    │
│     Backend #1     Backend #2     Backend #3                                │
│     ┌────────┐     ┌────────┐     ┌────────┐                               │
│     │ Org A  │     │ Org C  │     │ Org E  │                               │
│     │ Org B  │     │ Org D  │     │ Org F  │                               │
│     │  ...   │     │  ...   │     │  ...   │                               │
│     └────────┘     └────────┘     └────────┘                               │
│                                                                             │
│   结果：                                                                     │
│   ✅ 同一 Org 的 Runner、Pod、前端请求 → 同一 Backend                        │
│   ✅ Terminal Binding 直接内存通信（延迟 < 1ms）                             │
│   ✅ 跨实例通信 = 0                                                          │
│   ✅ 不需要 Redis Pub/Sub 做消息转发                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### 4.4.3 NGINX 配置

```nginx
http {
    # 从 Header 或 Query Param 提取 org_id
    map $http_x_organization_id $org_id {
        default $http_x_organization_id;
        ""      $arg_org_id;  # Header 为空时 fallback 到 query param
    }

    upstream backend {
        # 基于 org_id 一致性 hash
        hash $org_id consistent;

        server backend-1:8080;
        server backend-2:8080;
        server backend-3:8080;
    }

    server {
        listen 443 ssl;
        server_name api.agentsmesh.io;

        # WebSocket 支持
        location /ws/ {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Organization-ID $org_id;

            # WebSocket 超时
            proxy_read_timeout 3600s;
            proxy_send_timeout 3600s;
        }

        # REST API
        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Organization-ID $org_id;
        }
    }
}
```

#### 4.4.4 Kubernetes Ingress 配置

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agentsmesh-api
  annotations:
    nginx.ingress.kubernetes.io/upstream-hash-by: "$http_x_organization_id"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    # fallback 到 query param
    nginx.ingress.kubernetes.io/configuration-snippet: |
      set $org_id $http_x_organization_id;
      if ($org_id = "") {
        set $org_id $arg_org_id;
      }
spec:
  ingressClassName: nginx
  rules:
    - host: api.agentsmesh.io
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: backend
                port:
                  number: 8080
```

#### 4.4.5 客户端改动

| 客户端 | 方式 | 说明 |
|--------|------|------|
| **前端 REST API** | `X-Organization-ID` Header | 从 JWT 或登录态获取 |
| **前端 WebSocket** | Query param `?org_id=xxx` | 浏览器 WebSocket 不支持自定义 Header |
| **Runner** | `X-Organization-ID` Header | 配置文件中已有 org_id |

**Runner 连接示例：**

```
wss://api.agentsmesh.io/ws/runner?node_id=xxx&token=xxx
Header: X-Organization-ID: 123
```

**前端 WebSocket 连接示例：**

```
wss://api.agentsmesh.io/ws/terminal?org_id=123&pod_key=xxx
```

#### 4.4.6 Backend 验证

Backend 需要验证请求中的 `org_id` 与认证信息一致：

```go
// 中间件：验证 org_id 与认证信息一致
func ValidateOrgID() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 Header 或 Query 获取 org_id
        orgID := c.GetHeader("X-Organization-ID")
        if orgID == "" {
            orgID = c.Query("org_id")
        }

        // 从认证信息获取实际 org_id
        claims := auth.GetClaims(c)
        if claims != nil && claims.OrganizationID != orgID {
            c.AbortWithStatusJSON(403, gin.H{"error": "org_id mismatch"})
            return
        }

        c.Next()
    }
}
```

#### 4.4.7 一致性 Hash 的优势

| 特性 | 说明 |
|------|------|
| **增减实例平滑** | 添加/移除 Backend 时，只有少量 Org 需要迁移 |
| **负载均衡** | Org 均匀分布在各 Backend 上 |
| **故障恢复** | 单实例故障时，其 Org 自动迁移到其他实例 |
| **无状态** | Backend 无需维护跨实例状态 |

#### 4.4.8 大 Org 处理（未来优化）

如果某个 Org 特别大（占 30%+ 流量），可以采用虚拟分片：

```
小 Org: org_id 直接 hash
大 Org: org_id + shard_key 组合 hash
        org_big:shard_0 → Backend #1
        org_big:shard_1 → Backend #2
        org_big:shard_2 → Backend #3
```

**当前阶段不需要实现**，等遇到大客户再考虑。

#### 4.4.9 简化效果

| 项目 | 简化前 | 简化后 |
|------|--------|--------|
| 跨实例通信 | Redis Pub/Sub | ❌ 不需要 |
| Pod 位置发现 | Redis Hash | ❌ 不需要 |
| 终端数据转发 | 透明代理模式 | ❌ 不需要 |
| Redis 消息量 | 20 万条/秒 | ~0 |
| 架构复杂度 | 高 | **低** |
| 代码改动 | 大 | **小** |

---

## 5. 代码优化方案

### 5.1 ConnectionManager 分片锁

**目标：** 将单一全局锁拆分为 256 个分片，减少锁竞争。

**实现：**

```go
const numShards = 256

type ConnectionManager struct {
    shards [numShards]*connectionShard
    logger *slog.Logger
    // ... callbacks
}

type connectionShard struct {
    connections map[int64]*RunnerConnection
    mu          sync.RWMutex
}

func NewConnectionManager(logger *slog.Logger) *ConnectionManager {
    cm := &ConnectionManager{
        logger: logger,
    }
    for i := 0; i < numShards; i++ {
        cm.shards[i] = &connectionShard{
            connections: make(map[int64]*RunnerConnection),
        }
    }
    return cm
}

func (cm *ConnectionManager) getShard(runnerID int64) *connectionShard {
    return cm.shards[runnerID%numShards]
}

func (cm *ConnectionManager) AddConnection(runnerID int64, conn *websocket.Conn) *RunnerConnection {
    shard := cm.getShard(runnerID)
    shard.mu.Lock()
    defer shard.mu.Unlock()

    // Close existing connection if any
    if existing, ok := shard.connections[runnerID]; ok {
        existing.Close()
    }

    rc := &RunnerConnection{
        RunnerID: runnerID,
        Conn:     conn,
        Send:     make(chan []byte, 256),
        LastPing: time.Now(),
    }
    shard.connections[runnerID] = rc
    return rc
}

func (cm *ConnectionManager) GetConnection(runnerID int64) *RunnerConnection {
    shard := cm.getShard(runnerID)
    shard.mu.RLock()
    defer shard.mu.RUnlock()
    return shard.connections[runnerID]
}
```

**预期效果：**
- 锁竞争降低 256 倍
- 支持更高并发连接操作

### 5.2 心跳批量聚合

**目标：** 将高频心跳写入聚合为批量操作，减少数据库压力。

**实现：**

```go
type HeartbeatBatcher struct {
    redis    *redis.Client
    db       *gorm.DB
    buffer   map[int64]*HeartbeatItem
    mu       sync.Mutex
    interval time.Duration
}

type HeartbeatItem struct {
    RunnerID  int64
    Timestamp time.Time
    Status    string
    Pods      []PodInfo
}

func NewHeartbeatBatcher(redis *redis.Client, db *gorm.DB) *HeartbeatBatcher {
    b := &HeartbeatBatcher{
        redis:    redis,
        db:       db,
        buffer:   make(map[int64]*HeartbeatItem),
        interval: 5 * time.Second,
    }
    go b.flushLoop()
    return b
}

func (b *HeartbeatBatcher) Add(runnerID int64, data *HeartbeatData) {
    // 1. 立即更新 Redis（实时状态）
    ctx := context.Background()
    b.redis.HSet(ctx, fmt.Sprintf("runner:%d:status", runnerID), map[string]interface{}{
        "last_heartbeat": time.Now().Unix(),
        "status":         "online",
    })
    b.redis.Expire(ctx, fmt.Sprintf("runner:%d:status", runnerID), 60*time.Second)

    // 2. 加入批量队列（延迟写 DB）
    b.mu.Lock()
    b.buffer[runnerID] = &HeartbeatItem{
        RunnerID:  runnerID,
        Timestamp: time.Now(),
        Status:    "online",
    }
    b.mu.Unlock()
}

func (b *HeartbeatBatcher) flushLoop() {
    ticker := time.NewTicker(b.interval)
    for range ticker.C {
        b.flush()
    }
}

func (b *HeartbeatBatcher) flush() {
    b.mu.Lock()
    batch := b.buffer
    b.buffer = make(map[int64]*HeartbeatItem)
    b.mu.Unlock()

    if len(batch) == 0 {
        return
    }

    // 批量 UPDATE
    tx := b.db.Begin()
    for _, item := range batch {
        tx.Exec(
            "UPDATE runners SET last_heartbeat_at = ?, status = ? WHERE id = ?",
            item.Timestamp, item.Status, item.RunnerID,
        )
    }
    tx.Commit()
}
```

**预期效果：**
- 数据库写入从 3,333/秒 降至 ~667/秒（5秒聚合）
- 实时状态仍通过 Redis 保证

### 5.3 Scrollback Buffer LRU + Redis

**目标：** 将 30GB 内存占用降至 ~1GB，同时保证功能。

**实现：**

```go
import lru "github.com/hashicorp/golang-lru/v2"

type TerminalRouter struct {
    connectionManager *ConnectionManager
    logger            *slog.Logger
    redis             *redis.Client

    // LRU cache for hot scrollback buffers
    scrollbackCache *lru.Cache[string, *ScrollbackBuffer]

    // ... other fields
}

func NewTerminalRouter(cm *ConnectionManager, redis *redis.Client, logger *slog.Logger) *TerminalRouter {
    cache, _ := lru.New[string, *ScrollbackBuffer](10000) // 10K entries, ~1GB

    return &TerminalRouter{
        connectionManager: cm,
        logger:            logger,
        redis:             redis,
        scrollbackCache:   cache,
    }
}

func (tr *TerminalRouter) getScrollback(podKey string) *ScrollbackBuffer {
    // 1. Check LRU cache
    if buf, ok := tr.scrollbackCache.Get(podKey); ok {
        return buf
    }

    // 2. Load from Redis
    ctx := context.Background()
    data, err := tr.redis.Get(ctx, "scrollback:"+podKey).Bytes()
    if err != nil && err != redis.Nil {
        tr.logger.Warn("failed to load scrollback from redis", "error", err)
    }

    buf := NewScrollbackBuffer(DefaultScrollbackSize)
    if len(data) > 0 {
        buf.data = data
    }

    tr.scrollbackCache.Add(podKey, buf)
    return buf
}

func (tr *TerminalRouter) saveScrollback(podKey string, buf *ScrollbackBuffer) {
    ctx := context.Background()
    data := buf.GetData()
    if len(data) > 0 {
        tr.redis.Set(ctx, "scrollback:"+podKey, data, time.Hour)
    }
}

// 在 handleTerminalOutput 中异步保存
func (tr *TerminalRouter) handleTerminalOutput(runnerID int64, data *TerminalOutputData) {
    podKey := data.PodKey
    buf := tr.getScrollback(podKey)
    buf.Write(data.Data)

    // 异步保存到 Redis（每 N 秒或每 N 次写入）
    go tr.saveScrollback(podKey, buf)

    // ... broadcast to clients
}
```

**预期效果：**
- 内存从 30GB 降至 ~1GB
- 冷数据自动 evict 到 Redis
- 热数据仍在内存，性能不受影响

### 5.4 WebSocket Hub 分片

**目标：** 将单一 Hub 拆分为多个分片，提高广播并发能力。

**实现：**

```go
import "hash/fnv"

const hubShards = 64

type ShardedHub struct {
    shards [hubShards]*Hub
}

func NewShardedHub() *ShardedHub {
    sh := &ShardedHub{}
    for i := 0; i < hubShards; i++ {
        sh.shards[i] = NewHub()
        go sh.shards[i].Run()
    }
    return sh
}

func (sh *ShardedHub) getShard(podKey string) *Hub {
    h := fnv.New32a()
    h.Write([]byte(podKey))
    return sh.shards[h.Sum32()%hubShards]
}

func (sh *ShardedHub) BroadcastToPod(podKey string, message []byte) {
    hub := sh.getShard(podKey)
    hub.BroadcastToPod(podKey, message)
}

func (sh *ShardedHub) RegisterClient(client *Client) {
    hub := sh.getShard(client.PodKey)
    hub.Register(client)
}

func (sh *ShardedHub) UnregisterClient(client *Client) {
    hub := sh.getShard(client.PodKey)
    hub.Unregister(client)
}
```

**预期效果：**
- 广播处理并发度提升 64 倍
- 单个 pod 的消息仍保证顺序

---

## 6. 数据库优化

### 6.1 连接池配置

```go
func New(cfg config.DatabaseConfig) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        return nil, err
    }

    sqlDB, _ := db.DB()

    // 优化后的连接池配置
    sqlDB.SetMaxIdleConns(50)
    sqlDB.SetMaxOpenConns(300)
    sqlDB.SetConnMaxLifetime(time.Hour)
    sqlDB.SetConnMaxIdleTime(10 * time.Minute)

    return db, nil
}
```

### 6.2 读写分离

```go
type DBCluster struct {
    master  *gorm.DB
    slaves  []*gorm.DB
    counter uint64
}

func (c *DBCluster) Writer() *gorm.DB {
    return c.master
}

func (c *DBCluster) Reader() *gorm.DB {
    idx := atomic.AddUint64(&c.counter, 1) % uint64(len(c.slaves))
    return c.slaves[idx]
}
```

### 6.3 索引优化

```sql
-- 心跳更新优化（覆盖索引）
CREATE INDEX idx_runners_heartbeat ON runners(id) INCLUDE (last_heartbeat_at, status);

-- AgentPod 查询优化
CREATE INDEX idx_pods_runner_status ON pods(runner_id, status);

-- 终端活动查询
CREATE INDEX idx_pods_activity ON pods(last_activity_at) WHERE status = 'active';
```

---

## 7. 部署架构

### 7.1 单 Region 配置（支撑 ~35,000 Runner）

| 组件 | 规格 | 数量 | CPU | 内存 | 存储 |
|------|------|------|-----|------|------|
| Backend | c6i.2xlarge | 5 | 8 核 | 16 GB | 100 GB |
| PostgreSQL 主 | r6i.2xlarge | 1 | 8 核 | 64 GB | 500 GB |
| PostgreSQL 从 | r6i.xlarge | 2 | 4 核 | 32 GB | 500 GB |
| Redis | r6g.xlarge | 6 | 4 核 | 32 GB | 100 GB |
| Ingress | c6i.xlarge | 2 | 4 核 | 8 GB | 50 GB |

### 7.2 全球三 Region 部署（10 万 Runner）

| Region | Runner 分布 | Backend 实例 |
|--------|-------------|--------------|
| us-east-1 | 40,000 | 6 |
| eu-west-1 | 35,000 | 5 |
| ap-northeast-1 | 25,000 | 4 |

### 7.3 成本估算

| 项目 | 月成本 |
|------|--------|
| 计算资源 | $8,500 |
| 存储 | $640 |
| 数据传输 | $900 |
| **按需总计** | **~$12,850** |
| **预留实例 (1年)** | **~$8,000** |
| **Savings Plans (3年)** | **~$6,500** |

---

## 8. 监控告警

### 8.1 关键指标

| 指标 | 告警阈值 | 说明 |
|------|----------|------|
| `agentsmesh_runner_connections_total` | > 12,000/实例 | 连接数过高 |
| `agentsmesh_heartbeat_latency_p99` | > 500ms | 心跳延迟 |
| `agentsmesh_db_connections_used_ratio` | > 80% | 连接池饱和 |
| `agentsmesh_redis_memory_used_ratio` | > 80% | Redis 内存 |
| `agentsmesh_websocket_error_rate` | > 0.1% | WebSocket 错误 |
| `go_goroutines` | > 50,000/实例 | Goroutine 泄漏 |

### 8.2 Prometheus 指标

```go
var (
    runnerConnections = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "agentsmesh_runner_connections_total",
        Help: "Total number of connected runners",
    })

    heartbeatLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "agentsmesh_heartbeat_latency_seconds",
        Help:    "Heartbeat processing latency",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
    })

    dbQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "agentsmesh_db_query_duration_seconds",
        Help:    "Database query duration",
        Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
    }, []string{"operation"})
)
```

---

## 9. 实施路线图

### Phase 1: 基础优化（支撑 1 万 Runner）

- [ ] ConnectionManager 分片锁实现
- [ ] 数据库连接池扩容到 200
- [ ] Redis Runner 状态缓存
- [ ] Prometheus 监控接入
- [ ] 基准测试

**预计周期：** 2 周

### Phase 2: 中等规模（支撑 5 万 Runner）

- [ ] 心跳批量聚合
- [ ] Scrollback LRU + Redis
- [ ] 数据库读写分离
- [ ] WebSocket Hub 分片
- [ ] 多 Backend 实例部署
- [ ] 负载测试

**预计周期：** 3 周

### Phase 3: 大规模（支撑 10 万+ Runner）

- [ ] Redis Cluster 部署
- [ ] 多 Region 部署
- [ ] 数据库索引优化
- [ ] 全链路压测
- [ ] 灾备演练

**预计周期：** 4 周

---

## 10. 风险评估

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| 分片锁实现 bug | 中 | 高 | 充分单元测试，灰度发布 |
| Redis 单点故障 | 低 | 高 | Redis Cluster + 哨兵 |
| 数据库主从延迟 | 中 | 中 | 关键读走主库 |
| 网络分区 | 低 | 高 | 多 AZ 部署，自动故障转移 |

---

## 11. 参考资料

- [WebSocket at Scale](https://centrifugal.dev/docs/server/overview)
- [Go sync.RWMutex Performance](https://go.dev/doc/articles/race_detector)
- [Redis Cluster Specification](https://redis.io/docs/reference/cluster-spec/)
- [PostgreSQL Connection Pooling](https://www.pgbouncer.org/)

---

## 附录 A: 性能测试方案

### A.1 测试工具

```bash
# 使用 k6 进行 WebSocket 负载测试
k6 run --vus 10000 --duration 5m websocket_test.js
```

### A.2 测试场景

1. **连接建立测试**：10,000 并发连接建立
2. **心跳吞吐测试**：模拟 100,000 Runner 心跳
3. **终端数据测试**：10% 活跃 pod，5KB/s 数据流
4. **故障恢复测试**：单实例宕机后的重连

### A.3 性能目标

| 指标 | 目标值 |
|------|--------|
| 连接建立延迟 P99 | < 1s |
| 心跳处理延迟 P99 | < 500ms |
| 终端数据延迟 P99 | < 100ms |
| 故障恢复时间 | < 30s |
