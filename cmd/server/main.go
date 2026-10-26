// Command server 组合根：读 SERVER_PORT 装配监听地址并起 HTTP 服务。
// 装配期 fail-fast：缺键/非法直接退出非零，不兜底不重试。
package main

import (
	"fmt"
	"log"
	"log/slog"

	"saas-identity-platform-go/internal/config"
	"saas-identity-platform-go/internal/server"
)

func main() {
	port, err := config.Port()
	if err != nil {
		log.Fatalf("装配失败: %v", err)
	}
	// 绑全接口 ":port"（REQ-2026-002）：容器里绑 127.0.0.1 的话 host 端口映射
	// 打不进容器回环，deploy 探针必 connection refused。ktor 先例：
	// embeddedServer(Netty, port=…) 无 host 参数 = Netty 默认 0.0.0.0，dev/prod 同姿态。
	addr := fmt.Sprintf(":%d", port)
	srv := server.New(addr)
	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("服务退出: %v", err)
	}
}
