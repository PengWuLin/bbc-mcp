package tool

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func newGatewayStatus(deps *Dependencies) ToolDefinition {
	return ToolDefinition{
		Tool: mcp.NewTool("gateway_status",
			mcp.WithDescription("查询当前 minibbc 数据中心设备连接数"),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			toolPath := deps.Config.BbcTool.Path
			timeout := time.Duration(deps.Config.BbcTool.Timeout) * time.Second

			cmdCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, toolPath, "status", "gateway")
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("gateway_status: 执行命令失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("执行 bbc-tool 失败: %v", err)), nil
			}

			return mcp.NewToolResultText(string(output)), nil
		},
	}
}
