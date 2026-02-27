# E2E 测试用例

本目录包含 AgentsMesh 的端到端测试用例，采用结构化 YAML 格式编写，可供 Claude Code 或其他自动化测试工具执行。

## 目录结构

```
e2e/
├── README.md                          # 本文件
├── billing/                           # 账单模块
│   ├── subscription/                  # 订阅管理
│   │   ├── TC-SUB-001-status-display.yaml
│   │   ├── TC-SUB-002-plans-dialog.yaml
│   │   ├── TC-SUB-003-cancel-at-period-end.yaml
│   │   ├── TC-SUB-004-cancel-immediately.yaml
│   │   ├── TC-SUB-005-reactivate.yaml
│   │   └── TC-SUB-006-plan-upgrade.yaml
│   ├── seats/                         # 席位管理
│   │   ├── TC-SEAT-001-display.yaml
│   │   ├── TC-SEAT-002-add-dialog.yaml
│   │   └── TC-SEAT-003-based-plan-limit.yaml
│   ├── billing-cycle/                 # 计费周期
│   │   ├── TC-CYCLE-001-display.yaml
│   │   ├── TC-CYCLE-002-monthly-to-yearly.yaml
│   │   └── TC-CYCLE-003-yearly-to-monthly.yaml
│   ├── promo-code/                    # 优惠码
│   │   ├── TC-PROMO-001-display.yaml
│   │   ├── TC-PROMO-002-valid-code.yaml
│   │   └── TC-PROMO-003-invalid-code.yaml
│   └── quota/                         # 配额检查
│       ├── TC-QUOTA-001-users.yaml
│       ├── TC-QUOTA-002-runners.yaml
│       └── TC-QUOTA-003-repositories.yaml
├── extensions/                        # Extensions 模块（Skills + MCP Servers）
│   ├── TC-EXT-001-full-capabilities-flow.yaml  # 完整能力管理端到端流程
│   ├── skills/                        # Skills 测试
│   │   ├── TC-SKILL-001-capabilities-tab-display.yaml  # Capabilities Tab Skills 显示
│   │   ├── TC-SKILL-002-add-skill-dialog.yaml          # 添加 Skill 对话框
│   │   ├── TC-SKILL-003-install-from-market.yaml       # 从 Marketplace 安装
│   │   ├── TC-SKILL-004-install-from-github.yaml       # 从 GitHub URL 安装
│   │   ├── TC-SKILL-005-toggle-and-uninstall.yaml      # 启用/禁用和卸载
│   │   ├── TC-SKILL-006-source-link.yaml               # 来源链接显示
│   │   └── TC-SKILL-007-api-crud.yaml                  # API CRUD 流程
│   ├── mcp/                           # MCP Servers 测试
│   │   ├── TC-MCP-EXT-001-mcp-tab-display.yaml         # MCP Tab 显示
│   │   ├── TC-MCP-EXT-002-add-mcp-dialog.yaml          # 添加 MCP 对话框
│   │   ├── TC-MCP-EXT-003-install-from-market.yaml     # 从 Market 安装（无环境变量）
│   │   ├── TC-MCP-EXT-004-install-from-market-with-env.yaml # 从 Market 安装（需环境变量）
│   │   ├── TC-MCP-EXT-005-install-custom.yaml          # 安装自定义 MCP Server
│   │   ├── TC-MCP-EXT-006-edit-env-vars.yaml           # 编辑环境变量
│   │   ├── TC-MCP-EXT-007-toggle-and-uninstall.yaml    # 启用/禁用和卸载
│   │   ├── TC-MCP-EXT-008-source-link.yaml             # 来源链接显示
│   │   └── TC-MCP-EXT-009-api-crud.yaml                # API CRUD 流程
│   └── settings/                      # 组织 Settings Extensions 管理
│       ├── TC-EXTSET-001-extensions-page.yaml           # Extensions 页面显示
│       ├── TC-EXTSET-002-skill-registries.yaml          # Skill Registries 管理
│       └── TC-EXTSET-003-mcp-templates.yaml             # MCP Templates 浏览
└── runner/                            # Runner 管理模块
    ├── list/                          # Runner 列表
    │   ├── TC-RUNNER-001-list-all.yaml       # 列出所有 Runners
    │   ├── TC-RUNNER-002-list-available.yaml # 列出可用 Runners
    │   └── TC-RUNNER-003-get-single.yaml     # 获取单个 Runner
    ├── tokens/                        # 注册令牌管理
    │   ├── TC-TOKEN-001-list.yaml            # 列出注册令牌
    │   ├── TC-TOKEN-002-create.yaml          # 创建注册令牌
    │   ├── TC-TOKEN-003-revoke.yaml          # 吊销注册令牌
    │   └── TC-TOKEN-004-full-crud-flow.yaml  # 完整 CRUD 流程
    ├── config/                        # Runner 配置
    │   ├── TC-CONFIG-001-update.yaml         # 更新 Runner 配置
    │   └── TC-CONFIG-002-disable-enable.yaml # 禁用/启用 Runner
    ├── delete/                        # Runner 删除
    │   └── TC-DELETE-001-basic.yaml          # 删除 Runner
    ├── grpc-tokens/                   # gRPC 注册令牌
    │   ├── TC-GRPC-001-list.yaml             # 列出 gRPC 令牌
    │   ├── TC-GRPC-002-generate.yaml         # 生成 gRPC 令牌
    │   ├── TC-GRPC-003-delete.yaml           # 删除 gRPC 令牌
    │   └── TC-GRPC-004-full-crud-flow.yaml   # 完整 CRUD 流程
    ├── ui/                            # UI 页面测试
    │   ├── TC-UI-001-list-page.yaml          # Runner 列表页面
    │   ├── TC-UI-002-add-runner-dialog.yaml  # 添加 Runner 对话框
    │   ├── TC-UI-003-runner-config-dialog.yaml # 配置对话框
    │   ├── TC-UI-004-delete-confirmation.yaml  # 删除确认
    │   └── TC-UI-005-full-management-flow.yaml # 完整管理流程
    ├── admin/                         # Admin Runner 管理
    │   ├── TC-ADMIN-001-list.yaml            # Admin 列出所有 Runners
    │   ├── TC-ADMIN-002-get-single.yaml      # Admin 获取单个 Runner
    │   ├── TC-ADMIN-003-disable-enable.yaml  # Admin 禁用/启用
    │   ├── TC-ADMIN-004-delete.yaml          # Admin 删除
    │   └── TC-ADMIN-005-full-management-flow.yaml # Admin 完整流程
    └── registration/                  # Runner 注册与 Pod 创建
        ├── TC-REG-001-multi-runner-registration.yaml  # 多 Runner 注册完整流程
        ├── TC-REG-002-runner-online-status.yaml       # Runner 在线状态验证
        └── TC-REG-003-pod-creation-flow.yaml          # Pod 创建完整流程
```

