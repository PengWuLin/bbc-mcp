APP_NAME    := bbc-mcp
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "latest")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS    := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

GO         := go
DOCKER     := docker
DOCKER_IMG := $(APP_NAME):$(VERSION)
CONFIG     := etc/$(APP_NAME).yaml
OUTDIR     := bin

.PHONY: build build-linux run test docker-build docker-run docker-stop docker-up docker-down docker-logs clean docker-clean

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
	BBC_MCP_CONFIG=$(CONFIG) $(OUTDIR)/$(APP_NAME)

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

# ---- docker-compose ----

docker-up:
	$(DOCKER) compose up -d

docker-down:
	$(DOCKER) compose down

docker-logs:
	$(DOCKER) compose logs -f
