# BBC MCP 服务详细设计文档

> 版本：1.0
> 日期：2026-06-06
> 基于：docs/task1.md 需求规格

---

## 1. 概述

### 1.1 项目背景

BBC MCP 服务是一个基于 MCP（Model Context Protocol）协议的中间层服务，为 AI 助手提供对 minibbc 数据中心设备信息的安全查询能力。服务通过 MCP Tool 机制暴露三个核心查询功能，使 AI 助手能够实时获取设备连接数、租户设备列表和设备连接状态。

### 1.2 技术选型

| 维度 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.26+ | 项目已确立 |
| MCP 框架 | mark3labs/mcp-go v0.54.1 | 项目已有依赖，提供 Server/Tool/SSE 完整实现 |
| 传输协议 | SSE (Server-Sent Events) | 参考示例实现，支持 HTTP 长连接 |
| 数据库 | MySQL | 通过 database/sql + go-sql-driver 访问 scloudb |
| 缓存 | Redis | 通过 go-redis 访问设备状态信息 |
| 配置管理 | YAML + 环境变量 | 结构化配置，支持多环境 |

### 1.3 设计原则

- **可扩展性**：Tool 注册采用插件化模式，新增 Tool 无需修改核心启动逻辑
- **关注点分离**：Tool Handler / Repository / Infrastructure 三层隔离
- **防御性编程**：分页硬上限、SQL 参数化、命令路径白名单
- **参考对齐**：API 风格和项目结构参照 example 目录已有示例

---

## 2. 项目结构

```
bbc-mcp/
├── etc/
│   └── bbc-mcp.yaml              # 服务配置文件
├── internal/
│   ├── config/
│   │   └── config.go             # 配置结构定义与加载
│   ├── infrastructure/
│   │   ├── mysql.go              # MySQL 连接池初始化
│   │   └── redis.go              # Redis 客户端初始化
│   ├── repository/
│   │   └── device_repo.go        # 设备数据访问层（MySQL + Redis）
│   ├── tool/
│   │   ├── registry.go           # Tool 注册中心（可扩展核心）
│   │   ├── gateway_status.go     # Tool: 查询网关连接数
│   │   ├── device_list.go        # Tool: 查询租户设备列表
│   │   └── device_status.go      # Tool: 查询设备连接状态
│   └── types/
│       └── device.go             # 设备相关数据结构
├── cmd/
│   └── bbc-mcp/
│       └── main.go               # 服务入口
├── example/                      # 参考示例（已有）
├── docs/                         # 文档
├── go.mod
└── go.sum
```

---

## 3. 配置设计

### 3.1 配置文件 etc/bbc-mcp.yaml

```yaml
Server:
  Name: bbc-mcp
  Host: localhost
  Port: 9000

Database:
  Host: 127.0.0.1
  Port: 3306
  Name: scloudb
  User: root
  Password: "your_password"
  MaxOpenConns: 10
  MaxIdleConns: 5

Redis:
  Host: 127.0.0.1
  Port: 6379
  Password: ""
  DB: 0

BbcTool:
  Path: /usr/local/bin/bbc-tool
  Timeout: 30
```

### 3.2 配置结构体 internal/config/config.go

```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    BbcTool  BbcToolConfig
}

type ServerConfig struct {
    Name string
    Host string
    Port int
}

type DatabaseConfig struct {
    Host         string
    Port         int
    Name         string
    User         string
    Password     string
    MaxOpenConns int
    MaxIdleConns int
}

type RedisConfig struct {
    Host     string
    Port     int
    Password string
    DB       int
}

type BbcToolConfig struct {
    Path    string
    Timeout int // 秒
}
```

---

## 4. MCP Tool 详细设计

### 4.1 Tool 注册中心（可扩展核心）

`internal/tool/registry.go` 定义 Tool 注册接口，所有 Tool 通过 `RegisterAll` 统一注入 MCP Server。新增 Tool 只需：

1. 在 `internal/tool/` 下新建文件
2. 实现 `ToolDefinition` 接口
3. 在 `RegisterAll` 中追加注册调用

