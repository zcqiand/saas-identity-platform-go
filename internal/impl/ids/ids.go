// Package ids — 运行期 ID 生成（RFC 4122 v4，零三方依赖；version-lock 不收 uuid 库）。
package ids

import (
	crand "crypto/rand"
	"fmt"
)

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
