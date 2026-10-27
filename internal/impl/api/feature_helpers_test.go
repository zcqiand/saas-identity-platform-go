package api_test

// feature_helpers_test.go — Feature 测试共享助手（api_test 外部测试包）。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/impl/harness"
	"saas-identity-platform-go/internal/testsupport"
)

// doJSON 通用请求：带 Bearer 的 JSON 请求（body 为 nil 时不写体）。
func doJSON(mux http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	mux.ServeHTTP(rec, req)
	return rec
}

// postJSON 兼容早期用例的 POST 助手。
func postJSON(mux http.Handler, path string, body any) *httptest.ResponseRecorder {
	return doJSON(mux, http.MethodPost, path, "", body)
}

// decodeBody 解析响应体为 map（空体返回空 map）。
func decodeBody(rec *httptest.ResponseRecorder) map[string]any {
	out := map[string]any{}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return out
}

// featureMux 一条事务贯穿整个用例：server 的 DB 与断言查询共用同一连接
// （未提交写入只在本连接可见；Cleanup 统一回滚）。
func featureMux(t *testing.T) (http.Handler, testsupport.AuthFixtures, *gorm.DB) {
	t.Helper()
	tx := testsupport.Tx(t)
	fx := testsupport.SeedAuth(t, tx)
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101,http://localhost:5102,http://localhost:5103,http://localhost:5108")
	return harness.Assemble(tx), fx, tx
}
