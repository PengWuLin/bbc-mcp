# BBC MCP 短信云续期 设计文档

> 版本：1.0
> 日期：2026-06-18
> 基于：docs/task10.md 需求规格

---

## 1. 概述

### 1.1 背景

当前已支持短信云套餐查询（`sms_package_list`），但缺少套餐续期能力。需要新增续期工具，支持对指定租户的套餐进行续期操作：更新 MySQL 中的过期时间，并清除对应的 Redis 缓存。

短信云使用独立的 Redis 实例，需要单独配置。

### 1.2 目标

- 新增 `sms_package_renew` MCP 工具，支持按套餐 ID 和租户 ID 续期
- 续期默认时间为当前时间加一年（23:59:00）
- 删除对应租户的 Redis 缓存，使续期立即生效
- 支持独立的短信云 Redis 配置（IP、密码等）

### 1.3 非目标

- 不支持自定义续期时长（固定一年）
- 不支持批量续期（每次续期一个套餐）
- 不修改现有的 `sms_package_list` 工具

---

## 2. 总体方案

```
┌─────────────────────────────────────────────┐
│  MCP Client 调用 sms_package_renew           │
│  入参: id (套餐ID), ccode (租户ID)            │
└────────────────────┬────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│  tool/sms_package_renew.go                   │
│  1. 校验 SMSDB 和 SMSRedis 已配置             │
│  2. 提取并校验入参                             │
│  3. 调用 Repository.RenewExpireTime()         │
└────────────────────┬────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│  repository/sms_package_repo.go              │
│  RenewExpireTime(id, ccode):                 │
│    a. UPDATE package SET expire_time=?       │
│       WHERE id=? LIMIT 1  (MySQL/csmsdb)    │
│    b. DEL csms:corp:<ccode>:package:availnum │
│       (SMS Redis)                            │
└─────────────────────────────────────────────┘
```

**续期时间计算**：`time.Now().AddDate(1, 0, 0)` 取日期部分 + `23:59:00`。

**Redis Key 规则**：`csms:corp:<ccode>:package:availnum`，其中 `<ccode>` 为入参租户 ID。

---

## 3. 详细设计

### 3.1 配置扩展 `internal/config/config.go`

新增 `SMSRedis` 配置字段，复用现有 `RedisConfig` 类型：

```go
type Config struct {
    // ... 现有字段保持不变 ...
    SMSRedis RedisConfig `yaml:"SMSRedis"`  // 新增
}
```

`decryptSecrets()` 中新增 SMSRedis 密码解密：

```go
// SMSRedis.Password
if c.SMSRedis.Password != "" {
    pw, err := crypto.DecryptIfNeeded(c.SMSRedis.Password)
    if err != nil {
        return fmt.Errorf("解密 SMSRedis.Password 失败: %w", err)
    }
    c.SMSRedis.Password = pw
}
```

### 3.2 依赖扩展 `internal/tool/registry.go`

```go
type Dependencies struct {
    DB         *sql.DB
    SMSDB      *sql.DB
    Redis      *redis.Client
    SMSRedis   *redis.Client    // 新增：短信云独立 Redis
    Config     *config.Config
    K8sClients map[string]*k8s.Client
}
```

### 3.3 入口初始化 `cmd/bbc-mcp/main.go`

在 `main()` 中新增 SMSRedis 初始化逻辑（模式与 SMSDB 一致）：

```go
var smsRedis *redis.Client
if cfg.SMSRedis.Host != "" {
    smsRedis, err = infrastructure.NewRedisClient(&cfg.SMSRedis)
    if err != nil {
        log.Fatalf("初始化短信云 Redis 失败: %v", err)
    }
    defer smsRedis.Close()
    log.Printf("短信云 Redis 连接成功: %s:%d", cfg.SMSRedis.Host, cfg.SMSRedis.Port)
}
```

传入 Dependencies：