```go
package tool

import (
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// ToolDefinition 描述一个 MCP Tool 的注册信息
type ToolDefinition struct {
    Tool    mcp.Tool
    Handler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

// RegisterAll 将所有 Tool 注册到 MCP Server
// 新增 Tool 时只需在此追加一行
func RegisterAll(s *server.MCPServer, deps *Dependencies) {
    definitions := []ToolDefinition{
        newGatewayStatus(deps),
        newDeviceList(deps),
        newDeviceStatus(deps),
    }
    for _, def := range definitions {
        s.AddTool(def.Tool, def.Handler)
    }
}

// Dependencies 所有 Tool 共享的外部依赖
type Dependencies struct {
    DB      *sql.DB
    Redis   *redis.Client
    Config  *config.Config
}
```

### 4.2 Tool: gateway_status — 查询网关连接数

**功能**：执行 `bbc-tool status gateway` 命令，返回 minibbc 数据中心当前设备连接数。

| 项目 | 说明 |
|------|------|
| Tool 名称 | `gateway_status` |
| 描述 | 查询当前 minibbc 数据中心设备连接数 |
| 输入参数 | 无 |
| 返回内容 | 命令标准输出的文本内容 |
| 错误场景 | 命令执行超时、命令不存在、非零退出码 |

**实现要点**：

```
internal/tool/gateway_status.go
```

- 使用 `os/exec.CommandContext` 执行命令，传入配置中的 `BbcTool.Path`
- 命令参数固定为 `["status", "gateway"]`
- 通过 `context.WithTimeout` 设置超时（配置中的 BbcTool.Timeout）
- 成功时将 stdout 以 `mcp.NewToolResultText` 返回
- 失败时返回 `mcp.NewToolResultError` 并附带错误信息
- 安全约束：只执行白名单路径下的 bbc-tool，不接受用户输入的命令或参数

**伪代码**：

```
func gatewayStatusHandler(ctx, req):
    toolPath = deps.Config.BbcTool.Path
    timeout = deps.Config.BbcTool.Timeout

    cmdCtx, cancel = context.WithTimeout(ctx, timeout * time.Second)
    defer cancel()

    cmd = exec.CommandContext(cmdCtx, toolPath, "status", "gateway")
    output, err = cmd.CombinedOutput()
    if err != nil:
        return NewToolResultError("执行 bbc-tool 失败: " + err.Error())

    return NewToolResultText(string(output))
```

### 4.3 Tool: device_list — 查询租户设备列表

**功能**：根据租户 ID 查询设备列表，支持按设备名称模糊搜索和分页。

| 项目 | 说明 |
|------|------|
| Tool 名称 | `device_list` |
| 描述 | 查询某个租户的设备列表，支持按名称筛选和分页 |
| 参数 `ccode` | string, 必填, 租户 ID |
| 参数 `name` | string, 可选, 设备名称（模糊匹配） |
| 参数 `offset` | integer, 必填, 查询偏移量 |
| 返回内容 | 设备列表 JSON（每项含 id, name, zp_id, status） |
| 分页限制 | 硬编码 LIMIT 20，不可被调用方覆盖 |

**Tool Schema 定义**：

```go
mcp.NewTool("device_list",
    mcp.WithDescription("查询某个租户的设备列表，支持按名称筛选和分页"),
    mcp.WithString("ccode",
        mcp.Required(),
        mcp.Description("租户ID（公司代码）"),
    ),
    mcp.WithString("name",
        mcp.Description("设备名称（可选，模糊匹配）"),
    ),
    mcp.WithInteger("offset",
        mcp.Required(),
        mcp.Description("查询偏移量"),
    ),
)
```

**SQL 逻辑**（在 `internal/repository/device_repo.go`）：

- 无 name 参数：
  ```sql
  SELECT id, name, zp_id, status FROM device WHERE ccode = ? LIMIT 20 OFFSET ?
  ```
- 有 name 参数：
  ```sql
  SELECT id, name, zp_id, status FROM device WHERE ccode = ? AND name LIKE ? LIMIT 20 OFFSET ?
  ```
  其中 name 参数值格式为 `%name%`（在 Go 代码中拼接，非 SQL 拼接）

**返回结构**：

```json
{
  "total": true,
  "devices": [
    {
      "id": 260860,
      "name": "sz1_af_Connect_7000",
      "zp_id": null,
      "status": 0
    }
  ],
  "limit": 20,
  "offset": 0
}
```

**防御性约束**：
- `ccode` 不能为空，否则返回参数错误
- `offset` 不能为负数，否则置为 0
- LIMIT 20 硬编码在 SQL 中，不作为参数暴露
- 所有 SQL 参数通过 `?` 占位符传递，防止注入
- LIKE 通配符在 Go 侧拼接为 `%name%`，而非直接拼接 SQL

