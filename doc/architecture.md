# gozeroX 项目技术文档

## 一、项目简介

gozeroX 是一个类 Twitter/微博的社交平台后端系统，采用 **go-zero 微服务架构**，将业务拆分为 5 个独立服务。项目为毕业设计作品，**可读性优先于性能优化**，代码力求清晰易懂。

---

## 二、整体架构

### 2.1 服务拓扑

```
                         ┌──────────────────────┐
                         │   Nginx Gateway       │
                         │      :8888            │
                         └──────────┬───────────┘
                                    │
      ┌──────────┬──────────┬───────┴───────┬──────────┐
      ▼          ▼          ▼               ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│usercenter│ │ content  │ │ interact │ │  notice  │ │recommend │
│ API:1001 │ │ API:1002 │ │ API:1003 │ │ API:1004 │ │ API:1005 │
└────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘
     │            │            │             │            │
     ▼            ▼            ▼             ▼            ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ RPC:2001 │ │ RPC:2002 │ │ RPC:2003 │ │ RPC:2004 │ │ RPC:2005 │
└────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘
     │            │            │             │            │
     │            │       ┌────┴────┐        │       ┌────┴────┐
     │            │       │ MQ      │        │       │HTTP调用 │
     │            │       │(Kafka)  │        │       │Python   │
     │            │       └─────────┘        │       └─────────┘
     │            │                          │
     └────────────┴──────────┬───────────────┘
                             ▼
                    ┌─────────────────┐
                    │  PostgreSQL 17  │
                    │  Redis 7.4      │
                    │  Kafka 3.9      │
                    └─────────────────┘
```

### 2.2 通信方式

| 调用方 | 被调用方 | 方式 | 说明 |
|--------|---------|------|------|
| Nginx | 各服务 API | HTTP 反向代理 | 统一入口，路由分发 |
| API 层 | 同服务 RPC 层 | gRPC | 服务内部分层调用 |
| RPC 层 | 其他服务 RPC 层 | gRPC | 跨服务调用（如互动服务调用户中心） |
| RPC 层 | Kafka | 异步消息 | Write-Behind 写策略、事件通知 |
| MQ Consumer | PostgreSQL | 同步写入 | 异步消费后落库 |
| recommendService RPC | Python 推荐服务 | HTTP | 同步召回推荐结果 |

### 2.3 分层职责

每个微服务统一采用 **API → RPC → Model** 三层架构：

```
app/{serviceName}/
├── cmd/
│   ├── api/                 # HTTP API 层
│   │   ├── desc/            # .api 定义文件
│   │   ├── etc/             # YAML 配置
│   │   └── internal/
│   │       ├── config/      # 配置映射结构体
│   │       ├── handler/     # HTTP Handler（路由注册）
│   │       ├── logic/       # 业务编排（调 RPC，不写业务逻辑）
│   │       ├── svc/         # ServiceContext（持有 RPC 客户端）
│   │       └── types/       # 请求/响应类型
│   ├── rpc/                 # gRPC RPC 层
│   │   ├── pb/              # .proto 定义 + 生成代码
│   │   ├── etc/             # YAML 配置
│   │   └── internal/
│   │       ├── config/      # 配置映射结构体
│   │       ├── logic/       # 核心业务逻辑
│   │       ├── server/      # gRPC Server 实现
│   │       └── svc/         # ServiceContext（持有 DB、Redis、Kafka 等）
│   └── mq/                  # Kafka 消费者（部分服务有）
│       └── internal/
│           ├── config/
│           ├── consumer/    # 消费者逻辑
│           └── svc/
└── model/                   # 数据模型（PostgreSQL CRUD）
```

**关键设计原则：**

- **API 层**只做参数校验和 RPC 调用，不包含业务逻辑
- **RPC 层**不校验参数（由上游 API 保证），专注业务处理
- **Model 层**封装数据库操作，基于 goctl 自动生成的 CRUD 代码

---

## 三、各服务模块详解

### 3.1 usercenter — 用户中心服务

| 层 | 端口 |
|----|------|
| API | 1001 |
| RPC | 2001 |

**职责：** 用户注册、登录、信息查询。

**核心 RPC 接口：**

