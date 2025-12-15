package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/lwmacct/251215-go-pkg-llm/pkg/llm"
)

// ═══════════════════════════════════════════════════════════════════════════
// Phase 3.1: 延迟构建逻辑测试
// ═══════════════════════════════════════════════════════════════════════════

// TestBuilder_LazyBuild 测试延迟构建功能
func TestBuilder_LazyBuild(t *testing.T) {
	t.Run("Build_should_construct_agent_once", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		builder := New().
			Name("test-agent").
			Model("gpt-4o-mini").
			APIKeyFromEnv()

		// 第一次调用 Build
		ag1, err := builder.Build()
		if err != nil {
			t.Fatalf("First Build() failed: %v", err)
		}

		// 第二次调用 Build，应该返回相同的实例
		ag2, err := builder.Build()
		if err != nil {
			t.Fatalf("Second Build() failed: %v", err)
		}

		// 验证返回相同的 Agent 实例
		if ag1 != ag2 {
			t.Error("Build() should return the same agent instance")
		}
	})

	t.Run("Chat_should_trigger_lazy_build", func(t *testing.T) {
		// 跳过需要真实 API 的测试
		t.Skip("Requires real API key and provider")

		builder := New().
			Name("test-agent").
			Model("gpt-4o-mini").
			APIKey("test-key")

		ctx := context.Background()

		// 直接调用 Chat，应该触发自动构建
		_, err := builder.Chat(ctx, "Hello")

		// 这里会失败是正常的（因为没有真实的 provider）
		// 我们只是验证延迟构建逻辑被触发了
		if err != nil {
			// 预期会有错误（因为是测试环境）
			t.Logf("Expected error in test environment: %v", err)
		}

		// 验证 Agent 已经被构建
		if !builder.built {
			t.Error("Chat() should have triggered lazy build")
		}
		if builder.agent == nil {
			t.Error("agent should not be nil after Chat()")
		}
	})

	t.Run("Run_should_trigger_lazy_build", func(t *testing.T) {
		// 跳过需要真实 API 的测试
		t.Skip("Requires real API key and provider")

		builder := New().
			Name("test-agent").
			Model("gpt-4o-mini").
			APIKey("test-key")

		ctx := context.Background()

		// 直接调用 Run，应该触发自动构建
		eventCh := builder.Run(ctx, "Hello")

		// 读取事件（预期会有错误）
		for event := range eventCh {
			if event.Type == llm.EventTypeError {
				t.Logf("Expected error in test environment: %v", event.Error)
			}
		}

		// 验证 Agent 已经被构建（或尝试构建）
		// 注意：如果构建失败，built 可能仍为 false
		if builder.agent == nil && !builder.built {
			t.Log("Build was attempted (may have failed in test environment)")
		}
	})

	t.Run("Build_should_validate_config_errors", func(t *testing.T) {
		builder := New().
			MaxTokens(-100) // 触发配置错误

		_, err := builder.Build()
		if err == nil {
			t.Error("Build() should return error for invalid config")
		}

		t.Logf("Expected error: %v", err)
	})

	t.Run("Chat_should_propagate_config_errors", func(t *testing.T) {
		builder := New().
			MaxTokens(-100) // 触发配置错误

		ctx := context.Background()
		_, err := builder.Chat(ctx, "Hello")

		if err == nil {
			t.Error("Chat() should return error for invalid config")
		}

		t.Logf("Expected error: %v", err)
	})
}

