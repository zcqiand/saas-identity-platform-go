package config

import "testing"

// REQ-2026-001 AC-3：SERVER_PORT 缺失/为空必须 fail-fast 报错，禁止默认值兜底。
func TestPortMissingFailsFast(t *testing.T) {
	t.Setenv("SERVER_PORT", "")
	if _, err := Port(); err == nil {
		t.Fatal("SERVER_PORT 为空时应返回错误（fail-fast），而不是兜底默认端口")
	}
}

// REQ-2026-001 AC-1：合法端口值透传。
func TestPortValidPassthrough(t *testing.T) {
	t.Setenv("SERVER_PORT", "5111")
	port, err := Port()
	if err != nil {
		t.Fatalf("合法端口不应报错: %v", err)
	}
	if port != 5111 {
		t.Fatalf("期望 5111，得到 %d", port)
	}
}

func TestPortNonNumericRejected(t *testing.T) {
	t.Setenv("SERVER_PORT", "not-a-port")
	if _, err := Port(); err == nil {
		t.Fatal("非数字端口应返回错误")
	}
}

func TestPortOutOfRangeRejected(t *testing.T) {
	t.Setenv("SERVER_PORT", "70000")
	if _, err := Port(); err == nil {
		t.Fatal("超出 1-65535 的端口应返回错误")
	}
}

// REQ-2026-003 Task 3：DATABASE_URL 缺失必须 fail-fast（禁默认值兜底）。
func TestDatabaseURLMissingFailsFast(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := DatabaseURL(); err == nil {
		t.Fatal("DATABASE_URL 为空时应返回错误（fail-fast），而不是兜底连接串")
	}
}

func TestDatabaseURLPassthrough(t *testing.T) {
	const dsn = "postgresql://postgres:postgres@127.0.0.1:54321/saas_dev"
	t.Setenv("DATABASE_URL", dsn)
	got, err := DatabaseURL()
	if err != nil {
		t.Fatalf("合法连接串不应报错: %v", err)
	}
	if got != dsn {
		t.Fatalf("期望 %q，得到 %q", dsn, got)
	}
}
