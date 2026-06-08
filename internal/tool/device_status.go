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

func newDeviceStatus(deps *Dependencies) ToolDefinition {
	repo := repository.NewDeviceRepository(deps.DB, deps.Redis)

	return ToolDefinition{
		Tool: mcp.NewTool("device_status",
			mcp.WithDescription("查询某个设备的详细信息和实时连接状态"),
			mcp.WithInteger("id",
				mcp.Required(),
				mcp.Description("设备ID"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := req.Params.Arguments.(map[string]interface{})

			idFloat, ok := args["id"].(float64)
			if !ok {
				return mcp.NewToolResultError("参数错误: id 必须为整数"), nil
			}
			id := int(idFloat)

			device, err := repo.GetByID(ctx, id)
			if err != nil {
				log.Printf("device_status: 查询失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
			}
			if device == nil {
				return mcp.NewToolResultError("设备不存在"), nil
			}

			var realtime map[string]string

			// 从 Redis 获取基础实时信息
			basicInfo, basicErr := repo.GetDeviceBasicInfo(ctx, id)
			if basicErr == nil {
				realtime = basicInfo

				// 合并状态信息
				statusInfo, statusErr := repo.GetDeviceOnlineStatus(ctx, id)
				if statusErr == nil {
					for k, v := range statusInfo {
						realtime[k] = v
					}
				}
			}

			result := types.DeviceStatusResult{
				Device:   types.SanitizeDevice(device),
				Realtime: realtime,
			}

			jsonBytes, err := json.Marshal(result)
			if err != nil {
				log.Printf("device_status: JSON序列化失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("序列化失败: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}
