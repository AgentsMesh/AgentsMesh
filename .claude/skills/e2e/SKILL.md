---
name: e2e
description: |
  执行 E2E 测试用例。
  根据测试用例的验证类型（ui/api/database）选择正确的工具执行验证。
user-invocable: true
---

# E2E 测试执行指南

执行 `e2e/` 目录下的端到端测试用例。

## 🚨 强制要求：UI 测试必须执行

**⚠️ 禁止跳过 UI 测试！UI 测试是 E2E 测试的核心，必须使用 MCP Chrome DevTools 执行真实的浏览器验证。**

### UI 测试执行规则

1. **不可跳过**：任何 `verification.type: ui` 的测试步骤都必须执行
2. **不可替代**：禁止用 API 调用代替 UI 验证
3. **必须使用浏览器**：使用 MCP Chrome DevTools 工具（`mcp__chrome-devtools__*`）
4. **验证真实渲染**：确保页面元素真实渲染，而非仅验证数据存在

### 为什么 UI 测试重要？

- API 测试只验证数据，UI 测试验证用户体验
- 前端渲染问题只能通过 UI 测试发现
- 交互逻辑（点击、输入、导航）必须在浏览器中验证
- 响应式布局和样式问题需要实际渲染才能发现

## ⚠️ 关键：使用当前 Worktree 的开发环境

**每个 worktree 有独立的 Docker 环境和端口配置！**

### 1. 定位当前 Worktree 的 dev 环境

```bash
# 当前工作目录的 deploy/dev 目录
cd <当前worktree根目录>/deploy/dev

# 例如：
# /Users/stone/Works/AIO/AgentsMesh-Worktrees/feature-payment-membership/deploy/dev
# /Users/stone/Works/AIO/AgentsMesh-Worktrees/feature-xxx/deploy/dev
```

### 2. 获取当前环境的端口配置

**必须先 source 当前 worktree 的 .env 文件获取正确端口：**

```bash
cd <当前worktree>/deploy/dev
source .env

# 查看关键端口
echo "WEB_PORT: ${WEB_PORT}"           # Web/Nginx 端口（用于浏览器和 API）
echo "POSTGRES_PORT: ${POSTGRES_PORT}" # 数据库端口
echo "REDIS_PORT: ${REDIS_PORT}"       # Redis 端口
```

### 3. Docker 容器命名规则

容器名称包含 worktree 目录名，格式为：`agentsmesh-<worktree-dir>-<service>-1`

```bash
# 查看当前环境的容器
docker compose ps

# 示例容器名：
# agentsmesh-feature-payment-membership-postgres-1
# agentsmesh-feature-payment-membership-backend-1
# agentsmesh-feature-payment-membership-web-1
```

### 4. 确认环境运行中

```bash
cd <当前worktree>/deploy/dev
docker compose ps  # 确认所有服务 Up 且 healthy
```

---

## 环境变量与 URL 构建

### 浏览器 URL（UI 测试）

```bash
# 正确：使用 .env 中的 WEB_PORT
source deploy/dev/.env
URL="http://localhost:${WEB_PORT}/dev-org/settings?scope=organization&tab=billing"

# 错误：硬编码端口
URL="http://localhost/..."      # ❌ 可能不是当前环境
URL="http://localhost:80/..."   # ❌ 端口可能不对
```

### API URL

```bash
source deploy/dev/.env
API_BASE="http://localhost:${WEB_PORT}/api/v1"

# 示例
curl -s "${API_BASE}/orgs/dev-org/billing/subscription"
```

### 数据库连接

```bash
source deploy/dev/.env

# 获取容器名（基于 worktree 目录名）
WORKTREE_DIR=$(basename "$(git rev-parse --show-toplevel)")
POSTGRES_CONTAINER="agentsmesh-${WORKTREE_DIR}-postgres-1"

# 执行 SQL
docker exec ${POSTGRES_CONTAINER} psql -U agentsmesh -d agentsmesh -c "SELECT 1"
```

---

## 测试数据

| 数据 | 值 |
|------|-----|
| 测试用户邮箱 | dev@agentsmesh.local |
| 测试用户密码 | devpass123 |
| 测试组织 slug | dev-org |
| 账单页面路径 | /dev-org/settings?scope=organization&tab=billing |

---

## 验证方式与工具选择

### ⚠️ 重要：根据 `verification.type` 选择正确的工具

| 类型 | 说明 | 使用的工具 | 优先级 |
|------|------|-----------|--------|
| `ui` | 浏览器页面验证 | **MCP Chrome DevTools** | 🔴 最高优先级，禁止跳过 |
| `api` | API 调用验证 | **Bash (curl)** | 正常优先级 |
| `database` | 数据库查询验证 | **Bash (docker exec psql)** | 正常优先级 |

### 🚨 UI 测试不可跳过的原因

