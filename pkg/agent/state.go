package agent

// State Agent 运行状态
type State string

const (
	StateReady    State = "ready"    // 就绪，可以开始对话
	StateRunning  State = "running"  // 运行中，正在处理请求
	StateStopping State = "stopping" // 停止中，等待当前请求完成
	StateStopped  State = "stopped"  // 已停止，不再接受请求
)
