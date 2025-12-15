// Package agent 提供通用的 AI Agent 框架
//
// Agent 是独立的一等公民，可直接使用，无需依赖 Actor 系统。
// 如需并发安全的 Actor 包装，请使用 [pkg/actor/agent] 包。
//
// # 两层 API 设计
//
// 本包提供两层 API，满足不同场景需求：
//
//   - L1: Fluent Builder (推荐) - 链式配置，IDE 友好，参见 [New]
//   - L2: Functional Options - 完全控制，动态配置，参见 [NewAgent]
//
// 完整使用示例请参考 example_test.go。
//
// # 配置加载
//
// 支持多种配置方式：
//   - [Builder.FromJSON]: JSON 字符串配置
//   - [Builder.FromYAML]: YAML 字符串配置
//   - [Builder.FromFile]: 自动识别文件格式
//   - [Builder.FromEnv]: 环境变量配置
//
// 配置文件支持 Go 模板语法：
//   - env "VAR": 获取环境变量
//   - env "VAR" "default": 带默认值
//   - {{.VAR}}: 直接访问变量 (Taskfile 风格)
//
// # Provider 自动创建
//
// 如果不手动设置 Provider，NewAgent 会根据配置自动创建：
//   - 优先使用 cfg.APIKey（可从 JSON 模板获取）
//   - 其次从环境变量探测 API Key
//
// 支持的环境变量（按优先级）：
//   - OPENAI_API_KEY
//   - ANTHROPIC_API_KEY
//   - OPENROUTER_API_KEY
//   - LLM_API_KEY
//
// # 核心组件
//
//   - [Agent]: Agent 核心实现
//   - [Builder]: Fluent Builder 构建器
//   - [Config]: Agent 配置
//   - [Result]: 对话完成结果
//
// # 扩展点
//
// 通过接口实现自定义扩展：
//   - [llm.Provider]: LLM 调用（OpenAI, Anthropic, 本地模型等）
//   - [tool.Tool]: 工具定义
//   - [tool.Registry]: 工具管理
//
// # 与 Actor 集成
//
// 如需将 Agent 包装为 Actor 以获得并发安全，请使用 [pkg/actor/agent] 包。
//
// # 包文件组织
//
//   - agent.go: Agent 核心类型定义和公开 API
//   - builder.go: Fluent Builder API
//   - options.go: 函数式选项
//   - run_blocking.go: 非流式执行引擎
//   - run_streaming.go: 流式执行引擎
//   - tool_execution.go: 工具调用执行
package agent