```
❌ 错误做法：
   "UI 测试需要浏览器，我先跳过，只执行 API 测试"
   "MCP 工具不可用，改用 curl 验证"

✅ 正确做法：
   1. 确认 Chrome 浏览器已打开并可通过 MCP 连接
   2. 使用 mcp__chrome-devtools__* 工具执行所有 UI 验证
   3. 如果 MCP 不可用，报告问题而非跳过测试
```

---

## UI 验证 (`verification.type: ui`)

使用 **MCP Chrome DevTools** 工具进行浏览器验证。

### 执行前准备

```bash
# 1. 获取端口
source deploy/dev/.env
echo "浏览器访问: http://localhost:${WEB_PORT}"
```

### 工具列表

| 工具 | 用途 |
|------|------|
| `mcp__chrome-devtools__new_page` | 打开新页面 |
| `mcp__chrome-devtools__navigate_page` | 导航到 URL |
| `mcp__chrome-devtools__take_snapshot` | 获取页面 A11Y 快照（推荐） |
| `mcp__chrome-devtools__take_screenshot` | 截图 |
| `mcp__chrome-devtools__click` | 点击元素 |
| `mcp__chrome-devtools__fill` | 填写输入框 |
| `mcp__chrome-devtools__wait_for` | 等待文本出现 |

### 执行流程

```
1. 先获取端口
   source deploy/dev/.env
   # WEB_PORT=xxx

2. 打开/导航到目标页面（使用正确端口）
   mcp__chrome-devtools__new_page(url: "http://localhost:${WEB_PORT}/login")

3. 登录（如需要）
   mcp__chrome-devtools__fill(uid: "email-input", value: "dev@agentsmesh.local")
   mcp__chrome-devtools__fill(uid: "password-input", value: "devpass123")
   mcp__chrome-devtools__click(uid: "login-button")

4. 导航到测试页面
   mcp__chrome-devtools__navigate_page(url: "http://localhost:${WEB_PORT}/dev-org/settings?scope=organization&tab=billing")

5. 等待页面加载
   mcp__chrome-devtools__wait_for(text: "当前方案")

6. 获取页面快照
   mcp__chrome-devtools__take_snapshot()

7. 验证快照中包含预期元素
   - 检查快照文本是否包含预期内容
   - 检查元素是否存在（按钮、输入框等）
   - 检查元素状态（disabled、enabled）
```

### 示例：验证页面元素存在

```yaml
# 测试用例要求
verification:
  type: ui
  check:
    - exists: 'heading "优惠码"'
    - contains: "输入优惠码获取套餐时长"
```

执行步骤：
1. `source deploy/dev/.env` → 获取 WEB_PORT
2. `mcp__chrome-devtools__navigate_page` → 导航到账单页面
3. `mcp__chrome-devtools__wait_for` → 等待 "优惠码" 文本
4. `mcp__chrome-devtools__take_snapshot` → 获取快照
5. 分析快照：确认包含 "优惠码" 标题和描述文本

---

## API 验证 (`verification.type: api`)

使用 **curl** 进行 API 调用验证。

### 获取认证 Token

```bash
# 切换到当前 worktree 的 deploy/dev 目录
cd <当前worktree>/deploy/dev
source .env

TOKEN=$(curl -s -X POST "http://localhost:${WEB_PORT}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@agentsmesh.local","password":"devpass123"}' | jq -r '.token')

echo "Token: ${TOKEN}"
```

### API 调用模板

```bash
source deploy/dev/.env

# GET 请求
curl -s -X GET "http://localhost:${WEB_PORT}/api/v1/orgs/dev-org/billing/subscription" \
  -H "Authorization: Bearer $TOKEN" | jq

# POST 请求
curl -s -X POST "http://localhost:${WEB_PORT}/api/v1/orgs/dev-org/billing/checkout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order_type":"subscription","plan_name":"pro","success_url":"http://localhost/success","cancel_url":"http://localhost/cancel"}' | jq
```

---

## 数据库验证 (`verification.type: database`)

使用 **docker exec psql** 执行 SQL 查询。

### 获取正确的容器名

```bash
cd <当前worktree>/deploy/dev
source .env

# 方法1：从 docker compose 获取
POSTGRES_CONTAINER=$(docker compose ps -q postgres)

# 方法2：根据 worktree 目录名构建
WORKTREE_DIR=$(basename "$(git rev-parse --show-toplevel)")
POSTGRES_CONTAINER="agentsmesh-${WORKTREE_DIR}-postgres-1"
```

### 执行 SQL 查询

```bash
source deploy/dev/.env
WORKTREE_DIR=$(basename "$(git rev-parse --show-toplevel)")

docker exec agentsmesh-${WORKTREE_DIR}-postgres-1 psql -U agentsmesh -d agentsmesh -c "
  SELECT status, plan_id, seat_count
  FROM subscriptions
  WHERE organization_id = (SELECT id FROM organizations WHERE slug = 'dev-org');
"
```

### Setup/Cleanup SQL

