package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"bbc-mcp/internal/config"
	"bbc-mcp/internal/infrastructure"
	"bbc-mcp/internal/tool"
)

type BBCMCPServer struct {
	server *server.MCPServer
	deps   *tool.Dependencies
}

func NewBBCMCPServer(deps *tool.Dependencies) *BBCMCPServer {
	s := server.NewMCPServer(
		"bbc-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)
	tool.RegisterAll(s, deps)
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

func main() {
	cfgPath := "etc/bbc-mcp.yaml"
	if v := os.Getenv("BBC_MCP_CONFIG"); v != "" {
		cfgPath = v
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
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

	deps := &tool.Dependencies{
		DB:     db,
		Redis:  redisClient,
		Config: cfg,
	}

	bbcServer := NewBBCMCPServer(deps)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	sseServer := bbcServer.ServeSSE(addr)

	// 优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("收到信号 %v，正在关闭服务...", sig)
		sseServer.Shutdown(context.Background())
	}()

	log.Printf("bbc-mcp SSE server 启动于 %s", addr)
	if err := sseServer.Start(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
