package tool

import (
	"context"
	"database/sql"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"

	"bbc-mcp/internal/config"
	"bbc-mcp/internal/k8s"
)

type ToolDefinition struct {
	Tool    mcp.Tool
	Handler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type Dependencies struct {
	DB         *sql.DB
	SMSDB      *sql.DB
	Redis      *redis.Client
	SMSRedis   *redis.Client
	Config     *config.Config
	K8sClients map[string]*k8s.Client
}

func RegisterAll(s *server.MCPServer, deps *Dependencies) {
	definitions := []ToolDefinition{
		newDatacenterStatus(deps),
		newDeviceList(deps),
		newDeviceStatus(deps),
		newSMSPackageList(deps),
			newSMSPackageRenew(deps),
	}
	for _, def := range definitions {
		s.AddTool(def.Tool, def.Handler)
	}
}
