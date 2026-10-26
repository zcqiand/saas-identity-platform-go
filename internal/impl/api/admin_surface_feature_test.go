package api_test

// admin_surface_feature_test.go — Task 9 admin 面（admin-clients 6 op +
// admin-tenants 5 op）。JWT 必需、无租户校验（dev 无 scope 区分镜像）。

import (
	"net/http"
	"testing"

	"saas-identity-platform-go/internal/gen/entity"
)

// fn: M04.F01.I01
// fn: M04.F01.I02
// fn: M04.F01.I03
// fn: M04.F01.I04
// fn: M04.F01.I05
// fn: M04.F02.I01
func TestAdminClientsCrudLifecycle(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	base := "/api/v1/admin/clients"

	// 创建：validity 缺省/<=0 → 3600/86400（应用层默认，非 DB DEFAULT）；200 非 201
	rec := doJSON(mux, http.MethodPost, base, tok, map[string]any{
		"clientId": "go-admin-app-1", "clientName": "Admin App",
		"clientSecret": "sec-1", "grantTypes": "password",
		"redirectUris":        "http://localhost/cb",
		"accessTokenValidity": -5, // <=0 → 3600
	})
	if rec.Code != 200 {
		t.Fatalf("创建 client 应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	created := decodeBody(rec)
	if created["accessTokenValidity"] != float64(3600) || created["refreshTokenValidity"] != float64(86400) {
		t.Fatalf("validity 应用层缺省 3600/86400，得 %v/%v",
			created["accessTokenValidity"], created["refreshTokenValidity"])
	}
	if created["status"] != float64(1) {
		t.Fatalf("新 client status 应 1，得 %v", created["status"])
	}
	clientCode, _ := created["clientId"].(string)
	if clientCode != "go-admin-app-1" {
		t.Fatalf("clientId 回显不符: %v", created["clientId"])
	}

	// 缺必填 → 400（springboot @NotNull 契约面，CT I64：空 body/缺字段必须 400 不落库）
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{})
	if rec.Code != 400 {
		t.Fatalf("空 body 创建应 400，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{
		"clientId": "go-admin-app-x", "clientName": "缺 redirectUris",
		"clientSecret": "sec-x", "grantTypes": "password",
	})
	if rec.Code != 400 {
		t.Fatalf("缺 redirectUris 创建应 400，得 %d %s", rec.Code, rec.Body.String())
	}

	// 列表信封
	rec = doJSON(mux, http.MethodGet, base+"?page=0&pageSize=50", tok, nil)
	if rec.Code != 200 {
		t.Fatalf("列表应 200，得 %d", rec.Code)
	}
	if items, _ := decodeBody(rec)["items"].([]any); len(items) == 0 {
		t.Fatal("列表为空")
	}

	// 详情（按业务字符串 client_id 寻址，非行 UUID）
	rec = doJSON(mux, http.MethodGet, base+"/go-admin-app-1", tok, nil)
	if rec.Code != 200 || decodeBody(rec)["clientName"] != "Admin App" {
		t.Fatalf("详情应 200/Admin App，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodGet, base+"/no-such-client", tok, nil)
	if rec.Code != 404 {
		t.Fatalf("未知 client 应 404，得 %d", rec.Code)
	}

	// 更新：只应用 clientName/redirectUris/scopes（其余字段忽略）
	rec = doJSON(mux, http.MethodPatch, base+"/go-admin-app-1", tok, map[string]any{
		"clientName": "Admin App v2", "scopes": "read write",
		"grantTypes": "authorization_code", // 忽略
	})
	if rec.Code != 200 {
		t.Fatalf("更新应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	upd := decodeBody(rec)
	if upd["clientName"] != "Admin App v2" || upd["scopes"] != "read write" {
		t.Fatalf("更新未应用 clientName/scopes: %v", upd)
	}
	if upd["grantTypes"] != "password" {
		t.Fatalf("grantTypes 应被忽略，得 %v", upd["grantTypes"])
	}

	// 状态：status 无条件应用（0=disabled；springboot oracle 语义，CT I49 往返；
	// rails present? blank 0 不写是已定性家族偏差，go 随 oracle）
	rec = doJSON(mux, http.MethodPatch, base+"/go-admin-app-1/status", tok,
		map[string]any{"status": 0})
	if rec.Code != 200 || decodeBody(rec)["status"] != float64(0) {
		t.Fatalf("status=0 应应用（disabled），得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPatch, base+"/go-admin-app-1/status", tok,
		map[string]any{"status": 2})
	if rec.Code != 200 || decodeBody(rec)["status"] != float64(2) {
		t.Fatalf("status=2 应应用，得 %d %s", rec.Code, rec.Body.String())
	}

	// 删除：404 if missing；204；删后 get 404
	rec = doJSON(mux, http.MethodDelete, base+"/no-such-client", tok, nil)
	if rec.Code != 404 {
		t.Fatalf("删未知 client 应 404（destroy! 预查），得 %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodDelete, base+"/go-admin-app-1", tok, nil)
	if rec.Code != 204 {
		t.Fatalf("删除应 204，得 %d", rec.Code)
	}
	var n int64
	tx.Model(&entity.OauthClient{}).Where("client_id = ?", "go-admin-app-1").Count(&n)
	if n != 0 {
		t.Fatal("删除后行应不存在")
	}
}

// fn: M00.F01.I01
// fn: M00.F01.I02
// fn: M00.F01.I03
// fn: M00.F01.I04
// fn: M00.F01.I05
func TestAdminTenantsCrudLifecycle(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	base := "/api/v1/admin/tenants"

	// 创建：tenantKey/name 必填 400
	rec := doJSON(mux, http.MethodPost, base, tok, map[string]any{"name": "缺 key"})
	if rec.Code != 400 {
		t.Fatalf("缺 tenantKey 应 400，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{"tenantKey": "go_t_admin"})
	if rec.Code != 400 {
		t.Fatalf("缺 name 应 400，得 %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodPost, base, tok, map[string]any{
		"tenantKey": "go_t_admin", "name": "Admin Tenant",
	})
	if rec.Code != 200 {
		t.Fatalf("创建租户应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	created := decodeBody(rec)
	// DTO status 恒 'active'（springboot toDto 忠实镜像 quirk）
	if created["status"] != "active" {
		t.Fatalf("DTO status 恒 active，得 %v", created["status"])
	}
	tenantID, _ := created["id"].(string)
	if tenantID == "" {
		t.Fatal("创建响应缺 id")
	}

	// 列表信封
	rec = doJSON(mux, http.MethodGet, base+"?page=0&pageSize=50", tok, nil)
	if rec.Code != 200 {
		t.Fatalf("列表应 200，得 %d", rec.Code)
	}
	if items, _ := decodeBody(rec)["items"].([]any); len(items) == 0 {
		t.Fatal("列表为空")
	}

	// 详情 + 未知 404
	rec = doJSON(mux, http.MethodGet, base+"/"+tenantID, tok, nil)
	if rec.Code != 200 || decodeBody(rec)["tenantKey"] != "go_t_admin" {
		t.Fatalf("详情应 200/go_t_admin，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodGet, base+"/11111111-1111-4111-8111-111111111111", tok, nil)
	if rec.Code != 404 {
		t.Fatalf("未知租户应 404，得 %d", rec.Code)
	}

	// 更新：name 应用；status suspended → DB 0，DTO 仍 active（quirk 勿修）
	rec = doJSON(mux, http.MethodPatch, base+"/"+tenantID, tok, map[string]any{
		"name": "Admin Tenant v2", "status": "suspended",
	})
	if rec.Code != 200 {
		t.Fatalf("更新应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	upd := decodeBody(rec)
	if upd["name"] != "Admin Tenant v2" || upd["status"] != "active" {
		t.Fatalf("更新形状不符（status 恒 active）： %v", upd)
	}
	var row entity.Tenant
	tx.Where("id = ?", tenantID).First(&row)
	if row.Status != 0 {
		t.Fatalf("suspended 应落库 status=0，得 %d", row.Status)
	}

	// 删除：204 且幂等（不预查）
	rec = doJSON(mux, http.MethodDelete, base+"/"+tenantID, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("删除应 204，得 %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodDelete, base+"/"+tenantID, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("重复删除应幂等 204，得 %d", rec.Code)
	}
}