| 方法 | 说明 |
|------|------|
| `Register` | 注册新用户，返回用户信息 + JWT Token |
| `Login` | 手机号 + 密码登录，返回用户信息 + JWT Token |
| `GetUserInfo` | 查询单个用户详细信息（含关注/粉丝/推文数） |
| `BatchGetUserBrief` | 批量查询用户简要信息（用于评论区头像展示等） |

**技术要点：**
- JWT Token 由 RPC 层的 ServiceContext 统一生成（HS256 签名，24h 过期）
- 用户信息通过 Redis Hash 缓存（`user:info:{uid}`），减少数据库查询
- `BatchGetUserBrief` 支持批量查询，被其他服务广泛调用

---

### 3.2 contentService — 内容服务

| 层 | 端口 |
|----|------|
| API | 1002 |
| RPC | 2002 |

**职责：** 推文的创建、查询、删除。

**核心 RPC 接口：**

| 方法 | 说明 |
|------|------|
| `CreateTweet` | 发布推文（支持图片、标签，可选公开/私密） |
| `DeleteTweet` | 删除推文（需验证 uid 归属） |
| `ListTweetsUid` | 用户主页推文列表（游标分页） |
| `GetTweetBySnowTid` | 单条推文查询 |
| `BatchGetTweets` | 批量推文查询（推荐服务使用） |

**技术要点：**
- 推文 ID 使用雪花算法生成（`pkg/idgen`），保证全局唯一且有序
- 推文详情缓存 1 小时（`tweet:info:{snowTid}` Hash）
- 推文创建后异步发送推荐入库事件到 Kafka `recommend_tweet` topic
- 推文删除后同步清理 Redis 缓存，并发送删除事件到 Kafka

---

### 3.3 interactService — 互动服务

| 层 | 端口 |
|----|------|
| API | 1003 |
| RPC | 2003 |
| MQ | Kafka Consumer |

**职责：** 评论、点赞、互动数据管理。

**核心 RPC 接口：**

| 方法 | 说明 |
|------|------|
| `CreateComment` | 发表评论/回复（支持多级） |
| `DeleteComment` | 删除评论 |
| `GetComments` | 获取推文顶级评论列表 |
| `GetReplies` | 获取评论的回复列表 |
| `LikeTweet` | 点赞/取消点赞推文 |
| `LikeComment` | 点赞/取消点赞评论 |
| `GetUserAllLikes` | 获取用户所有点赞关系（登录时同步到前端） |

**Kafka MQ 消费者（interactMQ）：**

| Consumer | 消费 Topic | 职责 |
|----------|-----------|------|
| CommentConsumer | `comment_create` | 异步写入评论到数据库 |
| LikeTweetConsumer | `like_tweet` | 异步写入推文点赞到数据库 |
| LikeCommentConsumer | `like_comment` | 异步写入评论点赞到数据库 |

**技术要点：**
- 采用 **Write-Behind 模式**：RPC 先写 Redis 缓存 + 返回响应，再通过 Kafka 异步写 PostgreSQL
- 评论 ID 使用雪花算法生成，在 RPC 层生成后立即返回给前端
- 推文的 `like_count` / `comment_count` 通过 Redis Hash 字段 `HIncrBy` 实时更新
- 前端登录时通过 `GetUserAllLikes` 一次性拉取所有点赞关系，增量更新，后续操作做本地判断，减少请求
- 互动事件异步发送到 Kafka `recommend_interaction` topic，供 Python 推荐服务消费
- 消费失败的消息会被推入 Redis List 做重试（如 `failed:comment:create`）

---

### 3.4 noticeService — 通知服务

| 层 | 端口 |
|----|------|
| API | 1004 |
| RPC | 2004 |
| MQ | Kafka Consumer |

**职责：** 通知的生成、存储、查询、已读管理。

**核心 RPC 接口：**

| 方法 | 说明 |
|------|------|
| `GetNotices` | 获取通知列表（点赞 + 评论混合，游标分页） |
| `GetUnreadCount` | 获取未读数量（点赞未读、评论未读、总未读） |
| `MarkRead` | 标记已读（支持全部/按类型标记） |

**Kafka MQ 消费者（noticeMQ）：**

| 消费 Topic | 职责 |
|-----------|------|
| `notice` | 消费互动事件，生成通知记录 |