```go
deps := &tool.Dependencies{
    // ...
    SMSRedis: smsRedis,
}
```

### 3.4 Repository 扩展 `internal/repository/sms_package_repo.go`

`SMSPackageRepository` 增加 Redis 字段，`NewSMSPackageRepository` 增加 redis 参数（可为 nil）：

```go
type SMSPackageRepository struct {
    db    *sql.DB
    redis *redis.Client
}

func NewSMSPackageRepository(db *sql.DB, rds *redis.Client) *SMSPackageRepository {
    return &SMSPackageRepository{db: db, redis: rds}
}
```

新增 `RenewExpireTime` 方法：

```go
func (r *SMSPackageRepository) RenewExpireTime(ctx context.Context, id int, ccode string) error {
    // 1. 更新 MySQL
    ctxDB, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    expireTime := time.Now().AddDate(1, 0, 0).Format("2006-01-02") + " 23:59:00"
    result, err := r.db.ExecContext(ctxDB,
        "UPDATE package SET expire_time=? WHERE id=? LIMIT 1",
        expireTime, id)
    if err != nil {
        log.Printf("repository: 续期套餐失败(id=%d): %v", id, err)
        return fmt.Errorf("renew sms package: %w", err)
    }
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return fmt.Errorf("套餐不存在: id=%d", id)
    }

    // 2. 删除 Redis 缓存
    if r.redis != nil {
        ctxRedis, cancel := context.WithTimeout(ctx, 3*time.Second)
        defer cancel()

        key := fmt.Sprintf("csms:corp:%s:package:availnum", ccode)
        if err := r.redis.Del(ctxRedis, key).Err(); err != nil {
            log.Printf("repository: 删除 Redis 缓存失败(key=%s): %v", key, err)
            // 缓存删除失败不阻塞续期结果
        }
    }

    return nil
}
```

**注意**：`sms_package_list` 调用处需要适配新的构造函数签名，第三个参数传 `nil`（查询无需 Redis）。

### 3.5 新工具 `internal/tool/sms_package_renew.go`

```go
func newSMSPackageRenew(deps *Dependencies) ToolDefinition {
    return ToolDefinition{
        Tool: mcp.NewTool("sms_package_renew",
            mcp.WithDescription("对指定租户的短信套餐进行续期，续期时长为一年，到期时间为当前时间加一年的 23:59:00。\n\n参数：\n  - id (integer, 必填): 套餐 ID\n  - ccode (string, 必填): 租户 ID（公司代码）\n\n返回值：\n  成功时返回 {\"status\":\"success\",\"id\":2670,\"expire_time\":\"2027-06-18 23:59:00\"}\n  失败时返回错误信息\n\n注意事项：\n  - 短信云数据库未配置时返回错误\n  - 套餐不存在时返回错误"),
            mcp.WithInteger("id",
                mcp.Required(),
                mcp.Description("套餐ID"),
            ),
            mcp.WithString("ccode",
                mcp.Required(),
                mcp.Description("租户ID（公司代码）"),
            ),
        ),
        Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
            if deps.SMSDB == nil {
                return mcp.NewToolResultError("短信云数据库未配置"), nil
            }

            args, _ := req.Params.Arguments.(map[string]interface{})

            idFloat, ok := args["id"].(float64)
            if !ok {
                return mcp.NewToolResultError("参数错误: id 必须为整数"), nil
            }
            id := int(idFloat)

            ccode, ok := args["ccode"].(string)
            if !ok || ccode == "" {
                return mcp.NewToolResultError("参数错误: ccode 必须为非空字符串"), nil
            }

            repo := repository.NewSMSPackageRepository(deps.SMSDB, deps.SMSRedis)
            if err := repo.RenewExpireTime(ctx, id, ccode); err != nil {
                log.Printf("sms_package_renew: 续期失败: %v", err)
                return mcp.NewToolResultError(fmt.Sprintf("续期失败: %v", err)), nil
            }

            expireTime := time.Now().AddDate(1, 0, 0).Format("2006-01-02") + " 23:59:00"
            result := map[string]interface{}{
                "status":      "success",
                "id":          id,
                "expire_time": expireTime,
            }
            jsonBytes, _ := json.Marshal(result)
            return mcp.NewToolResultText(string(jsonBytes)), nil
        },
    }
}
```

