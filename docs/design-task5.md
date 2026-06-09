# BBC MCP 工具整改 — Gateway Status 原生集成与多集群支持设计文档

> 版本：1.0
> 日期：2026-06-09
> 基于：docs/task5.md 需求规格

---

## 1. 概述

### 1.1 背景

当前 `gateway_status` Tool 通过 `os/exec` 调用外部 `bbc-tool` 二进制文件获取网关设备连接数。该方式存在以下痛点：

- **部署依赖**：每个 MCP 服务实例必须附带 `bbc-tool` 可执行文件
- **集群扩展困难**：业务存在多个 K8s 集群，当前仅支持单一集群，无法按需查询不同集群
- **维护成本**：两个独立项目的版本需保持同步

### 1.2 目标

- 将 `bbc-tool status gateway` 核心逻辑移植到 MCP 服务内部，消除外部二进制依赖
- 保留原有 CLI 调用方式作为可切换选项
- 支持多 K8s 集群配置，Tool 可按集群名称查询

### 1.3 非目标

- 不移植 bbc-tool 的其他子命令（`db info/switch`、`status heartbeat`、`backup k8s`）
- 不改变 `gateway_status` 的 MCP Tool 接口签名（向后兼容，新增可选参数）

---

## 2. 总体方案

```
┌──────────────────────────────────────────────────────────────────┐
│  etc/bbc-mcp.yaml                                                 │
│  Gateway:                                                         │
│    Mode: native            # "native" 或 "cli"                     │
│    Native:                                                        │
│      Namespace: xcentral                                          │
│      StatefulSet: cloudbbc-gateway                                │
│      Container: cloudbbc-goproxy-container                        │
│      Port: 5000                                                   │
│  K8sClusters:                                                     │
│    default:                                                       │
│      Server: https://192.168.1.100:6443                           │
│      Token: "eyJhbGci..."                                         │
│      CAData: "LS0tLS1CRUdJTi..."                                  │
│    sz-4:                                                          │
│      Server: https://192.168.2.100:6443                           │
│      Token: "eyJhbGci..."                                         │
└──────────────────────────┬───────────────────────────────────────┘
                           │
          ┌────────────────┴────────────────┐
          ▼                                  ▼
┌─────────────────────────┐    ┌──────────────────────────────┐
│ Mode: native            │    │ Mode: cli                     │
│                         │    │                              │
│ internal/k8s/client.go  │    │ os/exec bbc-tool             │
│   ↓                     │    │   status gateway             │
│ GetStatefulSet replicas │    │                              │
│ ListPods by selector    │    │ (现有逻辑，保持不变)             │
│ ExecInPod netstat       │    │                              │
│   ↓                     │    │                              │
│ 计算 connections/3      │    │                              │
└──────────┬──────────────┘    └──────────────┬───────────────┘
           │                                  │
           └──────────────┬───────────────────┘
                          ▼
              ┌───────────────────────┐
              │ gateway_status Tool   │
              │ 返回设备连接数表格     │
              └───────────────────────┘
```

**关键设计决策**：通过 `Gateway.Mode` 配置项控制使用原生 K8s 调用还是 CLI 调用，无需重新编译。

---

## 3. 配置设计

### 3.1 新增配置结构

```go
type GatewayConfig struct {
    Mode   string               `yaml:"Mode"`   // "native" 或 "cli"
    Native GatewayNativeConfig  `yaml:"Native"`
}

type GatewayNativeConfig struct {
    Namespace   string `yaml:"Namespace"`
    StatefulSet string `yaml:"StatefulSet"`
    Container   string `yaml:"Container"`
    Port        int    `yaml:"Port"`
}

type K8sClusterConfig struct {
    Server string `yaml:"Server"` // K8s API server URL
    Token  string `yaml:"Token"`  // Bearer token
    CAData string `yaml:"CAData"` // CA certificate (base64, optional)
}
```

### 3.2 完整配置示例 `etc/bbc-mcp.yaml`

