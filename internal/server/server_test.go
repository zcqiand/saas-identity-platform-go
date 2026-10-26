package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// REQ-2026-001 AC-2：家族探活契约形状 GET /health → {"ok":true,"mode":"go"}。
// 基础设施端点不挂功能 ID（ktor REQ-2026-001/003 先例，trace fns 留空）。
func TestHealthShape(t *testing.T) {
	rec := httptest.NewRecorder()
	New("127.0.0.1:0").Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", rec.Code)
	}
	var body struct {
		OK   bool   `json:"ok"`
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体不是合法 JSON: %v", err)
	}
	if !body.OK || body.Mode != "go" {
		t.Fatalf("期望 {\"ok\":true,\"mode\":\"go\"}，得到 %s", rec.Body.String())
	}
}

func TestHealthRejectsNonGet(t *testing.T) {
	rec := httptest.NewRecorder()
	New("127.0.0.1:0").Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /health 期望 405，得到 %d", rec.Code)
	}
}
