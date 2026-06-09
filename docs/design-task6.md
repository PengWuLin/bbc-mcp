# BBC MCP 服务 — 系统 Prompt 设计文档

> 版本：1.0
> 日期：2026-06-09
> 基于：docs/task6.md 需求规格

---

## 1. 概述

### 1.1 背景

BBC MCP 服务业务性很强，返回的数据中设备类型、状态码、Redis 字段等均使用数字编码或缩写键名，AI 助手无法仅从 JSON 返回值理解业务含义。此外，三个 MCP Tool 各有特定的参数约束、分页限制和错误语义，需要在系统层面提供统一的使用指引。

### 1.2 目标

- 提供系统级 Prompt，向 AI 助手说明服务的业务背景、数据字段含义和各 Tool 的使用方法
- Prompt 内容覆盖：数据库表字段详细说明、Redis 字段说明、Tool 参数与返回值说明、设备/状态码映射表
- 参考 example1 的 Prompt 注册模式实现

### 1.3 非目标

- 不创建多轮对话式 Prompt（当前为单向知识注入）
- 不将 Prompt 内容写入外部文件（直接写入 Go 代码，便于版本管理）

---

## 2. MCP Prompt 机制说明

基于 example1 的实现模式，MCP Prompt 的生命周期如下：

```
┌──────────────────────────────────────────────────────┐
│  Server: s.AddPrompt(promptDef, handlerFunc)         │
│                                                      │
│  promptDef := mcp.NewPrompt("name",                  │
│      mcp.WithPromptDescription("描述"),               │
│      mcp.WithArgument("arg", mcp.ArgumentDescription  │
│          ("参数说明")),                                │
│  )                                                   │
│                                                      │
│  handler returns: mcp.NewGetPromptResult(             │
│      "描述",                                          │
│      []mcp.PromptMessage{                            │
│          mcp.NewPromptMessage(mcp.RoleAssistant,      │
│              mcp.NewTextContent("内容...")),           │
│      },                                              │
│  )                                                   │
└──────────────────────────┬───────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────┐
│  Client:                                             │
│  1. ListPrompts() -> 发现可用 prompt 列表              │
│  2. GetPrompt({Name: "xxx", Arguments: {...}})       │
│     -> 获取渲染后的消息内容                              │
└──────────────────────────────────────────────────────┘
```

**关键点**：
- Prompt 内容在服务端生成，客户端通过 MCP 协议获取，不硬编码在客户端
- 支持参数化 Prompt（通过 `Arguments` 实现），也支持无参数静态 Prompt
- 服务端需声明 `WithPromptCapabilities(true)` 告知客户端支持 Prompt

---

## 3. 系统 Prompt 设计

### 3.1 Prompt 注册

注册一个名为 `bbc-guide` 的 Prompt，无参数，返回完整的服务使用指南。

```go
s.AddPrompt(mcp.NewPrompt("bbc-guide",
    mcp.WithPromptDescription("BBC MCP 服务使用指南：包含数据库字段说明、设备/状态码映射、Tool 使用说明"),
), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    return mcp.NewGetPromptResult(
        "BBC MCP 服务使用指南",
        []mcp.PromptMessage{
            mcp.NewPromptMessage(mcp.RoleAssistant,
                mcp.NewTextContent(bbcGuideContent),
            ),
        },
    ), nil
})
```

### 3.2 Prompt 内容结构

```
一、服务概述
二、设备数据库字段说明（device 表全部字段 + 编码映射）
三、Redis 实时数据字段说明
四、Tool 使用说明
   4.1 gateway_status
   4.2 device_list
   4.3 device_status
五、注意事项
```

### 3.3 完整 Prompt 内容

---

#### 一、服务概述

```
你正在使用 BBC MCP 服务，该服务提供对 minibbc 数据中心设备信息的安全查询能力。
通过以下三个 MCP Tool，你可以查询设备连接数、租户设备列表和设备实时连接状态。
```

