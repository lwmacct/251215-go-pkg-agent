# Agent Package

<!--TOC-->

- [文件组织](#文件组织) `:19+44`
- [设计原则](#设计原则) `:63+26`
  - [1. 职责分离](#1-职责分离) `:65+9`
  - [2. 渐进式披露](#2-渐进式披露) `:74+7`
  - [3. 可测试性](#3-可测试性) `:81+8`
- [使用示例](#使用示例) `:89+33`
  - [快速开始 (L1 API)](#快速开始-l1-api) `:91+10`
  - [完全控制 (L2 API)](#完全控制-l2-api) `:101+10`
  - [配置文件](#配置文件) `:111+11`

<!--TOC-->

通用 AI Agent 框架实现。

## 文件组织

本包采用模块化文件组织，职责清晰：

```
pkg/agent/
├── agent.go            # Agent 核心类型定义和公开 API
│                       # - Agent struct 定义
│                       # - Run(), Chat(), Status(), Close() 等公开方法
│                       # - NewAgent() 构造函数
│
├── builder.go          # Fluent Builder API (L1 API)
│                       # - 链式配置方法
│                       # - JSON/YAML 配置加载
│                       # - IDE 友好的自动补全
│
├── options.go          # 函数式选项模式 (L2 API)
│                       # - WithName(), WithModel() 等选项
│                       # - 完全控制的配置方式
│
├── run_blocking.go     # 非流式执行引擎
│                       # - runLoopBlocking(): 阻塞式对话循环
│                       # - callProviderBlocking(): 非流式 Provider 调用
│
├── run_streaming.go    # 流式执行引擎
│                       # - runLoopStreaming(): 流式对话循环
│                       # - callProviderStreaming(): 流式 Provider 调用
│                       # - 实时文本增量处理
│
├── tool_execution.go   # 工具调用编排
│                       # - executeToolsWithEvents(): 工具执行和事件发送
│                       # - extractToolCalls(): 从消息中提取工具调用
│
├── helpers.go          # 内部辅助方法
│                       # - appendMessage(): 线程安全的消息管理
│                       # - buildProviderOptions(): 构建 Provider 配置
│                       # - generateAgentID(): ID 生成
│                       # - truncateString(): 字符串处理
│
├── aliases.go          # 类型别名导出
├── doc.go              # 包文档
└── example_test.go     # 使用示例
```

## 设计原则

### 1. 职责分离

每个文件专注于单一功能模块：

- **核心 API** (`agent.go`): 对外暴露的接口，不包含实现细节
- **执行引擎** (`run_*.go`): 独立的执行策略，便于扩展和测试
- **工具系统** (`tool_execution.go`): 工具编排逻辑，独立可复用
- **配置层** (`builder.go`, `options.go`): 多种配置方式，满足不同需求

### 2. 渐进式披露

提供两层 API，从简单到复杂：

- **L1: Fluent Builder** - 新手友好，IDE 自动补全
- **L2: Functional Options** - 高级用户，完全控制

### 3. 可测试性

模块化设计使得每个组件都可以独立测试：

- 执行引擎可以 mock Provider
- 工具执行可以独立测试
- 配置加载可以单元测试

## 使用示例

### 快速开始 (L1 API)

```go
ag, err := agent.New().
    Name("assistant").
    System("You are helpful").
    APIKeyFromEnv().
    Build()
```

### 完全控制 (L2 API)

```go
ag, err := agent.NewAgent(
    agent.WithName("assistant"),
    agent.WithPrompt("You are helpful"),
    agent.WithAPIKey(os.Getenv("API_KEY")),
)
```

### 配置文件

```go
// JSON
ag, err := agent.New().FromFile("config.json").Build()

// YAML
ag, err := agent.New().FromFile("config.yaml").Build()
```

完整文档请参考 `go doc github.com/lwmacct/251215-go-pkg-agent/pkg/agent`
