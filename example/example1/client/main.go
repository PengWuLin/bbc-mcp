package main

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"log"
)

func main() {
	ctx := context.Background()

	// 创建 SSE MCP 客户端
	cli, err := client.NewSSEMCPClient("http://localhost:9000/sse")
	if err != nil {
		log.Fatalf("Failed to create SSE MCP client: %v", err)
	}
	defer cli.Close()

	// 启动 SSE 连接
	if err := cli.Start(ctx); err != nil {
		log.Fatalf("Failed to start SSE MCP client: %v", err)
	}

	// 初始化客户端
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "example1-client",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	initResult, err := cli.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Printf("Connected to server: %s v%s\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	fmt.Printf("Protocol version: %s\n\n", initResult.ProtocolVersion)

	// 1. 列出工具
	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("=== Tools (%d) ===\n", len(tools.Tools))
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// 2. 测试计算器工具
	fmt.Println("=== Testing calculate tool ===")
	testCalculate(ctx, cli, "add", 10, 5)
	testCalculate(ctx, cli, "subtract", 10, 5)
	testCalculate(ctx, cli, "multiply", 10, 5)
	testCalculate(ctx, cli, "divide", 10, 5)
	fmt.Println()

	// 3. 列出资源
	resources, err := cli.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		log.Fatalf("Failed to list resources: %v", err)
	}

	fmt.Printf("=== Resources (%d) ===\n", len(resources.Resources))
	for _, resource := range resources.Resources {
		fmt.Printf("- %s: %s (MIME: %s)\n", resource.URI, resource.Name, resource.MIMEType)
	}
	fmt.Println()

	// 4. 读取 README 资源
	if len(resources.Resources) > 0 {
		fmt.Println("=== Reading resource ===")
		uri := resources.Resources[0].URI
		readResult, err := cli.ReadResource(ctx, mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: uri,
			},
		})
		if err != nil {
			log.Printf("Failed to read resource: %v\n", err)
		} else {
			for _, content := range readResult.Contents {
				if tc, ok := content.(mcp.TextResourceContents); ok {
					fmt.Printf("Content from %s:\n%s\n", tc.URI, tc.Text)
				}
			}
		}
		fmt.Println()
	}

	// 5. 列出提示词
	prompts, err := cli.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		log.Fatalf("Failed to list prompts: %v", err)
	}

	fmt.Printf("=== Prompts (%d) ===\n", len(prompts.Prompts))
	for _, prompt := range prompts.Prompts {
		fmt.Printf("- %s: %s\n", prompt.Name, prompt.Description)
	}
	fmt.Println()

	// 6. 获取问候提示词
	if len(prompts.Prompts) > 0 {
		fmt.Println("=== Getting prompt ===")
		getPromptResult, err := cli.GetPrompt(ctx, mcp.GetPromptRequest{
			Params: mcp.GetPromptParams{
				Name:      prompts.Prompts[0].Name,
				Arguments: map[string]string{"name": "Claude"},
			},
		})
		if err != nil {
			log.Printf("Failed to get prompt: %v\n", err)
		} else {
			fmt.Printf("Description: %s\n", getPromptResult.Description)
			for _, msg := range getPromptResult.Messages {
				fmt.Printf("Message (Role: %s): %v\n", msg.Role, msg.Content)
			}
		}
	}
}

func testCalculate(ctx context.Context, cli *client.Client, operation string, x, y float64) {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "calculate",
			Arguments: map[string]any{
				"operation": operation,
				"x":         x,
				"y":         y,
			},
		},
	}

	result, err := cli.CallTool(ctx, request)
	if err != nil {
		log.Printf("Error calling calculate(%s, %f, %f): %v\n", operation, x, y, err)
		return
	}

	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			fmt.Printf("%g %s %g = %s\n", x, operation, y, c.Text)
		}
	}
}