### 3.6 工具注册 `internal/tool/registry.go`

在 `RegisterAll` 中追加：

```go
newSMSPackageRenew(deps),
```

---

## 4. 文件清单

### 4.1 新增文件

| 文件 | 职责 |
|------|------|
| `internal/tool/sms_package_renew.go` | `sms_package_renew` MCP 工具定义 |

### 4.2 修改文件

| 文件 | 改动内容 |
|------|----------|
| `internal/config/config.go` | 新增 `SMSRedis RedisConfig` 字段 + `decryptSecrets()` 解密 |
| `internal/tool/registry.go` | `Dependencies` 新增 `SMSRedis *redis.Client`；`RegisterAll` 注册新工具 |
| `internal/tool/sms_package_list.go` | 适配 `NewSMSPackageRepository` 新签名（增加 redis 参数，传 nil） |
| `internal/repository/sms_package_repo.go` | 增加 redis 字段，构造函数增加 redis 参数，新增 `RenewExpireTime` 方法 |
| `cmd/bbc-mcp/main.go` | 初始化 SMSRedis 客户端，传入 Dependencies |
| `etc/bbc-mcp.yaml` | 新增 `SMSRedis` 配置段 |
| `etc/bbc-mcp-docker.yaml` | 同上 |

### 4.3 无需改动的文件

| 文件 | 说明 |
|------|------|
| `internal/infrastructure/redis.go` | 复用 `NewRedisClient`，无需修改 |
| `internal/infrastructure/mysql.go` | 复用 `NewMySQLDataSource`，无需修改 |
| `internal/crypto/crypto.go` | 复用现有加密方案 |
| `internal/types/sms_package.go` | 无需新增类型 |
| `internal/auth/` | 无影响 |

---

## 5. 安全设计

- SMSRedis 密码通过 AES-256-GCM 加密存储，与现有 Database/Redis 密码一致
- 续期操作经过 Bearer Token 认证（auth 中间件统一拦截）
- SQL 使用参数化查询，防止注入
- Redis DEL 操作失败不阻塞续期结果（缓存可被动过期）

---

## 6. 向后兼容

- `SMSRedis` 配置为空时（Host 为空字符串），SMSRedis 客户端为 nil，不影响现有功能
- `SMSPackageRepository` 的 redis 参数可为 nil，当 SMSRedis 未配置时缓存删除步骤跳过
- 现有 `sms_package_list` 行为完全不变（redis 传 nil）
- 配置文件可逐步迁移：旧的配置文件缺少 `SMSRedis` 段时使用零值（空 Host），不会导致启动失败

---

## 7. 实施步骤

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | `Config` 新增 `SMSRedis` 字段 + 解密 | config.go |
| 2 | `Dependencies` 新增 `SMSRedis` | registry.go |
| 3 | `SMSPackageRepository` 增加 redis 字段和 `RenewExpireTime` 方法 | sms_package_repo.go |
| 4 | 创建 `sms_package_renew.go` 工具文件 | sms_package_renew.go |
| 5 | `RegisterAll` 注册新工具；`sms_package_list` 适配新签名 | registry.go, sms_package_list.go |
| 6 | `main.go` 初始化 SMSRedis 并注入 | main.go |
| 7 | 配置文件中新增 `SMSRedis` 段 | bbc-mcp.yaml, bbc-mcp-docker.yaml |
| 8 | `go vet ./...` + `go build` 验证编译 | - |
| 9 | 启动服务验证续期功能和缓存删除 | - |