**技术要点：**
- **点赞通知聚合**：同一推文/评论被多次点赞，只生成一条通知记录，记录最近点赞的 2 个用户和总点赞数
  - 例：「用户A、用户B 等 10 人赞了你的推文」
- **评论通知独立**：每条评论/回复生成独立通知记录，保留评论内容
- 通知列表通过 `updated_at` 游标分页，保证最新通知在前
- interactService 在执行点赞/评论操作时，异步向 Kafka `notice` topic 发送通知事件

---

### 3.5 recommendService — 推荐服务

| 层 | 端口 |
|----|------|
| API | 1005 |
| RPC | 2005 |

**职责：** 个性化推荐首页推文流。

**核心 RPC 接口：**

| 方法 | 说明 |
|------|------|
| `RecommendFeed` | 获取推荐推文列表（游标分页） |
| `SearchRecommend` | 搜索推荐（预留接口） |

**技术要点：**
- **Go + Python 混合架构**：Go 端负责 RPC 服务和缓存管理，Python 端负责推荐算法（CLIP 多模态 + 向量召回）
- 推荐流程：`用户请求 → Go RPC → HTTP 调用 Python recall 接口 → 拿到推文 ID 列表 → 调用 contentService RPC 批量获取推文详情 → 返回`
- **缓存预热策略**：每次请求获取 3 倍所需数量（最少 24 条），存入 Redis 缓存（60s TTL），后续请求直接命中缓存
- 缓存 Key：`recommend:feed:{uid}:{cursor}`（推文 ID 集合）和 `recommend:feed_meta:{uid}:{cursor}`（元信息 Hash）
- Python 推荐内核的消费端由 Python 服务独立实现，消费 `recommend_tweet` 和 `recommend_interaction` 两个 Kafka topic

---

## 四、技术亮点

### 4.1 Write-Behind 异步写入

点赞和评论等高并发操作采用 **Write-Behind 模式**：

```
客户端请求 → RPC 层
               ├── 1. 雪花 ID 生成
               ├── 2. 写入 Redis 缓存（计数更新）
               ├── 3. 推送 Kafka 消息
               └── 4. 立即返回响应给客户端

Kafka Consumer → 异步写入 PostgreSQL
```

- 客户端无需等待数据库写入完成，响应速度极快
- Redis 保证数据实时可见，Kafka 保证最终一致性
- 消费失败时写入 Redis 失败队列，支持重试

### 4.2 Kafka 事件驱动架构

系统通过 Kafka 构建了松耦合的事件驱动架构：

```
contentService ──→ recommend_tweet topic ──→ Python 推荐服务
                                         ──→ (推文入库/向量化)

interactService ─→ recommend_interaction ─→ Python 推荐服务
                                         ──→ (兴趣向量更新)

interactService ─→ notice topic ──────────→ noticeMQ
                                         ──→ (通知记录生成)

interactService ─→ comment_create ────────→ interactMQ
              ──→ like_tweet ──────────────→ interactMQ
              ──→ like_comment ────────────→ interactMQ
                                         ──→ (异步写库)
```

- 生产者使用**单例 Pusher 池**（`GetPusher(topic)`），double-check locking 保证并发安全
- 同一服务既是生产者（RPC 层）也是消费者（MQ 层），职责清晰

### 4.3 通知聚合

点赞通知采用**聚合策略**：

- 同一目标（推文/评论）被多人点赞时，只保留一条通知记录
- 记录 `recent_uid1`、`recent_uid2`（最近两个点赞用户）和 `total_count`
- 前端可展示：「用户A、用户B 等 10 人赞了你的推文」
- 使用数据库 `ON CONFLICT` 处理并发点赞时的插入冲突

### 4.4 游标分页

所有列表接口统一使用**游标分页**（基于时间戳），避免传统 offset 分页在数据量大时的性能问题：

- 首次请求 `cursor = 0`，服务端返回数据 + 下一页 cursor
- 后续请求携带上次返回的 cursor，服务端通过 `WHERE created_at < cursor` 查询
- 适合社交场景的时间线展示，天然支持"加载更多"

### 4.5 雪花 ID 生成器

`pkg/idgen` 实现了标准 Snowflake 算法，为推文、评论、通知等生成全局唯一、有序的 64 位 ID：

