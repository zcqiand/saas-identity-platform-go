// Package server 装配 HTTP 服务（REQ-2026-003 Task 3 起：路由面归
// internal/impl/harness.Assemble，本包只做 addr+handler → *http.Server 包装）。
package server

import "net/http"

// New 返回监听 addr、以 handler 为路由根的 HTTP Server。
func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler}
}
