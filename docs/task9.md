# BBC MCP 工具说明优化设计文档

> 版本：1.0
> 日期：2026-06-10
> 基于：docs/task9.md 需求规格

---

## 1. 概述

### 1.1 背景

当前 MCP Tool 存在两个问题：

1. **工具名冲突**：`gateway_status` 与 openclaw 的 gateway status 严重重合，AI 助手无法区分两者。业务本质是查询 "数据中心状态"，而非 "网关状态"。

2. **Tool 描述信息不足**：当前四个 Tool 的描述仅有一句话，openclaw/AI 助手读取后无法完全理解工具用途、参数含义、返回值结构及编码映射。虽然 bbc-guide Prompt 中有详细说明，但 Tool 描述是 AI 助手做工具选择判断时最先读取的信息，应自包含关键业务上下文。

### 1.2 目标

- `gateway_status` 重命名为 `datacenter_status`，消除与 openclaw 的命名冲突
- 四个 Tool 的描述均扩充为自包含的完整说明：功能、参数、返回值结构、编码映射、注意事项
- 保持 Tool 接口签名不变（仅改名），向后兼容

### 1.3 非目标

- 不改变 Tool 的业务逻辑和 Handler 实现
- 不修改 bbc-guide Prompt 内容（Prompt 仍保留完整参考文档）
- 不改变 MCP 协议交互方式

---

## 2. 总体方案

```
┌──────────────────────────────────────────────────────────────────┐
│  变更前                                                            │
│  Tool: gateway_status  → 描述: "查询当前 minibbc 数据中心设备连接数" │
│  Tool: device_list     → 描述: "查询某个租户的设备列表..."         │
│  Tool: device_status   → 描述: "查询某个设备的详细信息..."         │
│  Tool: sms_package_list → 描述: "查询某个租户的短信云套餐信息"      │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  变更后                                                            │
│  Tool: datacenter_status  → 描述: 功能+返回值结构+编码说明+注意事项 │
│  Tool: device_list        → 描述: 功能+参数+返回结构+分页+注意事项  │
│  Tool: device_status      → 描述: 功能+返回结构+字段编码+注意事项   │
│  Tool: sms_package_list   → 描述: 功能+表字段+返回结构+注意事项     │
└──────────────────────────────────────────────────────────────────┘
```

**设计原则**：Tool 描述是 AI 助手的第一信息来源，需做到看完描述就能正确调用和解读返回值。从 bbc-guide Prompt 中提取每个 Tool 的关键信息精简后嵌入描述。

---

## 3. 详细设计

### 3.1 Tool 重命名：`gateway_status` → `datacenter_status`

**原因**：`gateway_status` 与 openclaw 的 gateway status 工具名重合，AI 助手无法区分。"数据中心状态"（datacenter status）更准确地反映了该工具的查询内容——minibbc 数据中心各集群 Pod 的连接数和设备数。

**变更范围**：

| 层级 | 变更 |
|------|------|
| MCP Tool Name | `"gateway_status"` → `"datacenter_status"` |
| Go 函数名 | `newGatewayStatus` → `newDatacenterStatus` |
| Go 函数名 | `gatewayStatusNative` → `datacenterStatusNative` |
| Go 函数名 | `gatewayStatusCLI` → `datacenterStatusCLI` |
| 日志前缀 | `"gateway_status:"` → `"datacenter_status:"` |
| 注册调用 | `newGatewayStatus(deps)` → `newDatacenterStatus(deps)` |

**影响范围分析**：

| 文件 | 是否受改名影响 |
|------|:---:|
| `internal/tool/datacenter_status.go` | ✓ 全部函数、Tool Name、日志（文件更名为 datacenter_status.go） |
| `internal/tool/registry.go` | ✓ RegisterAll 注册调用 |
| `internal/prompt/guide.go` | ✓ bbcGuideContent 中引用 `gateway_status` 的段落 |
| `test/client/main.go` | ✓ 测试调用 `testGatewayStatus` → `testDatacenterStatus` |
| `internal/tool/device_list.go` | ✗ 无关 |
| `internal/tool/device_status.go` | ✗ 无关 |
| `internal/tool/sms_package_list.go` | ✗ 无关 |
| `internal/config/config.go` | ✗ 无关（Gateway 配置段保持不变） |

### 3.2 Tool 描述增强

每个 Tool 的 `mcp.WithDescription(...)` 扩充为自包含说明。描述结构统一：