```bash
source deploy/dev/.env
WORKTREE_DIR=$(basename "$(git rev-parse --show-toplevel)")
POSTGRES_CONTAINER="agentsmesh-${WORKTREE_DIR}-postgres-1"

# 执行 setup SQL
docker exec ${POSTGRES_CONTAINER} psql -U agentsmesh -d agentsmesh -c "
  INSERT INTO promo_codes (code, ...) VALUES ('TESTCODE', ...);
"

# 执行 cleanup SQL
docker exec ${POSTGRES_CONTAINER} psql -U agentsmesh -d agentsmesh -c "
  DELETE FROM promo_code_uses WHERE ...;
"
```

---

## 完整执行流程

### 0. 确认环境（最重要！）

```bash
# 切换到当前 worktree 的 dev 环境
cd <当前worktree>/deploy/dev
source .env

# 确认服务运行中
docker compose ps

# 记录关键信息
echo "WEB_PORT: ${WEB_PORT}"
echo "WORKTREE: $(basename "$(git rev-parse --show-toplevel)")"
```

### 1. 读取测试用例

```
读取 YAML 文件，解析：
- preconditions: 前置条件
- setup: 初始化数据
- steps: 测试步骤
- cleanup: 清理数据
```

### 2. 执行 Setup

如果有 `setup.sql`，先执行数据库初始化。

### 3. 逐步执行 Steps

对每个 step：

```
1. 读取 action 描述
2. 根据 verification.type 选择工具：
   - ui → MCP Chrome DevTools（使用 ${WEB_PORT}）
   - api → curl（使用 ${WEB_PORT}）
   - database → docker exec psql（使用正确容器名）
3. 执行操作
4. 验证 expected 结果
5. 记录 PASS/FAIL
```

### 4. 执行 Cleanup

执行清理 SQL，恢复测试环境。

### 5. 输出报告

```
=== TC-XXX-001: 测试用例名称 ===
环境: feature-payment-membership (WEB_PORT=8080)

Step 1: [操作描述] ✅ PASS
Step 2: [操作描述] ✅ PASS
Step 3: [操作描述] ❌ FAIL - 期望 "xxx"，实际 "yyy"

结果: 2/3 PASS
```

---

## 常见问题

### Q: 页面无法访问 / Connection refused？

1. **检查端口是否正确**：
   ```bash
   source deploy/dev/.env && echo "WEB_PORT: ${WEB_PORT}"
   ```
2. **确认服务运行中**：
   ```bash
   docker compose ps
   ```
3. **不要使用默认端口 80**，每个 worktree 端口不同

### Q: 数据库容器找不到？

1. 容器名包含 worktree 目录名：
   ```bash
   # 查看实际容器名
   docker compose ps
   ```

### Q: API 返回 401？

1. Token 可能过期，重新获取
2. 确认使用的是当前环境的 WEB_PORT

### Q: UI 测试用 API 模拟？

**🚨 错误做法！** UI 测试必须使用 MCP Chrome DevTools 进行真实的浏览器验证。

### Q: 可以跳过 UI 测试吗？

**🚨 绝对不可以！** UI 测试是 E2E 测试的核心：

1. **不要因为 MCP 不可用就跳过**：先解决 MCP 连接问题
2. **不要用 API 替代**：API 测试无法验证前端渲染
3. **不要因为"太慢"就跳过**：UI 测试发现的问题往往是最关键的

如果 MCP Chrome DevTools 无法连接：
```
1. 确认 Chrome 浏览器已打开
2. 确认 MCP 服务已配置并运行
3. 尝试 mcp__chrome-devtools__list_pages 检查连接
4. 如仍无法连接，报告问题，不要跳过测试
```

---

## 命令速查

| 操作 | 命令/工具 |
|------|----------|
| 获取端口 | `source deploy/dev/.env && echo $WEB_PORT` |
| 获取容器名 | `docker compose ps` |
| 导航页面 | `mcp__chrome-devtools__navigate_page` |
| 获取快照 | `mcp__chrome-devtools__take_snapshot` |
| 截图 | `mcp__chrome-devtools__take_screenshot` |
| 点击 | `mcp__chrome-devtools__click` |
| 输入 | `mcp__chrome-devtools__fill` |
| 等待文本 | `mcp__chrome-devtools__wait_for` |
| API 调用 | `curl -s "http://localhost:${WEB_PORT}/api/v1/..."` |
| 数据库查询 | `docker exec <container> psql -U agentsmesh -d agentsmesh -c "SQL"` |

---

## 注意事项

1. **🚨 UI 测试禁止跳过**：所有 `verification.type: ui` 的测试必须使用 MCP Chrome DevTools 执行
2. **UI ≠ API**：绝对不要用 API 调用代替 UI 验证
3. **先确认环境**：执行前先 `source deploy/dev/.env` 获取正确端口
4. **不要硬编码端口**：使用 `${WEB_PORT}` 变量
5. **区分验证类型**：严格按 `verification.type` 选择工具
6. **容器名包含 worktree 名**：不要使用通用容器名
7. **执行 Cleanup**：测试后清理数据，避免影响其他测试
