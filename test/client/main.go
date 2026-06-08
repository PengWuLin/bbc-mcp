package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	serverURL := "http://192.168.31.67:9000"
	if v := os.Getenv("BBC_MCP_SERVER_URL"); v != "" {
		serverURL = v
	}
	token := os.Getenv("BBC_MCP_TOKEN")
	if token == "" {
		token = "mf8KPh6f5ma8RhOkIeYYVCK15jtAg4CLUTTlkGHc"
	}

	ctx := context.Background()

	cli, err := newSSEClient(ctx, serverURL, token)
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

	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("列出 Tools 失败: %v", err)
	}

	fmt.Printf("=== 可用 Tools (%d) ===\n", len(tools.Tools))
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	fmt.Println("=== 测试 gateway_status ===")
	testGatewayStatus(ctx, cli)
	fmt.Println()

	fmt.Println("=== 测试 device_list ===")
	testDeviceList(ctx, cli, "56626662", "", 0)
	testDeviceList(ctx, cli, "56626662", "Connect", 0)
	fmt.Println()

	fmt.Println("=== 测试 device_status ===")
	testDeviceStatus(ctx, cli, 223166)
	fmt.Println()

	if os.Getenv("BBC_MCP_TEST_RATELIMIT") == "1" {
		testRateLimit(serverURL, token)
	}
	if os.Getenv("BBC_MCP_TEST_AUTH") == "1" {
		testAuth(serverURL, token)
	}
}

func newSSEClient(_ context.Context, serverURL, token string) (*client.Client, error) {
	return client.NewSSEMCPClient(serverURL+"/sse",
		client.WithHeaders(map[string]string{
			"Authorization": "Bearer " + token,
		}),
	)
}

func testRateLimit(serverURL, token string) {
	fmt.Println("=== 测试限流 ===")
	var wg sync.WaitGroup
	results := make(chan string, 2)

	for i := range 2 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cli, err := newSSEClient(context.Background(), serverURL, token)
			if err != nil {
				results <- fmt.Sprintf("goroutine %d: 创建客户端失败: %v", idx, err)
				return
			}
			defer cli.Close()

			if err := cli.Start(context.Background()); err != nil {
				results <- fmt.Sprintf("goroutine %d: 启动失败: %v", idx, err)
				return
			}

			time.Sleep(time.Duration(idx*100) * time.Millisecond)

			_, err = cli.CallTool(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "gateway_status",
				},
			})
			if err != nil {
				results <- fmt.Sprintf("goroutine %d: 调用失败 (限流生效): %v", idx, err)
			} else {
				results <- fmt.Sprintf("goroutine %d: 调用成功", idx)
			}
		}(i)
	}

	wg.Wait()
	close(results)
	for r := range results {
		fmt.Println(r)
	}
}

func testAuth(serverURL, token string) {
	fmt.Println("=== 测试认证 ===")

	tests := []struct {
		name   string
		header string
	}{
		{"无 Authorization 头", ""},
		{"格式错误的 Authorization", "InvalidFormat token123"},
		{"无效 token", "Bearer invalid-token"},
	}

	for _, tt := range tests {
		fmt.Printf("测试: %s\n", tt.name)
		opts := []transport.ClientOption{}
		if tt.header != "" {
			opts = append(opts, client.WithHeaders(map[string]string{
				"Authorization": tt.header,
			}))
		}

		cli, err := client.NewSSEMCPClient(serverURL+"/sse", opts...)
		if err != nil {
			fmt.Printf("  创建客户端失败: %v\n", err)
			continue
		}

		err = cli.Start(context.Background())
		if err != nil {
			fmt.Printf("  连接失败 (预期): %v\n", err)
			continue
		}
		cli.Close()
		fmt.Printf("  连接成功 (未预期)\n")
	}

	fmt.Printf("测试: 有效 token\n")
	cli, err := newSSEClient(context.Background(), serverURL, token)
	if err != nil {
		fmt.Printf("  创建客户端失败: %v\n", err)
		return
	}
	if err := cli.Start(context.Background()); err != nil {
		fmt.Printf("  连接失败: %v\n", err)
		return
	}
	defer cli.Close()
	fmt.Printf("  连接成功\n")
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
