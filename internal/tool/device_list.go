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
			mcp.WithDescription("按租户 ID 查询设备列表，支持设备名称模糊搜索和分页。\n\n参数：\n  - ccode (string, 必填): 租户 ID（公司代码），不能为空\n  - name (string, 可选): 设备名称，支持模糊匹配（LIKE %name%）\n  - offset (integer, 必填): 分页偏移量，>=0，负数自动置 0\n\n返回值：\n  {\"total\":true, \"limit\":20, \"offset\":0, \"devices\":[{\"id\":260860,\"name\":\"sz1_af\",\"zp_id\":400,\"status\":3}]}\n  - limit: 固定每页 20 条，不可覆盖\n  - zp_id: 接入服务器 ID（200=奥飞, 400=深圳四区, 800=深圳一区）\n  - status: 设备状态码（0=未激活, 1=已启用, 2=离线, 3=在线, 4=告警, 5=停用）\n  - type 字段不返回，查设备类型请用 device_status\n\n注意事项：\n  - 超时 5s\n  - 租户无设备时返回空数组 []"),
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