#### 二、设备数据库字段说明

`device` 表字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 设备主键 ID |
| `gwid` | string | 网关设备 ID |
| `ccode` | int | 公司/租户 ID |
| `type` | int | 设备类型，见类型码映射表 |
| `name` | string | 设备名称（用户自定义） |
| `pwd` | string | 设备密码（总是返回 `"***"`） |
| `model` | string | 设备型号 |
| `is_nfv` | bool | 是否 NFV 设备 |
| `prod_line` | int | 产品线编号 |
| `prod_name` | string | 产品名称 |
| `version` | string | 当前软件版本 |
| `is_custom` | bool | 是否定制版本（0=否, 1=是） |
| `active_time` | timestamp | 设备激活时间 |
| `lst_online_time` | timestamp | 最近一次上线时间 |
| `lst_offline_time` | timestamp | 最近一次离线时间 |
| `status` | int | 设备状态，见状态码映射表 |
| `ip` | string | 外网 IP 地址 |
| `mac` | string | 外网 MAC 地址 |
| `eth0_mac` | string | eth0 网卡 MAC 地址 |
| `zp_id` | int | 接入服务器 ID ，200 为奥飞机房，400为深圳四区，800为深圳一区|
| `branch_id` | int | 分支机构 ID |
| `parent_id` | int | 父设备 ID（-1 表示无） |
| `tpl_id` | int | 策略模板 ID |
| `license` | string | 授权序列号 |
| `device_belong` | int | 设备归属：0=旧设备, 1=安服, 2=XDR, 3=云图 |
| `create_source` | int | 创建来源 |
| `update_source` | int | 更新来源 |
| `last_update_time` | timestamp | 最后更新时间 |
| `desc` | string | 设备描述 |
| `sn` | string | 硬件序列号 |
| `region_id` | int | 区域编码 |
| `tags` | string | 设备标签 |
| `platforms` | string | 平台标识（逗号分隔，如 sase,xdr） |
| `addr` | string | 详细地址 |
| `order_id` | string | 订单编号 |
| `org_id` | string | 组织 ID |
| `alias` | string | 设备别名 |
| `manager` | string | 负责人 |
| `email` | string | 负责人邮箱 |
| `remark` | string | 备注 |
| `conn_code` | string | 连接码 |
| `conn_code_gen_time` | datetime | 连接码生成时间 |
| `add_type` | int | 添加方式 |
| `create_time` | timestamp | 创建时间 |
| `vid` | string | V-ID |

**设备类型码映射（`type` 字段）：**

| 编码 | 设备类型 |
|------|----------|
| 1 | ABOS |
| 2 | AC (Access Controller) |
| 3 | AF (Application Firewall) |
| 4 | MIG |
| 5 | WOC |
| 6 | SG |
| 7 | EDR |
| 8 | CM |
| 9 | CG |

**设备状态码映射（`status` 字段）：**

| 编码 | 状态 | 说明 |
|------|------|------|
| 0 | 未激活 | 设备尚未激活使用 |
| 1 | 已启用 | 聚合状态，包含离线/在线/告警子状态 |
| 2 | 离线 | 设备当前不在线 |
| 3 | 在线 | 设备正常运行中 |
| 4 | 告警 | 设备在线但存在告警 |
| 5 | 停用 | 设备已被停用 |

**设备归属映射（`device_belong` 字段）：**

| 编码 | 归属 |
|------|------|
| 0 | 旧设备 |
| 1 | 安服 (AnFu) |
| 2 | XDR |
| 3 | 云图 (YunTu) |

#### 三、Redis 实时数据字段说明

`device_status` Tool 返回的 `realtime` 字段来自 Redis，字段含义如下：