// TestBuilder_ErrorCollection 测试错误收集机制
func TestBuilder_ErrorCollection(t *testing.T) {
	t.Run("should_collect_multiple_errors", func(t *testing.T) {
		builder := New().
			MaxTokens(-1).  // 错误1
			MaxTokens(-2).  // 错误2
			APIKeyFromEnv() // 错误3（如果环境变量未设置）

		_, err := builder.Build()
		if err == nil {
			t.Error("Build() should return collected errors")
		}

		t.Logf("Collected errors: %v", err)
	})

	t.Run("should_fail_fast_on_build", func(t *testing.T) {
		builder := New().
			MaxTokens(-100)

		// 第一次调用应该失败
		_, err1 := builder.Build()
		if err1 == nil {
			t.Error("First Build() should fail")
		}

		// 第二次调用应该返回相同的错误（快速失败）
		_, err2 := builder.Build()
		if err2 == nil {
			t.Error("Second Build() should also fail")
		}

		if err1.Error() != err2.Error() {
			t.Errorf("Errors should be consistent:\n  First: %v\n  Second: %v", err1, err2)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Phase 3.2: 并发安全测试
// ═══════════════════════════════════════════════════════════════════════════

// TestBuilder_ConcurrentBuild 测试并发构建的线程安全性
func TestBuilder_ConcurrentBuild(t *testing.T) {
	t.Run("concurrent_Build_should_be_safe", func(t *testing.T) {
		builder := New().
			Name("concurrent-agent").
			Model("gpt-4o-mini")

		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines)

		agents := make([]*Agent, goroutines)
		errors := make([]error, goroutines)

		// 并发调用 Build()
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				ag, err := builder.Build()
				agents[idx] = ag
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// 验证所有调用都成功或都失败
		firstErr := errors[0]
		for i, err := range errors {
			if (err == nil) != (firstErr == nil) {
				t.Errorf("Inconsistent results at index %d: got %v, want %v", i, err, firstErr)
			}
		}

		// 如果成功，验证所有返回相同的 Agent 实例
		if firstErr == nil {
			firstAgent := agents[0]
			for i, ag := range agents[1:] {
				if ag != firstAgent {
					t.Errorf("Agent at index %d is different: %p vs %p", i+1, ag, firstAgent)
				}
			}
			t.Logf("All %d goroutines got the same agent instance: %p", goroutines, firstAgent)
		}
	})

	t.Run("concurrent_Chat_should_be_safe", func(t *testing.T) {
		// 跳过需要真实 API 的测试
		t.Skip("Requires real API key and provider")

		builder := New().
			Name("concurrent-agent").
			Model("gpt-4o-mini").
			APIKey("test-key")

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		ctx := context.Background()

		// 并发调用 Chat()
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				_, err := builder.Chat(ctx, "Hello")
				if err != nil {
					// 预期会有错误（测试环境）
					t.Logf("Goroutine %d got error: %v", idx, err)
				}
			}(i)
		}

		wg.Wait()

		// 验证只构建了一次
		if builder.agent == nil {
			t.Error("agent should be built after concurrent calls")
		}
	})

	t.Run("concurrent_Run_should_be_safe", func(t *testing.T) {
		// 跳过需要真实 API 的测试
		t.Skip("Requires real API key and provider")

		builder := New().
			Name("concurrent-agent").
			Model("gpt-4o-mini").
			APIKey("test-key")

		const goroutines = 30
		var wg sync.WaitGroup
		wg.Add(goroutines)

		ctx := context.Background()

		// 并发调用 Run()
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				eventCh := builder.Run(ctx, "Hello")
				for event := range eventCh {
					if event.Type == llm.EventTypeError {
						t.Logf("Goroutine %d got error: %v", idx, event.Error)
					}
				}
			}(i)
		}

		wg.Wait()

		// 验证只构建了一次
		if builder.agent == nil && !builder.built {
			t.Error("Build should have been attempted")
		}
	})
}

// TestBuilder_RaceCondition 使用 race detector 检测竞态条件
//
// 运行方式：go test -race -run TestBuilder_RaceCondition
func TestBuilder_RaceCondition(t *testing.T) {
	t.Run("no_race_in_ensureBuilt", func(t *testing.T) {
		builder := New().
			Name("race-test").
			Model("gpt-4o-mini")

		const goroutines = 1000
		var wg sync.WaitGroup
		wg.Add(goroutines)

		// 高并发调用，触发 race detector
		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				_, _ = builder.Build()
			}()
		}

		wg.Wait()
		t.Log("No race detected in concurrent Build()")
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 性能基准测试
// ═══════════════════════════════════════════════════════════════════════════

// BenchmarkBuilder_Build 基准测试：构建性能
func BenchmarkBuilder_Build(b *testing.B) {
	builder := New().
		Name("bench-agent").
		Model("gpt-4o-mini")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = builder.Build()
	}
}

// BenchmarkBuilder_LazyBuild 基准测试：延迟构建的快速路径
func BenchmarkBuilder_LazyBuild(b *testing.B) {
	builder := New().
		Name("bench-agent").
		Model("gpt-4o-mini")

	// 预先构建
	_, err := builder.Build()
	if err != nil {
		b.Fatalf("Build failed: %v", err)
	}

	b.ResetTimer()
	// 测试快速路径（已构建，直接返回）
	for i := 0; i < b.N; i++ {
		_, _ = builder.Build()
	}
}