```yaml
Server:
  Name: bbc-mcp
  Host: 0.0.0.0
  Port: 9000

Database:
  Host: 192.168.31.200
  Port: 3306
  Name: scloudb
  User: root
  Password: "dJm5tI8fqUsMJ1332LQep1YMyWVTtCZ/o632cBxpEPTyu9L20+8="
  MaxOpenConns: 10
  MaxIdleConns: 5

Redis:
  Host: 192.168.31.200
  Port: 6379
  Password: ""
  DB: 0

Gateway:
  Mode: native
  Native:
    Namespace: xcentral
    StatefulSet: cloudbbc-gateway
    Container: cloudbbc-goproxy-container
    Port: 5000

K8sClusters:
  default:
    Server: https://192.168.1.100:6443
    Token: "eyJhbGciOiJSUzI1NiIs..."
    CAData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t..."

Auth:
  Tokens:
    - "mf8KPh6f5ma8RhOkIeYYVCK15jtAg4CLUTTlkGHc"
```

### 3.3 配置说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Gateway.Mode` | string | `"cli"` | `"native"` = 进程内 K8s 调用，`"cli"` = 调用 bbc-tool 二进制 |
| `Gateway.Native.Namespace` | string | `"xcentral"` | K8s 命名空间 |
| `Gateway.Native.StatefulSet` | string | `"cloudbbc-gateway"` | StatefulSet 名称 |
| `Gateway.Native.Container` | string | `"cloudbbc-goproxy-container"` | 容器名称 |
| `Gateway.Native.Port` | int | `5000` | TCP 监听端口 |
| `K8sClusters.<name>.Server` | string | - | K8s API Server URL，如 `https://192.168.1.100:6443` |
| `K8sClusters.<name>.Token` | string | - | Bearer token 用于认证 |
| `K8sClusters.<name>.CAData` | string | `""` | CA 证书（base64 编码），为空则跳过 TLS 验证 |

**默认值说明**：`Gateway.Mode` 默认为 `"cli"`，确保未配置 `Gateway` 时保持现有 CLI 行为，完全向后兼容。

---

## 4. MCP Tool 变更

### 4.1 gateway_status 接口变更

**新增可选参数 `cluster`**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `cluster` | string | 否 | K8s 集群名称，不填则查询所有集群 |

**Tool Schema**：

```go
mcp.NewTool("gateway_status",
    mcp.WithDescription("查询当前 minibbc 数据中心设备连接数"),
    mcp.WithString("cluster",
        mcp.Description("K8s 集群名称（可选，不填则查询所有集群）"),
    ),
)
```

**行为**：
- `cluster` 为空：遍历 `K8sClusters` 中所有集群，分别查询并聚合结果
- `cluster` 指定：仅查询指定集群
- `Gateway.Mode = "cli"` 时：忽略 `K8sClusters`，直接调用 bbc-tool（与当前行为一致）

---

## 5. 详细设计

### 5.1 新增 `internal/k8s/client.go` — K8s 客户端

从 bbc-tool 移植最小化子集，仅封装网关状态查询所需能力。

```go
package k8s

import (
    "context"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

// Client wraps the K8s clientset for gateway status operations.
type Client struct {
    clientset kubernetes.Interface
    config    *rest.Config
    Name      string
}

// NewClient creates a K8s client from API server URL, bearer token, and optional CA data.
func NewClient(name, server, token, caData string) (*Client, error)

// GetStatefulSetReplicas returns the replica count of a StatefulSet.
func (c *Client) GetStatefulSetReplicas(ctx context.Context, namespace, name string) (int, error)

// GetPodStatus returns the phase of a pod (Running, Pending, etc.).
func (c *Client) GetPodStatus(ctx context.Context, namespace, podName string) (string, error)

// ExecCommand executes a command inside a container and returns stdout.
func (c *Client) ExecCommand(ctx context.Context, namespace, podName, container string, cmd []string) (string, error)
```

K8s 连接通过 `rest.Config` 直接构建，不使用 kubeconfig 文件：
- `Host` 设置为 `K8sClusters.<name>.Server`
- `BearerToken` 设置为 `K8sClusters.<name>.Token`
- `TLSClientConfig.CAData` 从 `K8sClusters.<name>.CAData` 解码（base64）
- 若未配置 CAData，跳过 TLS 证书验证

**接口设计考量**：`kubernetes.Interface` 为接口类型，支持单元测试时注入 `fake.NewSimpleClientset()`。

### 5.2 新增 `internal/gateway/native.go` — 原生网关状态查询

移植 bbc-tool `status gateway` 的核心逻辑。

