package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"saas-identity-platform-go/internal/impl/harness"
)

// REQ-2026-001 AC-2：家族探活契约形状 GET /health → {"ok":true,"mode":"go"}。
// /health 由 harness.Assemble 注册（基础设施端点不挂功能 ID——ktor 先例，
// trace fns 留空）。
func TestHealthShape(t *testing.T) {
	rec := httptest.NewRecorder()
	New("127.0.0.1:0", harness.Assemble(nil)).Handler.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != 200 {
		t.Fatalf("期望 200，得到 %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"mode":"go"`) {
		t.Fatalf(`期望 {"ok":true,"mode":"go"}，得到 %s`, body)
	}
}

// REQ-2026-003 Task 3：New 包装语义——handler 原样成为路由根。
func TestNewWrapsHandler(t *testing.T) {
	srv := New("127.0.0.1:5111", nil)
	if srv.Addr != "127.0.0.1:5111" || srv.Handler != nil {
		t.Fatalf("New 未按 addr/handler 包装: %+v", srv)
	}
}
