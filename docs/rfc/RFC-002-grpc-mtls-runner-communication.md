# RFC-002: gRPC + mTLS Runner 通信协议升级

| 属性 | 值 |
|------|-----|
| **状态** | Draft |
| **作者** | AgentsMesh Team |
| **创建日期** | 2025-01-15 |
| **目标** | 升级 Backend ↔ Runner 通信协议，防止服务端/客户端伪造攻击 |

---

## 1. 概述

### 1.1 背景

AgentsMesh 的 Runner 是用户自托管的守护进程，运行在用户的机器上执行 AI Agent 任务。当前 Runner 与 Backend 通过 WebSocket + JSON 通信，使用 `node_id + auth_token` 认证。

在用户自托管场景下，存在以下安全风险：
- **伪造服务端**：攻击者可通过 DNS 劫持/中间人攻击伪装成 Backend，向 Runner 下发恶意指令
- **伪造 Runner**：攻击者获取 auth_token 后可冒充合法 Runner 接入系统
- **Token 泄露**：auth_token 长期有效，泄露后难以快速撤销

### 1.2 目标

- 防止服务端伪造攻击（核心目标）
- 防止 Runner 伪造攻击
- 支持证书吊销和轮换
- 保持 Frontend ↔ Backend 通信不变

### 1.3 非目标

- Frontend 通信协议变更（保持 JSON/WebSocket）
- 性能优化（gRPC 的性能提升是附带收益）

---

## 2. 方案设计

### 2.1 目标架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                         Frontend (Next.js)                                   │
│                                                                              │
│              REST API (JSON)              WebSocket (JSON)                   │
│                      ↓                          ↓                            │
│                              保持现状不变                                     │
└──────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Backend (Go/Gin)                                    │
│                                                                              │
│   HTTP Server (:8080)              gRPC Server (:9090)                       │
│   服务 Frontend                    服务 Runner (mTLS)                        │
│                                                                              │
│                    ┌─────────────────────────┐                               │
│                    │     PKI Service         │                               │
│                    │  (加载 Root CA 签发证书)  │                               │
│                    └─────────────────────────┘                               │
└──────────────────────────────────────────────────────────────────────────────┘
                                     │
                            gRPC + mTLS (双向认证)
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Runner (Go daemon)                                  │
│                                                                              │
│                  X.509 客户端证书 + Protobuf 消息                             │
│                  只信任 AgentsMesh Root CA                                    │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 mTLS 双向认证机制

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│                       AgentsMesh Root CA                                      │
│                    (存放在独立部署仓库，git-crypt 加密)                        │
│                                 │                                            │
│                  ┌──────────────┴──────────────┐                             │
│                  │                             │                             │
│                  ▼                             ▼                             │
│         Backend 服务端证书              Runner 客户端证书                     │
│         (启动时生成或预置)              (注册时由 CA 签发)                     │
│                                                                              │
│   连接时双向验证：                                                            │
│   • Runner 验证 Backend → 必须是 Root CA 签发 → 防伪造服务端                  │
│   • Backend 验证 Runner → 必须是 Root CA 签发 → 防伪造客户端                  │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

**关键设计**：Runner 只信任 AgentsMesh 私有 CA，不信任系统 CA。攻击者即使获得 Let's Encrypt 等公共 CA 签发的证书，也无法伪造服务端。

### 2.3 安全防护效果

| 威胁 | 防护 | 说明 |
|------|------|------|
| 伪造服务端 | ✅ 完全阻止 | Runner 只信任私有 CA，攻击者无法获得有效服务端证书 |
| 伪造 Runner | ✅ 完全阻止 | Backend 只信任私有 CA 签发的客户端证书 |
| 中间人攻击 | ✅ 完全阻止 | 双向验证，攻击者无法同时伪造两端 |
| 消息窃听 | ✅ 完全阻止 | TLS 1.3 加密 |
| 消息篡改 | ✅ 完全阻止 | TLS 内置完整性校验 |
| 证书泄露 | ✅ 可快速响应 | CRL 吊销 + 短有效期 (90天) |

### 2.4 CPU 开销分析

mTLS 相比普通 TLS 的额外开销主要在握手阶段：

| 阶段 | 开销 | 说明 |
|------|------|------|
| TLS 握手 | +1-2ms | 验证客户端证书 |
| 数据传输 | 无差异 | 握手后加密方式相同 |

