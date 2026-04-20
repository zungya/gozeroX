# gozeroX

类 Twitter/微博的社交平台后端，基于 **go-zero** 微服务框架构建。

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.26 |
| 框架 | go-zero v1.10 |
| 数据库 | PostgreSQL 17 |
| 缓存 | Redis 7.4 |
| 消息队列 | Kafka 3.9 |
| 容器化 | Docker Compose / OrbStack |
| 监控 | Prometheus + Grafana |

## 服务架构

```
Nginx Gateway (:8888)
        │
   ┌────┼────┬─────────────┬──────┐
   ▼    ▼    ▼             ▼      ▼
user  content interact  notice  recommend
:1001  :1002   :1003    :1004    :1005
   │    │       │         │       │
   ▼    ▼       ▼         ▼       ▼
 RPC   RPC     RPC       RPC     RPC → Python
:2001  :2002  :2003     :2004   :2005
   │    │       │         │
   └────┼───────┼─────────┘
        ▼       ▼
   PostgreSQL  Redis  Kafka
```

每个服务采用 **API → RPC → Model** 三层架构，API 层负责参数校验与路由，RPC 层负责核心业务逻辑。

## 核心功能

- 用户注册 / 登录（JWT 认证）
- 推文发布（支持图片、标签）、查询、删除
- 评论系统（支持多级回复）
- 点赞（推文 / 评论）
- 通知系统（点赞聚合、评论独立）
- 推荐系统（Go + Python 混合架构）

## 项目结构

```
gozeroX/
├── app/                        # 业务微服务
│   ├── usercenter/             # 用户中心
│   ├── contentService/         # 内容服务
│   ├── interactService/        # 互动服务
│   ├── noticeService/          # 通知服务
│   └── recommendService/       # 推荐服务
├── pkg/                        # 公共库（缓存、错误码、JWT、ID 生成）
├── deploy/                     # 部署配置（Nginx、Prometheus、SQL 初始化脚本）
├── docker-compose-env.yml      # 基础设施容器
├── docker-compose.yml          # 应用容器
└── go.work                     # Go workspace
```

## 快速开始

```bash
# 启动基础设施
docker compose -f docker-compose-env.yml up -d

# 同步 workspace
go work sync

# 启动服务（示例）
go run app/usercenter/cmd/api/usercenter.go -f app/usercenter/cmd/api/etc/usercenter.yaml
go run app/usercenter/cmd/rpc/usercenter.go -f app/usercenter/cmd/rpc/etc/usercenter.yaml
```