```go
package gateway

// PodConnectionInfo 单个 Pod 的连接信息
type PodConnectionInfo struct {
    PodName     string `json:"pod_name"`
    Status      string `json:"status"`
    Connections int    `json:"connections"`
    Devices     int    `json:"devices"`     // connections / 3
    Error       string `json:"error,omitempty"`
}

// ClusterGatewayStatus 单个集群的网关状态
type ClusterGatewayStatus struct {
    Cluster string              `json:"cluster"`
    Pods    []PodConnectionInfo `json:"pods"`
}

// QueryGatewayStatus queries gateway connection counts on a single K8s cluster.
func QueryGatewayStatus(ctx context.Context, client k8s.Client, cfg GatewayNativeConfig) (*ClusterGatewayStatus, error)
```

**查询流程**（与 bbc-tool 一致）：

```
1. GetStatefulSet(namespace, statefulSetName) → 获取 spec.replicas
2. 对每个 replica (pod-0, pod-1, ...)，并发执行：
   a. 通过 field selector "metadata.name=<podName>" ListPods
   b. 检查 Pod phase 是否为 Running
   c. 若 Running：ExecInPod 执行命令
      netstat -antp 2>/dev/null | grep EST | grep ':<port>' | wc -l
   d. 解析 stdout 为整数连接数
3. devices = connections / 3（整除）
4. 聚合汇总：total connections, total devices
```

**关键差异**（相比 bbc-tool）：
- 不包含进度条 / spinner 等终端 UI 组件（MCP 是 API 服务，非交互式 CLI）
- 不包含 watch 模式（MCP Tool 是单次请求-响应模型，watch 由上层 AI Agent 轮询实现）
- 去除了 bbc-tool 中的 `pterm`、`tablewriter`、`fatih/color` 等 UI 依赖

### 5.3 修改 `internal/tool/gateway_status.go` — 模式分发

```go
func newGatewayStatus(deps *Dependencies) ToolDefinition {
    return ToolDefinition{
        Tool: mcp.NewTool("gateway_status",
            mcp.WithDescription("查询当前 minibbc 数据中心设备连接数"),
            mcp.WithString("cluster",
                mcp.Description("K8s 集群名称（可选，不填则查询所有集群）"),
            ),
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            cluster, _ := req.Params.Arguments["cluster"].(string)

            switch deps.Config.Gateway.Mode {
            case "native":
                return gatewayStatusNative(ctx, deps, cluster)
            default:
                return gatewayStatusCLI(ctx, deps)
            }
        },
    }
}
```

- `gatewayStatusNative`：遍历 `K8sClients`，对每个集群调用 `gateway.QueryGatewayStatus`
- `gatewayStatusCLI`：复用现有 `os/exec` 逻辑

### 5.4 修改 `internal/tool/registry.go` — 扩展依赖

```go
type Dependencies struct {
    DB         *sql.DB
    Redis      *redis.Client
    Config     *config.Config
    K8sClients map[string]*k8s.Client  // 新增
}
```

### 5.5 修改 `cmd/bbc-mcp/main.go` — K8s 客户端初始化

在 `config.Load` 之后、`tool.RegisterAll` 之前，初始化 K8s 客户端：

```go
k8sClients := make(map[string]*k8s.Client)
if cfg.Gateway.Mode == "native" {
    for name, clusterCfg := range cfg.K8sClusters {
        client, err := k8s.NewClient(name, clusterCfg.Server, clusterCfg.Token, clusterCfg.CAData)
        if err != nil {
            log.Fatalf("初始化 K8s 客户端 %s 失败: %v", name, err)
        }
        k8sClients[name] = client
    }
}
```

---

## 6. 新增文件与改动

### 6.1 新增文件

| 文件 | 职责 |
|------|------|
| `internal/k8s/client.go` | K8s 客户端封装（GetStatefulSetReplicas, ListPods, ExecCommand） |
| `internal/k8s/client_test.go` | K8s 客户端单元测试（使用 fake clientset） |
| `internal/gateway/native.go` | 原生网关状态查询（netstat 连接计数逻辑） |

### 6.2 修改文件

| 文件 | 改动内容 |
|------|----------|
| `internal/config/config.go` | 新增 `GatewayConfig`、`GatewayNativeConfig`、`K8sClusterConfig` 结构体 |
| `internal/tool/gateway_status.go` | 根据 `Gateway.Mode` 分发 native/cli 实现 |
| `internal/tool/registry.go` | `Dependencies` 新增 `K8sClients` |
| `cmd/bbc-mcp/main.go` | 启动时初始化 K8s 客户端 |
| `etc/bbc-mcp.yaml` | 新增 `Gateway` 和 `K8sClusters` 配置段 |
| `etc/bbc-mcp-docker.yaml` | 同上 |
| `go.mod` | 新增 `k8s.io/client-go`、`k8s.io/api`、`k8s.io/apimachinery` 依赖 |