**错误场景**：
- 数据库连接失败
- 查询执行超时（建议 context 超时 5s）
- 参数校验不通过

### 4.4 Tool: device_status — 查询设备连接状态

**功能**：根据设备 ID 查询设备的详细信息和实时连接状态，数据来源为 MySQL + Redis。

| 项目 | 说明 |
|------|------|
| Tool 名称 | `device_status` |
| 描述 | 查询某个设备的详细信息和实时连接状态 |
| 参数 `id` | integer, 必填, 设备 ID |
| 返回内容 | 设备基础信息（MySQL）+ 实时状态（Redis）合并后的 JSON |

**Tool Schema 定义**：

```go
mcp.NewTool("device_status",
    mcp.WithDescription("查询某个设备的详细信息和实时连接状态"),
    mcp.WithInteger("id",
        mcp.Required(),
        mcp.Description("设备ID"),
    ),
)
```

**数据获取流程**：

```
1. MySQL 查询: SELECT * FROM device WHERE id = ? LIMIT 1
2. 若 MySQL 无记录, 返回设备不存在错误
3. Redis 查询基础信息: HGETALL device:<id>:status:basic
4. Redis 查询状态信息: HGETALL device_<id>
5. 合并 MySQL 数据 + Redis 数据, 返回完整 JSON
```

**Redis Key 规则**：

| 数据类型 | Key 格式 | 命令 | 说明 |
|----------|----------|------|------|
| 基础信息 | `device:{id}:status:basic` | `HGETALL` | 包含 product_name, tenant, heartbeat, cpu, mem, disk, version 等 |
| 状态信息 | `device_{id}` | `HGETALL` | 包含 status 字段（0=未激活, 1=启用, 2=离线, 3=在线, 4=告警, 5=停用） |

**返回结构**：

```json
{
  "device": {
    "id": 223166,
    "gwid": "",
    "ccode": 56626662,
    "type": 3,
    "name": "sz1_af_Connect_7000",
    "pwd": "***",
    "model": null,
    "status": 0,
    "ip": null,
    "mac": null,
    "zp_id": null,
    "org_id": "0",
    "alias": "sz1_af_Connect_7000",
    "sn": "",
    "version": null
  },
  "realtime": {
    "product_name": "AF",
    "tenant": "56626662",
    "zp_id": "8001",
    "cpu": "98",
    "mem": "41",
    "disk": "64",
    "version": "8.0.7",
    "session_num": "20",
    "user": "10",
    "bandwidth": "0",
    "last_online_time": "2026-06-05-17:56:07",
    "last_offline_time": "2026-05-14-16:41:11",
    "status": "3"
  }
}
```

**安全约束**：
- `pwd` 字段在返回时脱敏为 `"***"`
- Redis 查询失败不影响 MySQL 数据返回，`realtime` 字段设为 `null` 并附说明
- 设备不存在时返回明确的错误信息

---

## 5. 基础设施层设计

### 5.1 MySQL 连接管理 internal/infrastructure/mysql.go

```go
package infrastructure

func NewMySQLDataSource(cfg *config.DatabaseConfig) (*sql.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(cfg.MaxOpenConns)
    db.SetMaxIdleConns(cfg.MaxIdleConns)
    db.SetConnMaxLifetime(time.Hour)

    // 健康检查
    if err = db.Ping(); err != nil {
        return nil, err
    }

    return db, nil
}
```

### 5.2 Redis 客户端管理 internal/infrastructure/redis.go

```go
package infrastructure

func NewRedisClient(cfg *config.RedisConfig) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password: cfg.Password,
        DB:       cfg.DB,
    })

    // 健康检查
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }

    return client, nil
}
```

---

## 6. 数据访问层设计

### 6.1 DeviceRepository internal/repository/device_repo.go