**对 AgentsMesh 的影响**：Runner 使用长连接（gRPC stream），握手只发生一次，总体 CPU 增加 < 1%。

---

## 3. 详细设计

### 3.1 Root CA 管理

**存储位置**：独立部署仓库 (git-crypt 加密)

```
deploy-repo/  (git-crypt 加密)
├── agentsmesh/
│   ├── ca.crt          # Root CA 公钥
│   ├── ca.key          # Root CA 私钥
│   ├── server.crt      # Backend 服务端证书 (可选预生成)
│   └── server.key
```

**Backend 配置**：

```yaml
# config.yaml
pki:
  ca_cert_file: /secrets/ca.crt
  ca_key_file: /secrets/ca.key
  server_cert_file: /secrets/server.crt  # 可选，不配则启动时生成
  server_key_file: /secrets/server.key
  cert_validity_days: 90                 # Runner 证书有效期
```

### 3.2 Proto 文件定义

新建 `proto/runner/v1/runner.proto`：

```protobuf
syntax = "proto3";

package runner.v1;

option go_package = "github.com/anthropic/agentsmesh/proto/runner/v1;runnerv1";

// Runner 服务定义
service RunnerService {
  // 双向流：Runner ↔ Backend 主通信通道
  rpc Connect(stream RunnerMessage) returns (stream ServerMessage);
}

// Runner -> Backend 消息
message RunnerMessage {
  oneof payload {
    InitializeRequest initialize = 1;
    InitializedConfirm initialized = 2;
    HeartbeatData heartbeat = 3;
    PodCreatedEvent pod_created = 4;
    PodTerminatedEvent pod_terminated = 5;
    TerminalOutputEvent terminal_output = 6;
    AgentStatusEvent agent_status = 7;
    PtyResizedEvent pty_resized = 8;
    ErrorEvent error = 9;
  }
  int64 timestamp = 15;
}

// Backend -> Runner 消息
message ServerMessage {
  oneof payload {
    InitializeResult initialize_result = 1;
    CreatePodCommand create_pod = 2;
    TerminatePodCommand terminate_pod = 3;
    TerminalInputCommand terminal_input = 4;
    TerminalResizeCommand terminal_resize = 5;
    SendPromptCommand send_prompt = 6;
  }
  int64 timestamp = 15;
}

// 初始化请求
message InitializeRequest {
  int32 protocol_version = 1;
  RunnerInfo runner_info = 2;
}

message RunnerInfo {
  string version = 1;
  string node_id = 2;
  int32 mcp_port = 3;
  string os = 4;
  string arch = 5;
  string hostname = 6;
}

// 初始化响应
message InitializeResult {
  int32 protocol_version = 1;
  ServerInfo server_info = 2;
  repeated AgentTypeInfo agent_types = 3;
  repeated string features = 4;
}

message ServerInfo {
  string version = 1;
}

message AgentTypeInfo {
  string slug = 1;
  string name = 2;
  string command = 3;
  repeated string default_args = 4;
}

// 初始化完成确认
message InitializedConfirm {
  repeated string available_agents = 1;
}

// 心跳数据
message HeartbeatData {
  string node_id = 1;
  repeated PodInfo pods = 2;
}

message PodInfo {
  string pod_key = 1;
  string status = 2;
  string agent_status = 3;
}

// Pod 创建命令
message CreatePodCommand {
  string pod_key = 1;
  string launch_command = 2;
  repeated string launch_args = 3;
  map<string, string> env_vars = 4;
  repeated FileToCreate files_to_create = 5;
  WorkDirConfig work_dir_config = 6;
  string prompt = 7;
}

message FileToCreate {
  string path = 1;
  string content = 2;
  int32 mode = 3;
}

message WorkDirConfig {
  string type = 1;           // "worktree", "tempdir", "path"
  string repo_path = 2;
  string branch_name = 3;
  string base_branch = 4;
  string path = 5;
}

// 终止 Pod 命令
message TerminatePodCommand {
  string pod_key = 1;
  bool force = 2;
}

// 终端输入命令
message TerminalInputCommand {
  string pod_key = 1;
  bytes data = 2;
}

// 终端调整大小命令
message TerminalResizeCommand {
  string pod_key = 1;
  int32 cols = 2;
  int32 rows = 3;
}

// 发送 Prompt 命令
message SendPromptCommand {
  string pod_key = 1;
  string prompt = 2;
}

// Pod 创建事件
message PodCreatedEvent {
  string pod_key = 1;
  int32 pid = 2;
}

// Pod 终止事件
message PodTerminatedEvent {
  string pod_key = 1;
  int32 exit_code = 2;
  string error_message = 3;
}

// 终端输出事件
message TerminalOutputEvent {
  string pod_key = 1;
  bytes data = 2;  // 直接二进制，无需 Base64 编码
}

// Agent 状态事件
message AgentStatusEvent {
  string pod_key = 1;
  string status = 2;
}

// PTY 调整大小事件
message PtyResizedEvent {
  string pod_key = 1;
  int32 cols = 2;
  int32 rows = 3;
}

// 错误事件
message ErrorEvent {
  string pod_key = 1;
  string code = 2;
  string message = 3;
  map<string, string> details = 4;
}
```