## 测试用例格式

```yaml
id: TC-XXX-001
name: 测试用例名称
description: 测试用例描述
priority: critical | high | medium | low
must_execute: true  # 🚨 UI 测试必须设置为 true
module: billing/subscription
tags:
  - ui          # 标记为 UI 测试
  - mcp-required  # 标记需要 MCP Chrome DevTools

preconditions:
  - 前置条件

setup:
  sql: |
    -- 可选的数据库初始化 SQL

steps:
  - action: 操作描述
    expected: 预期结果
    verification:
      type: ui | api | database
      details: 验证详情

cleanup:
  - sql: |
      -- 清理 SQL
```

### 🚨 UI 测试强制执行规则

UI 测试（`verification.type: ui`）是 E2E 测试的核心，**禁止跳过**：

- `priority: critical` - UI 测试必须设置为最高优先级
- `must_execute: true` - 标记为必须执行
- `tags: [ui, mcp-required]` - 标记需要 MCP Chrome DevTools

**执行 UI 测试时：**
1. 必须使用 MCP Chrome DevTools 工具
2. 禁止用 API 调用代替浏览器验证
3. 如果 MCP 不可用，报告问题而非跳过测试

## 执行测试

### 使用 Claude Code 执行

```
请执行 e2e/billing/subscription/TC-SUB-001-status-display.yaml 测试用例
```

