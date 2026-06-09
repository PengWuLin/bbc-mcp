# BBC MCP 安全整改 — 配置文件密码加密设计文档

> 版本：1.0
> 日期：2026-06-09
> 基于：docs/task4.md 需求规格

---

## 1. 概述

### 1.1 背景

当前 `etc/bbc-mcp.yaml` 和 `etc/bbc-mcp-docker.yaml` 中 MySQL 和 Redis 的密码均以明文存储，存在密码泄露风险。一旦配置文件被提交到版本控制系统或被未授权访问，数据库和缓存服务的密码将直接暴露。

### 1.2 目标

- 配置文件中的 `Database.Password` 和 `Redis.Password` 字段存储密文，而非明文
- 服务启动时使用主密钥解密密码后再传递给 MySQL / Redis 连接层
- 提供独立的加解密命令行工具，供运维人员生成密文配置
- 对现有代码侵入尽量小，不影响业务逻辑

### 1.3 非目标

- 不加密 `tools/config.yaml`（该文件属于外部 bbc-tool CLI，不在本服务范围内）
- 不加密 Auth.Tokens（Token 本身非密码，且变更频率高）
- 不引入外部密钥管理服务（如 Vault），通过环境变量传递主密钥即可满足当前安全等级

---

## 2. 总体方案

```
┌──────────────────────────────────────────────────────────────────┐
│                        运维人员                                   │
│  使用 encrypt-tool 工具输入明文密码 + 主密钥 → 生成密文           │
│  将密文写入 etc/bbc-mcp.yaml                                     │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  etc/bbc-mcp.yaml                                                │
│  Database:                                                       │
│    Password: "base64ciphertext"                                   │
│  Redis:                                                          │
│    Password: "base64ciphertext"                                   │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  config.Load()                                                   │
│  1. yaml.Unmarshal 读取配置                                      │
│  2. 尝试 base64 解码 + AES-GCM 解密 Password 字段                  │
│  3. 解密失败则认为是明文，保持不变（兼容过渡期）                      │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  infrastructure.NewMySQLDataSource / NewRedisClient              │
│  使用已解密的明文密码连接（无需改动）                              │
└──────────────────────────────────────────────────────────────────┘
```

**主密钥来源**：环境变量 `BBC_MCP_KEY`，由运维在启动容器或进程时注入，不写入任何文件。

---

## 3. 加密算法选择

### 3.1 算法：AES-256-GCM

| 维度 | 选择 | 说明 |
|------|------|------|
| 算法 | AES-256-GCM | NIST 标准，同时提供机密性和完整性校验 |
| 密钥长度 | 256 bits (32 bytes) | 环境变量 `BBC_MCP_KEY` 的 SHA-256 派生 |
| 模式 | GCM (Galois/Counter Mode) | AEAD 模式，自带认证标签防篡改 |
| Nonce | 12 bytes 随机生成 | GCM 标准 nonce 长度 |
| 密文格式 | base64(nonce + ciphertext + tag) | 无前缀裸密文，不暴露加密算法信息 |

### 3.2 密文格式说明

密文为 base64 编码的二进制数据，结构如下：

```
base64(12-byte-nonce || ciphertext || 16-byte-GCM-tag)
```

密文不含任何前缀或算法标识，外观类似普通 base64 字符串，不暴露加密算法信息。解密时先尝试 base64 解码，再执行 AES-GCM 解密；若任一步骤失败，则认为该值为明文直接使用。

### 3.3 密钥派生

用户提供的 `BBC_MCP_KEY` 可能是任意长度的字符串（密码短语），通过 SHA-256 哈希派生为固定 32 字节的 AES-256 密钥：

```
encryptionKey = SHA256(BBC_MCP_KEY)
```

这样用户可以使用任意长度、便于记忆的密码短语，无需手动管理 32 字节随机密钥。

---

## 4. 新增文件与改动

### 4.1 新增文件清单

| 文件 | 职责 |
|------|------|
| `internal/crypto/crypto.go` | AES-256-GCM 加解密核心逻辑 |
| `internal/crypto/crypto_test.go` | 加解密单元测试 |
| `cmd/encrypt-tool/main.go` | 命令行加解密工具入口 |

### 4.2 修改文件清单

| 文件 | 改动内容 |
|------|----------|
| `internal/config/config.go` | `Load()` 在解析 YAML 后调用 `decryptPasswords()` 解密密码字段 |
| `etc/bbc-mcp.yaml` | `Password` 字段替换为密文 |
| `etc/bbc-mcp-docker.yaml` | `Password` 字段替换为密文 |
| `Makefile` | 新增 `encrypt-tool` 构建目标 |
| `docker-compose.yaml` | 新增 `BBC_MCP_KEY` 环境变量 |
| `go.mod` | 无新增依赖（全部使用标准库 `crypto/aes`、`crypto/sha256` 等） |

---

## 5. 详细设计

### 5.1 加解密模块 `internal/crypto/crypto.go`

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "errors"
    "fmt"
    "io"
    "os"
)