```go
package repository

type DeviceRepository struct {
    db    *sql.DB
    redis *redis.Client
}

func NewDeviceRepository(db *sql.DB, redis *redis.Client) *DeviceRepository {
    return &DeviceRepository{db: db, redis: redis}
}

// ListByCcode 按租户 ID 查询设备列表（分页）
func (r *DeviceRepository) ListByCcode(ctx context.Context, ccode string, name string, offset int) ([]DeviceItem, error)

// GetByID 按 ID 查询设备完整信息
func (r *DeviceRepository) GetByID(ctx context.Context, id int) (*Device, error)

// GetDeviceBasicInfo 从 Redis 获取设备基础实时信息
func (r *DeviceRepository) GetDeviceBasicInfo(ctx context.Context, id int) (map[string]string, error)

// GetDeviceOnlineStatus 从 Redis 获取设备在线状态
func (r *DeviceRepository) GetDeviceOnlineStatus(ctx context.Context, id int) (map[string]string, error)
```

**DeviceItem 结构**（列表返回用）：

```go
type DeviceItem struct {
    ID     int     `json:"id"`
    Name   string  `json:"name"`
    ZpID   *int    `json:"zp_id"`
    Status *int    `json:"status"`
}
```

**Device 结构**（详情返回用，映射 device 表全字段）：

```go
type Device struct {
    ID              int        `json:"id"`
    GwID            string     `json:"gwid"`
    Ccode           int        `json:"ccode"`
    Type            int        `json:"type"`
    Name            string     `json:"name"`
    Pwd             string     `json:"-"`            // JSON 序列化时忽略
    Model           *string    `json:"model"`
    IsNfv           *bool      `json:"is_nfv"`
    ProdLine        *int       `json:"prod_line"`
    ProdName        *string    `json:"prod_name"`
    Version         *string    `json:"version"`
    IsCustom        *bool      `json:"is_custom"`
    ActiveTime      *time.Time `json:"active_time"`
    LstOnlineTime   *time.Time `json:"lst_online_time"`
    LstOfflineTime  *time.Time `json:"lst_offline_time"`
    Status          *int       `json:"status"`
    IP              *string    `json:"ip"`
    Mac             *string    `json:"mac"`
    Eth0Mac         *string    `json:"eth0_mac"`
    ZpID            *int       `json:"zp_id"`
    BranchID        *int       `json:"branch_id"`
    ParentID        *int       `json:"parent_id"`
    TplID           *int       `json:"tpl_id"`
    License         *string    `json:"license"`
    DeviceBelong    int        `json:"device_belong"`
    CreateSource    int        `json:"create_source"`
    UpdateSource    int        `json:"update_source"`
    LastUpdateTime  *time.Time `json:"last_update_time"`
    Desc            *string    `json:"desc"`
    SN              *string    `json:"sn"`
    RegionID        int        `json:"region_id"`
    Tags            *string    `json:"tags"`
    Platforms       *string    `json:"platforms"`
    Addr            *string    `json:"addr"`
    OrderID         *string    `json:"order_id"`
    OrgID           string     `json:"org_id"`
    Alias           string     `json:"alias"`
    Manager         *string    `json:"manager"`
    Email           *string    `json:"email"`
    Remark          *string    `json:"remark"`
    ConnCode        *string    `json:"conn_code"`
    ConnCodeGenTime *time.Time `json:"conn_code_gen_time"`
    AddType         *int       `json:"add_type"`
    CreateTime      *time.Time `json:"create_time"`
    VID             string     `json:"vid"`
}
```

**Pwd 脱敏方案**：Device 结构体中 `Pwd` 字段标记 `json:"-"` 不序列化，在 Handler 层组装返回时使用独立的 `DeviceResponse` 结构体，将 Pwd 设为 `"***"`。

---

## 7. 服务启动流程

### 7.1 cmd/bbc-mcp/main.go

```
func main():
    1. 加载配置 etc/bbc-mcp.yaml
    2. 初始化 MySQL 连接
    3. 初始化 Redis 连接
    4. 构建 Dependencies（注入 DB、Redis、Config）
    5. 创建 MCP Server:
         - 名称: "bbc-mcp"
         - 版本: "1.0.0"
         - 开启 Tool 能力
    6. 调用 tool.RegisterAll(server, deps) 注册所有 Tool
    7. 创建 SSE Server:
         - BaseURL: http://{Host}:{Port}
    8. 启动 SSE Server, 监听 :{Port}
    9. 优雅关闭: 监听 SIGINT/SIGTERM → 关闭 DB/Redis 连接
```

参考 example2 的封装模式，使用 `MCPServer` 结构体包裹 `server.MCPServer`：