### 3.3 数据库变更

新增 `backend/migrations/000025_add_runner_certificates.up.sql`：

```sql
-- Runner 证书表
CREATE TABLE runner_certificates (
    id BIGSERIAL PRIMARY KEY,
    runner_id BIGINT REFERENCES runners(id) ON DELETE CASCADE,
    serial_number VARCHAR(64) UNIQUE NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    issued_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    revocation_reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runner_certs_runner_id ON runner_certificates(runner_id);
CREATE INDEX idx_runner_certs_serial ON runner_certificates(serial_number);
CREATE INDEX idx_runner_certs_expires ON runner_certificates(expires_at);

-- Runner 表添加证书相关字段
ALTER TABLE runners ADD COLUMN cert_serial_number VARCHAR(64);
ALTER TABLE runners ADD COLUMN cert_expires_at TIMESTAMP;
```

### 3.4 PKI Service 实现

新增 `backend/internal/infra/pki/service.go`：

```go
package pki

import (
    "crypto"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/pem"
    "math/big"
    "os"
    "time"
)

type Service struct {
    caCert     *x509.Certificate
    caKey      crypto.PrivateKey
    serverCert tls.Certificate
    certPool   *x509.CertPool

    validityDays int
}

type Config struct {
    CACertFile     string
    CAKeyFile      string
    ServerCertFile string
    ServerKeyFile  string
    ValidityDays   int
}

func NewService(cfg *Config) (*Service, error) {
    // 加载 CA 证书和私钥
    caCertPEM, err := os.ReadFile(cfg.CACertFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read CA cert: %w", err)
    }

    caKeyPEM, err := os.ReadFile(cfg.CAKeyFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read CA key: %w", err)
    }

    caCert, caKey, err := parseCA(caCertPEM, caKeyPEM)
    if err != nil {
        return nil, fmt.Errorf("failed to parse CA: %w", err)
    }

    // 构建 CA 证书池
    certPool := x509.NewCertPool()
    certPool.AddCert(caCert)

    // 加载或生成服务端证书
    serverCert, err := loadOrGenerateServerCert(cfg, caCert, caKey)
    if err != nil {
        return nil, fmt.Errorf("failed to load/generate server cert: %w", err)
    }

    validityDays := cfg.ValidityDays
    if validityDays <= 0 {
        validityDays = 90
    }

    return &Service{
        caCert:       caCert,
        caKey:        caKey,
        serverCert:   serverCert,
        certPool:     certPool,
        validityDays: validityDays,
    }, nil
}

// IssueRunnerCertificate 为 Runner 签发客户端证书
func (s *Service) IssueRunnerCertificate(nodeID, orgSlug string) (certPEM, keyPEM []byte, serialNumber string, expiresAt time.Time, err error) {
    // 生成 ECDSA 密钥对
    key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, nil, "", time.Time{}, err
    }

    // 生成序列号
    serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
    if err != nil {
        return nil, nil, "", time.Time{}, err
    }

    now := time.Now()
    expiresAt = now.Add(time.Duration(s.validityDays) * 24 * time.Hour)

    // 创建证书模板
    template := &x509.Certificate{
        SerialNumber: serial,
        Subject: pkix.Name{
            CommonName:   nodeID,
            Organization: []string{orgSlug},
            OrganizationalUnit: []string{"runners"},
        },
        NotBefore:   now,
        NotAfter:    expiresAt,
        KeyUsage:    x509.KeyUsageDigitalSignature,
        ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
    }

    // 用 CA 私钥签名
    certDER, err := x509.CreateCertificate(rand.Reader, template, s.caCert, &key.PublicKey, s.caKey)
    if err != nil {
        return nil, nil, "", time.Time{}, err
    }

    // 编码为 PEM
    certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
    keyDER, _ := x509.MarshalECPrivateKey(key)
    keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

    return certPEM, keyPEM, serial.String(), expiresAt, nil
}

// ServerCert 返回服务端 TLS 证书
func (s *Service) ServerCert() tls.Certificate {
    return s.serverCert
}

// CACertPool 返回 CA 证书池（用于验证客户端证书）
func (s *Service) CACertPool() *x509.CertPool {
    return s.certPool
}

// CACertPEM 返回 CA 证书 PEM（返回给 Runner）
func (s *Service) CACertPEM() []byte {
    return pem.EncodeToMemory(&pem.Block{
        Type:  "CERTIFICATE",
        Bytes: s.caCert.Raw,
    })
}

// IsRevoked 检查证书是否已吊销（需要数据库查询）
func (s *Service) IsRevoked(serialNumber string) bool {
    // TODO: 查询数据库 runner_certificates 表
    return false
}
```

