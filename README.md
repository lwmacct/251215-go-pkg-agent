# Agent Package

<!--TOC-->

- [文件组织](#文件组织) `:26+76`
- [设计原则](#设计原则) `:102+27`
  - [1. 职责分离](#1-职责分离) `:104+9`
  - [2. 渐进式披露](#2-渐进式披露) `:113+8`
  - [3. 可测试性](#3-可测试性) `:121+8`
- [使用示例](#使用示例) `:129+64`
  - [零配置 (L0 API)](#零配置-l0-api) `:131+8`
  - [快速开始 (L1 API)](#快速开始-l1-api) `:139+12`
  - [完全控制 (L2 API)](#完全控制-l2-api) `:151+11`
  - [配置文件](#配置文件) `:162+10`
  - [流式输出](#流式输出) `:172+10`
  - [添加工具](#添加工具) `:182+11`
- [Quick Start](#quick-start) `:193+14`
  - [Init Development Environment](#init-development-environment) `:195+6`
  - [List All Available Tasks](#list-all-available-tasks) `:201+6`
- [Related Links](#related-links) `:207+4`

<!--TOC-->

通用 AI Agent 框架实现。

## 文件组织

本包采用模块化文件组织，按功能分组：

```
pkg/agent/
├── 核心实现
│   ├── agent.go            # Agent 核心类型和公开 API
│   │                       # - Agent struct 定义
│   │                       # - ID(), Name(), ParentID() 身份方法
│   │                       # - Run(), Chat() 执行方法
│   │                       # - Status(), Messages(), Config() 查询方法
│   │                       # - AddTool(), RemoveTool() 工具管理
│   │                       # - Close() 生命周期
│   │
│   ├── types.go            # 核心类型定义
│   │                       # - Status: 状态快照
│   │                       # - Result: 对话结果
│   │                       # - AgentEvent: 事件流类型
│   │                       # - AgentInterface: Agent 公开接口
│   │                       # - AgentFactory: 工厂接口
│   │
│   ├── state.go            # Agent 运行状态
│   │                       # - State 类型和常量
│   │                       # - Ready/Running/Stopping/Stopped
│   │
│   └── config.go           # 配置管理 (Koanf 集成)
│                           # - Config struct 定义
│                           # - LoadConfig(): 多源加载
│                           # - 支持 YAML/JSON/环境变量/模板语法
│
├── API 层
│   ├── quick.go            # L0 快速 API
│   │                       # - Quick(): 零配置一次性调用
│   │                       # - 自动探测环境变量
│   │
│   ├── builder.go          # L1 Fluent Builder API
│   │                       # - New(): 创建 Builder
│   │                       # - 链式配置方法
│   │                       # - FromFile(), FromEnv() 配置加载
│   │                       # - Build(), Chat(), Run() 构建和执行
│   │
│   └── options.go          # L2 函数式选项 API
│                           # - NewAgent(): 创建 Agent
│                           # - With*() 系列选项函数
│                           # - CloneAgent(): 克隆 Agent
│
├── 执行引擎
│   ├── run_blocking.go     # 非流式执行引擎
│   │                       # - runLoopBlocking(): 阻塞式对话循环
│   │                       # - callProviderBlocking(): 非流式调用
│   │
│   ├── run_streaming.go    # 流式执行引擎
│   │                       # - runLoopStreaming(): 流式对话循环
│   │                       # - callProviderStreaming(): 流式调用
│   │                       # - 实时文本增量处理
│   │
│   └── tool_execution.go   # 工具调用编排
│                           # - executeToolsWithEvents(): 工具执行
│                           # - 支持重试和 panic recovery
│
├── 工具
│   ├── helpers.go          # 内部辅助方法
│   │                       # - appendMessage(): 线程安全消息管理
│   │                       # - buildProviderOptions(): 构建配置
│   │                       # - generateAgentID(): ID 生成
│   │
│   └── retry.go            # 重试机制
│                           # - RetryConfig: 重试配置
│                           # - retryWithBackoff(): 指数退避算法
│
└── 文档
    ├── doc.go              # 包文档
    └── example_test.go     # 使用示例
```

## 设计原则

### 1. 职责分离

每个文件专注于单一功能模块：

- **核心实现** (`agent.go`, `types.go`, `state.go`, `config.go`): 类型定义和核心逻辑
- **API 层** (`quick.go`, `builder.go`, `options.go`): 三种风格的用户接口
- **执行引擎** (`run_*.go`, `tool_execution.go`): 独立的执行策略
- **工具** (`helpers.go`, `retry.go`): 可复用的工具函数

### 2. 渐进式披露

提供三层 API，从简单到复杂：

- **L0: Quick API** - 零配置，一行代码调用
- **L1: Fluent Builder** - 链式配置，IDE 友好
- **L2: Functional Options** - 完全控制，动态配置

### 3. 可测试性

模块化设计使得每个组件都可以独立测试：

- 执行引擎可以 mock Provider
- 工具执行可以独立测试
- 配置加载可以单元测试

## 使用示例

### 零配置 (L0 API)

```go
// 自动从环境变量探测配置
result, err := agent.Quick(ctx, "翻译成法语: Hello")
fmt.Println(result.Text)
```

### 快速开始 (L1 API)

```go
ag, err := agent.New().
    Name("assistant").
    System("You are helpful").
    APIKeyFromEnv().
    Build()

result, _ := ag.Chat(ctx, "Hello")
```

### 完全控制 (L2 API)

```go
ag, err := agent.NewAgent(
    agent.WithName("assistant"),
    agent.WithPrompt("You are helpful"),
    agent.WithAPIKey(os.Getenv("API_KEY")),
    agent.WithModel("anthropic/claude-sonnet-4"),
)
```

### 配置文件

```go
// YAML 配置
ag, err := agent.New().FromFile("agent.yaml").Build()

// 环境变量
ag, err := agent.New().FromEnv("AGENT_").Build()
```

### 流式输出

```go
for event := range ag.Run(ctx, "写一首诗", agent.WithStreaming(true)) {
    if event.Type == llm.EventTypeText {
        fmt.Print(event.Text)  // 实时输出
    }
}
```

### 添加工具

```go
ag, err := agent.New().
    Name("tool-agent").
    Tools(searchTool, calcTool).
    Build()
```

完整文档请参考 `go doc github.com/lwmacct/251215-go-pkg-agent/pkg/agent`

## Quick Start

### Init Development Environment

```shell
pre-commit install
```

### List All Available Tasks

```shell
task -a
```

## Related Links

- Use [Taskfile](https://taskfile.dev) to manage the project's CLI
- Use [Pre-commit](https://pre-commit.com/) to manage and maintain multi-language pre-commit hooks
