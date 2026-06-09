# BBC MCP 安全整改 — Token 加密设计文档

> 版本：1.0
> 日期：2026-06-09
> 基于：docs/task7.md 需求规格

---

## 1. 概述

### 1.1 背景

当前配置文件中 `Auth.Tokens`（Bearer 认证令牌）和 `K8sClusters.*.Token`（Kubernetes 服务账户令牌）均以明文存储，一旦配置文件泄露，攻击者可直接使用这些令牌访问 MCP 服务和 K8s 集群。

`Database.Password` 和 `Redis.Password` 已在 Task 4 中实现了 AES-256-GCM 加密，本次整改将相同的加密方案扩展到 Token 字段。

### 1.2 目标

- `Auth.Tokens[]` 中每个 Token 存储密文（与 Password 相同的 base64+AES-GCM 格式）
- `K8sClusters.*.Token` 存储密文
- 服务启动时自动解密，业务代码无需改动
- 复用现有 `encrypt-tool` CLI 生成密文

### 1.3 非目标

- 不改造 `BbcTool.Path`、`Server` 等非敏感字段
- 不引入新的加密算法或密钥体系

---

## 2. 总体方案

复用 Task 4 的加密基础设施（`internal/crypto`），扩展 `config.Load()` 的解密范围。

```
┌──────────────────────────────────────────────────────────────────┐
│  etc/bbc-mcp.yaml                                                 │
│  Auth:                                                             │
│    Tokens:                                                         │
│      - "base64-ciphertext-1"    ← 加密后的 Bearer Token            │
│      - "base64-ciphertext-2"    ← 加密后的 Bearer Token            │
│  K8sClusters:                                                      │
│    bbc-shenzhen4:                                                  │
│      Server: "https://..."                                         │
│      Token: "base64-ciphertext" ← 加密后的 K8s ServiceAccount Token│
│      Insecure: true                                                │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  config.Load()                                                    │
│  1. yaml.Unmarshal 读取配置                                       │
│  2. applyDefaults()                                               │
│  3. decryptSecrets(): 遍历 Auth.Tokens 和 K8sClusters.*.Token      │
│     每个值调用 crypto.DecryptIfNeeded() — 密文解密，明文透传        │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  业务层（auth、k8s）使用已解密的明文 Token，完全无感知              │
└──────────────────────────────────────────────────────────────────┘
```

**加密方案**：与 Task 4 完全一致 — AES-256-GCM，nonce+ciphertext+tag → base64 编码，无前缀。
**解密策略**：`DecryptIfNeeded` — base64 解码失败或 GCM 认证失败则视为明文透传。

---

## 3. 详细设计

### 3.1 配置解密扩展 `internal/config/config.go`

将现有 `decryptPasswords()` 重命名为 `decryptSecrets()`，并扩展解密范围：

```go
func (c *Config) decryptSecrets() error {
    // Database.Password
    if c.Database.Password != "" {
        pw, err := crypto.DecryptIfNeeded(c.Database.Password)
        if err != nil {
            return fmt.Errorf("解密 Database.Password 失败: %w", err)
        }
        c.Database.Password = pw
    }

    // Redis.Password
    if c.Redis.Password != "" {
        pw, err := crypto.DecryptIfNeeded(c.Redis.Password)
        if err != nil {
            return fmt.Errorf("解密 Redis.Password 失败: %w", err)
        }
        c.Redis.Password = pw
    }

    // Auth.Tokens
    for i, token := range c.Auth.Tokens {
        if token != "" {
            decrypted, err := crypto.DecryptIfNeeded(token)
            if err != nil {
                return fmt.Errorf("解密 Auth.Tokens[%d] 失败: %w", i, err)
            }
            c.Auth.Tokens[i] = decrypted
        }
    }

    // K8sClusters.*.Token
    for name, cluster := range c.K8sClusters {
        if cluster.Token != "" {
            decrypted, err := crypto.DecryptIfNeeded(cluster.Token)
            if err != nil {
                return fmt.Errorf("解密 K8sClusters.%s.Token 失败: %w", name, err)
            }
            cluster.Token = decrypted
            c.K8sClusters[name] = cluster
        }
    }

    return nil
}
```

`Load()` 中将调用从 `decryptPasswords()` 改为 `decryptSecrets()`。

### 3.2 加密工具 — 无需改动

`cmd/encrypt-tool/main.go` 无需任何修改，`encrypt`/`decrypt`/`genkey` 命令通用支持任意字符串的加解密：

```
# 加密 Bearer Token
encrypt-tool encrypt "mf8KPh6f5ma8RhOkIeYYVCK15jtAg4CLUTTlkGHc"

# 加密 K8s Token
encrypt-tool encrypt "eyJhbGciOiJS..."
```

### 3.3 crypto 模块 — 无需改动

`internal/crypto/crypto.go` 无需任何修改，`DecryptIfNeeded` 已是通用方法。

---

## 4. 文件清单

### 4.1 修改文件

| 文件 | 改动内容 |
|------|----------|
| `internal/config/config.go` | `decryptPasswords()` → `decryptSecrets()`，新增 Auth.Tokens 和 K8sClusters 解密逻辑 |
| `etc/bbc-mcp.yaml` | Auth.Tokens 和 K8sClusters.*.Token 替换为密文 |
| `etc/bbc-mcp-docker.yaml` | 同上 |

### 4.2 无需改动

| 文件 | 说明 |
|------|------|
| `internal/crypto/crypto.go` | `DecryptIfNeeded` 通用方法，无需修改 |
| `cmd/encrypt-tool/main.go` | 通用加解密 CLI，无需修改 |
| `internal/auth/` | 收到的 Token 已解密，无感知 |
| `internal/k8s/client.go` | 收到的 Token 已解密，无感知 |
| `cmd/bbc-mcp/main.go` | 无影响 |

---

## 5. 向后兼容

- `DecryptIfNeeded` 的行为保证了向后兼容：明文 Token 透传，密文 Token 自动解密
- 支持逐步迁移：可以部分 Token 加密、部分保持明文
- 空值安全：空字符串不执行解密操作

---

## 6. 实施步骤

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | 将 `decryptPasswords()` 重命名为 `decryptSecrets()`，增加 Auth.Tokens 和 K8sClusters Token 解密循环 | config.go |
| 2 | 使用 encrypt-tool 加密所有 Token，替换两个 YAML 配置中的明文 | bbc-mcp.yaml, bbc-mcp-docker.yaml |
| 3 | `go vet ./...` + `go build` 验证编译 | - |
| 4 | 启动服务验证认证和 K8s 连接正常 | - |