### 3.5 gRPC Server 实现

新增 `backend/internal/api/grpc/server.go`：

```go
package grpc

import (
    "crypto/tls"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"

    "agentsmesh/internal/infra/pki"
    runnerv1 "agentsmesh/proto/runner/v1"
)

func NewServer(pkiService *pki.Service, runnerService *runner.Service) *grpc.Server {
    // 配置 mTLS
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{pkiService.ServerCert()},
        ClientAuth:   tls.RequireAndVerifyClientCert,  // 强制客户端证书
        ClientCAs:    pkiService.CACertPool(),         // 只信任自己的 CA
        MinVersion:   tls.VersionTLS13,
    }

    creds := credentials.NewTLS(tlsConfig)

    server := grpc.NewServer(
        grpc.Creds(creds),
        grpc.MaxRecvMsgSize(16 * 1024 * 1024),  // 16MB
        grpc.MaxSendMsgSize(16 * 1024 * 1024),
    )

    runnerv1.RegisterRunnerServiceServer(server, &RunnerServer{
        runnerService: runnerService,
        pkiService:    pkiService,
    })

    return server
}
```

新增 `backend/internal/api/grpc/runner_server.go`：

```go
package grpc

import (
    "context"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/peer"
    "google.golang.org/grpc/status"

    runnerv1 "agentsmesh/proto/runner/v1"
)

type RunnerServer struct {
    runnerv1.UnimplementedRunnerServiceServer

    runnerService *runner.Service
    pkiService    *pki.Service
    connManager   *runner.GRPCConnectionManager
}

func (s *RunnerServer) Connect(stream runnerv1.RunnerService_ConnectServer) error {
    // 从 TLS 握手中提取客户端证书
    p, ok := peer.FromContext(stream.Context())
    if !ok {
        return status.Error(codes.Unauthenticated, "no peer info")
    }

    tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
    if !ok || len(tlsInfo.State.PeerCertificates) == 0 {
        return status.Error(codes.Unauthenticated, "no client certificate")
    }

    clientCert := tlsInfo.State.PeerCertificates[0]

    // 从证书中提取 Runner 身份
    nodeID := clientCert.Subject.CommonName
    orgSlug := ""
    if len(clientCert.Subject.Organization) > 0 {
        orgSlug = clientCert.Subject.Organization[0]
    }
    serialNumber := clientCert.SerialNumber.String()

    // 检查证书是否已吊销
    if s.pkiService.IsRevoked(serialNumber) {
        return status.Error(codes.Unauthenticated, "certificate revoked")
    }

    // 验证 Runner 在数据库中存在且未被禁用
    r, err := s.runnerService.GetByNodeID(stream.Context(), nodeID)
    if err != nil {
        return status.Error(codes.Unauthenticated, "runner not found")
    }
    if !r.IsEnabled {
        return status.Error(codes.Unauthenticated, "runner disabled")
    }

    // 处理双向流
    return s.handleStream(r, stream)
}

func (s *RunnerServer) handleStream(r *runner.Runner, stream runnerv1.RunnerService_ConnectServer) error {
    // 注册连接
    conn := s.connManager.AddConnection(r.ID, stream)
    defer s.connManager.RemoveConnection(r.ID)

    // 启动消息接收循环
    for {
        msg, err := stream.Recv()
        if err != nil {
            return err
        }

        // 处理消息
        s.handleMessage(r.ID, msg)
    }
}
```

