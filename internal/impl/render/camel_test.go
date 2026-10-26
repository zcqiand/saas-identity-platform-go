package render

// camel_test.go — render_camel 深变换语义（镜像 rails ApiSupport::JsonCamelizeKey）：
// map 键 snake→camel、nil 值键直接丢弃（非 "field":null）、数组保序递归、标量原样。

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCamelizeDropsNilKeys(t *testing.T) {
	in := map[string]any{"user_id": "u1", "display_name": nil, "ok": true}
	got := Camelize(in)
	want := map[string]any{"userId": "u1", "ok": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nil 键应被丢弃：got %v want %v", got, want)
	}
}

func TestCamelizeNestedAndSlice(t *testing.T) {
	in := map[string]any{
		"available_tenants": []any{
			map[string]any{"tenant_id": "t1", "role_name": nil},
		},
		"memberships": []map[string]any{{"tenant_key": "k1"}, {"role_ids": []string{}}},
		"page_meta":   map[string]any{"page_size": 20},
	}
	got := Camelize(in)
	want := map[string]any{
		"availableTenants": []any{map[string]any{"tenantId": "t1"}},
		"memberships":      []map[string]any{{"tenantKey": "k1"}, {"roleIds": []string{}}},
		"pageMeta":         map[string]any{"pageSize": 20},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("嵌套/数组深变换不符：got %v want %v", got, want)
	}
}

func TestCamelizeScalarsPassthrough(t *testing.T) {
	if Camelize("x") != "x" || Camelize(42) != 42 || Camelize(nil) != nil {
		t.Fatal("标量应原样透传")
	}
	s := []any{"a", nil, "b"}
	got := Camelize(s).([]any)
	if len(got) != 3 || got[0] != "a" || got[2] != "b" {
		t.Fatal("数组应保序（nil 元素保留——只有 map 的 nil 值键丢弃）")
	}
}

func TestWriteJSONCamelizesAndSetsHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, 200, map[string]any{"access_token": "tok", "refresh_token": nil})
	if rec.Code != 200 {
		t.Fatalf("期望 200，得到 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type 应为 application/json，得到 %q", ct)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("响应体不是合法 JSON: %v", err)
	}
	if _, ok := out["accessToken"]; !ok {
		t.Fatalf("应含 camel 键 accessToken：%s", rec.Body.String())
	}
	if _, ok := out["refreshToken"]; ok {
		t.Fatalf("nil 值键不得出现：%s", rec.Body.String())
	}
}
