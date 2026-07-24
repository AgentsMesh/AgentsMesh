# Lens B —— 职责与因果（含实现层硬核内容）

这是本 skill 最独特、最能抓**真实结构性 bug** 的 lens：职责归属、机制/策略分离、因果与不变量（并发/状态契约）、属性三分。

## 5. 职责划分 (Responsibility Assignment)

**检查项：**
- [ ] 每个职责都有明确的 owner（一个职责只属于一个对象/模块）
- [ ] 数据和操作该数据的行为放在一起（避免贫血模型）
- [ ] 跨对象的协作通过显式接口而非共享状态
- [ ] 领域行为留在领域对象，应用编排留在应用服务
- [ ] 没有「职责窃取」(behavior 应该在 A 但放到了 B)
- [ ] 没有「职责扩散」(一个职责拆到了多个对象都管一点)

**职责分配的优先顺序（GRASP 升级版）：**
```
1. 信息专家：拥有数据的对象先承担相关行为
2. 创建者：B 包含/聚合 A 时，由 B 创建 A
3. 控制器：用例的协调由 facade-controller 承担
4. 纯虚构：当 1-3 都导致破坏内聚时，引入服务/工厂
```

**违规模式：**
- 行为散落：A 持有数据但 B/C/D 各自处理 A 的不同方面
- 协调失位：上层 controller 知道下层细节、做了下层应做的判断
- 服务越权：service 拥有了应属于实体的领域规则
- 数据漂移：同一数据在多个 owner 间被修改，没有 single-writer
- "经理类"：XxxManager 持有大量异质职责，只是命名让你以为它有理

**反例 → 修复：**
```
❌ class Order:           # 贫血
       items: List
   class OrderService:
       def total(o): sum(i.price * i.qty for i in o.items)
       def discount(o, code): ...
       def freeze(o): ...

✅ class Order:
       def total() -> Money: ...      # 信息专家：拥有 items 就拥有计算
       def apply_discount(code): ...  # 领域规则
       def freeze(): ...              # 状态转换
   class OrderApplicationService:
       def place_order(...):          # 仅编排：调用 Order 行为
```

## 6. 机制与策略分离 (Mechanism vs Policy)

**核心原则：机制提供「能怎么做」(how-to)，策略决定「该不该做、用哪种」(what / when)。**
变更频率不同的代码不应耦合在同一层：机制稳定，策略易变。

**检查项：**
- [ ] 「能力」和「决策」是否在不同层（机制层暴露 capability，策略层调用并决定参数）
- [ ] 重试次数、超时时长、阈值这类策略参数是否硬编码进了机制
- [ ] 业务规则（哪些用户能 X、什么场景触发 Y）是否散落进了基础设施
- [ ] 是否存在「机制泄漏」：调用方必须知道机制内部才能正确使用
- [ ] 策略是否可注入/可配置/可在测试中替换

**典型分离场景：**
```
机制（mechanism / capability）          策略（policy / decision）
─────────────────────────────────       ──────────────────────────
HTTP client (能发请求)           ↔     重试 3 次 + 指数退避
文件系统 (能读写)                ↔     权限决策、路径白名单
TokenBucket 限流 (能发令牌)      ↔     每秒 100 个 / 突发 50 的配额
Process spawn (能拉子进程)       ↔     超时 30s、SIGTERM grace 500ms
Background task store (能登记)   ↔     evict_terminal 保留 1h 后清理
```

**违规模式：**
- 机制类内写死决策：`fn fetch() { for _ in 0..3 { try; sleep(500) } }` — 3 和 500 是策略
- 策略层直接做机制工作：上层 controller 自己开 socket、自己 retry — 跨层
- "全能配置"：试图通过一个配置字段同时决定机制和策略
- 决策耦合到具体实现：策略写死「用 redis 缓存」而不是「缓存 trait + 注入 impl」

**反例 → 修复：**
```
❌ struct HttpClient {
       fn fetch(url) {                     # 机制 + 策略混杂
           for _ in 0..3 { ... }           # 重试次数硬编码
       }
   }

✅ struct HttpClient { fn fetch(url) -> Result<...> }    # 纯机制
   struct RetryPolicy { attempts: u32, backoff: Duration }
   fn fetch_with_retry(client, policy, url) { ... }      # 策略 + 机制组合
```

## 7. 因果关系与不变量 (Causality & Invariants) —— 抓真 bug 的核心

**核心原则：每一次状态变化都应该有明确的因（trigger）、清晰的传播路径、可枚举的后果。**

