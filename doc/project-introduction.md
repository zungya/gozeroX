# gozeroX 项目介绍

## 一、项目简介

gozeroX 是一个**类 Twitter / 微博的社交平台后端系统**，采用 go-zero 微服务框架构建。项目将业务拆分为 5 个独立服务（用户中心、内容、互动、通知、推荐），配合 PostgreSQL、Redis、Kafka 等中间件，实现了一个完整的社交平台后端。

> **设计理念**：项目为毕业设计作品，**可读性优先于性能优化**，代码力求清晰易懂。

---

## 二、核心功能

### 2.1 用户系统
- 手机号 + 密码注册 / 登录
- JWT Token 认证（HS256 签名，24h 过期）
- 用户信息查询（昵称、头像、简介、关注/粉丝/推文数）
- 批量用户简要信息查询（供其他服务使用）

### 2.2 推文系统
- 推文发布（支持图片 URL 列表、标签、公开/私密设置）
- 推文删除（软删除，校验 uid 归属）
- 用户主页推文列表（游标分页）
- 推文详情查询、批量推文查询

### 2.3 评论系统
- 多级评论体系：根评论（comment）+ 回复（reply）
- 发表评论 / 回复，删除评论
- 获取推文的顶级评论列表（游标分页）
- 获取评论下的回复列表（游标分页）

### 2.4 点赞系统
- 推文点赞 / 取消点赞
- 评论点赞 / 取消点赞
- 用户所有点赞关系一次性拉取（登录时同步到前端，后续本地判断）
- 增量同步机制（基于 `updated_at` 游标）

### 2.5 通知系统
- 点赞通知**聚合展示**：同一推文/评论被多人点赞，只显示一条通知
  - 示例：「用户A、用户B 等 10 人赞了你的推文」
- 评论通知**独立展示**：每条评论 / 回复生成独立通知
- 未读数统计（点赞未读、评论未读、总未读）
- 标记已读（全部已读 / 按类型标记）

### 2.6 推荐系统
- Go + Python 混合架构
- Python 端基于 CLIP 多模态 + 向量召回算法
- Go 端负责 RPC 服务和缓存预热
- 推荐结果缓存（60s TTL），减少对 Python 服务的请求压力

---

## 三、系统架构

### 3.1 服务拓扑

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
     │            │       ┌────┴────┐        │       ┌────┴────┐
     │            │       │Kafka MQ │        │       │HTTP调用 │
     │            │       └─────────┘        │       │Python   │
     │            │                          │       │  :2006  │
     └────────────┴──────────┬───────────────┘       └─────────┘
                             ▼
                    ┌─────────────────┐
                    │  PostgreSQL 17  │
                    │  Redis 7.4      │
                    │  Kafka 3.9      │
                    └─────────────────┘
```

### 3.2 分层架构

每个微服务统一采用 **API → RPC → Model** 三层架构：

```
app/{serviceName}/
├── cmd/
│   ├── api/                 # HTTP API 层
│   │   ├── desc/            # .api 定义文件（goctl 生成代码的源）
│   │   ├── etc/             # YAML 配置
│   │   └── internal/
│   │       ├── config/      # 配置映射结构体
│   │       ├── handler/     # HTTP Handler（路由注册）
│   │       ├── logic/       # 业务编排（调 RPC，不含业务逻辑）
│   │       ├── svc/         # ServiceContext（持有 RPC 客户端）
│   │       └── types/       # 请求/响应类型
│   ├── rpc/                 # gRPC RPC 层
│   │   ├── pb/              # .proto 定义 + goctl 生成的 Go 代码
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

**设计原则：**
- **API 层**只做参数校验和 RPC 调用，不包含业务逻辑
- **RPC 层**不校验参数（由上游 API 保证），专注业务处理
- **Model 层**封装数据库操作，基于 goctl 自动生成的 CRUD 代码

### 3.3 服务依赖关系

