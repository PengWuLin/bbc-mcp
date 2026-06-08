# mcp server 迭代三设计 — 打包与构建

## 概述
本迭代为 mcp server 制作 Docker 镜像构建文件和 Makefile 自动化构建脚本，实现一键构建、打包、部署。

## 需求

### Dockerfile
- 制作 Docker 镜像构建文件
- 支持多阶段构建，镜像体积尽量小
- 配置文件通过挂载方式注入，不打包进镜像

### Makefile
- 制作 Makefile 自动化构建脚本
- 支持 `build` / `docker-build` / `run` / `test` / `clean` 等目标

## 设计

### Dockerfile 设计

#### 方案选择
采用**多阶段构建**，第一阶段编译 Go 二进制，第二阶段使用 `alpine` 轻量镜像运行，最终镜像体积约 15MB。

#### 目录规划

```
bbc-mcp/
├── Dockerfile
├── Makefile
├── cmd/bbc-mcp/main.go
├── etc/bbc-mcp.yaml          # 配置文件（挂载，不打包）
├── internal/...
└── ...
```

#### Dockerfile

```dockerfile
# ============================================================
# Stage 1: build
# ============================================================
FROM golang:1.26.2-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /build/bbc-mcp ./cmd/bbc-mcp/

# ============================================================
# Stage 2: run
# ============================================================
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/bbc-mcp .

EXPOSE 9000

ENTRYPOINT ["./bbc-mcp"]
```

#### 使用方式

```bash
# 构建镜像
docker build -t bbc-mcp:latest .

# 运行容器（挂载配置文件）
docker run -d \
  --name bbc-mcp \
  -p 9000:9000 \
  -v $(pwd)/etc/bbc-mcp.yaml:/app/etc/bbc-mcp.yaml:ro \
  bbc-mcp:latest
```

### Makefile 设计

#### 变量

```makefile
APP_NAME    := bbc-mcp
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

GO         := go
DOCKER     := docker
DOCKER_IMG := $(APP_NAME):$(VERSION)
CONFIG     := etc/$(APP_NAME).yaml
```

#### 目标

| 目标 | 说明 |
|------|------|
| `build` | 编译当前平台的二进制到 `bin/` 目录 |
| `build-linux` | 交叉编译 Linux amd64 二进制 |
| `run` | 本地编译并运行，加载 `etc/bbc-mcp.yaml` 配置 |
| `test` | 运行 `go vet` + 单元测试 |
| `docker-build` | 构建 Docker 镜像 |
| `docker-run` | 构建并启动 Docker 容器 |
| `docker-stop` | 停止并删除 Docker 容器 |
| `clean` | 清理 `bin/` 编译产物 |
| `docker-clean` | 删除 Docker 镜像 |

#### Makefile

```makefile
APP_NAME    := bbc-mcp
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

GO         := go
DOCKER     := docker
DOCKER_IMG := $(APP_NAME):$(VERSION)
CONFIG     := etc/$(APP_NAME).yaml
OUTDIR     := bin

.PHONY: build build-linux run test docker-build docker-run docker-stop clean docker-clean

# ---- build ----

build: $(OUTDIR)/$(APP_NAME)

$(OUTDIR)/$(APP_NAME):
	@mkdir -p $(OUTDIR)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(OUTDIR)/$(APP_NAME) ./cmd/$(APP_NAME)/

build-linux:
	@mkdir -p $(OUTDIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -ldflags="$(LDFLAGS)" -o $(OUTDIR)/$(APP_NAME)-linux-amd64 ./cmd/$(APP_NAME)/

# ---- run ----

run: build
	$(OUTDIR)/$(APP_NAME) --config=$(CONFIG)

# ---- test ----

test:
	$(GO) vet ./...
	$(GO) test ./... -count=1

# ---- docker ----

docker-build:
	$(DOCKER) build -t $(DOCKER_IMG) .

docker-run: docker-build
	$(DOCKER) run -d \
		--name $(APP_NAME) \
		-p 9000:9000 \
		-v $(realpath $(CONFIG)):/app/$(CONFIG):ro \
		$(DOCKER_IMG)

docker-stop:
	-$(DOCKER) stop $(APP_NAME)
	-$(DOCKER) rm $(APP_NAME)

# ---- clean ----

clean:
	rm -rf $(OUTDIR)

docker-clean:
	-$(DOCKER) rmi $(DOCKER_IMG)
```

### .dockerignore

为避免将无关文件发送到 Docker 构建上下文，需要创建 `.dockerignore` 文件：

```
.git
.gitignore
.idea
.vscode
.claude
*.md
docs/
example/
test/
bin/
Dockerfile
Makefile
.dockerignore
```

## 实现步骤

### Step 1: 创建 .dockerignore
- 排除 Git、IDE 配置、文档、测试等无关文件

### Step 2: 创建 Dockerfile
- 多阶段构建：golang:1.26.2-alpine 编译 → alpine:3.22 运行
- 暴露 9000 端口

### Step 3: 创建 Makefile
- 实现 build / run / test / docker-build / docker-run / clean 等目标

### Step 4: 验证
- `make build` 本地编译
- `make docker-build` 构建镜像
- `make docker-run` 启动容器，验证服务可正常访问

## 镜像体积估算

| 阶段 | 基础镜像 | 大小 |
|------|----------|------|
| builder | golang:1.26.2-alpine | ~380MB（仅构建用） |
| run | alpine:3.22 | ~9MB |
| 最终镜像 | bbc-mcp | ~15MB（alpine + 二进制） |

## 注意事项

1. **CGO_ENABLED=0**：纯静态编译，确保可在 alpine/scratch 中运行
2. **-ldflags="-s -w"**：去除调试信息，减小二进制体积
3. **配置文件挂载**：配置文件通过 `-v` 挂载而非打包进镜像，便于不同环境切换
4. **ca-certificates**：alpine 运行阶段安装 CA 证书，确保 HTTPS 请求正常
5. **tzdata**：安装时区数据，确保日志时间正确
6. **Linux 构建兼容性**：`make build-linux` 支持 macOS 上交叉编译 Linux 二进制