**检查项：**
- [ ] 每个 mutable 状态有 single-writer（或显式声明的多写者契约）
- [ ] 状态变更的 trigger 在代码中显式可达（不是「魔法地发生」）
- [ ] 变更的传播路径可追踪（emit event / call dependent / write store 等）
- [ ] 关键不变量由代码结构保护，不靠注释或开发者自觉
- [ ] 不存在「隐性因果」：A 写字段后期望 B 自动感知，但没有订阅机制
- [ ] 不存在「时序耦合」：两个调用必须按特定顺序，但顺序未被类型/方法签名强制
- [ ] race condition 已识别并通过 lock / channel / atomic / actor 模式隔离

**因果分析的关键问题（对每个 mutable state X）：**
```
1. 谁能写 X？（唯一写者 vs 多写者）
2. 写 X 的 trigger 是什么？（事件 / 调用 / 定时器）
3. X 变化后谁需要感知？通过什么机制？
4. 如果传播链中断（emit 失败、消费者掉线），系统会进入什么状态？
5. X 的「合法」取值范围是什么？这个范围由什么代码保护？
```

**不变量分类：**
| 类型 | 例子 | 保护方式 |
|---|---|---|
| 结构不变量 | List 永远有序 | 私有字段 + 控制写入方法 |
| 顺序不变量 | status_tx.send 必须先于 ack.send | 单一 actor 写入顺序固化 |
| 资源不变量 | log file 内容是 reader 可见的超集 | write_all 先于 in-memory push |
| 因果不变量 | 子进程退出 ⇒ 监督者收到 | wait() 主导，禁止旁路 kill |

**违规模式：**
- 「魔法状态」：字段被多个互不知情的地方修改
- 「广播无序」：多个写者并发 emit 同 channel，消费侧不知正确顺序
- 「不变量靠注释」：`// must call init() before exec()`，但调用者可绕过
- 「沉默错误」：emit 失败被 swallow，下游永远等不到信号，没有补偿机制
- 「双相变更」：同时改本地状态和远端，缺少 commit/rollback 边界

**反例 → 修复：**
```
❌ struct Task {
       status: AtomicEnum,       # 谁都能 store()
   }
   // 各处零散 task.status.store(...)

✅ struct Task {
       status_tx: watch::Sender<Status>,       # 单一写者
       status_watch: watch::Receiver<Status>,  # 多读者
   }
   // 只有 monitor actor 持 status_tx，外部只能 borrow_and_update
```

> 这类问题（多写者、race、传播链断裂）几乎都是 75–100 分的真 bug，应优先报告。

## 8. 属性建模 (Attribute Modeling)

**核心原则：每个属性应被归类为 identity / value / derived 中的一种，并据此选择存储与同步策略。**

| 类型 | 特征 | 存储 | 同步策略 |
|---|---|---|---|
| **Identity** | 用于标识对象，creation 时确定，永不变 | 不可变字段（`id: Uuid`）| 无需同步 |
| **Value** | 业务真实状态 | 可变字段 + 写入控制 | single-writer |
| **Derived** | 由其他属性推导 | 优先即时计算；缓存仅在测量到瓶颈时 | 缓存需绑定 invalidation 信号 |

**检查项：**
- [ ] 每个属性是否能明确分类为上述三种之一
- [ ] Identity 字段是否真正不可变（`pub` 暴露但无 setter 不算保护）
- [ ] Value 字段的写者是否明确受限
- [ ] Derived 字段如果缓存了，是否有明确的 invalidation 机制
- [ ] 没有「属性归属错位」：A 的属性写在 B 上、用 B 的方法访问
- [ ] 没有「冗余属性」：同一信息存在两份且不同步（如 `created_at: Instant` + `created_at_unix_ms: u64`）
- [ ] 属性命名表达「是什么」不是「怎么算」（`total_price` ✓ ；`computed_field` ✗）

**违规模式：**
- 把 derived 字段暴露为 public mutable（外部直接覆盖，导致与基础属性不一致）
- 缓存 derived 值但没设计 invalidation：基础属性变了 cache 还是旧值
- "上帝属性"：一个字段塞多种语义（`status: i32` 其中 0=init, 1=running, -1=err, -99=未设置）→ 应拆 enum + Option
- 用同一字段表达 identity 和 value：`pub name: String` 既做主键又做显示名
- "意图丢失"：fields 之间隐含约束（`start_time <= end_time`）但没有 invariant 保护

**反例 → 修复：**
```
❌ struct Task {
       pub id: u64,                    # identity，但 pub mut 可改 ❌
       pub status_code: i32,           # 多语义混塞
       pub cached_summary: String,     # derived，无 invalidation
   }

✅ struct Task {
       id: TaskId,                     # newtype + 不可变（无 setter）
       status: TaskStatus,             # enum，三态分明
       fn summary(&self) -> String { format!(...) }   # derived on demand
   }
```

**与因果关系的联动：**
- Identity 是因果分析的「主键」，所有事件都通过 id 关联
- Value 是因果的「读写靶子」，single-writer 契约施加于此
- Derived 是因果的「投影」，永远只读，invalidation = 重新派生
