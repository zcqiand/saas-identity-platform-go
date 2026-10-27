package harness

// route_parity_test.go — 路由面契约门（REQ-2026-003 Task 3）
//
// SSOT = generated/api_manifest.json（genmanifest 从 shared openapi.yaml 产出）。
// 断言一：manifest 47 op 每条都能被装配出的 ServeMux 以「方法+路径」精确命中
// （pattern 逐字节相等，{clientId} 段用探针值替换）。
// 断言二：路径对、方法错 → 405（拦「catch-all 注册」类漂移）；
//         契约外的根路径 → 404（infra 端点只有 /health，ktor/laravel 先例）。
//
// 本测试不连库（装配只注册路由，handler 壳不触 DB）——rails RouteParityTest
// 无库先例同构。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type manifestOp struct {
	OperationID string `json:"operationId"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Tag         string `json:"tag"`
}

type manifestFile struct {
	Schema     int          `json:"schema"`
	Operations []manifestOp `json:"operations"`
}

func loadManifest(t *testing.T) manifestFile {
	t.Helper()
	// 测试 cwd = 本包目录（internal/impl/harness），仓根在上三级
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "generated", "api_manifest.json"))
	if err != nil {
		t.Fatalf("读 api_manifest.json: %v（先跑 bash scripts/gen-shared.sh）", err)
	}
	var m manifestFile
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("解析 api_manifest.json: %v", err)
	}
	if m.Schema != 1 || len(m.Operations) == 0 {
		t.Fatalf("manifest 形状异常: schema=%d ops=%d", m.Schema, len(m.Operations))
	}
	return m
}

// probePath 把 {param} 段替换成探针值（ServeMux 单段通配必有命中）。
func probePath(pattern string) string {
	segs := strings.Split(pattern, "/")
	for i, seg := range segs {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			segs[i] = "probe1"
		}
	}
	return strings.Join(segs, "/")
}

func expectedPattern(op manifestOp) string {
	return op.Method + " " + op.Path
}

func TestManifestOperationsHaveExactRoutes(t *testing.T) {
	m := loadManifest(t)
	mux := assembleMux(nil)

	for _, op := range m.Operations {
		req := httptest.NewRequest(op.Method, probePath(op.Path), nil)
		h, pattern := mux.Handler(req)
		if pattern == "" {
			t.Errorf("manifest %s %s (%s) 无匹配路由", op.Method, op.Path, op.OperationID)
			continue
		}
		if pattern != expectedPattern(op) {
			t.Errorf("manifest %s %s (%s) 命中 pattern=%q，期望 %q", op.Method, op.Path, op.OperationID, pattern, expectedPattern(op))
		}
		if h == nil {
			t.Errorf("manifest %s %s handler 为 nil", op.Method, op.Path)
		}
	}
}

func TestWrongMethodYields405(t *testing.T) {
	m := loadManifest(t)
	mux := assembleMux(nil)

	// 方法错 → ServeMux 写 405；若路由被注册成方法无关 catch-all，这里会拿到
	// 2xx/5xx——红。两个合法例外，均须放行：
	//  1) 通配碰撞：错方法请求命中「另一条同方法、路径形状等价」的 manifest op
	//     （如 PATCH .../members/invitations 落 PATCH .../members/{userId}）——
	//     这是正确路由，不是漂移；
	//  2) 命中了那条路由但参数绑定失败（wrapper 400）——路由面本身正确，
	//     绑定行为归各域实现的 schema 测试管。
	// 必须走 mux.ServeHTTP：绕过会丢 PathValue 绑定，wrapper 400 是探针伪影。
	for _, op := range m.Operations {
		wrong := "PATCH"
		if op.Method == "PATCH" {
			wrong = "PUT"
		}
		req := httptest.NewRequest(wrong, probePath(op.Path), nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code == http.StatusMethodNotAllowed {
			continue
		}
		equivalent := false
		for _, other := range m.Operations {
			if other.Method != wrong {
				continue
			}
			if pathShapeMatches(other.Path, probePath(op.Path)) {
				equivalent = true
				break
			}
		}
		if !equivalent {
			t.Errorf("%s %s 用 %s 应 405（或命中等价 manifest op），得 %d", op.Method, op.Path, wrong, rec.Code)
		}
	}
}

// pathShapeMatches 判断 opPath 的形状（{param} 视为单段通配）能否匹配实探路径。
func pathShapeMatches(opPath, actual string) bool {
	segs := strings.Split(opPath, "/")
	for i, seg := range segs {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			segs[i] = "[^/]+"
		} else {
			segs[i] = regexp.QuoteMeta(seg)
		}
	}
	re, err := regexp.Compile("^" + strings.Join(segs, "/") + "$")
	if err != nil {
		return false
	}
	return re.MatchString(actual)
}

func TestOutsideContractIs404(t *testing.T) {
	mux := assembleMux(nil)
	// 契约外根路径必须 404——API 面只认生成物（硬规则 §4）；infra 面=/health + /api-docs + 根跳转（都不进 manifest）
	var req *http.Request
	var rec *httptest.ResponseRecorder
	for _, p := range []string{"/api", "/api/v1", "/internal/secret"} {
		req = httptest.NewRequest(http.MethodGet, p, nil)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("契约外路径 %s 应 404，得 %d", p, rec.Code)
		}
	}
	// 根路径：rails `root to: redirect(...)` 镜像，302 跳 /api-docs/index.html（302 而非 404）
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Errorf("GET / 应 302 跳 Swagger UI（rails root redirect 镜像），得 %d", rec.Code)
	}
	// infra：GET /health 在（家族探活契约，不进 manifest）
	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /health 应 200，得 %d", rec.Code)
	}
}