```
[一句话功能概述]

参数：
  - <参数名> (<类型>, <必填/可选>): <说明> [编码映射/约束]

返回值：
  <结构说明，含关键字段的编码映射>

注意事项：
  - <关键行为/约束>
```

#### 3.2.1 `datacenter_status` 描述

```
查询 minibbc 数据中心各 K8s 集群的设备连接状态。返回每个集群的 Pod 连接数（TCP ESTABLISHED）和设备数（connections/3）。

参数：
  - cluster (string, 可选): K8s 集群名称，不填则查询所有已配置集群

返回值（JSON 数组）：
  [{"cluster":"bbc-shenzhen4", "pods":[{"pod_name":"cloudbbc-gateway-0","status":"Running","connections":60,"devices":20}]}]
  - connections: TCP ESTABLISHED 连接数
  - devices: connections / 3（每设备维持 3 个连接）
  - 若 Pod 非 Running 状态，error 字段会有说明；若指定 cluster 不存在，返回错误

注意事项：
  - 查询超时 30s
  - 单次请求串行查询所有集群
```

#### 3.2.2 `device_list` 描述

```
按租户 ID 查询设备列表，支持设备名称模糊搜索和分页。

参数：
  - ccode (string, 必填): 租户 ID（公司代码），不能为空
  - name (string, 可选): 设备名称，支持模糊匹配（LIKE %name%）
  - offset (integer, 必填): 分页偏移量，>=0，负数自动置 0

返回值：
  {"total":true, "limit":20, "offset":0, "devices":[{"id":260860,"name":"sz1_af","zp_id":400,"status":3}]}
  - limit: 固定每页 20 条，不可覆盖
  - zp_id: 接入服务器 ID（200=奥飞, 400=深圳四区, 800=深圳一区）
  - status: 设备状态码（0=未激活, 1=已启用, 2=离线, 3=在线, 4=告警, 5=停用）
  - type 字段不返回，查设备类型请用 device_status

注意事项：
  - 超时 5s
  - 租户无设备时返回空数组 []
```

#### 3.2.3 `device_status` 描述

```
按设备 ID 查询设备完整信息（MySQL）和实时运行状态（Redis）。

参数：
  - id (integer, 必填): 设备 ID

返回值：
  {"device": {...35个字段}, "realtime": {"cpu":"98","mem":"41",...}}
  - device.type: 设备类型（1=ABOS,2=AC,3=AF,4=MIG,5=WOC,6=SG,7=EDR,8=CM,9=CG）
  - device.status: 持久化状态码（0=未激活,1=已启用,2=离线,3=在线,4=告警,5=停用）
  - device.device_belong: 设备归属（0=旧设备,1=安服,2=XDR,3=云图）
  - device.zp_id: 接入服务器 ID（200=奥飞,400=深圳四区,800=深圳一区）
  - device.pwd: 永远返回 "***"，密码已脱敏
  - realtime: 实时采集数据（CPU/内存/磁盘/会话数/在线用户等 18 个字段），可能为 null（Redis 不可用时）
  - realtime.status: 实时状态码，优先于 device.status 使用

注意事项：
  - MySQL 超时 5s，Redis 超时 3s
  - 设备不存在返回错误"设备不存在"
  - realtime 为 null 时不视为错误
```

#### 3.2.4 `sms_package_list` 描述

```
按租户 ID 查询短信云套餐信息。

参数：
  - corp_id (integer, 必填): 租户 ID

返回值（JSON 数组）：
  [{"id":1292,"corp_id":26912728,"order_id":"201905178888","pkg_type":1,"pkg_name":"50条短信套餐",
    "pkg_total_num":50,"pkg_avail_num":0,"pkg_price":2.4,
    "package_desc":"套餐允许发送50条短信","purchase_time":"...","expire_time":"..."}]
  - pkg_total_num: 套餐总短信条数
  - pkg_avail_num: 剩余可用条数
  - expire_time: 套餐过期时间

注意事项：
  - 超时 5s
  - 短信云数据库未配置时返回错误"短信云数据库未配置"
  - 租户无套餐时返回空数组 []
```

### 3.3 Prompt 同步更新

`bbc-guide` Prompt 中 `4.1` 节的标题和引用需同步更新：

```
- ### 4.1 gateway_status — 查询设备连接数
+ ### 4.1 datacenter_status — 查询数据中心设备连接数
```

