package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"

	"bbc-mcp/internal/repository"
)

func newSMSPackageList(deps *Dependencies) ToolDefinition {
	return ToolDefinition{
		Tool: mcp.NewTool("sms_package_list",
			mcp.WithDescription("按租户 ID 查询短信云套餐信息。\n\n参数：\n  - corp_id (integer, 必填): 租户 ID\n\n返回值（JSON 数组）：\n  [{\"id\":1292,\"corp_id\":26912728,\"order_id\":\"201905178888\",\"pkg_type\":1,\"pkg_name\":\"50条短信套餐\",\n    \"pkg_total_num\":50,\"pkg_avail_num\":0,\"pkg_price\":2.4,\n    \"package_desc\":\"套餐允许发送50条短信\",\"purchase_time\":\"...\",\"expire_time\":\"...\"}]\n  - pkg_total_num: 套餐总短信条数\n  - pkg_avail_num: 剩余可用条数\n  - expire_time: 套餐过期时间\n\n注意事项：\n  - 超时 5s\n  - 短信云数据库未配置时返回错误\"短信云数据库未配置\"\n  - 租户无套餐时返回空数组 []"),
			mcp.WithInteger("corp_id",
				mcp.Required(),
				mcp.Description("租户ID"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if deps.SMSDB == nil {
				return mcp.NewToolResultError("短信云数据库未配置"), nil
			}

			args, _ := req.Params.Arguments.(map[string]interface{})
			idFloat, ok := args["corp_id"].(float64)
			if !ok {
				return mcp.NewToolResultError("参数错误: corp_id 必须为整数"), nil
			}
			corpID := int(idFloat)

			repo := repository.NewSMSPackageRepository(deps.SMSDB)
			packages, err := repo.QueryByCorpID(ctx, corpID)
			if err != nil {
				log.Printf("sms_package_list: 查询失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("查询失败: %v", err)), nil
			}

			jsonBytes, _ := json.Marshal(packages)
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}
