// Package config 装配运行环境配置：键只来自 env，缺失/非法即报错。
// 禁止默认值兜底（suite-hard-rules §1）。
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Port 读取 SERVER_PORT（multi-repo-family §6 家族统一 key）并校验为合法端口。
func Port() (int, error) {
	raw := os.Getenv("SERVER_PORT")
	if raw == "" {
		return 0, fmt.Errorf("缺少环境变量 SERVER_PORT（全集见 .env.example；禁默认值兜底）")
	}
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("SERVER_PORT 不是合法端口数字 %q: %w", raw, err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("SERVER_PORT 超出 1-65535: %d", port)
	}
	return port, nil
}
