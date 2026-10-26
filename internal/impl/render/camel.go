// Package render — 响应渲染（镜像 rails render_camel / ApiSupport::JsonCamelizeKey）。
//
// 语义：map 键 snake→camel 深变换、nil 值键直接丢弃（可选字段缺失 = 键不出现，
// 不是 "field":null）、数组保序递归、标量原样。生成 DTO（genapi models）自带
// camel JSON tag，直接编码即可；本包服务的是 service 层构造的动态 map 形状
// （login payload、菜单树等）。
package render

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Camelize 深变换 v 并返回新值（不修改入参）。
func Camelize(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, val := range typed {
			if val == nil {
				continue // nil 值键直接丢弃（rails 口径）
			}
			out[camelKey(k)] = Camelize(val)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, val := range typed {
			out[i] = Camelize(val)
		}
		return out
	default:
		return v
	}
}

func camelKey(k string) string {
	parts := strings.Split(k, "_")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
		} else {
			b.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	return b.String()
}

// WriteJSON 以 camel 语义编码响应（service 层动态 map 用；DTO 直出也走此处，
// Camelize 对非 map/slice 值原样透传，无副作用）。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	if err := enc.Encode(Camelize(v)); err != nil {
		// 头已写，只能中断半截响应——编码失败是编程错误，fail-loud 于日志侧
		panic(err)
	}
}
