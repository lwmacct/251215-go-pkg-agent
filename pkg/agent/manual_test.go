package agent_test

import (
	"os"
	"testing"

	"github.com/lwmacct/251207-go-pkg-cfgm/pkg/cfgm"
	"github.com/lwmacct/251215-go-pkg-agent/pkg/agent"
)

// TestLoginReal 实际登录测试。
//
// 手动运行:
//
// MANUAL=1 go test -v -run Test$ ./pkg/agent/
func Test(t *testing.T) {
	if os.Getenv("MANUAL") == "" {
		t.Skip()
	}

	cfg := cfgm.MustLoad(agent.DefaultConfig())

	// 打印完整配置
	t.Logf("Loaded config:\n%s", cfgm.MarshalYAML(cfg))

	t.Log("测试完成")
}