| Redis 字段 | 说明                                   |
|------------|--------------------------------------|
| `product_name` | 产品名称（如 AF, AC 等）                     |
| `tenant` | 租户 ID（对应 ccode）                      |
| `zp_id` | 接入服务器 ID，200 为奥飞机房，400为深圳四区，800为深圳一区 |
| `cpu` | CPU 使用率（百分比）                         |
| `mem` | 内存使用率（百分比）                           |
| `disk` | 磁盘使用率（百分比）                           |
| `version` | 当前软件版本                               |
| `session_num` | 会话数                                  |
| `user` | 在线用户数                                |
| `bandwidth` | 带宽占用                                 |
| `last_online_time` | 最近上线时间                               |
| `last_offline_time` | 最近离线时间                               |
| `status` | 实时状态（编码同上表）                          |
| `bbc:heartbeat` | BBC 心跳标识                             |
| `device_id` | 设备 ID                                |
| `gateway_id` | 网关 ID                                |
| `send` | 发送流量                                 |
| `recv` | 接收流量                                 |
| `vm_num` | 虚拟机数量                                |
| `minibbc` | minibbc 标识                           |

#### 四、Tool 使用说明

##### 4.1 `gateway_status` — 查询设备连接数

```
功能：查询 minibbc 数据中心网关各 Pod 的 ESTABLISHED 连接数和设备数。

参数：
  - cluster (string, 可选): K8s 集群名称，不填则查询所有已配置集群

返回值：
  原生模式(native)：JSON 数组，每项包含：
    - cluster: 集群名称
    - pods: Pod 连接信息列表 [{pod_name, status, connections, devices, error}]
    - connections: TCP ESTABLISHED 连接数
    - devices: connections / 3（每设备 3 个连接）
  CLI 模式：bbc-tool status gateway 命令原始输出

注意事项：
  - 若 Pod 状态非 Running，其 error 字段会有说明
  - 若指定的 cluster 名称不存在，返回错误提示
```

##### 4.2 `device_list` — 查询租户设备列表

```
功能：根据租户 ID 查询设备列表，支持按名称模糊搜索和分页。

参数：
  - ccode (string, 必填): 租户 ID（公司代码），不能为空
  - name (string, 可选): 设备名称，支持模糊匹配
  - offset (integer, 必填): 分页偏移量，负数自动置为 0

返回值：
  JSON 对象：
  {
    "total": true,
    "devices": [
      {"id": 260860, "name": "sz1_af_Connect_7000", "zp_id": null, "status": 0}
    ],
    "limit": 20,    // 每页固定 20 条
    "offset": 0     // 当前偏移量
  }

注意事项：
  - 每页固定返回最多 20 条记录，LIMIT 不可被调用方覆盖
  - name 参数为空时返回该租户所有设备（分页）
  - name 不为空时执行 LIKE %name% 模糊查询
  - 若 ccode 为空字符串，返回参数错误
  - status 字段含义见设备状态码映射表
  - type 字段含义见设备类型码映射表（device_list 不返回 type，需使用 device_status 查询）
```

##### 4.3 `device_status` — 查询设备详情与实时状态

```
功能：根据设备 ID 查询设备完整信息（MySQL）和实时连接状态（Redis）。

参数：
  - id (integer, 必填): 设备 ID

返回值：
  JSON 对象：
  {
    "device": {
      "id": 223166,
      "gwid": "",
      "ccode": 56626662,
      "type": 3,          // 详见设备类型码映射表
      "name": "sz1_af_Connect_7000",
      "pwd": "***",       // 密码已脱敏
      "model": null,
      "status": 3,        // 详见设备状态码映射表
      ...                  // 共 35 个字段
    },
    "realtime": {
      "product_name": "AF",
      "tenant": "56626662",
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

注意事项：
  - "pwd" 字段永远返回 "***"，不会泄露真实密码
  - "realtime" 字段可能为 null（Redis 不可用时），不视为错误
  - "realtime" 中的 status 字段为实时值，优先级高于 MySQL 中的持久化状态
  - 设备不存在时返回错误："设备不存在"
```