// GetMasterKey 从环境变量 BBC_MCP_KEY 获取主密钥
func GetMasterKey() (string, error) {
    key := os.Getenv("BBC_MCP_KEY")
    if key == "" {
        return "", errors.New("crypto: 环境变量 BBC_MCP_KEY 未设置")
    }
    return key, nil
}

// deriveKey 从密码短语派生 32 字节 AES-256 密钥
func deriveKey(masterKey string) []byte {
    h := sha256.Sum256([]byte(masterKey))
    return h[:]
}

// Encrypt 使用 AES-256-GCM 加密明文，返回 base64 编码的密文
func Encrypt(plaintext, masterKey string) (string, error) {
    key := deriveKey(masterKey)
    block, err := aes.NewCipher(key)
    // ...
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
    combined := append(nonce, ciphertext...)
    return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt 解密 base64 编码的密文
func Decrypt(encoded, masterKey string) (string, error) {
    combined, err := base64.StdEncoding.DecodeString(encoded)
    // ... AES-GCM 解密 ...
    return string(plaintext), nil
}

// DecryptIfNeeded 尝试解密，失败则返回原值（兼容明文过渡期）
func DecryptIfNeeded(encoded string) (string, error) {
    if encoded == "" {
        return encoded, nil
    }
    // 尝试 base64 解码
    combined, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return encoded, nil // 非 base64，视为明文
    }
    key, err := GetMasterKey()
    if err != nil {
        return "", err
    }
    dek := deriveKey(key)
    // ... AES-GCM 解密 ...
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return encoded, nil // 解密失败，视为明文
    }
    return string(plaintext), nil
}

// GenerateKey 生成随机 32 字节密钥（hex 编码）
func GenerateKey() (string, error) {
    b := make([]byte, 32)
    rand.Read(b)
    return hex.EncodeToString(b), nil
}
```

### 5.2 配置加载改造 `internal/config/config.go`

在 `Load()` 函数中，YAML 解析完成后调用 `decryptPasswords()` 对密码字段就地解密：

```go
func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        log.Printf("config: 读取配置文件失败: %v", err)
        return nil, err
    }
    cfg := &Config{}
    if err := yaml.Unmarshal(data, cfg); err != nil {
        log.Printf("config: 解析配置失败: %v", err)
        return nil, err
    }

    // 解密密码字段
    if err := cfg.decryptPasswords(); err != nil {
        log.Printf("config: 解密密码失败: %v", err)
        return nil, err
    }

    return cfg, nil
}

func (c *Config) decryptPasswords() error {
    if c.Database.Password != "" {
        pw, err := crypto.DecryptIfNeeded(c.Database.Password)
        if err != nil {
            return fmt.Errorf("解密 Database.Password 失败: %w", err)
        }
        c.Database.Password = pw
    }
    if c.Redis.Password != "" {
        pw, err := crypto.DecryptIfNeeded(c.Redis.Password)
        if err != nil {
            return fmt.Errorf("解密 Redis.Password 失败: %w", err)
        }
        c.Redis.Password = pw
    }
    return nil
}
```

### 5.3 加密工具 `cmd/encrypt-tool/main.go`

命令行工具，提供 `encrypt` 和 `decrypt` 两个子命令：

```
用法:
  encrypt-tool encrypt <明文密码>    使用 BBC_MCP_KEY 环境变量加密密码
  encrypt-tool decrypt <密文>        使用 BBC_MCP_KEY 环境变量解密密码
  encrypt-tool genkey                生成随机主密钥（供运维参考）

示例:
  export BBC_MCP_KEY="my-secret-master-key"
  encrypt-tool encrypt "YkRPTXf6EIChMOvz"
  # 输出: SsDB/PRMydaCaoAd7vIyLvTDN/BJT3c31KWIfHtkzAk1jZWr92zTd8FH7T0=

  encrypt-tool decrypt "SsDB/PRMydaCaoAd7vIyLvTDN/BJT3c31KWIfHtkzAk1jZWr92zTd8FH7T0="
  # 输出: YkRPTXf6EIChMOvz
```

`genkey` 子命令使用 `crypto/rand` 生成 32 字节安全随机数并以 hex 编码输出，供运维人员选用作为主密钥。

### 5.4 基础设施层 — 无需改动

`internal/infrastructure/mysql.go` 和 `redis.go` 收到的 `cfg.Password` 已经是解密后的明文，与当前行为一致，无需修改。

---

## 6. 部署变更

### 6.1 docker-compose.yaml 变更

新增 `BBC_MCP_KEY` 环境变量：

```yaml
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    image: bbc-mcp:latest
    container_name: bbc-mcp-app
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./etc/bbc-mcp-docker.yaml:/app/etc/bbc-mcp.yaml:ro
      - /root/code/tools/:/app/tools
    environment:
      BBC_MCP_CONFIG: /app/etc/bbc-mcp.yaml
      BBC_MCP_KEY: ${BBC_MCP_KEY}        # 新增：从宿主机环境变量传入
