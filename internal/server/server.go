// Package server 装配 HTTP 服务。/health 是家族统一探活端点（基础设施端点，
// 不进契约面/功能树——ktor REQ-2026-001/003 先例）：GET → {"ok":true,"mode":"go"}。
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// New 返回装配好路由与监听地址的 HTTP Server。
func New(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	return &http.Server{Addr: addr, Handler: mux}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(map[string]any{"ok": true, "mode": "go"}); err != nil {
		slog.Error("health 响应编码失败", "err", err)
	}
}
