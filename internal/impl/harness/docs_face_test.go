package harness

// docs_face_test.go — Swagger 文档面（家族先例 rails /api-docs + springboot
// springdoc；基础设施端点不进契约面/功能树，与 /health 同类，ktor CI 冒烟同款）：
// /api-docs → 302 /api-docs/index.html；index.html 引 swagger-ui；openapi.json
// 与 shared 契约同源且 servers 清空（同源 try-it-out 打本域）。

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsFaceRedirectsToIndex(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101")
	h := Assemble(nil)
	// rails config/routes.rb `root to: redirect('/api-docs/index.html')` 镜像：
	// 根路径与 /api-docs 都 302 跳 Swagger UI（prod 实证 rails / → 302）
	for _, path := range []string{"/", "/api-docs"} {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 302 {
			t.Fatalf("GET %s 期望 302，得到 %d", path, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/api-docs/index.html" {
			t.Fatalf("GET %s Location 期望 /api-docs/index.html，得到 %q", path, loc)
		}
	}
}

func TestDocsFaceServesSwaggerUI(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101")
	h := Assemble(nil)
	req := httptest.NewRequest("GET", "/api-docs/index.html", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET /api-docs/index.html 期望 200，得到 %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "swagger-ui") {
		t.Fatal("index.html 应引 swagger-ui 静态件")
	}
}

func TestDocsFaceOpenAPISameOrigin(t *testing.T) {
	t.Setenv("SAAS_CORS_ALLOWED_ORIGINS", "http://localhost:5101")
	h := Assemble(nil)
	req := httptest.NewRequest("GET", "/api-docs/openapi.json", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("GET /api-docs/openapi.json 期望 200，得到 %d", rec.Code)
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("openapi.json 非法 JSON: %v", err)
	}
	if doc["openapi"] == nil {
		t.Fatal("openapi.json 缺 openapi 版本键（非 OpenAPI 文档）")
	}
	servers, ok := doc["servers"].([]any)
	if !ok || len(servers) != 0 {
		t.Fatalf("servers 应为空数组（同源 try-it-out 打本域）: %v", doc["servers"])
	}
}
