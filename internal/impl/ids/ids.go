// Package ids — 运行期 ID 生成（RFC 4122 v4，零三方依赖；version-lock 不收 uuid 库）。
package ids

import (
	crand "crypto/rand"
	"fmt"
	"regexp"
)

// uuidRe RFC 4122 任意版本形制（rails ApplicationController::UUID_RE 同形）。
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsValidUUID 路径参数 uuid 形制校验（不合格式 → 调用方转 400，防 PG 22P02 裸 500）。
func IsValidUUID(s string) bool { return uuidRe.MatchString(s) }

// NewUUID 生成 v4 UUID 字符串；系统熵源不可用属运行期灾难，直接 panic（禁吞错）。
func NewUUID() string {
	var b [16]byte
	if _, err := crand.Read(b[:]); err != nil {
		panic(fmt.Errorf("ids: 系统熵源不可用: %w", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
