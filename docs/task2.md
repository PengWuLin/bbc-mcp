# mcp server 迭代二设计

## 概述
本迭代为 mcp server 添加安全性和稳定性功能，包括限流和认证机制。

## 需求

### 限流
该 mcp 服务的 qps 限制为 1
- 所有 http handler 需要加锁，同一时间只能响应一个指令
- 如果当前指令还没有执行完成，新的请求立刻报错
- 错误消息需要清晰，告知客户端服务繁忙

### 认证
- 添加 OAuth 2.0 / 令牌认证
- 支持 `Authorization: "Bearer <token>"` 头
- token 从配置文件中读取

## 设计

### 限流设计

#### 方案选择
采用**单进程单令牌**方式，使用一个互斥锁（Mutex）确保同一时间只有一个请求被处理。

#### 实现位置
在 SSE 服务器层面添加限流中间件，而不是在每个 handler 内部添加锁。

#### 限流器组件

```go
package ratelimit

import (
    "sync"
)

type RateLimiter struct {
    mu   sync.Mutex
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{}
}

func (r *RateLimiter) TryAcquire() bool {
    acquired := r.mu.TryLock()
    return acquired
}

func (r *RateLimiter) Release() {
    r.mu.Unlock()
}

func (r *RateLimiter) Process(f func()) error {
    if !r.TryAcquire() {
        return errors.New("服务繁忙，请稍后重试")
    }
    defer r.Release()
    f()
    return nil
}
```

#### 集成方式

使用 mcp-go 提供的 `WithHandlerHook` 钩子，在处理请求前进行限流检查：

```go
rateLimiter := ratelimit.NewRateLimiter()

sseServer := server.NewSSEServer(
    mcpServer,
    server.WithHandlerHook(func(ctx context.Context, method string, next server.NextHookFunc) (*mcp.CallToolResult, error) {
        if !rateLimiter.TryAcquire() {
            return nil, errors.New("服务繁忙，请稍后重试")
        }
        defer rateLimiter.Release()
        return next(ctx)
    }),
    // ... 其他选项
)
```

#### 错误响应格式
限流失败时返回 JSON-RPC 错误：
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "服务繁忙，请稍后重试"
  },
  "id": null
}
```

### 认证设计

#### 方案选择
采用**固定 Token 验证**方式，从配置文件读取预定义的 token 列表。

#### 配置文件更新

```yaml
# etc/bbc-mcp.yaml
Server:
  Name: bbc-mcp
  Host: 0.0.0.0
  Port: 9000

Auth:
  Tokens:
    - "mf8KPh6f5ma8RhOkIeYYVCK15jtAg4CLUTTlkGHc"
    - "another-token-here"
```

#### 认证中间件

```go
package auth

import (
    "errors"
    "strings"
)

type AuthMiddleware struct {
    tokens map[string]bool
}

func NewAuthMiddleware(tokens []string) *AuthMiddleware {
    m := make(map[string]bool)
    for _, t := range tokens {
        m[t] = true
    }
    return &AuthMiddleware{tokens: m}
}

func (a *AuthMiddleware) Validate(authHeader string) error {
    if authHeader == "" {
        return errors.New("缺少认证信息")
    }
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || parts[0] != "Bearer" {
        return errors.New("认证格式错误，应为: Bearer <token>")
    }
    token := parts[1]
    if !a.tokens[token] {
        return errors.New("无效的认证令牌")
    }
    return nil
}
```

#### 集成方式

在 SSE 处理层添加认证检查：

```go
authMiddleware := auth.NewAuthMiddleware(cfg.Auth.Tokens)

sseServer := server.NewSSEServer(
    mcpServer,
    server.WithSSECORS(server.WithCORSAllowedOrigins("*")),
    server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
    server.WithUseFullURLForMessageEndpoint(false),
    server.WithSSEHandlerWrapper(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if err := authMiddleware.Validate(authHeader); err != nil {
                http.Error(w, err.Error(), http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }),
)
```

## 实现步骤

### Step 1: 配置文件更新
- 在 `etc/bbc-mcp.yaml` 添加 `Auth` 字段
- 更新 `internal/config/config.go` 解析新的配置

### Step 2: 实现限流组件
- 创建 `internal/ratelimit/ratelimiter.go`
- 实现 `RateLimiter` 结构和方法

### Step 3: 实现认证组件
- 创建 `internal/auth/middleware.go`
- 实现 `AuthMiddleware` 结构和方法

### Step 4: 集成到服务器
- 更新 `cmd/bbc-mcp/main.go`
- 初始化限流器和认证中间件
- 使用 `WithHandlerHook` 和 `WithSSEHandlerWrapper` 集成

### Step 5: 测试
- 测试限流功能：并发请求验证限流生效
- 测试认证功能：验证无效 token 被拒绝
- 测试正常流程：验证正常请求不受影响

## 文件结构

```
bbc-mcp/
├── cmd/bbc-mcp/
│   └── main.go                 # 集成限流和认证
├── internal/
│   ├── auth/
│   │   └── middleware.go       # 认证中间件
│   ├── config/
│   │   └── config.go           # 配置解析
│   └── ratelimit/
│       └── ratelimiter.go      # 限流器
├── etc/
│   └── bbc-mcp.yaml            # 配置文件（含 Auth 配置）
└── test/
    └── client/
        └── main.go             # 测试客户端（支持 Authorization 头）
```

## 测试用例

### 限流测试
1. 发送两个并发请求，第一个成功，第二个返回限流错误
2. 第一个请求完成后，后续请求可以正常处理

### 认证测试
1. 无 `Authorization` 头 → 401 错误
2. 格式错误的 `Authorization` 头 → 401 错误
3. 无效 token → 401 错误
4. 有效 token → 正常处理

## 注意事项

1. **线程安全**：限流器和认证中间件都需要是线程安全的
2. **性能影响**：认证检查应尽量高效，避免影响正常请求
3. **日志记录**：认证失败和限流事件应记录日志，便于排查问题
4. **配置热更新**：考虑后续支持配置热更新，无需重启服务