其余 Prompt 内容不变（仍保留完整参考文档）。

---

## 4. 代码变更

### 4.1 `internal/tool/datacenter_status.go` — 重命名 + 描述增强

```go
// 函数名重命名
func newDatacenterStatus(deps *Dependencies) ToolDefinition {
    return ToolDefinition{
        Tool: mcp.NewTool("datacenter_status",
            mcp.WithDescription("查询 minibbc 数据中心各 K8s 集群的设备连接状态..."),
            mcp.WithString("cluster",
                mcp.Description("K8s 集群名称（可选，不填则查询所有集群）"),
            ),
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            cluster := getStringArg(req, "cluster")
            if deps.Config.Gateway.Mode == "native" {
                return datacenterStatusNative(ctx, deps, cluster)
            }
            return datacenterStatusCLI(ctx, deps)
        },
    }
}

func datacenterStatusNative(ctx context.Context, deps *Dependencies, cluster string) (*mcp.CallToolResult, error) { ... }
func datacenterStatusCLI(ctx context.Context, deps *Dependencies) (*mcp.CallToolResult, error) { ... }
```

### 4.2 其余 Tool 文件 — 仅改描述

`device_list.go`、`device_status.go`、`sms_package_list.go` —— 仅修改 `mcp.WithDescription(...)` 参数。

---

## 5. 文件清单

### 5.1 修改文件

| 文件 | 改动内容 |
|------|----------|
| `internal/tool/datacenter_status.go` | 文件重命名 + 函数/日志重命名 + Tool Name 改为 `datacenter_status` + 描述增强 |
| `internal/tool/device_list.go` | `WithDescription` 扩充 |
| `internal/tool/device_status.go` | `WithDescription` 扩充 |
| `internal/tool/sms_package_list.go` | `WithDescription` 扩充 |
| `internal/tool/registry.go` | `newGatewayStatus` → `newDatacenterStatus` |
| `internal/prompt/guide.go` | `gateway_status` → `datacenter_status` |
| `test/client/main.go` | `testGatewayStatus` → `testDatacenterStatus`，Tool Name 更新 |

### 5.2 无需改动

| 文件 | 原因 |
|------|------|
| `cmd/bbc-mcp/main.go` | Tool 注册通过 `RegisterAll` 间接调用，无直接引用 |
| `etc/*.yaml` | 配置不变 |
| `internal/config/config.go` | `Gateway` 配置段语义不变 |
| `internal/gateway/native.go` | 仅被网关 Tool 调用，接口不变 |
| `internal/k8s/` | 无关 |

---

## 6. 向后兼容

- **MCP Client 侧**：Tool Name 从 `gateway_status` 变为 `datacenter_status`，Client 需更新调用名称。这是一个 breaking change，但由于服务仍在开发阶段且与 openclaw 的命名冲突更严重，此时改名代价最低。
- **API 行为**：除名称外，参数、返回值格式、错误处理均不变。
- **配置文件**：`Gateway.Mode` 等配置字段保持原样，不随 Tool 名变化。

---

## 7. 测试策略

| 用例 | 说明 |
|------|------|
| ListTools | 验证 `datacenter_status` 出现在 Tool 列表中，`gateway_status` 不再出现 |
| CallTool `datacenter_status` | 功能与改名前的 `gateway_status` 完全一致 |
| CallTool `gateway_status` | 预期失败（Tool not found） |
| GetPrompt `bbc-guide` | 验证 Prompt 中引用了 `datacenter_status` |
| 其余 3 个 Tool | 回归验证功能不受影响，描述信息可读 |

---

## 8. 实施步骤

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | 文件重命名为 `datacenter_status.go`，内部函数/日志/Tool Name 全部重命名，增强描述 | datacenter_status.go |
| 2 | 扩充 `device_list.go` 描述 | device_list.go |
| 3 | 扩充 `device_status.go` 描述 | device_status.go |
| 4 | 扩充 `sms_package_list.go` 描述 | sms_package_list.go |
| 5 | `RegisterAll` 中的调用更新 | registry.go |
| 6 | Prompt 中引用更新 | guide.go |
| 7 | 测试客户端更新 Tool Name 和函数名 | main.go |
| 8 | `go vet ./...` + `go build ./...` | - |
| 9 | 启动服务，MCP Client 验证 ListTools / CallTool / GetPrompt | - |