或执行整个模块：

```
请执行 e2e/billing/subscription/ 目录下的所有测试用例
```

### 验证方式

| 类型 | 说明 | 示例 |
|------|------|------|
| `ui` | 浏览器快照验证 | 检查页面元素、文本、按钮状态 |
| `api` | API 调用验证 | curl 请求，验证状态码和响应 |
| `database` | 数据库查询验证 | psql 执行 SQL，验证数据状态 |

## 测试数据

| 数据 | 值 |
|------|-----|
| 测试用户邮箱 | dev@agentsmesh.local |
| 测试用户密码 | devpass123 |
| Admin 用户邮箱 | admin@agentsmesh.local |
| Admin 用户密码 | adminpass123 |
| 测试组织 slug | dev-org |
| 默认订阅计划 | pro |
| 账单页面路径 | /dev-org/settings?scope=organization&tab=billing |
| Runner 管理页面路径 | /dev-org/runners |

## Runner 模块测试覆盖

Runner E2E 测试覆盖以下功能：

### API 测试

| 接口 | 测试用例 | 说明 |
|------|----------|------|
| `GET /orgs/:slug/runners` | TC-RUNNER-001 | 列出组织内所有 Runners |
| `GET /orgs/:slug/runners/available` | TC-RUNNER-002 | 列出可用 Runners |
| `GET /orgs/:slug/runners/:id` | TC-RUNNER-003 | 获取单个 Runner |
| `PUT /orgs/:slug/runners/:id` | TC-CONFIG-001/002 | 更新 Runner 配置、禁用/启用 |
| `DELETE /orgs/:slug/runners/:id` | TC-DELETE-001 | 删除 Runner |
| `GET /orgs/:slug/runners/tokens` | TC-TOKEN-001 | 列出注册令牌 |
| `POST /orgs/:slug/runners/tokens` | TC-TOKEN-002 | 创建注册令牌 |
| `DELETE /orgs/:slug/runners/tokens/:id` | TC-TOKEN-003 | 吊销注册令牌 |
| `GET /orgs/:slug/runners/grpc/tokens` | TC-GRPC-001 | 列出 gRPC 令牌 |
| `POST /orgs/:slug/runners/grpc/tokens` | TC-GRPC-002 | 生成 gRPC 令牌 |
| `DELETE /orgs/:slug/runners/grpc/tokens/:id` | TC-GRPC-003 | 删除 gRPC 令牌 |

### Admin API 测试

| 接口 | 测试用例 | 说明 |
|------|----------|------|
| `GET /api/v1/admin/runners` | TC-ADMIN-001 | Admin 列出所有 Runners |
| `GET /api/v1/admin/runners/:id` | TC-ADMIN-002 | Admin 获取单个 Runner |
| `POST /api/v1/admin/runners/:id/disable` | TC-ADMIN-003 | Admin 禁用 Runner |
| `POST /api/v1/admin/runners/:id/enable` | TC-ADMIN-003 | Admin 启用 Runner |
| `DELETE /api/v1/admin/runners/:id` | TC-ADMIN-004 | Admin 删除 Runner |

### UI 测试

| 页面/功能 | 测试用例 | 说明 |
|----------|----------|------|
| Runner 列表页面 | TC-UI-001 | 页面显示和状态统计 |
| 添加 Runner 对话框 | TC-UI-002 | 注册命令和令牌生成 |
| Runner 配置对话框 | TC-UI-003 | 配置编辑和保存 |
| 删除确认对话框 | TC-UI-004 | 删除确认流程 |
| 完整管理流程 | TC-UI-005 | 端到端管理操作 |

### 多 Runner 注册与 Pod 创建测试

| 测试用例 | 说明 | 验证类型 |
|----------|------|----------|
| TC-REG-001 | 多 Runner 注册完整流程 | UI + Docker + API + DB |
| TC-REG-002 | Runner 在线状态验证 | API + DB |
| TC-REG-003 | Pod 创建完整流程 | UI + API + DB |