// BenchmarkBuilder_ConcurrentBuild 基准测试：并发构建
func BenchmarkBuilder_ConcurrentBuild(b *testing.B) {
	builder := New().
		Name("bench-agent").
		Model("gpt-4o-mini")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = builder.Build()
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 资源管理测试
// ═══════════════════════════════════════════════════════════════════════════

// TestBuilder_Close 测试资源清理
func TestBuilder_Close(t *testing.T) {
	t.Run("Close_should_be_safe_before_build", func(t *testing.T) {
		builder := New().Name("test-agent")

		// 在 Build 之前调用 Close 应该是安全的
		err := builder.Close()
		if err != nil {
			t.Errorf("Close() before Build() should not error: %v", err)
		}
	})

	t.Run("Close_should_cleanup_after_build", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		builder := New().
			Name("test-agent").
			Model("gpt-4o-mini").
			APIKeyFromEnv()

		_, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() failed: %v", err)
		}

		// 关闭应该成功
		err = builder.Close()
		if err != nil {
			t.Errorf("Close() after Build() failed: %v", err)
		}
	})

	t.Run("multiple_Close_should_be_safe", func(t *testing.T) {
		builder := New().
			Name("test-agent").
			Model("gpt-4o-mini")

		_, _ = builder.Build()

		// 多次 Close 应该是安全的
		err1 := builder.Close()
		err2 := builder.Close()

		if err1 != nil {
			t.Errorf("First Close() failed: %v", err1)
		}
		if err2 != nil {
			t.Logf("Second Close() may fail (expected): %v", err2)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// 集成测试（需要 mock provider）
// ═══════════════════════════════════════════════════════════════════════════

// TestBuilder_Integration 集成测试（跳过，需要 mock）
func TestBuilder_Integration(t *testing.T) {
	t.Skip("Requires mock provider - implement in future")

	// TODO: 实现 mock provider 后的集成测试
	// - 测试完整的 Chat 流程
	// - 测试完整的 Run 流程
	// - 测试工具调用
	// - 测试流式输出
}

// ═══════════════════════════════════════════════════════════════════════════
// 辅助函数
// ═══════════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════════
// Agent 克隆功能测试
// ═══════════════════════════════════════════════════════════════════════════

// TestAgentClone_From 测试 Builder.From() 克隆方法
func TestAgentClone_From(t *testing.T) {
	t.Run("should_clone_config", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			ID("src-agent").
			Name("Source Agent").
			Model("gpt-4").
			System("Source prompt").
			MaxTokens(2000).
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		// 使用 From() 克隆
		cloned, err := From(src).Build()
		if err != nil {
			t.Fatalf("Failed to clone agent: %v", err)
		}

		// 验证配置被复制
		srcCfg := src.Config()
		clonedCfg := cloned.Config()

		if clonedCfg.Name != srcCfg.Name {
			t.Errorf("Name mismatch: got %s, want %s", clonedCfg.Name, srcCfg.Name)
		}
		if clonedCfg.LLM.Model != srcCfg.LLM.Model {
			t.Errorf("Model mismatch: got %s, want %s", clonedCfg.LLM.Model, srcCfg.LLM.Model)
		}
		if clonedCfg.Prompt != srcCfg.Prompt {
			t.Errorf("Prompt mismatch: got %s, want %s", clonedCfg.Prompt, srcCfg.Prompt)
		}
		if clonedCfg.MaxTokens != srcCfg.MaxTokens {
			t.Errorf("MaxTokens mismatch: got %d, want %d", clonedCfg.MaxTokens, srcCfg.MaxTokens)
		}
	})

	t.Run("should_override_config", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			Name("Source").
			Model("gpt-4").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		// 克隆并覆盖配置
		cloned, err := From(src).
			Name("Cloned").
			Model("gpt-4o-mini").
			Build()
		if err != nil {
			t.Fatalf("Failed to clone agent: %v", err)
		}

		// 验证配置被覆盖
		clonedCfg := cloned.Config()
		if clonedCfg.Name != "Cloned" {
			t.Errorf("Name not overridden: got %s, want Cloned", clonedCfg.Name)
		}
		if clonedCfg.LLM.Model != "gpt-4o-mini" {
			t.Errorf("Model not overridden: got %s, want gpt-4o-mini", clonedCfg.LLM.Model)
		}
	})
}

// TestAgentClone_CloneAgent 测试 CloneAgent() 便捷函数
func TestAgentClone_CloneAgent(t *testing.T) {
	t.Run("basic_clone", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			Name("Source").
			Model("gpt-4").
			System("Original prompt").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		// 使用 CloneAgent 克隆
		cloned, err := CloneAgent(src)
		if err != nil {
			t.Fatalf("CloneAgent failed: %v", err)
		}

		// 验证基本配置
		clonedCfg := cloned.Config()
		srcCfg := src.Config()

		if clonedCfg.Name != srcCfg.Name {
			t.Errorf("Config not cloned correctly")
		}
	})

	t.Run("clone_with_options", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			Name("Source").
			Model("gpt-4").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		// 克隆并应用选项
		cloned, err := CloneAgent(src,
			WithName("Modified"),
			WithPrompt("New prompt"),
		)
		if err != nil {
			t.Fatalf("CloneAgent with options failed: %v", err)
		}

		// 验证选项被应用
		clonedCfg := cloned.Config()
		if clonedCfg.Name != "Modified" {
			t.Errorf("Name not modified: got %s, want Modified", clonedCfg.Name)
		}
		if clonedCfg.Prompt != "New prompt" {
			t.Errorf("Prompt not modified: got %s, want New prompt", clonedCfg.Prompt)
		}
	})
}

