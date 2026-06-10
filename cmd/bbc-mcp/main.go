package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"bbc-mcp/internal/auth"
	"bbc-mcp/internal/config"
	"bbc-mcp/internal/infrastructure"
	"bbc-mcp/internal/k8s"
	"bbc-mcp/internal/prompt"
	"bbc-mcp/internal/ratelimit"
	"bbc-mcp/internal/tool"
)

type BBCMCPServer struct {
	server *server.MCPServer
	deps   *tool.Dependencies
}

func NewBBCMCPServer(deps *tool.Dependencies, hooks *server.Hooks) *BBCMCPServer {
	s := server.NewMCPServer(
		"bbc-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithHooks(hooks),
	)
	tool.RegisterAll(s, deps)

	guidePrompt, guideHandler := prompt.NewBBCGuidePrompt()
	s.AddPrompt(guidePrompt, guideHandler)

	return &BBCMCPServer{server: s, deps: deps}
}

func (s *BBCMCPServer) ServeSSE(addr string) *server.SSEServer {
	return server.NewSSEServer(
		s.server,
		server.WithSSECORS(
			server.WithCORSAllowedOrigins("*"),
		),
		server.WithBaseURL(fmt.Sprintf("http://%s", addr)),
		server.WithUseFullURLForMessageEndpoint(false),
	)
}

func createRateLimitHooks(rl *ratelimit.RateLimiter) *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddOnRequestInitialization(func(ctx context.Context, id any, message any) error {
		if !rl.TryAcquire() {
			return ratelimit.ErrRateLimitExceeded
		}
		return nil
	})

	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		rl.Release()
	})

	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		rl.Release()
	})

	return hooks
}

func main() {
	cfgPath := "etc/bbc-mcp.yaml"
	if v := os.Getenv("BBC_MCP_CONFIG"); v != "" {
		cfgPath = v
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	k8sClients := make(map[string]*k8s.Client)
	if cfg.Gateway.Mode == "native" {
		for name, clusterCfg := range cfg.K8sClusters {
			client, err := k8s.NewClient(name, clusterCfg.Server, clusterCfg.Token, clusterCfg.CAData, clusterCfg.Insecure)
			if err != nil {
				log.Fatalf("初始化 K8s 客户端 %s 失败: %v", name, err)
			}
			k8sClients[name] = client
			log.Printf("K8s 客户端 %s 初始化成功", name)
		}
	}

	db, err := infrastructure.NewMySQLDataSource(&cfg.Database)
	if err != nil {
		log.Fatalf("初始化 MySQL 失败: %v", err)
	}
	defer db.Close()

	redisClient, err := infrastructure.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}
	defer redisClient.Close()

	var smsDB *sql.DB
	if cfg.SMSDatabase.Host != "" {
		smsDB, err = infrastructure.NewMySQLDataSource(&cfg.SMSDatabase)
		if err != nil {
			log.Fatalf("初始化短信云数据库失败: %v", err)
		}
		defer smsDB.Close()
		log.Printf("短信云数据库连接成功: %s:%d/%s",
			cfg.SMSDatabase.Host, cfg.SMSDatabase.Port, cfg.SMSDatabase.Name)
	}

	deps := &tool.Dependencies{
		DB:         db,
		SMSDB:      smsDB,
		Redis:      redisClient,
		Config:     cfg,
		K8sClients: k8sClients,
	}

	rateLimiter := ratelimit.NewRateLimiter()
	hooks := createRateLimitHooks(rateLimiter)

	bbcServer := NewBBCMCPServer(deps, hooks)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	sseServer := bbcServer.ServeSSE(addr)

	authMiddleware := auth.NewAuthMiddleware(cfg.Auth.Tokens)

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if err := authMiddleware.Validate(authHeader); err != nil {
			log.Printf("auth: 认证失败: %v (from %s)", err, r.RemoteAddr)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		sseServer.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:    addr,
		Handler: wrappedHandler,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("收到信号 %v，正在关闭服务...", sig)
		httpServer.Shutdown(context.Background())
	}()

	log.Printf("bbc-mcp SSE server 启动于 %s", addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败: %v", err)
	}
}
