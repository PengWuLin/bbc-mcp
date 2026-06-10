package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"bbc-mcp/internal/gateway"
)

func newDatacenterStatus(deps *Dependencies) ToolDefinition {
	return ToolDefinition{
		Tool: mcp.NewTool("datacenter_status",
			mcp.WithDescription("查询 minibbc 数据中心各 K8s 集群的设备连接状态。返回每个集群的 Pod 连接数（TCP ESTABLISHED）和设备数（connections/3）。\n\n参数：\n  - cluster (string, 可选): K8s 集群名称，不填则查询所有已配置集群\n\n返回值（JSON 数组）：\n  [{\"cluster\":\"bbc-shenzhen4\", \"pods\":[{\"pod_name\":\"cloudbbc-gateway-0\",\"status\":\"Running\",\"connections\":60,\"devices\":20}]}]\n  - connections: TCP ESTABLISHED 连接数\n  - devices: connections / 3（每设备维持 3 个连接）\n  - 若 Pod 非 Running 状态，error 字段会有说明；若指定 cluster 不存在，返回错误\n\n注意事项：\n  - 查询超时 30s\n  - 单次请求串行查询所有集群"),
			mcp.WithString("cluster",
				mcp.Description("K8s 集群名称（可选，不填则查询所有集群）"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			cluster := getStringArg(req, "cluster")

			if deps.Config.Gateway.Mode == "native" {
				return datacenterStatusNative(ctx, deps, cluster)
			}
			return datacenterStatusCLI(ctx, deps)
		},
	}
}

func getStringArg(req mcp.CallToolRequest, key string) string {
	args, ok := req.Params.Arguments.(map[string]any)
	if !ok {
		return ""
	}
	v, _ := args[key].(string)
	return v
}

func datacenterStatusNative(ctx context.Context, deps *Dependencies, cluster string) (*mcp.CallToolResult, error) {
	allClients := deps.K8sClients
	if len(allClients) == 0 {
		return mcp.NewToolResultError("未配置 K8s 集群"), nil
	}

	targetNames := make([]string, 0, len(allClients))
	if cluster != "" {
		if _, ok := allClients[cluster]; !ok {
			return mcp.NewToolResultError(fmt.Sprintf("集群 %s 不存在", cluster)), nil
		}
		targetNames = append(targetNames, cluster)
	} else {
		for name := range allClients {
			targetNames = append(targetNames, name)
		}
	}

	cfg := deps.Config.Gateway.Native
	var allResults []gateway.ClusterGatewayStatus
	for _, name := range targetNames {
		client := allClients[name]
		result, err := gateway.QueryGatewayStatus(ctx, client, cfg)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("查询集群 %s 失败: %v", name, err)), nil
		}
		allResults = append(allResults, *result)
	}

	output, _ := json.MarshalIndent(allResults, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func datacenterStatusCLI(ctx context.Context, deps *Dependencies) (*mcp.CallToolResult, error) {
	toolPath := deps.Config.BbcTool.Path
	timeout := time.Duration(deps.Config.BbcTool.Timeout) * time.Second

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, toolPath, "status", "gateway")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("datacenter_status: 执行命令失败，输出：%s，错误： %v", output, err)
		return mcp.NewToolResultText(string(output)), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}
