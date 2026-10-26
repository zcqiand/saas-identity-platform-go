package api_test

// me_feature_test.go — tag: me 的 Feature 测试（真库 saas_test，事务回滚隔离）。
// 语义逐 rails MeController + Me::MenuTree 平移：whoami 全量 memberships +
// resolved tenant；menus 按 client 分组建树（根 parentId 零 UUID → 键剔除、
// 仅根层按 sort_order 升序、title 非 name）；tenants 裸数组；switch 换签 JWT +
// 临时 refresh（saas-rt-，不落库），无 membership/禁用 → 404。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/impl/auth"
	"saas-identity-platform-go/internal/testsupport"
)

func getWithBearer(mux http.Handler, path, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(rec, req)
	return rec
}

func loginForToken(t *testing.T, mux http.Handler, fx testsupport.AuthFixtures) string {
	t.Helper()
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": fx.User.Username, "password": fx.Password, "clientId": fx.Client.ClientId,
	})
	if rec.Code != 200 {
		t.Fatalf("login 前置失败: %d %s", rec.Code, rec.Body.String())
	}
	tok, _ := decodeBody(rec)["accessToken"].(string)
	if tok == "" {
		t.Fatal("login 未返回 accessToken")
	}
	return tok
}

// fn: M01.F01.I01
func TestWhoamiReturnsMembershipsAndResolvedTenant(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)
	rec := getWithBearer(mux, "/api/v1/me", tok)
	if rec.Code != 200 {
		t.Fatalf("whoami 应 200，得到 %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	if body["id"] != fx.User.Id {
		t.Fatalf("whoami id 不符: %v", body["id"])
	}
	if body["currentTenantId"] != fx.Tenant.Id {
		t.Fatalf("currentTenantId 不符: %v", body["currentTenantId"])
	}
	ms, ok := body["memberships"].([]any)
	if !ok || len(ms) != 1 {
		t.Fatalf("memberships 应为单元素数组: %v", body["memberships"])
	}
	m0, _ := ms[0].(map[string]any)
	if ids, ok := m0["roleIds"].([]any); !ok || len(ids) != 0 {
		t.Fatalf("membership roleIds 应为数组（此处未挂角色）: %v", m0["roleIds"])
	}
}

