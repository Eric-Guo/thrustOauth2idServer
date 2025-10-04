[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/Eric-Guo/thrustOauth2idServer)

# thrustOauth2idServer (http, monolith)

## Web 生成参数

```bash
sponge web http --module-name=thrust_oauth2id --server-name=thrustOauth2idServer --project-name=oauth2idThruster --d
b-driver=sqlite --db-dsn='/Users/guochunzhong/git/sso/oauth2id/storage/oauth2id_dev.sqlite3' --db-table=users --extended-api=true --embed=true
```

## 概述

1. **服务名称与定位**  
   - 是什么类型的服务？（如：用户管理微服务、订单处理 API、数据同步任务等）
   - 解决什么问题？（如：提供用户注册/登录能力、处理电商订单生命周期等）

2. **核心功能**  
   - 用 1-3 句话概括主要功能。

3. **服务边界**（可选）  
   - 如果是微服务，说明与其他服务的关系（如：依赖哪些服务/被哪些服务调用）。

## 技术栈

- 编程语言：go
- Web 框架：gin
- 配置管理：viper
- 日志：zap
- ORM: gorm
- 数据库：sqlite
- 缓存：go-redis
- 监控：prometheus+grafana
- 链路追踪：opentracing+jaeger
- 其他：...

## 目录结构

```text
.
├─ cmd                          # 应用程序入口目录
│   └─ thrustOauth2idServer                     # 服务名称
│       ├─ initial              # 初始化逻辑 (如配置加载、服务初始化等)
│       └─ main.go              # 主程序入口文件
├─ configs                      # 配置文件目录 (yaml 格式配置模板)
├─ deployments                  # 部署相关脚本 (二进制、Docker、K8S 部署)
├─ docs                         # 项目文档 (API 文档、设计文档等)
├─ internal                     # 内部实现代码 (对外不可见)
│   ├─ cache                    # 缓存相关实现 (Redis 或本地内存缓存封装)
│   ├─ config                   # 配置解析和结构体定义
│   ├─ dao                      # 数据访问层 (Database Access Object)
│   ├─ ecode                    # 错误码定义
│   ├─ handler                  # 业务逻辑处理层 (类似 Controller)
│   ├─ model                    # 数据模型/实体定义
│   ├─ routers                  # 路由定义和中间件
│   ├─ server                   # 服务启动
│   └─ types                    # 请求/响应结构体定义
├─ scripts                      # 实用脚本 (如代码生成、构建、运行、部署等)
├─ go.mod                       # Go 模块定义文件 (声明依赖)
├─ go.sum                       # Go 模块校验文件 (自动生成)
├─ Makefile                     # 项目构建自动化脚本
└─ README.md                    # 项目说明文档
```

代码采用分层架构，完整调用链路如下：

`cmd/thrustOauth2idServer/main.go` → `internal/server/http.go` → `internal/routers/router.go` → `internal/handler` → `internal/dao` → `internal/model`

其中 handler 层主要负责 API 处理，若需处理更复杂业务逻辑，建议在 handler 和 dao 之间额外添加业务逻辑层（如 `service`、`logic` 或 `biz` 等，自己定义）。

## 快速开始

### 1. 生成 openapi 文档

```bash
make docs
```

注：仅当新增或修改 API 时需要执行该命令，API 未变更时无需重复执行。

### 2. 编译和运行

```bash
make run
```

### 3. 测试 API

在浏览器访问 [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)，测试 HTTP API。

### 4. 启动 Rails 上游应用

1. 编辑 `configs/thrustOauth2idServer.yml`，将 `upstream.enabled` 设为 `true`，并配置 `command`、`args`、`workingDirectory`、`targetPort` 等字段，例如：

   ```yaml
   upstream:
     enabled: true
     workingDirectory: "/path/to/rails/app"
     command: "bin/rails"
     args:
       - "server"
     targetPort: 3000
   ```

2. 启动服务 `make run Config=configs/thrustOauth2idServer.yml`，Thruster 会在启动 HTTP 代理的同时拉起 Rails/Puma 进程，并通过 `PORT` 环境变量传递 `targetPort`。

## 开发指南

点击查看详细的 [**开发指南**](https://go-sponge.com/zh/guide/web/based-on-sql.html)。

## 部署

- [裸机部署](https://go-sponge.com/zh/deployment/binary.html)
- [Docker 部署](https://go-sponge.com/zh/deployment/docker.html)
- [K8S 部署](https://go-sponge.com/zh/deployment/kubernetes.html)
