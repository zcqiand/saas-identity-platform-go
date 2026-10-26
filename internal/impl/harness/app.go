// Package harness — HTTP 装配（REQ-2026-003）。
//
// Assemble 把生成路由（genapi.HandlerFromMux，逐 op 注册到同一 mux）与基础设施
// 端点（GET /health，家族探活契约 {"ok":true,"mode":"go"}，不进功能树不动树
// ——ktor 先例）装到同一个 ServeMux。db 允许 nil（路由壳阶段；各域实现接管后
// 经 service 触库）。未命中路径走 ServeMux 默认 404——契约外无路由（硬规则 §4）。
package harness

import (
	"net/http"

	"gorm.io/gorm"

	genapi "saas-identity-platform-go/internal/gen/api"
	apiserver "saas-identity-platform-go/internal/impl/api"
)

// Assemble 返回装配好的根 ServeMux。db 仅透传给 api Server（壳阶段可为 nil）。
func Assemble(db *gorm.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	_ = genapi.HandlerFromMux(apiserver.NewServer(), mux)
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"mode":"go"}`))
}
