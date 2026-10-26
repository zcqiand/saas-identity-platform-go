package family

// refresh_tokens.go — switch 端点的临时 refresh 形状（springboot
// JwtIssuer.generateRefreshToken 镜像）。"saas-rt-{userId}-{ts-ms}-{rand}"；
// 不落库不持久化 refresh 行。

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// EphemeralRefreshToken 生成一次性形状 refresh（仅响应体携带，无库行）。
func EphemeralRefreshToken(userID string) string {
	b := make([]byte, 24)
	if _, err := crand.Read(b); err != nil {
		panic(fmt.Errorf("family: 系统熵源不可用: %w", err))
	}
	return fmt.Sprintf("saas-rt-%s-%d-%s", userID, time.Now().UnixMilli(),
		base64.RawURLEncoding.EncodeToString(b))
}