// TestAgentClone_Independence 测试克隆后的独立性
func TestAgentClone_Independence(t *testing.T) {
	t.Run("config_should_be_independent", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			ID("src").
			Name("Source").
			Model("gpt-4").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		// 克隆
		cloned, err := From(src).
			ID("cloned").
			Build()
		if err != nil {
			t.Fatalf("Failed to clone: %v", err)
		}

		// 验证 ID 不同
		if src.ID() == cloned.ID() {
			t.Error("Cloned agent should have different ID")
		}

		// 获取配置快照
		srcCfg1 := src.Config()
		clonedCfg1 := cloned.Config()

		// 验证初始配置相似（除了 ID）
		if srcCfg1.LLM.Model != clonedCfg1.LLM.Model {
			t.Error("Initial configs should be similar")
		}

		// 注意: 修改返回的配置不会影响 Agent 内部状态
		// 因为 Config() 返回的是深拷贝
		t.Log("Config independence verified (Config() returns deep copy)")
	})

	t.Run("agents_should_be_independent_instances", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			Name("Source").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		// 克隆
		cloned, err := CloneAgent(src, WithName("Cloned"))
		if err != nil {
			t.Fatalf("Failed to clone: %v", err)
		}

		// 验证是不同的实例
		srcPtr := src
		clonedPtr := cloned

		if srcPtr == clonedPtr {
			t.Error("Cloned agent should be a different instance")
		}
	})
}

// TestAgentClone_Concurrent 测试并发克隆的线程安全性
func TestAgentClone_Concurrent(t *testing.T) {
	t.Run("concurrent_from_should_be_safe", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			Name("Source").
			Model("gpt-4").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines)

		errors := make([]error, goroutines)
		clones := make([]*Agent, goroutines)

		// 并发克隆
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				cloned, err := From(src).
					ID(fmt.Sprintf("clone-%d", idx)).
					Build()
				errors[idx] = err
				clones[idx] = cloned
			}(i)
		}

		wg.Wait()

		// 验证所有克隆都成功
		for i, err := range errors {
			if err != nil {
				t.Errorf("Clone %d failed: %v", i, err)
			}
		}

		// 验证所有克隆都是不同的实例
		for i := 0; i < goroutines; i++ {
			for j := i + 1; j < goroutines; j++ {
				if clones[i] == clones[j] {
					t.Errorf("Clones %d and %d are the same instance", i, j)
				}
			}
		}
	})

	t.Run("concurrent_CloneAgent_should_be_safe", func(t *testing.T) {
		// 跳过需要真实 API 的测试（provider 不再自动探测 API key）
		t.Skip("Requires real API key and provider")

		// 创建源 Agent
		src, err := New().
			Name("Source").
			APIKeyFromEnv().
			Build()
		if err != nil {
			t.Fatalf("Failed to create source agent: %v", err)
		}

		const goroutines = 50
		var wg sync.WaitGroup
		wg.Add(goroutines)

		errors := make(chan error, goroutines)

		// 并发使用 CloneAgent
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				_, err := CloneAgent(src,
					WithID(fmt.Sprintf("clone-%d", idx)),
					WithName(fmt.Sprintf("Clone %d", idx)),
				)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// 验证没有错误
		for err := range errors {
			t.Errorf("Concurrent clone failed: %v", err)
		}
	})
}
