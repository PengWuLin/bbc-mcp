package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	serverURL := "http://127.0.0.1:9000"
	if v := os.Getenv("BBC_MCP_SERVER_URL"); v != "" {
		serverURL = v
	}

	ctx := context.Background()

	cli, err := client.NewSSEMCPClient(serverURL + "/sse")
	if err != nil {
		log.Fatalf("创建 SSE MCP 客户端失败: %v", err)
	}
	defer cli.Close()

	if err := cli.Start(ctx); err != nil {
		log.Fatalf("启动 SSE 连接失败: %v", err)
	}

	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "bbc-mcp-test-client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	initResult, err := cli.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("初始化客户端失败: %v", err)
	}
	fmt.Printf("Connected to server: %s v%s\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	fmt.Printf("Protocol version: %s\n\n", initResult.ProtocolVersion)

	// 列出所有可用 Tools
	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("列出 Tools 失败: %v", err)
	}

	fmt.Printf("=== 可用 Tools (%d) ===\n", len(tools.Tools))
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// 测试 gateway_status
	fmt.Println("=== 测试 gateway_status ===")
	testGatewayStatus(ctx, cli)
	fmt.Println()

	// 测试 device_list
	fmt.Println("=== 测试 device_list ===")
	testDeviceList(ctx, cli, "56626662", "", 0)        // 不带 name
	testDeviceList(ctx, cli, "56626662", "Connect", 0) // 带 name 模糊搜索
	fmt.Println()

	// 测试 device_status
	fmt.Println("=== 测试 device_status ===")
	testDeviceStatus(ctx, cli, 223166)
	fmt.Println()
}

func testGatewayStatus(ctx context.Context, cli *client.Client) {
	result, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "gateway_status",
		},
	})
	if err != nil {
		log.Printf("gateway_status 调用失败: %v", err)
		return
	}

	printResult(result)
}

func testDeviceList(ctx context.Context, cli *client.Client, ccode, name string, offset int) {
	args := map[string]any{
		"ccode":  ccode,
		"offset": float64(offset),
	}
	if name != "" {
		args["name"] = name
	}

	result, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "device_list",
			Arguments: args,
		},
	})
	if err != nil {
		log.Printf("device_list 调用失败 (ccode=%s, name=%q): %v", ccode, name, err)
		return
	}

	fmt.Printf("ccode=%s name=%q:\n", ccode, name)
	printPrettyJSON(result)
}

func testDeviceStatus(ctx context.Context, cli *client.Client, id int) {
	result, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "device_status",
			Arguments: map[string]any{
				"id": float64(id),
			},
		},
	})
	if err != nil {
		log.Printf("device_status 调用失败 (id=%d): %v", id, err)
		return
	}

	fmt.Printf("id=%d:\n", id)
	printPrettyJSON(result)
}

func printResult(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			fmt.Println(c.Text)
		}
	}
}

func printPrettyJSON(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(c.Text), "", "  "); err != nil {
				fmt.Println(c.Text)
			} else {
				fmt.Println(buf.String())
			}
		}
	}
}
