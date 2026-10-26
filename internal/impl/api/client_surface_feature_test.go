package api_test

// client_surface_feature_test.go — Task 8 client 面（client-menus CRUD/move/reorder、
// tenant-applications 订阅链、clients 匿名元数据）。单事务贯穿 server+断言。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"saas-identity-platform-go/internal/testsupport"
)

// decodeArr 裸数组响应解码（菜单列表/排序响应无信封，rails render_camel(map) 镜像）。
func decodeArr(rec *httptest.ResponseRecorder) []any {
	var out []any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		return nil
	}
	return out
}

// fn: M04.F04.I01
// fn: M04.F04.I02
// fn: M04.F04.I03
// fn: M04.F04.I04
// fn: M04.F04.I05
// fn: M04.F04.I06
// fn: M04.F04.I07
func TestClientMenusCrudMoveReorderDelete(t *testing.T) {
	mux, fx, db := featureMux(t)
	tok := loginForToken(t, mux, fx)
	base := "/api/v1/clients/" + fx.Client.ClientId + "/menus"

	// 创建：title 缺失 400；type 缺省 directory（rails present? ? code : 1）；parentId 缺省=根
	rec := doJSON(mux, http.MethodPost, base, tok, map[string]any{})
	if rec.Code != 400 {
		t.Fatalf("创建缺 title 应 400，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{"title": "菜单丙"})
	if rec.Code != 200 {
		t.Fatalf("创建菜单应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	created := decodeBody(rec)
	if created["type"] != "directory" {
		t.Fatalf("type 缺省应 directory，得 %v", created["type"])
	}
	if created["parentId"] != testsupport.RootMenuID {
		t.Fatalf("parentId 缺省应零 UUID，得 %v", created["parentId"])
	}
	menuID, _ := created["id"].(string)
	if menuID == "" {
		t.Fatal("创建响应缺 id")
	}

	// 列表：裸数组（非信封非树），按 sort_order；DTO 含 status/createdAt 无 updatedAt
	rec = doJSON(mux, http.MethodGet, base, tok, nil)
	if rec.Code != 200 {
		t.Fatalf("列表应 200，得 %d", rec.Code)
	}
	items := decodeArr(rec)
	if len(items) == 0 {
		t.Fatal("列表为空")
	}
	first, _ := items[0].(map[string]any)
	if _, has := first["status"]; !has {
		t.Fatal("菜单 DTO 缺 status")
	}
	if _, has := first["updatedAt"]; has {
		t.Fatal("菜单 DTO 不应含 updatedAt（rails 镜像）")
	}

	// 详情
	rec = doJSON(mux, http.MethodGet, base+"/"+menuID, tok, nil)
	if rec.Code != 200 || decodeBody(rec)["title"] != "菜单丙" {
		t.Fatalf("详情应 200/菜单丙，得 %d %s", rec.Code, rec.Body.String())
	}

	// 更新：只应用 title/path/component/perms/icon/sortOrder；parentId/type/status 忽略
	rec = doJSON(mux, http.MethodPatch, base+"/"+menuID, tok, map[string]any{
		"title": "菜单丁", "path": "/ding", "type": "button", "status": 0,
	})
	if rec.Code != 200 {
		t.Fatalf("更新应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	upd := decodeBody(rec)
	if upd["title"] != "菜单丁" || upd["path"] != "/ding" {
		t.Fatalf("更新未应用 title/path: %v", upd)
	}
	if upd["type"] != "directory" || upd["status"] != float64(1) {
		t.Fatalf("type/status 应被忽略，得 type=%v status=%v", upd["type"], upd["status"])
	}

	// 移动：parentId 非空则改挂
	target := testsupport.SeedMenus(t, db, fx).DirMenu
	rec = doJSON(mux, http.MethodPatch, base+"/"+menuID+"/parent", tok,
		map[string]any{"parentId": target.Id})
	if rec.Code != 200 || decodeBody(rec)["parentId"] != target.Id {
		t.Fatalf("移动应 200/新父，得 %d %s", rec.Code, rec.Body.String())
	}

	// 排序：indexOf(menuId) 写 sortOrder；响应=全量扁平裸数组
	rec = doJSON(mux, http.MethodPut, base+"/"+menuID+"/reorder", tok,
		map[string]any{"orderedMenuIds": []string{target.Id, menuID}})
	if rec.Code != 200 {
		t.Fatalf("排序应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	rows := decodeArr(rec)
	if rows == nil {
		t.Fatalf("排序响应应为全量裸数组，得 %s", rec.Body.String())
	}
	found := false
	for _, rowAny := range rows {
		m, _ := rowAny.(map[string]any)
		if m["id"] == menuID {
			found = true
			if m["sortOrder"] != float64(1) {
				t.Fatalf("reorder 应写 sortOrder=1，得 %v", m["sortOrder"])
			}
		}
	}
	if !found {
		t.Fatal("排序响应缺被排序菜单")
	}

	// 删除：204 且幂等（不预查）
	rec = doJSON(mux, http.MethodDelete, base+"/"+menuID, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("删除应 204，得 %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodDelete, base+"/"+menuID, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("重复删除应幂等 204，得 %d", rec.Code)
	}
}

// fn: M00.F05.I01
// fn: M00.F05.I02
// fn: M00.F05.I03
// fn: M00.F05.I04
func TestTenantApplicationsLifecycle(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)
	base := "/api/v1/tenants/" + fx.Tenant.Id + "/applications"

	// 列表信封（空集起步）
	rec := doJSON(mux, http.MethodGet, base, tok, nil)
	if rec.Code != 200 {
		t.Fatalf("列表应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	env := decodeBody(rec)
	if env["total"] != float64(0) {
		t.Fatalf("空列表 total 应 0，得 %v", env["total"])
	}

	// 订阅：未知/空 clientId 均先撞 find_by! → 404（rails ensure_subscribable! 顺序）
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{"clientId": "no-such-client"})
	if rec.Code != 404 {
		t.Fatalf("未知 clientId 应 404，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{"clientId": ""})
	if rec.Code != 404 {
		t.Fatalf("空 clientId 先撞查库应 404，得 %d", rec.Code)
	}
	expire := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{
		"clientId": fx.Client.ClientId, "expireTime": expire.Format(time.RFC3339),
	})
	if rec.Code != 200 {
		t.Fatalf("订阅应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	app := decodeBody(rec)
	if app["clientId"] != fx.Client.ClientId || app["status"] != float64(1) {
		t.Fatalf("订阅形状不符: %v", app)
	}

	// 重复订阅 400 clean message
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{"clientId": fx.Client.ClientId})
	if rec.Code != 400 {
		t.Fatalf("重复订阅应 400，得 %d %s", rec.Code, rec.Body.String())
	}
	if got, _ := decodeBody(rec)["message"].(string); got == "" {
		t.Fatal("重复订阅 400 应带 clean message")
	}

	// 更新：status=2 应用（注意 rails blank 语义：0 不应用，非 0 才写）
	rec = doJSON(mux, http.MethodPatch, base+"/"+fx.Client.ClientId, tok,
		map[string]any{"status": 2})
	if rec.Code != 200 || decodeBody(rec)["status"] != float64(2) {
		t.Fatalf("更新订阅应 200/status 2，得 %d %s", rec.Code, rec.Body.String())
	}

	// 移除：204 且幂等（行不存在也 204）
	rec = doJSON(mux, http.MethodDelete, base+"/"+fx.Client.ClientId, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("移除应 204，得 %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodDelete, base+"/"+fx.Client.ClientId, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("重复移除应幂等 204，得 %d", rec.Code)
	}
}

// fn: M04.F01.I06
func TestClientsGetClientAnonymous(t *testing.T) {
	mux, fx, _ := featureMux(t)
	// 无 token 匿名可读
	rec := doJSON(mux, http.MethodGet, "/api/v1/clients/"+fx.Client.ClientId, "", nil)
	if rec.Code != 200 {
		t.Fatalf("匿名 client 元数据应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	if body["clientId"] != fx.Client.ClientId || body["clientName"] != fx.Client.ClientName {
		t.Fatalf("client 元数据形状不符: %v", body)
	}
	rec = doJSON(mux, http.MethodGet, "/api/v1/clients/no-such-client", "", nil)
	if rec.Code != 404 {
		t.Fatalf("未知 client 应 404，得 %d", rec.Code)
	}
}