- 基于 2024-01-01 纪元，10 位机器 ID + 12 位序列号
- 单机每毫秒可生成 4096 个 ID
- `sync.Once` 保证全局单例，`sync.Mutex` 保证并发安全
- 处理了时钟回拨和序列号溢出的边界情况

### 4.6 统一错误码体系

`pkg/errorx` 定义了全系统统一的错误码规范：

- 格式：`模块码(2位) + 错误类型(2位) + 具体错误(2位)`
- 6 位数字编码，通过 `GetMsg(code)` 获取中文描述
- RPC 层统一返回 `{code, msg}` 结构，不抛 Go error
- API 层透传 code/msg 给前端

| 模块 | 编码范围 |
|------|---------|
| 通用 | 99xxxx |
| 用户 | 10xxxx |
| 推文 | 11xxxx |
| 互动 | 12xxxx |
| 通知 | 13xxxx |
| 推荐 | 14xxxx |

### 4.7 Redis 缓存管理器

`pkg/cache/Manager` 封装了统一的 Redis 操作接口：

- Key 命名规范：`{module}:{dataType}:{id}`（如 `tweet:info:789`、`user:info:123`）
- 支持 String、Set、Hash 三种数据结构的常用操作
- 所有写入操作自动处理 JSON 序列化
- Set 操作自动处理 int64 类型转换

### 4.8 推荐系统缓存预热

recommendService 的推荐结果缓存策略：

- 用户请求 N 条推荐时，实际向 Python 召回 3N 条（最少 24 条）
- 多出的部分存入 Redis 缓存（60s TTL）
- 用户下次翻页时直接命中缓存，减少对 Python 服务的请求压力
- 缓存采用 Set + Hash 双结构存储，分别存 ID 列表和元信息

---

## 五、公共库（pkg/）

| 包 | 文件 | 职责 |
|----|------|------|
| `cache` | `manager.go` | Redis 缓存管理器，统一 Key 命名和操作接口 |
| `errorx` | `errorx.go` | 统一错误码定义和消息映射 |
| `idgen` | `snowflake.go` | 雪花算法 ID 生成器（单例、并发安全） |
| `jwt` | `jwtmiddleware.go` | JWT 认证中间件（提取 user_id 存入 context） |
| `types` | `usercenter.go` | 公共类型定义 |

---

## 六、基础设施

### 端口一览

| 服务 | 端口 |
|------|------|
| Nginx Gateway | 8888 |
| usercenter API / RPC | 1001 / 2001 |
| contentService API / RPC | 1002 / 2002 |
| interactService API / RPC | 1003 / 2003 |
| noticeService API / RPC | 1004 / 2004 |
| recommendService API / RPC | 1005 / 2005 |
| Python 推荐服务 | 2006 |
| Redis | 36379 |
| PostgreSQL | 54329 |
| Kafka（外部访问） | 9094 |
| Prometheus | 9090 |
| Grafana | 3001 |
| Asynqmon | 8980 |

### Kafka Topics

| Topic | 生产者 | 消费者 | 用途 |
|-------|--------|--------|------|
| `comment_create` | interactService RPC | interactMQ | 异步写入评论 |
| `like_tweet` | interactService RPC | interactMQ | 异步写入推文点赞 |
| `like_comment` | interactService RPC | interactMQ | 异步写入评论点赞 |
| `notice` | interactService RPC | noticeMQ | 生成通知记录 |
| `tweet_operation` | contentService RPC | — | 推文操作事件 |
| `recommend_tweet` | contentService RPC | Python 服务 | 推文入库到向量数据库 |
| `recommend_interaction` | interactService RPC | Python 服务 | 更新用户兴趣向量 |

---

## 七、数据库设计

PostgreSQL 数据库 `gozerox_db`，初始化脚本位于 `deploy/script/postgre/init/`，按序号执行：

| 文件 | 内容 |
|------|------|
| `01_create_tables_user.sql` | 用户表 |
| `02_create_tables_content.sql` | 推文表 |
| `03_create_tables_comment.sql` | 评论表 |
| `04_create_tables_likes.sql` | 点赞表（推文点赞、评论点赞） |
| `05_create_tables_user_follow.sql` | 用户关注表 |
| `04_create_tables_interaction.sql` | 互动相关表（用户点赞同步表） |
| `05_create_tables_notice.sql` | 通知表（点赞通知、评论通知） |