### 6.3 不修改的文件

| 文件 | 原因 |
|------|------|
| `internal/tool/device_list.go` | 不受此改动影响 |
| `internal/tool/device_status.go` | 同上 |
| `internal/infrastructure/` | 同上 |
| `internal/repository/` | 同上 |
| `docker-compose.yaml` | 无需额外挂载（连接信息已在配置文件中） |

---

## 7. 新增依赖

| 依赖 | 用途 | 版本（参考 bbc-tool） |
|------|------|----------------------|
| `k8s.io/client-go` | K8s clientset, rest config, exec | v0.34.1 |
| `k8s.io/api` | K8s API 类型（core/v1, apps/v1） | v0.34.1 |
| `k8s.io/apimachinery` | K8s API 元类型（metav1） | v0.34.1 |

---

## 8. 测试策略

### 8.1 单元测试 `internal/k8s/client_test.go`

使用 `k8s.io/client-go/kubernetes/fake` 模拟：

| 用例 | 说明 |
|------|------|
| `TestGetStatefulSetReplicas` | 正常获取副本数 |
| `TestGetStatefulSetNotFound` | StatefulSet 不存在 |
| `TestGetPodStatus` | 获取 Pod 运行状态 |
| `TestExecCommandSuccess` | 正常执行命令并返回 stdout |
| `TestNewClientInvalidServer` | Server URL 无效 |

### 8.2 单元测试 `internal/gateway/native_test.go`

| 用例 | 说明 |
|------|------|
| `TestQueryGatewayStatusSinglePod` | 单 Pod 查询成功 |
| `TestQueryGatewayStatusMultiPod` | 多 Pod 并发查询 + 聚合 |
| `TestQueryGatewayStatusPodNotRunning` | Pod 非 Running 状态的处理 |
| `TestQueryGatewayStatusExecError` | Exec 失败的处理 |

### 8.3 集成验证

1. 准备一个可访问的 K8s 测试集群
2. 配置 `Gateway.Mode = native` + 对应的 `K8sClusters`
3. 启动服务，调用 `gateway_status` Tool
4. 对比 `native` 模式输出与 `cli` 模式（`bbc-tool status gateway`）输出一致性
5. 验证多集群聚合结果正确

---

## 9. 实施步骤

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | 新增 `GatewayConfig`、`GatewayNativeConfig`、`K8sClusterConfig` | config.go |
| 2 | 新增 K8s 客户端封装 | k8s/client.go |
| 3 | 新增 K8s 客户端单元测试 | k8s/client_test.go |
| 4 | 新增网关原生查询逻辑 | gateway/native.go |
| 5 | 新增网关查询单元测试 | gateway/native_test.go |
| 6 | 修改 `gateway_status.go` 支持模式分发 | gateway_status.go |
| 7 | 修改 `Dependencies` 新增 `K8sClients` 字段 | registry.go |
| 8 | 修改 `main.go` 启动时初始化 K8s 客户端 | main.go |
| 9 | 更新配置文件和 docker-compose.yaml | etc/*.yaml, docker-compose.yaml |
| 10 | 运行测试 + 集成验证 | - |

---

## 10. 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| K8s client-go 依赖体积大 | 编译时间增加，二进制体积增大 | 仅导入必要的子包（`kubernetes`、`rest`、`remotecommand`），避免全量导入 |
| K8s Token 过期 | native 模式认证失败 | 使用长期有效的 ServiceAccount Token，或定期轮换 |
| Pod Exec 超时 | 请求挂起 | 传入 ctx 控制超时，默认 30s（与现有 BbcTool.Timeout 对齐） |
| CAData 未配置 | TLS 连接不安全 | 仅测试环境允许跳过 TLS 验证，生产环境必须配置 CAData |

---

## 11. 向后兼容

- `Gateway.Mode` 默认值为 `"cli"`，未配置 `Gateway` 时保持现有 CLI 行为
- `cluster` 参数为可选，不传则查询所有集群，传统单集群用户无需修改调用方式
- `BbcTool` 配置段保留，CLI 模式下照常使用
- `gateway_status` Tool 名称和返回格式不变