```
usercenter-api  ──→ usercenter-rpc  ──→ PostgreSQL, Redis
content-api     ──→ content-rpc     ──→ PostgreSQL, Redis, Kafka, usercenter-rpc
interact-api    ──→ interact-rpc    ──→ PostgreSQL, Redis, Kafka, usercenter-rpc, content-rpc
interact-mq     ──→                   PostgreSQL, Redis, Kafka, content-rpc
notice-api      ──→ notice-rpc       ──→ PostgreSQL, Redis, usercenter-rpc
notice-mq       ──→                   PostgreSQL, Redis, Kafka
recommend-api   ──→ recommend-rpc    ──→ Redis, content-rpc, Python Recall(:2006)
```

---

## 四、项目目录结构

```
gozeroX/
├── app/                              # 业务微服务
│   ├── usercenter/                    # 用户中心服务
│   │   ├── cmd/api/                   #   HTTP API 层 (:1001)
│   │   ├── cmd/rpc/                   #   gRPC RPC 层 (:2001)
│   │   └── model/                     #   数据模型
│   ├── contentService/                # 内容服务
│   │   ├── cmd/api/                   #   HTTP API 层 (:1002)
│   │   ├── cmd/rpc/                   #   gRPC RPC 层 (:2002)
│   │   └── model/                     #   数据模型
│   ├── interactService/               # 互动服务
│   │   ├── cmd/api/                   #   HTTP API 层 (:1003)
│   │   ├── cmd/rpc/                   #   gRPC RPC 层 (:2003)
│   │   ├── cmd/mq/                    #   Kafka 消费者
│   │   └── model/                     #   数据模型（6 张表）
│   ├── noticeService/                 # 通知服务
│   │   ├── cmd/api/                   #   HTTP API 层 (:1004)
│   │   ├── cmd/rpc/                   #   gRPC RPC 层 (:2004)
│   │   ├── cmd/mq/                    #   Kafka 消费者
│   │   └── model/                     #   数据模型（2 张表）
│   └── recommendService/              # 推荐服务
│       ├── cmd/api/                   #   HTTP API 层 (:1005)
│       └── cmd/rpc/                   #   gRPC RPC 层 (:2005)
├── pkg/                               # 公共库
│   ├── cache/manager.go               #   Redis 缓存管理器
│   ├── errorx/errorx.go              #   统一错误码
│   ├── idgen/snowflake.go            #   雪花算法 ID 生成器
│   ├── jwt/jwtmiddleware.go          #   JWT 认证中间件
│   └── types/usercenter.go           #   公共类型定义
├── deploy/                            # 部署配置
│   ├── k8s/                           #   Kubernetes 清单（namespace + 基础设施 + 服务）
│   ├── nginx/conf.d/                  #   Nginx 网关配置
│   ├── prometheus/server/             #   Prometheus 采集配置
│   └── script/postgre/init/           #   SQL 初始化脚本（5 个）
├── data/                              # 持久化数据卷（已 gitignore）
├── doc/                               # 项目文档
├── docker-compose.yml                 # 应用服务容器（13 个）
├── docker-compose-env.yml             # 基础设施容器（5 个）
├── Dockerfile                         # 多阶段构建（Go 编译 → Alpine 运行）
├── build-images.sh                    # 构建所有 Docker 镜像
├── start-local.sh                     # 本地启动所有 Go 服务
├── stop-local.sh                      # 停止本地服务
├── test.sh                            # 集成测试脚本（12 个测试用例）
├── go.work                            # Go workspace（17 个模块）
└── readme.md                          # 项目说明
```

---

## 五、API 总览

### 5.1 用户中心 — usercenter

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/usercenter/v1/user/register` | 无 | 用户注册 |
| POST | `/usercenter/v1/user/login` | 无 | 用户登录（返回 JWT） |
| POST | `/usercenter/v1/user/detail` | JWT | 获取用户详情 |

### 5.2 内容服务 — contentService

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/contentService/v1/listTweets` | JWT | 用户推文列表（游标分页） |
| POST | `/contentService/v1/createTweet` | JWT | 发布推文 |
| DELETE | `/contentService/v1/deleteTweet/{snowTid}` | JWT | 删除推文 |

