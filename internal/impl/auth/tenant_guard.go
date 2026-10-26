package auth

import (
	"context"
	"net/http"
	"strings"

	familyerrors "saas-identity-platform-go/internal/impl/errors"
)

type claimsKeyType struct{}

// Authenticate Bearer 守卫核心：无/坏 token → 401 + WWW-Authenticate 并返回 false；
// 合法 → claims（生成 handler 入口直接调用——oapi-codegen 无中间件挂点，rails
// include TenantGuard 的等价物是每个 protected handler 首行 s.auth(w, r)）。
func Authenticate(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	raw := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		unauthorized(w)
		return nil, false
	}
	claims, err := Verify(strings.TrimPrefix(raw, prefix))
	if err != nil {
		unauthorized(w)
		return nil, false
	}
	return claims, true
}

// Guard Bearer 守卫中间件（Authenticate + claims 进 context）。
func Guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := Authenticate(w, r)
		if !ok {
			return
		}
		next(w, r.WithContext(WithClaims(r.Context(), claims)))
	}
}

// WithClaims 把 claims 挂进 context（ClaimsFrom 的写入侧）。
func WithClaims(ctx context.Context, claims map[string]any) context.Context {
	return context.WithValue(ctx, claimsKeyType{}, claims)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
	familyerrors.Write(w, familyerrors.ErrInvalidCredentials())
}

// ClaimsFrom 取守卫注入的 claims（未过守卫时 ok=false）。
func ClaimsFrom(ctx context.Context) (map[string]any, bool) {
	claims, ok := ctx.Value(claimsKeyType{}).(map[string]any)
	return claims, ok
}

// RequirePathTenant 校验路径 {tenantId} 与 claim tenant_id 一致，不一致 403
// （镜像 rails verify_path_tenant!）。claims 必须已由 Guard 注入。
func RequirePathTenant(r *http.Request) error {
	claims, ok := ClaimsFrom(r.Context())
	if !ok {
		return familyerrors.ErrInvalidCredentials()
	}
	current, _ := claims["tenant_id"].(string)
	if r.PathValue("tenantId") != current {
		return familyerrors.ErrForbidden("forbidden")
	}
	return nil
}

// WriteError 错误落响应薄包装（测试桩与本包自用；handler 侧直接用 errors.Write）。
func WriteError(w http.ResponseWriter, err error) {
	familyerrors.Write(w, err)
}
