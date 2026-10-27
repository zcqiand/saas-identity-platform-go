package harness

// cors_test.go — 家族 CORS 契约（final review lab-go Critical#1 对称面；rails
// config/initializers/cors.rb rack-cors 镜像）：白名单 origin 预检 204 + ACAO 回显
// + expose Authorization；非白名单不出 CORS 头；键缺失装配期 fail-fast。

import (
	"net/http/httptest"
	"testing"
)

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101,http://localhost:5102")
	h := Assemble(nil)
	req := httptest.NewRequest("OPTIONS", "/api/me", nil)
	req.Header.Set("Origin", "http://localhost:5101")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("白名单 origin 预检期望 204，得到 %d", rec.Code)
	}
	if ao := rec.Header().Get("Access-Control-Allow-Origin"); ao != "http://localhost:5101" {
		t.Fatalf("ACAO 应回显 origin: %q", ao)
	}
	if ae := rec.Header().Get("Access-Control-Expose-Headers"); ae != "Authorization" {
		t.Fatalf("expose 应含 Authorization: %q", ae)
	}
}

func TestCORSPreflightDisallowedOriginNoACAO(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101")
	h := Assemble(nil)
	req := httptest.NewRequest("OPTIONS", "/api/me", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ao := rec.Header().Get("Access-Control-Allow-Origin"); ao != "" {
		t.Fatalf("非白名单 origin 不应得 ACAO: %q", ao)
	}
}

func TestCORSActualRequestGetsACAO(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101")
	h := Assemble(nil)
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://localhost:5101")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ao := rec.Header().Get("Access-Control-Allow-Origin"); ao != "http://localhost:5101" {
		t.Fatalf("实际请求应带 ACAO: %q", ao)
	}
}

func TestCORSMissingEnvFailFast(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "")
	defer func() {
		if recover() == nil {
			t.Fatal("空 SAAS_CORS_ALLOWED_ORIGINS 应 fail-fast panic")
		}
	}()
	Assemble(nil)
}