```go
type BBCMCPServer struct {
    server *server.MCPServer
    deps   *tool.Dependencies
}

func NewBBCMCPServer(deps *tool.Dependencies) *BBCMCPServer {
    s := server.NewMCPServer(
        "bbc-mcp",
        "1.0.0",
        server.WithToolCapabilities(true),
    )
    tool.RegisterAll(s, deps)
    return &BBCMCPServer{server: s, deps: deps}
}

func (s *BBCMCPServer) ServeSSE(addr string) *server.SSEServer {
    return server.NewSSEServer(
        s.server,
        server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
    )
}
```

---

## 8. 新增依赖

当前 go.mod 仅有 `mark3labs/mcp-go`，需新增：

| 依赖 | 用途 |
|------|------|
| `github.com/go-sql-driver/mysql` | MySQL 驱动 |
| `github.com/redis/go-redis/v9` | Redis 客户端 |
| `gopkg.in/yaml.v3` | YAML 配置解析 |

---

## 9. 错误处理策略

| 错误类型 | 处理方式 | 返回给 MCP Client |
|----------|----------|-------------------|
| 参数校验失败 | 在 Handler 入口校验，提前返回 | `NewToolResultError("参数错误: ...")` |
| 数据库连接失败 | 启动时 Ping 检测，运行时重试 | `NewToolResultError("数据库连接失败")` |
| 数据库查询超时 | context 超时 5s | `NewToolResultError("查询超时")` |
| Redis 连接失败 | 启动时 Ping 检测 | `NewToolResultError("Redis 连接失败")` |
| Redis 查询无数据 | 不视为错误，返回 null | realtime 字段为 null |
| 设备不存在 | 判断 Rows 影响 | `NewToolResultError("设备不存在")` |
| bbc-tool 执行失败 | 捕获 exit code + stderr | `NewToolResultError("命令执行失败: ...")` |
| bbc-tool 超时 | context 超时 | `NewToolResultError("命令执行超时")` |

---

## 10. 安全设计

### 10.1 SQL 注入防护
- 所有 SQL 使用 `?` 参数化查询，禁止字符串拼接 SQL
- LIKE 通配符 `%name%` 在 Go 代码中构建后作为参数传入

### 10.2 命令执行安全
- bbc-tool 路径仅从配置文件读取，不接受用户输入
- 命令参数硬编码为 `["status", "gateway"]`，不接受外部参数
- 使用 `exec.CommandContext` 设置超时，防止命令挂起

### 10.3 数据脱敏
- `device.pwd` 字段返回时替换为 `"***"`，`json:"-"` 标签阻止自动序列化
- 日志中不打印数据库密码和 Redis 密码

### 10.4 分页限制
- `device_list` 的 LIMIT 20 硬编码在 SQL 中，不可通过参数覆盖
- 防止大量数据查询导致数据库或服务异常

---

## 11. 设备状态码映射

MySQL `device.status` 字段和 Redis `status` 字段的含义一致：

| 值 | 含义 |
|----|------|
| 0 | 未激活 |
| 1 | 启用（含离线/在线/告警子状态） |
| 2 | 离线 |
| 3 | 在线 |
| 4 | 告警 |
| 5 | 停用 |

Redis `device_{id}` 中的 `status` 字段提供实时状态（通常为 2/3/4），MySQL 中的 `status` 为持久化状态。查询设备连接状态时优先使用 Redis 的实时值。

---

## 12. 接口总览

| Tool 名称 | 输入参数 | 返回数据 | 数据源 |
|------------|----------|----------|--------|
| `gateway_status` | 无 | bbc-tool 命令输出文本 | bbc-tool CLI |
| `device_list` | ccode(必填), name(可选), offset(必填) | 设备列表 JSON | MySQL |
| `device_status` | id(必填) | 设备详情 + 实时状态 JSON | MySQL + Redis |

---

## 13. 扩展指南

新增一个 MCP Tool 的步骤：

1. 在 `internal/tool/` 下新建文件，如 `new_feature.go`
2. 实现 `func newNewFeature(deps *Dependencies) ToolDefinition`，返回 Tool Schema 和 Handler
3. 在 `internal/tool/registry.go` 的 `RegisterAll` 函数中追加 `newNewFeature(deps)` 调用
4. 如需新的数据源，在 `internal/repository/` 中添加对应方法
5. 如需新的基础设施，在 `internal/infrastructure/` 中初始化，并添加到 `Dependencies` 结构体

无需修改 `main.go` 或其他 Tool 文件，符合开闭原则。