// fn: M04.F04.I08
func TestMeMenusClientGroupedTreeNullsStripped(t *testing.T) {
	mux, fx, tx := featureMux(t)
	mf := testsupport.SeedMenus(t, tx, fx)
	tok := loginForToken(t, mux, fx)
	rec := getWithBearer(mux, "/api/v1/me/menus", tok)
	if rec.Code != 200 {
		t.Fatalf("menus 应 200，得到 %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	group, ok := body[fx.Client.ClientId].([]any)
	if !ok || len(group) != 1 {
		t.Fatalf("应按 clientId 分组且根层单元素: %v", body)
	}
	dir, _ := group[0].(map[string]any)
	if dir["id"] != mf.DirMenu.Id {
		t.Fatalf("根层应含目录（仅根层 sort_order 升序）: %v", dir)
	}
	if dir["title"] != "目录甲" { // title 非 name（记忆 saas-me-menus-title-not-name）
		t.Fatalf("菜单节点应含 title: %v", dir)
	}
	if dir["type"] != "directory" {
		t.Fatalf("type 应映射 directory: %v", dir["type"])
	}
	if _, exists := dir["parentId"]; exists {
		t.Fatalf("根节点 parentId 应被剔除（nil 键丢弃）: %v", dir)
	}
	children, ok := dir["children"].([]any)
	if !ok || len(children) != 1 {
		t.Fatalf("目录应挂一个子节点: %v", dir["children"])
	}
	child, _ := children[0].(map[string]any)
	if child["id"] != mf.ChildMenu.Id || child["parentId"] != mf.DirMenu.Id || child["type"] != "menu" {
		t.Fatalf("子节点形状不符: %v", child)
	}
}

// fn: M01.F03.I01
func TestMeTenantsBareActiveArray(t *testing.T) {
	mux, fx, tx := featureMux(t)
	testsupport.SeedSecondTenant(t, tx, fx, 2) // invited 不进本端点（只 active）
	tok := loginForToken(t, mux, fx)
	rec := getWithBearer(mux, "/api/v1/me/tenants", tok)
	if rec.Code != 200 {
		t.Fatalf("me/tenants 应 200，得到 %d: %s", rec.Code, rec.Body.String())
	}
	var arr []any
	if err := json.Unmarshal(rec.Body.Bytes(), &arr); err != nil || arr == nil {
		t.Fatalf("me/tenants 应为裸数组: %s", rec.Body.String())
	}
	if len(arr) != 1 {
		t.Fatalf("只应含 active membership: %v", arr)
	}
	m0, _ := arr[0].(map[string]any)
	if m0["status"] != "active" || m0["tenantId"] != fx.Tenant.Id {
		t.Fatalf("membership 形状不符: %v", m0)
	}
}

// fn: M01.F03.I02
func TestSwitchTenantIssuesNewToken(t *testing.T) {
	mux, fx, tx := featureMux(t)
	second := testsupport.SeedSecondTenant(t, tx, fx, 2) // invited 可切
	tok := loginForToken(t, mux, fx)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/tenants/"+second.Id+"/switch", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("switch 应 200，得到 %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	access, _ := body["accessToken"].(string)
	claims, err := auth.Verify(access)
	if err != nil {
		t.Fatalf("switch 签发的 JWT 应可验: %v", err)
	}
	if claims["tenant_id"] != second.Id || claims["sub"] != fx.User.Id {
		t.Fatalf("JWT claims 应指向新租户: %v", claims)
	}
	refresh, _ := body["refreshToken"].(string)
	if len(refresh) < 8 || refresh[:8] != "saas-rt-" {
		t.Fatalf("switch refresh 应为临时形状 saas-rt-: %q", refresh)
	}
	if body["tenantId"] != second.Id {
		t.Fatalf("tenantId 回显不符: %v", body["tenantId"])
	}
	// expiresAt 为 RFC3339 时间串（rails Time render 镜像，非毫秒数）
	expStr, _ := body["expiresAt"].(string)
	expAt, err := time.Parse(time.RFC3339Nano, expStr)
	if err != nil {
		t.Fatalf("expiresAt 应为 RFC3339 串: %v %q", err, expStr)
	}
	if expAt.Before(time.Now().Add(50 * time.Minute)) {
		t.Fatalf("expiresAt 应约 now+TTL: %v", expAt)
	}
	// 临时 refresh 不落库
	var n int64
	tx.Model(&entity.OauthRefreshToken{}).Where("refresh_token = ?", refresh).Count(&n)
	if n != 0 {
		t.Fatal("switch refresh 不应落库")
	}
}

// fn: M01.F03.I02
func TestSwitchTenant404Paths(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	// 未知租户 → 404
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/tenants/"+testsupport.NewUUID()+"/switch", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("未知租户 switch 应 404，得到 %d", rec.Code)
	}
	// 有租户无 membership → 404（裸租户，不挂任何 member）
	bare := entity.Tenant{
		Id: testsupport.NewUUID(), TenantKey: "go_bare_" + testsupport.NewUUID()[:8],
		Name: "Bare", Status: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := tx.Create(&bare).Error; err != nil {
		t.Fatalf("裸租户种子失败: %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/me/tenants/"+bare.Id+"/switch", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("无 membership switch 应 404，得到 %d: %s", rec.Code, rec.Body.String())
	}
	// disabled membership（status=0）→ 404
	disabledTenant := testsupport.SeedSecondTenant(t, tx, fx, 0)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/me/tenants/"+disabledTenant.Id+"/switch", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("disabled membership switch 应 404，得到 %d", rec.Code)
	}
}
