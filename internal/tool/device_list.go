package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"

	"bbc-mcp/internal/repository"
	"bbc-mcp/internal/types"
)

func newDeviceList(deps *Dependencies) ToolDefinition {
	repo := repository.NewDeviceRepository(deps.DB, deps.Redis)

	return ToolDefinition{
		Tool: mcp.NewTool("device_list",
			mcp.WithDescription("查询某个租户的设备列表，支持按名称筛选和分页"),
			mcp.WithString("ccode",
				mcp.Required(),
				mcp.Description("租户ID（公司代码）"),
			),
			mcp.WithString("name",
				mcp.Description("设备名称（可选，模糊匹配）"),
			),
			mcp.WithInteger("offset",
				mcp.Required(),
				mcp.Description("查询偏移量"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := req.Params.Arguments.(map[string]interface{})

			ccode, ok := args["ccode"].(string)
			if !ok || ccode == "" {
				return mcp.NewToolResultError("参数错误: ccode 不能为空"), nil
			}

			var name string
			if n, ok := args["name"].(string); ok {
				name = n
			}

			offset := 0
			if o, ok := args["offset"].(float64); ok {
				offset = int(o)
			}
			if offset < 0 {
				offset = 0
			}

			devices, err := repo.ListByCcode(ctx, ccode, name, offset)
			if err != nil {
				log.Printf("device_list: 查询失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
			}

			result := types.DeviceListResult{
				Total:   true,
				Devices: devices,
				Limit:   20,
				Offset:  offset,
			}

			jsonBytes, err := json.Marshal(result)
			if err != nil {
				log.Printf("device_list: JSON序列化失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("序列化失败: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}