#### TC-REG-001 测试流程

1. **通过 UI 生成注册令牌** - 在 Runner 管理页面生成多个 gRPC 注册令牌
2. **启动 Docker Runner** - 使用令牌启动多个 Runner 容器并注册
3. **验证 Runner 在线** - 确认多个 Runner 同时显示为 online 状态
4. **创建 Pod** - 从一个 Runner 创建 Pod
5. **验证 Pod 运行** - 确认 Pod 进入 running 状态，终端可用
6. **清理资源** - 停止容器并清理数据库

#### 执行要求

- 需要 Docker 环境
- 需要 MCP Chrome DevTools（UI 验证）
- Runner 容器需要能访问 backend 和 nginx 服务（同一 Docker 网络）

## Extensions 模块测试覆盖

Extensions E2E 测试覆盖 Skills 和 MCP Servers 的完整能力管理功能。

### 测试数据

| 数据 | 值 |
|------|-----|
| Extensions 设置页面路径 | /dev-org/settings?scope=organization&tab=extensions |
| Repository 页面路径 | /dev-org/repositories → 点击 Demo WebApp → Extensions Tab |
| MCP Market Seed 数据 | jira, postgres, slack, github, filesystem, memory |

### Skills UI 测试

| 页面/功能 | 测试用例 | 说明 |
|----------|----------|------|
| Capabilities Tab 显示 | TC-SKILL-001 | Skills 子标签页、org/user scope 区块、空状态 |
| 添加 Skill 对话框 | TC-SKILL-002 | 三种安装方式 Tab（Marketplace/GitHub/Upload）|
| Marketplace 安装 | TC-SKILL-003 | 搜索、安装、列表更新、已安装标记 |
| GitHub URL 安装 | TC-SKILL-004 | 填写 URL/Branch/Path 导入 |
| 启用/禁用和卸载 | TC-SKILL-005 | Switch 切换、确认对话框、卸载流程 |
| 来源链接 | TC-SKILL-006 | 外部链接图标、source_url 链接 |
| API CRUD | TC-SKILL-007 | Skills API 完整 CRUD 流程 |

### MCP Servers UI 测试

| 页面/功能 | 测试用例 | 说明 |
|----------|----------|------|
| MCP Tab 显示 | TC-MCP-EXT-001 | MCP 子标签页、org/user scope 区块、空状态 |
| 添加 MCP 对话框 | TC-MCP-EXT-002 | Market Templates/Custom Tab、模板列表、搜索 |
| Market 安装（无 env） | TC-MCP-EXT-003 | 选择 Filesystem 模板直接安装 |
| Market 安装（有 env） | TC-MCP-EXT-004 | 选择 Jira 模板、填写必填环境变量、Change 切换 |
| 自定义安装 | TC-MCP-EXT-005 | stdio 类型自定义配置、环境变量添加 |
| 编辑环境变量 | TC-MCP-EXT-006 | Settings 齿轮按钮、schema 模式、自由编辑模式 |
| 启用/禁用和卸载 | TC-MCP-EXT-007 | Switch 切换、确认对话框、卸载流程 |
| 来源链接 | TC-MCP-EXT-008 | Market 标签、repository_url 外部链接 |
| API CRUD | TC-MCP-EXT-009 | MCP Server API 完整 CRUD 流程 |

### 组织 Settings 测试

| 页面/功能 | 测试用例 | 说明 |
|----------|----------|------|
| Extensions 页面 | TC-EXTSET-001 | 页面标题、双 Tab 显示 |
| Skill Registries | TC-EXTSET-002 | 平台/组织 Registry、添加对话框、认证选项 |
| MCP Templates | TC-EXTSET-003 | 模板列表、搜索、分类、数量统计 |

### 端到端流程测试

| 测试用例 | 说明 | 验证类型 |
|----------|------|----------|
| TC-EXT-001 | 完整能力管理流程（Settings → Repo → Install → Edit → Toggle → Uninstall）| UI + API |
