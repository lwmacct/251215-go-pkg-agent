package agent_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm/provider/localmock"
	"github.com/lwmacct/251215-go-pkg-tool/pkg/tool"
)

// ═══════════════════════════════════════════════════════════════════════════
// 工具定义示例
// ═══════════════════════════════════════════════════════════════════════════

// SearchInput 搜索工具输入
type SearchInput struct {
	Query string `json:"query" jsonschema:"搜索关键词"`
}

// CalcInput 计算器输入
type CalcInput struct {
	A  int    `json:"a" jsonschema:"第一个数"`
	B  int    `json:"b" jsonschema:"第二个数"`
	Op string `json:"op" jsonschema:"运算符"`
}

// Example_basic 展示 L1 Fluent Builder API 的基本用法
//
// Fluent Builder 是推荐的 API，提供链式配置和良好的 IDE 自动补全。
func Example_basic() {
	// 创建 mock provider（实际使用时会自动从环境变量探测）
	provider := localmock.New(localmock.WithResponse("Hello! I'm your assistant."))
	defer func() { _ = provider.Close() }()

	// L1: Fluent Builder API
	ag, err := agent.New().
		Name("assistant").
		System("You are helpful").
		Provider(provider).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func() { _ = ag.Close() }()

	fmt.Println("Agent created:", ag.Name())
	// Output: Agent created: assistant
}

// Example_newAgent 展示 L2 Functional Options API
//
// Functional Options 提供完全控制，适合需要动态配置的场景。
func Example_newAgent() {
	provider := localmock.New(localmock.WithResponse("Hello!"))
	defer func() { _ = provider.Close() }()

	// L2: Functional Options API
	ag, err := agent.NewAgent(
		agent.WithName("assistant"),
		agent.WithPrompt("You are helpful"),
		agent.WithProvider(provider),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func() { _ = ag.Close() }()

	fmt.Println("Agent:", ag.Name())
	// Output: Agent: assistant
}

// Example_builderChat 展示使用 Builder 直接对话
func Example_builderChat() {
	provider := localmock.New(localmock.WithResponse("2"))
	defer func() { _ = provider.Close() }()

	// Builder 可以直接调用 Chat，无需显式 Build
	result, err := agent.New().
		Name("calculator").
		System("Answer math questions concisely").
		Provider(provider).
		Chat(context.Background(), "1+1=?")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(result.Text)
	// Output: 2
}

// Example_builderFromFile 展示从配置文件加载
func Example_builderFromFile() {
	provider := localmock.New(localmock.WithResponse("Configured!"))
	defer func() { _ = provider.Close() }()

	// 从 YAML 配置文件加载（支持 JSON/YAML 格式，支持模板语法）
	ag, err := agent.New().
		FromFile("testdata/agent.yaml").
		Provider(provider).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func() { _ = ag.Close() }()

	fmt.Println("Name:", ag.Name())
	// Output: Name: yaml-assistant
}

// Example_builderToYAML 展示导出配置为 YAML
func Example_builderToYAML() {
	// 构建配置并导出为 YAML
	yamlBytes := agent.New().
		Name("export-test").
		Model("gpt-4").
		ToYAML()

	// 验证包含关键字段
	yamlStr := string(yamlBytes)
	hasName := strings.Contains(yamlStr, "name:")
	hasModel := strings.Contains(yamlStr, "model:")

	fmt.Println("Has name:", hasName)
	fmt.Println("Has model:", hasModel)
	// Output:
	// Has name: true
	// Has model: true
}

// Example_builderTools 展示添加工具
func Example_builderTools() {
	provider := localmock.New(localmock.WithResponse("Tool added"))
	defer func() { _ = provider.Close() }()

	// 使用 tool.Func 快捷定义工具
	searchTool := tool.Func("search", "搜索",
		func(ctx context.Context, in SearchInput) (string, error) {
			return "Found: " + in.Query, nil
		})

	calcTool := tool.Func("calculator", "计算器",
		func(ctx context.Context, in CalcInput) (int, error) {
			switch in.Op {
			case "+":
				return in.A + in.B, nil
			default:
				return 0, nil
			}
		})

	// 创建带工具的 Agent
	ag, err := agent.New().
		Name("tool-agent").
		Tools(searchTool, calcTool).
		Provider(provider).
		Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer func() { _ = ag.Close() }()

	// 获取工具列表
	names := ag.ToolRegistry().Names()
	fmt.Println("Tools:", names)
	// Output: Tools: [search calculator]
}

// Example_from 展示从现有 Agent 创建变体
func Example_from() {
	provider := localmock.New(localmock.WithResponse("Variant"))
	defer func() { _ = provider.Close() }()

	// 创建基础 Agent
	base, _ := agent.New().
		Name("base").
		Model("gpt-4").
		Provider(provider).
		Build()
	defer func() { _ = base.Close() }()

	// 从基础 Agent 创建变体（需要重新设置 Provider）
	variant, _ := agent.From(base).
		Name("variant").
		Provider(provider).
		Build()
	defer func() { _ = variant.Close() }()

	fmt.Println("Base:", base.Name())
	fmt.Println("Variant:", variant.Name())
	// Output:
	// Base: base
	// Variant: variant
}

// Example_cloneAgent 展示克隆 Agent
func Example_cloneAgent() {
	provider := localmock.New(localmock.WithResponse("Cloned"))
	defer func() { _ = provider.Close() }()

	// 创建原始 Agent
	original, _ := agent.NewAgent(
		agent.WithName("original"),
		agent.WithPrompt("Original prompt"),
		agent.WithProvider(provider),
	)
	defer func() { _ = original.Close() }()

	// 克隆并修改（需要重新设置 Provider）
	cloned, _ := agent.CloneAgent(original,
		agent.WithName("cloned"),
		agent.WithProvider(provider),
	)
	defer func() { _ = cloned.Close() }()

	fmt.Println("Original:", original.Name())
	fmt.Println("Cloned:", cloned.Name())
	// Output:
	// Original: original
	// Cloned: cloned
}

// Example_streaming 展示流式执行
func Example_streaming() {
	// 使用 WithResponses 模拟流式响应
	provider := localmock.New(localmock.WithResponse("Stream response"))
	defer func() { _ = provider.Close() }()

	ag, _ := agent.New().
		Name("streamer").
		Provider(provider).
		Build()
	defer func() { _ = ag.Close() }()

	// 使用 Run 获取事件流
	ctx := context.Background()
	var finalText string
	for event := range ag.Run(ctx, "Hello") {
		if event.Result != nil {
			finalText = event.Result.Text
		}
	}

	fmt.Println(finalText)
	// Output: Stream response
}

// Example_multiTurn 展示多轮对话
func Example_multiTurn() {
	provider := localmock.New(localmock.WithResponses(
		"Hi there!",
		"I'm doing great, thanks!",
	))
	defer func() { _ = provider.Close() }()

	ag, _ := agent.New().
		Name("chat-agent").
		Provider(provider).
		Build()
	defer func() { _ = ag.Close() }()

	ctx := context.Background()

	// 第一轮对话
	result1, _ := ag.Chat(ctx, "Hello")
	fmt.Println("Turn 1:", result1.Text)

	// 第二轮对话（保持上下文）
	result2, _ := ag.Chat(ctx, "How are you?")
	fmt.Println("Turn 2:", result2.Text)

	// Output:
	// Turn 1: Hi there!
	// Turn 2: I'm doing great, thanks!
}