### 3.6 Runner 注册流程变更

**新的注册流程**：

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                        Runner 注册流程 (新)                                  │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. 用户执行 runner register                                                 │
│     └── POST /api/v1/runners/register (仍用 HTTPS，一次性 token 认证)         │
│                                                                              │
│  2. Backend 响应                                                             │
│     {                                                                        │
│       "runner_id": 123,                                                      │
│       "certificate": "-----BEGIN CERTIFICATE-----...",   ← Runner 证书      │
│       "private_key": "-----BEGIN EC PRIVATE KEY-----...", ← Runner 私钥     │
│       "ca_certificate": "-----BEGIN CERTIFICATE-----...", ← CA 公钥         │
│       "grpc_endpoint": "grpc.agentsmesh.io:9090"                              │
│     }                                                                        │
│                                                                              │
│  3. Runner 保存到本地                                                        │
│     ~/.agentsmesh/                                                            │
│     ├── config.yaml                                                          │
│     └── certs/                                                               │
│         ├── runner.crt                                                       │
│         ├── runner.key                                                       │
│         └── ca.crt                                                           │
│                                                                              │
│  4. 后续连接使用 gRPC + mTLS                                                 │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

**修改 Runner 注册 API** (`backend/internal/api/rest/runner.go`)：

```go
func (h *RunnerHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 验证 registration_token...

    // 创建 Runner 记录
    runner, err := h.runnerService.Create(c.Request.Context(), req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 签发客户端证书
    certPEM, keyPEM, serialNumber, expiresAt, err := h.pkiService.IssueRunnerCertificate(
        runner.NodeID,
        runner.Organization.Slug,
    )
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to issue certificate"})
        return
    }

    // 保存证书信息到数据库
    h.runnerService.UpdateCertificate(c.Request.Context(), runner.ID, serialNumber, expiresAt)

    // 返回证书
    c.JSON(200, gin.H{
        "runner_id":      runner.ID,
        "certificate":    string(certPEM),
        "private_key":    string(keyPEM),
        "ca_certificate": string(h.pkiService.CACertPEM()),
        "grpc_endpoint":  h.config.GRPCEndpoint,
    })
}
```

### 3.7 Runner gRPC 客户端

新增 `runner/internal/client/grpc_connection.go`：

```go
package client

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "os"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/keepalive"

    runnerv1 "agentsmesh/proto/runner/v1"
)

type GRPCConnection struct {
    serverAddr string
    certFile   string
    keyFile    string
    caFile     string

    conn    *grpc.ClientConn
    client  runnerv1.RunnerServiceClient
    stream  runnerv1.RunnerService_ConnectClient

    handler MessageHandler
    logger  *slog.Logger
}

func NewGRPCConnection(cfg *Config, handler MessageHandler, logger *slog.Logger) *GRPCConnection {
    return &GRPCConnection{
        serverAddr: cfg.GRPCEndpoint,
        certFile:   cfg.CertFile,
        keyFile:    cfg.KeyFile,
        caFile:     cfg.CAFile,
        handler:    handler,
        logger:     logger,
    }
}

func (c *GRPCConnection) Connect(ctx context.Context) error {
    // 加载客户端证书
    cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
    if err != nil {
        return fmt.Errorf("failed to load client certificate: %w", err)
    }

    // 加载 CA 证书
    caCert, err := os.ReadFile(c.caFile)
    if err != nil {
        return fmt.Errorf("failed to load CA certificate: %w", err)
    }

    caPool := x509.NewCertPool()
    if !caPool.AppendCertsFromPEM(caCert) {
        return fmt.Errorf("failed to parse CA certificate")
    }

    // 配置 mTLS - 只信任 AgentsMesh CA
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caPool,  // 关键：只信任私有 CA，不信任系统 CA
        MinVersion:   tls.VersionTLS13,
    }

    // 建立 gRPC 连接
    c.conn, err = grpc.DialContext(ctx, c.serverAddr,
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                30 * time.Second,
            Timeout:             10 * time.Second,
            PermitWithoutStream: true,
        }),
    )
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

    c.client = runnerv1.NewRunnerServiceClient(c.conn)
    c.stream, err = c.client.Connect(ctx)
    if err != nil {
        return fmt.Errorf("failed to establish stream: %w", err)
    }

    return nil
}

func (c *GRPCConnection) Send(msg *runnerv1.RunnerMessage) error {
    msg.Timestamp = time.Now().UnixMilli()
    return c.stream.Send(msg)
}

func (c *GRPCConnection) Recv() (*runnerv1.ServerMessage, error) {
    return c.stream.Recv()
}

func (c *GRPCConnection) Close() error {
    if c.stream != nil {
        c.stream.CloseSend()
    }
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}
```

