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
			mcp.WithDescription("按设备 ID 查询设备完整信息（MySQL）和实时运行状态（Redis）。\n\n参数：\n  - id (integer, 必填): 设备 ID\n\n返回值：\n  {\"device\": {...35个字段}, \"realtime\": {\"cpu\":\"98\",\"mem\":\"41\",...}}\n  - device.type: 设备类型（1=ABOS,2=AC,3=AF,4=MIG,5=WOC,6=SG,7=EDR,8=CM,9=CG）\n  - device.status: 持久化状态码（0=未激活,1=已启用,2=离线,3=在线,4=告警,5=停用）\n  - device.device_belong: 设备归属（0=旧设备,1=安服,2=XDR,3=云图）\n  - device.zp_id: 接入服务器 ID（200=奥飞,400=深圳四区,800=深圳一区）\n  - device.pwd: 永远返回 \"***\"，密码已脱敏\n  - realtime: 实时采集数据（CPU/内存/磁盘/会话数/在线用户等 18 个字段），可能为 null（Redis 不可用时）\n  - realtime.status: 实时状态码，优先于 device.status 使用\n\n注意事项：\n  - MySQL 超时 5s，Redis 超时 3s\n  - 设备不存在返回错误\"设备不存在\"\n  - realtime 为 null 时不视为错误"),
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
