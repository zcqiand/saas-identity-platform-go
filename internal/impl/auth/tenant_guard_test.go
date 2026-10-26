package auth

// tenant_guard_test.go — Bearer 守卫 + 路径租户一致性（镜像 rails tenant_guard concern）：
// 无/坏 Bearer → 401 + WWW-Authenticate: Bearer realm="api"；
// 路径 tenantId ≠ claim → 403 FORBIDDEN；claims 进 context。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuardMissingBearer401(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	Guard(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("无 Bearer 不应到 handler")
	}).ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("期望 401，得到 %d", rec.Code)
	}
	if wa := rec.Header().Get("WWW-Authenticate"); wa != `Bearer realm="api"` {
		t.Fatalf("WWW-Authenticate 不符: %q", wa)
	}
}

func TestGuardBadToken401(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	Guard(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("坏 token 不应到 handler")
	}).ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("期望 401，得到 %d", rec.Code)
	}
}

func TestGuardPassesClaimsViaContext(t *testing.T) {
	withJWTEnv(t)
	tok, err := Encode(map[string]any{"sub": "u1", "tenant_id": "t1"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	reached := false
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	Guard(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		claims, ok := ClaimsFrom(r.Context())
		if !ok || claims["sub"] != "u1" || claims["tenant_id"] != "t1" {
			t.Fatalf("claims 未进 context 或值不符: %v", claims)
		}
	}).ServeHTTP(rec, req)
	if !reached || rec.Code != 200 {
		t.Fatalf("合法 token 应放行: reached=%v code=%d", reached, rec.Code)
	}
}

func TestPathTenantMismatch403(t *testing.T) {
	withJWTEnv(t)
	tok, _ := Encode(map[string]any{"sub": "u1", "tenant_id": "t1"})

	// Guard→RequirePathTenant 全链：mux 已匹配路径参数由 SetPathValue 桩模拟
	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if err := RequirePathTenant(r); err != nil {
			WriteError(w, err)
			return
		}
		w.WriteHeader(200)
	}
	patched := Guard(withPathValue(handler, "tenantId", "t-other"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenants/t-other/members", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	patched(rec, req)
	if rec.Code != 403 {
		t.Fatalf("路径租户不符应 403，得到 %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "FORBIDDEN") {
		t.Fatalf("403 体应含 FORBIDDEN：%s", rec.Body.String())
	}
}

func TestPathTenantMatchPasses(t *testing.T) {
	withJWTEnv(t)
	tok, _ := Encode(map[string]any{"sub": "u1", "tenant_id": "t1"})
	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		if err := RequirePathTenant(r); err != nil {
			WriteError(w, err)
			return
		}
		w.WriteHeader(200)
	}
	patched := Guard(withPathValue(handler, "tenantId", "t1"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/tenants/t1/members", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	patched(rec, req)
	if rec.Code != 200 {
		t.Fatalf("租户一致应放行，得到 %d: %s", rec.Code, rec.Body.String())
	}
}

func TestClaimsFromEmptyContext(t *testing.T) {
	if _, ok := ClaimsFrom(context.Background()); ok {
		t.Fatal("空 context 不应有 claims")
	}
}

// withPathValue 模拟 ServeMux pattern 绑定后的 PathValue（Go 1.22 SetPathValue）。
func withPathValue(h http.HandlerFunc, key, val string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue(key, val)
		h(w, r)
	}
}
