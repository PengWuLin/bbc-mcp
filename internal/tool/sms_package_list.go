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
			mcp.WithDescription("查询某个租户的短信云套餐信息"),
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