```

### 6.2 运维操作流程变更

**改造前**：
1. 直接在 yaml 中写入明文密码
2. 启动服务

**改造后**：
1. 设置 `BBC_MCP_KEY` 环境变量（首次执行 `encrypt-tool genkey` 生成）
2. 使用 `encrypt-tool encrypt <明文>` 生成密文
3. 将密文写入 yaml 的 `Password` 字段
4. 启动服务时传入 `BBC_MCP_KEY`

### 6.3 Makefile 新增目标

```makefile
# ---- encrypt-tool ----
build-tool:
	@mkdir -p $(OUTDIR)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(OUTDIR)/encrypt-tool ./cmd/encrypt-tool/
```

---

## 7. 安全分析

### 7.1 威胁模型

| 威胁 | 缓解措施 |
|------|----------|
| 配置文件泄露（提交到 Git、复制到 U 盘等） | 密码以密文存储，无主密钥无法解密 |
| 主密钥泄露（环境变量被读取） | 仅运行中的进程可读取自身环境变量，需 root 权限访问 `/proc/<pid>/environ` |
| 密文被篡改 | GCM 模式自带认证标签，篡改后解密失败并报错 |
| 重放攻击 | 每次加密使用随机 nonce，相同明文每次生成不同密文 |
| 主密钥暴力破解 | 使用 SHA-256 派生 + AES-256，计算不可行 |

### 7.2 主密钥生命周期

- **生成**：运维通过 `encrypt-tool genkey` 生成随机密钥，或自行拟定密码短语
- **分发**：运维通过安全渠道（堡垒机环境变量、K8s Secret 等）注入到运行环境
- **轮换**：生成新密钥 → 用新密钥重新加密所有密码 → 更新配置文件 → 重启服务
- **吊销**：如怀疑密钥泄露，立即轮换密钥并更新所有数据库/Redis 密码

### 7.3 注意事项

- `BBC_MCP_KEY` 不会出现在任何日志或配置文件中
- 空密码（如本地 Redis 无密码）不会被加密，`DecryptIfNeeded` 在空字符串上短路
- 主密钥丢失后密文不可恢复，需妥善保管

---

## 8. 测试策略

### 8.1 单元测试 `internal/crypto/crypto_test.go`

| 用例 | 说明 |
|------|------|
| `TestEncryptDecrypt` | 正常加密后解密，验证明文一致 |
| `TestDecryptIfNeededPlaintextPassthrough` | 明文输入直接透传，不做解密 |
| `TestDecryptWrongKey` | 错误密钥解密应返回错误 |
| `TestDecryptTampered` | 篡改后的密文解密应失败（GCM 认证） |
| `TestEncryptSameInputDifferentOutput` | 相同明文两次加密结果不同（随机 nonce） |
| `TestDeriveKey` | 相同输入产生相同派生密钥 |
| `TestEmptyPassword` | 空密码不做加密/解密处理 |
| `TestDecryptIfNeededWithEncryptedValue` | 密文经 DecryptIfNeeded 正常解密 |
| `TestDecryptIfNeededWrongKeyFallsBack` | 错误密钥时 DecryptIfNeeded 回退返回原值 |
| `TestGenerateKey` | 生成随机密钥 |

### 8.2 集成验证

1. 使用 `encrypt-tool` 加密测试密码
2. 替换 `etc/bbc-mcp.yaml` 中数据库密码为密文
3. 设置 `BBC_MCP_KEY` 后启动服务
4. 验证 MySQL 和 Redis 连接正常
5. 验证各 MCP Tool 功能正常

---

## 9. 实施步骤

| 步骤 | 内容 | 涉及文件 |
|------|------|----------|
| 1 | 新建 `internal/crypto/crypto.go`，实现加解密核心逻辑 | crypto.go |
| 2 | 新建 `internal/crypto/crypto_test.go`，编写单元测试 | crypto_test.go |
| 3 | 修改 `internal/config/config.go`，Load() 增加解密步骤 | config.go |
| 4 | 新建 `cmd/encrypt-tool/main.go`，实现命令行工具 | main.go |
| 5 | 用 encrypt-tool 生成密文，替换两个 yaml 中的 Password | bbc-mcp.yaml, bbc-mcp-docker.yaml |
| 6 | docker-compose.yaml 新增 BBC_MCP_KEY 环境变量 | docker-compose.yaml |
| 7 | Makefile 新增 build-tool 目标 | Makefile |
| 8 | 运行单元测试 + 集成测试验证 | - |

---

## 10. 向后兼容

- 明文密码直接透传：`DecryptIfNeeded` 先尝试 base64 解码，非 base64 格式直接视为明文返回
- 即使字符串是 base64 格式，若 GCM 认证失败（密钥不匹配或非加密数据），同样视为明文返回
- 过渡期可逐步将明文替换为密文，无需一次性全部切换
- 基础设施层（infrastructure）完全无感知，无需任何改动
- 空密码不做处理，直接返回空字符串
