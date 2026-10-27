// cors.go — 家族 CORS 契约（lab-go final review Critical#1 对称补齐；rails
// config/initializers/cors.rb rack-cors 镜像）：origin 白名单读
// SAAS_CORS_ALLOWED_ORIGINS（逗号分隔，值=saas 前端 dev origin 5101/5102/5103/5108），
// fail-fast 无默认兜底（suite-hard-rules §1）。语义：origin 精确匹配回显 ACAO +
// expose Authorization；预检 OPTIONS → 204；非白名单 origin 不出 CORS 头（请求照常
// 处理，由浏览器拦——rack-cors 默认）。

package harness

import (
	"net/http"
	"os"
	"strings"
)

// corsAllowedOrigins env 白名单解析（缺失/空 → panic fail-fast；纯空白段剔除）。
func corsAllowedOrigins() map[string]bool {
	raw := os.Getenv("SAAS_CORS_ALLOWED_ORIGINS")
	if strings.TrimSpace(raw) == "" {
		panic("SAAS_CORS_ALLOWED_ORIGINS env required (comma-separated origin allowlist)")
	}
	out := map[string]bool{}
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out[o] = true
		}
	}
	return out
}

// withCORS CORS 中间件（Assemble 最外层；白名单装配期解析一次）。
func withCORS(allowed map[string]bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Expose-Headers", "Authorization")
			if r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods",
					"GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
