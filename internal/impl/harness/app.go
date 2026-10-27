// Package harness — HTTP 装配（REQ-2026-003）。
//
// Assemble 把生成路由（genapi.HandlerFromMux，逐 op 注册到同一 mux）与基础设施
// 端点（GET /health，家族探活契约 {"ok":true,"mode":"go"}，不进功能树不动树
// ——ktor 先例；/api-docs Swagger 文档面，rails routes.rb 同类基础设施）装到
// 同一个 ServeMux。db 允许 nil（路由壳阶段；各域实现接管后经 service 触库）。
// 未命中路径走 ServeMux 默认 404——契约外无路由（硬规则 §4）。
package harness

import (
	"embed"
	"io/fs"
	"net/http"

	"gorm.io/gorm"

	genapi "saas-identity-platform-go/internal/gen/api"
	apiserver "saas-identity-platform-go/internal/impl/api"
)

//go:embed docs/api-docs
var docsFaceFS embed.FS

// Assemble 返回装配好的根 ServeMux。db 仅透传给 api Server（壳阶段可为 nil）。
func Assemble(db *gorm.DB) http.Handler {
	// CORS 白名单中间件（rails rack-cors 镜像；SAAS_CORS_ALLOWED_ORIGINS fail-fast）
	return withCORS(corsAllowedOrigins(), assembleMux(db))
}

// assembleMux 裸 mux 装配（无 CORS 壳；route-parity 测试用 .Handler(pattern) 查表）。
func assembleMux(db *gorm.DB) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mountDocsFace(mux)
	_ = genapi.HandlerFromMux(apiserver.NewServer(db), mux)
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"mode":"go"}`))
}

// mountDocsFace 挂 Swagger 文档面（rails config/routes.rb 镜像：/api-docs 与根
// 302 跳 index.html；静态件 go:embed 进单二进制，openapi.json 由 gen-shared
// 从 shared 契约生成、servers 清空让 try-it-out 打本域）。
func mountDocsFace(mux *http.ServeMux) {
	sub, err := fs.Sub(docsFaceFS, "docs/api-docs")
	if err != nil {
		panic(err) // 装配期 fail-fast：embed 布局错误即崩，不兜底
	}
	redirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api-docs/index.html", http.StatusFound)
	}
	mux.HandleFunc("GET /api-docs", redirect)
	mux.HandleFunc("GET /api-docs/{$}", redirect)
	// FileServer/ServeFileFS 都会把 /index.html 301 规范化成 ./——rails 静态中间件
	// 语义是直出，这里读 embed 字节直接写（家族探针打的就是这个路径，不能跳）
	mux.HandleFunc("GET /api-docs/index.html", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.Handle("GET /api-docs/", http.StripPrefix("/api-docs/", http.FileServerFS(sub)))
}