### 5.3 互动服务 — interactService

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/interactService/v1/createComment` | JWT | 发表评论 / 回复 |
| DELETE | `/interactService/v1/deleteComment/{snowCid}` | JWT | 删除评论 |
| GET | `/interactService/v1/getComments` | JWT | 获取推文顶级评论 |
| GET | `/interactService/v1/getReplies` | JWT | 获取评论回复列表 |
| POST | `/interactService/v1/like` | JWT | 点赞 / 取消点赞 |
| GET | `/interactService/v1/getUserLikesAll` | JWT | 获取用户所有点赞关系 |

### 5.4 通知服务 — noticeService

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/noticeService/v1/getNotices` | JWT | 获取通知列表 |
| GET | `/noticeService/v1/getUnreadCount` | JWT | 获取未读数量 |
| POST | `/noticeService/v1/markRead` | JWT | 标记已读 |

### 5.5 推荐服务 — recommendService

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/recommendService/v1/feed` | JWT | 获取推荐推文流 |

---

## 六、数据库表结构

数据库名 `gozerox_db`，共 8 张业务表：

| 表名 | 说明 | 主键 |
|------|------|------|
| `user` | 用户主表 | `uid` (BIGSERIAL) |
| `tweet` | 推文表 | `snow_tid` (雪花ID) |
| `comment` | 根评论表 | `snow_cid` (雪花ID) |
| `reply` | 回复表（子评论） | `snow_cid` (雪花ID) |
| `likes_tweet` | 推文点赞表 | `snow_likes_id` (雪花ID) |
| `likes_comment` | 评论点赞表 | `snow_likes_id` (雪花ID) |
| `user_like_sync` | 用户点赞同步时间表 | `uid` |
| `notice_like` | 点赞通知表（聚合） | `snow_nid` (雪花ID) |
| `notice_comment` | 评论通知表（逐条） | `snow_nid` (雪花ID) |

**设计特点：**
- 所有业务 ID 使用雪花算法生成（全局唯一、时间有序）
- 所有时间字段使用毫秒级 Unix 时间戳（BIGINT）
- 状态字段统一：0=正常，1=删除，2=审核
- PostgreSQL 数组类型存储 `media_urls` 和 `tags`，配合 GIN 索引
- 使用视图过滤已删除数据（如 `tweet_normal`、`comment_normal`）

---

## 七、Kafka 事件流

```
contentService RPC ──→ recommend_tweet topic     ──→ Python 推荐服务（向量化入库）

interactService RPC ─→ comment_create topic      ──→ interactMQ（异步写评论入 DB）
                    ─→ like_tweet topic           ──→ interactMQ（异步写推文点赞入 DB）
                    ─→ like_comment topic         ──→ interactMQ（异步写评论点赞入 DB）
                    ─→ notice topic               ──→ noticeMQ（生成通知记录）
                    ─→ recommend_interaction topic──→ Python 推荐服务（更新兴趣向量）
```

---

## 八、端口一览

| 服务 | 端口 | 说明 |
|------|------|------|
| Nginx Gateway | 8888 | 统一入口 |
| usercenter API / RPC | 1001 / 2001 | 用户中心 |
| contentService API / RPC | 1002 / 2002 | 内容服务 |
| interactService API / RPC | 1003 / 2003 | 互动服务 |
| noticeService API / RPC | 1004 / 2004 | 通知服务 |
| recommendService API / RPC | 1005 / 2005 | 推荐服务 |
| Python 推荐服务 | 2006 | 召回服务 |
| PostgreSQL | 54329 | 数据库（外部映射端口） |
| Redis | 36379 | 缓存（外部映射端口） |
| Kafka | 9094 | 消息队列（外部访问端口） |
| Prometheus | 9090 | 监控 |
| Grafana | 3001 | 监控面板 |