#### 五、注意事项

```
1. 所有查询都有超时限制（数据库 5s，Redis 3s，K8s exec 30s）
2. 服务有全局 QPS=1 的限流，并发请求会收到"服务繁忙"错误
3. 请求需要 Bearer Token 认证，Header 格式：Authorization: Bearer <token>
4. device_list 和 device_status 返回的数据应该结合起来理解：
   - 先用 device_list 获取设备列表（含 id, name, status）
   - 再用 device_status 获取单个设备的详细信息和实时状态
5. status 字段在 MySQL 和 Redis 中独立存在：
   - MySQL status 为持久化状态，更新有延迟
   - Redis status 为实时采集状态，优先使用
```

---

## 4. 代码设计

### 4.1 新增文件 `internal/prompt/guide.go`

Prompt 内容封装为常量，通过函数返回注册所需的 Prompt 定义和 Handler。

```go
package prompt

import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
)

const bbcGuideContent = `...` // 完整的 Prompt 文本

// NewBBCGuidePrompt returns the prompt definition and handler for the bbc-guide prompt.
func NewBBCGuidePrompt() (mcp.Prompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error)) {
    prompt := mcp.NewPrompt("bbc-guide",
        mcp.WithPromptDescription("BBC MCP 服务使用指南：包含数据库字段说明、设备/状态码映射、Tool 使用说明"),
    )
    handler := func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
        return mcp.NewGetPromptResult(
            "BBC MCP 服务使用指南",
            []mcp.PromptMessage{
                mcp.NewPromptMessage(mcp.RoleAssistant, mcp.NewTextContent(bbcGuideContent)),
            },
        ), nil
    }
    return prompt, handler
}
```

### 4.2 修改 `cmd/bbc-mcp/main.go`

1. 启用 Prompt 能力：在 `NewMCPServer` 选项中添加 `server.WithPromptCapabilities(true)`
2. 注册 Prompt：在 `tool.RegisterAll` 之后调用 `prompt.NewBBCGuidePrompt()` 并 `s.AddPrompt`

```go
s := server.NewMCPServer(
    "bbc-mcp",
    "1.0.0",
    server.WithToolCapabilities(true),
    server.WithPromptCapabilities(true),  // 新增
    server.WithHooks(hooks),
)
tool.RegisterAll(s, deps)

// 注册系统 Prompt
guidePrompt, guideHandler := prompt.NewBBCGuidePrompt()
s.AddPrompt(guidePrompt, guideHandler)
```

---

## 5. 文件清单

### 5.1 新增文件

| 文件 | 职责 |
|------|------|
| `internal/prompt/guide.go` | 系统 Prompt 内容定义与注册函数 |

### 5.2 修改文件

| 文件 | 改动内容 |
|------|----------|
| `cmd/bbc-mcp/main.go` | 启用 `WithPromptCapabilities(true)` + 注册 `bbc-guide` Prompt |

---

## 6. 测试策略

### 6.1 Prompt 内容验证

1. 启动服务后通过 MCP 客户端调用 `ListPrompts`，验证 `bbc-guide` 出现在列表中
2. 调用 `GetPrompt({Name: "bbc-guide"})`，验证返回内容完整且格式正确
3. 逐一核对：设备类型码 9 种、状态码 6 种、设备归属 4 种、Redis 字段 18 个

### 6.2 回归验证

- 确认启用 Prompt 后，三个 Tool 功能不受影响
- 确认客户端未调用 Prompt 时不影响正常业务

---

## 7. 实施步骤

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | 新建 `internal/prompt/guide.go`，定义完整的 Prompt 内容 | guide.go |
| 2 | 修改 `main.go`：启用 Prompt 能力 + 注册 | main.go |
| 3 | 运行 `go vet` + `go test` 确保编译和测试通过 | - |
| 4 | 启动服务，通过 MCP 客户端验证 ListPrompts 和 GetPrompt | - |
