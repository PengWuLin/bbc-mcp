package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"bbc-mcp/internal/repository"
)

func newSMSPackageRenew(deps *Dependencies) ToolDefinition {
	return ToolDefinition{
		Tool: mcp.NewTool("sms_package_renew",
			mcp.WithDescription("对指定租户的短信套餐进行续期，续期时长为一年。\n\n参数：\n  - id (integer, 必填): 套餐 ID\n  - ccode (string, 必填): 租户 ID（公司代码）\n\n返回值：\n  成功时返回 {\"status\":\"success\",\"id\":2670,\"expire_time\":\"2027-06-18 23:59:00\"}\n  失败时返回错误信息\n\n注意事项：\n  - 短信云数据库未配置时返回错误\"短信云数据库未配置\"\n  - 套餐不存在时返回错误"),
			mcp.WithInteger("id",
				mcp.Required(),
				mcp.Description("套餐ID"),
			),
			mcp.WithString("ccode",
				mcp.Required(),
				mcp.Description("租户ID（公司代码）"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if deps.SMSDB == nil {
				return mcp.NewToolResultError("短信云数据库未配置"), nil
			}

			args, _ := req.Params.Arguments.(map[string]interface{})

			idFloat, ok := args["id"].(float64)
			if !ok {
				return mcp.NewToolResultError("参数错误: id 必须为整数"), nil
			}
			id := int(idFloat)

			ccode, ok := args["ccode"].(string)
			if !ok || ccode == "" {
				return mcp.NewToolResultError("参数错误: ccode 必须为非空字符串"), nil
			}

			repo := repository.NewSMSPackageRepository(deps.SMSDB, deps.SMSRedis)
			if err := repo.RenewExpireTime(ctx, id, ccode); err != nil {
				log.Printf("sms_package_renew: 续期失败: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("续期失败: %v", err)), nil
			}

			expireTime := time.Now().AddDate(1, 0, 0).Format("2006-01-02") + " 23:59:00"
			result := map[string]interface{}{
				"status":      "success",
				"id":          id,
				"expire_time": expireTime,
			}
			jsonBytes, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	}
}
