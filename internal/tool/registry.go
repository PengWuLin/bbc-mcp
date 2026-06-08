package tool

import (
	"context"
	"database/sql"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/redis/go-redis/v9"

	"bbc-mcp/internal/config"
)

type ToolDefinition struct {
	Tool    mcp.Tool
	Handler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type Dependencies struct {
	DB     *sql.DB
	Redis  *redis.Client
	Config *config.Config
}

func RegisterAll(s *server.MCPServer, deps *Dependencies) {
	definitions := []ToolDefinition{
		newGatewayStatus(deps),
		newDeviceList(deps),
		newDeviceStatus(deps),
	}
	for _, def := range definitions {
		s.AddTool(def.Tool, def.Handler)
	}
}