---

## 4. 实施计划

> **注意**：由于项目尚未上线，直接替换 WebSocket 为 gRPC，无需双协议并行。

### Phase 1: gRPC + mTLS 基础设施 (3-4 周)

- [ ] 定义 proto 文件
- [ ] 实现 PKI Service
- [ ] 实现 gRPC Server
- [ ] 修改 Runner 注册 API
- [ ] 数据库迁移
- [ ] 单元测试

### Phase 2: Runner gRPC 客户端 (2-3 周)

- [ ] 实现 GRPCConnection
- [ ] 修改 Runner 注册流程
- [ ] 证书保存和加载
- [ ] 重连逻辑
- [ ] 移除旧 WebSocket 客户端代码
- [ ] 集成测试

### Phase 3: 清理和文档 (1 周)

- [ ] 移除 Backend WebSocket Runner 端点
- [ ] 移除 Runner WebSocket 客户端代码
- [ ] 更新部署文档
- [ ] 更新 CLAUDE.md

---

## 5. 关键文件清单

### 新增文件

```
proto/
└── runner/v1/runner.proto           # gRPC 服务定义

backend/
├── internal/infra/pki/
│   └── service.go                   # PKI 服务
└── internal/api/grpc/
    ├── server.go                    # gRPC Server
    └── runner_server.go             # Runner 服务实现

runner/
└── internal/client/
    └── grpc_connection.go           # gRPC 客户端
```

### 修改文件

```
backend/
├── cmd/server/main.go               # 启动 gRPC Server
├── internal/config/config.go        # PKI 配置
└── internal/api/rest/runner.go      # 注册接口返回证书

runner/
├── cmd/runner/main.go               # 使用 gRPC 连接
└── internal/client/registration.go  # 保存证书
```

### 删除文件

```
backend/
└── internal/api/rest/ws/runner.go   # 旧 WebSocket Handler

runner/
└── internal/client/
    ├── connection.go                # 旧 WebSocket 连接
    └── client.go                    # 旧客户端实现
```

---

## 6. 验证计划

### 6.1 单元测试

```bash
# PKI 服务测试
cd backend && go test ./internal/infra/pki/... -v

# gRPC Server 测试
cd backend && go test ./internal/api/grpc/... -v
```

### 6.2 集成测试

```bash
# 无证书连接应被拒绝
grpcurl -plaintext localhost:9090 list  # 应该失败

# 有效证书连接应成功
grpcurl -cert runner.crt -key runner.key -cacert ca.crt \
    localhost:9090 runner.v1.RunnerService/Connect

# 伪造证书应被拒绝（使用非 AgentsMesh CA 签发的证书）
grpcurl -cert fake.crt -key fake.key -cacert public_ca.crt \
    localhost:9090 runner.v1.RunnerService/Connect  # 应该失败
```

### 6.3 端到端测试

```bash
# 1. 启动 Backend
cd deploy/dev && docker compose up -d

# 2. Runner 注册
./runner register --server https://localhost --token xxx

# 3. Runner 启动（应使用 gRPC 连接）
./runner run

# 4. 创建 Pod，验证终端输出正常
```

---

## 7. 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| CA 私钥泄露 | 存放在独立部署仓库，git-crypt 加密 |
| 证书过期未续期 | Runner 启动时检查，提前 7 天警告 |
| gRPC 端口防火墙 | 文档说明需开放 9090 端口 |

---

## 8. 参考资料

- [gRPC Authentication](https://grpc.io/docs/guides/auth/)
- [mTLS in Go](https://venilnoronha.io/a-step-by-step-guide-to-mtls-in-go)
- [X.509 Certificate Structure](https://datatracker.ietf.org/doc/html/rfc5280)
- [Kubernetes mTLS](https://kubernetes.io/docs/concepts/security/controlling-access/)
