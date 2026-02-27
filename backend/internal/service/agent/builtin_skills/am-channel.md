---
name: am-channel
description: |
  WHEN to use:
  - 开始新任务前检查是否有新消息
  - 任务状态发生变化需要同步（开始、完成、阻塞）
  - 需要与其他 Agent 交换信息
  - 需要更新 Ticket 状态

  WHEN NOT to use:
  - 独立的代码编写过程中
  - 不涉及协作的单独任务
user-invocable: false
---

# Channel 通信协议

你可以通过 Channel 与其他 Agent 进行异步通信。

## 可用工具

- `search_channels` - 搜索 Channel
- `get_channel` - 获取 Channel 详情
- `create_channel` - 创建新 Channel
- `get_channel_messages` - 获取 Channel 消息
- `send_channel_message` - 发送消息到 Channel
- `update_ticket` - 更新 Ticket 状态

## 检查消息

开始任务前，检查是否有新的任务分配或消息：

```
// 1. 查找相关 Channel
search_channels(name="project-x")

// 2. 获取最新消息
get_channel_messages(channel_id=123, limit=20)
```

重点关注：
- 任务分配消息
- 问题反馈
- @提及你的消息

## 状态同步

### 开始任务

```
// 通知团队
send_channel_message(
  channel_id=123,
  content="开始处理用户登录功能"
)

// 更新 Ticket 状态
update_ticket(ticket_id="AM-456", status="in_progress")
```

### 完成任务

```
// 通知团队
send_channel_message(
  channel_id=123,
  content="用户登录功能已完成，可以进行联调"
)

// 更新 Ticket 状态
update_ticket(ticket_id="AM-456", status="done")
```

### 遇到阻塞

```
send_channel_message(
  channel_id=123,
  content="遇到问题：缺少 API 文档，需要后端同学提供接口定义"
)
```

说明问题和需要的帮助，然后继续其他可进行的工作。

## 注意事项

- 状态变化时及时同步，保持团队信息透明
- 使用 @mention 明确通知相关 Pod
- 完成后如有后续任务，使用 am-delegate 委派
