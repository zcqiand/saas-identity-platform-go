package auth

import (
	"context"
	"net/http"
	"strings"

	familyerrors "saas-identity-platform-go/internal/impl/errors"
)

type claimsKeyType struct{}

// Guard Bearer 守卫中间件：无/坏 token → 401 + WWW-Authenticate；合法 → claims 进
// context 后放行（镜像 rails tenant_guard concern 的 token 面）。
func Guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(raw, prefix) {
			unauthorized(w)
			return
		}
		claims, err := Verify(strings.TrimPrefix(raw, prefix))
		if err != nil {
			unauthorized(w)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), claimsKeyType{}, claims)))
	}
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
