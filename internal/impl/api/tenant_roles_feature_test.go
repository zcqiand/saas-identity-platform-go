package api_test

// tenant_roles_feature_test.go — tag: tenant-roles / tenant-role-menus 的 Feature 测试
// （M00.F03 + M00.F04.I02/I03/I04）。语义逐 rails TenantRolesController /
// TenantRoleMenusController 平移：role.tenant_id≠path → 404 不泄露存在性；
// PUT menus 全量替换差量实现 + touch role；DELETE clear 幂等 204 不预查。

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/testsupport"
)

func rolesBase(fx testsupport.AuthFixtures) string {
	return "/api/v1/tenants/" + fx.Tenant.Id + "/roles"
}

func createRole(t *testing.T, mux http.Handler, tok string, fx testsupport.AuthFixtures, code string, isPreset bool) map[string]any {
	t.Helper()
	rec := doJSON(mux, http.MethodPost, rolesBase(fx), tok, map[string]any{
		"clientId": fx.Client.ClientId, "roleCode": code, "roleName": "Role " + code, "isPreset": isPreset,
	})
	if rec.Code != 200 {
		t.Fatalf("createRole 前置失败: %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody(rec)
}

// fn: M00.F03.I02
func TestTenantRolesCreateValidationAndDefaults(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)
	for _, missing := range []string{"clientId", "roleCode", "roleName"} {
		body := map[string]any{"clientId": "c", "roleCode": "x", "roleName": "n"}
		delete(body, missing)
		rec := doJSON(mux, http.MethodPost, rolesBase(fx), tok, body)
		if rec.Code != 400 || !strings.Contains(rec.Body.String(), missing) {
			t.Fatalf("缺 %s 应 400: %d %s", missing, rec.Code, rec.Body.String())
		}
	}
	rec := doJSON(mux, http.MethodPost, rolesBase(fx), tok, map[string]any{
		"clientId": fx.Client.ClientId, "roleCode": "r1", "roleName": "R1", "isPreset": true,
	})
	body := decodeBody(rec)
	if rec.Code != 200 || body["roleCode"] != "r1" || body["isPreset"] != true ||
		body["status"] != float64(1) || body["tenantId"] != fx.Tenant.Id {
		t.Fatalf("创建形状不符: %d %v", rec.Code, body)
	}
	if _, ok := body["id"]; !ok {
		t.Fatalf("创建应回 id: %v", body)
	}
}

// fn: M00.F03.I01
func TestTenantRolesListPaginated(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)
	createRole(t, mux, tok, fx, "la", false)
	createRole(t, mux, tok, fx, "lb", false)
	rec := getWithBearer(mux, rolesBase(fx), tok)
	if rec.Code != 200 {
		t.Fatalf("list 应 200: %d", rec.Code)
	}
	body := decodeBody(rec)
	if body["total"] != float64(2) {
		t.Fatalf("total 应为 2: %v", body["total"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items 应为 2: %v", items)
	}
}

// fn: M00.F03.I03 M00.F03.I04 M00.F03.I05
func TestTenantRolesGetUpdateDelete(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	role := createRole(t, mux, tok, fx, "crud", false)
	roleID := role["id"].(string)
	// 挂 menu 绑定 + member 绑定，删除后应被清理
	menu := entity.SysMenu{
		Id: testsupport.NewUUID(), ClientId: fx.Client.ClientId, ParentId: testsupport.RootMenuID,
		Title: "待清菜单", Type: 2, SortOrder: 1, Status: 1, CreatedAt: time.Now(),
	}
	if err := tx.Create(&menu).Error; err != nil {
		t.Fatalf("menu 种子失败: %v", err)
	}
	if err := tx.Create(&entity.SysRoleMenu{RoleId: roleID, MenuId: menu.Id}).Error; err != nil {
		t.Fatalf("role_menu 种子失败: %v", err)
	}
	if err := tx.Create(&entity.TenantMemberRole{MemberId: fx.Member.Id, RoleId: roleID}).Error; err != nil {
		t.Fatalf("member_role 种子失败: %v", err)
	}
	// 详情
	rec := getWithBearer(mux, rolesBase(fx)+"/"+roleID, tok)
	if rec.Code != 200 || decodeBody(rec)["roleCode"] != "crud" {
		t.Fatalf("详情应 200: %d %s", rec.Code, rec.Body.String())
	}
	// 更新只动 roleName/description
	rec = doJSON(mux, http.MethodPatch, rolesBase(fx)+"/"+roleID, tok,
		map[string]any{"roleName": "改名后"})
	if rec.Code != 200 || decodeBody(rec)["roleName"] != "改名后" {
		t.Fatalf("更新应生效: %d %s", rec.Code, rec.Body.String())
	}
	// 删除 → 204 + 关联清理
	rec = doJSON(mux, http.MethodDelete, rolesBase(fx)+"/"+roleID, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("删除应 204: %d", rec.Code)
	}
	var n int64
	tx.Model(&entity.SysRoleMenu{}).Where("role_id = ?", roleID).Count(&n)
	if n != 0 {
		t.Fatalf("role_menu 应清理: %d", n)
	}
	tx.Model(&entity.TenantMemberRole{}).Where("role_id = ?", roleID).Count(&n)
	if n != 0 {
		t.Fatalf("member_role 应清理: %d", n)
	}
	rec = getWithBearer(mux, rolesBase(fx)+"/"+roleID, tok)
	if rec.Code != 404 {
		t.Fatalf("删后详情应 404: %d", rec.Code)
	}
}

// fn: M00.F03.I03
func TestTenantRolesForeignRoleIs404(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	foreignTenant := entity.Tenant{
		Id: testsupport.NewUUID(), TenantKey: "go_fr_" + testsupport.NewUUID()[:8],
		Name: "FR", Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := tx.Create(&foreignTenant).Error; err != nil {
		t.Fatalf("外来租户种子失败: %v", err)
	}
	foreignRole := entity.SysRole{
		Id: testsupport.NewUUID(), TenantId: foreignTenant.Id, ClientId: fx.Client.ClientId,
		RoleCode: "fr", RoleName: "FR", Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := tx.Create(&foreignRole).Error; err != nil {
		t.Fatalf("外来 role 种子失败: %v", err)
	}
	rec := getWithBearer(mux, rolesBase(fx)+"/"+foreignRole.Id, tok)
	if rec.Code != 404 {
		t.Fatalf("跨租户 role 应 404 不泄露存在性: %d", rec.Code)
	}
}

// fn: M00.F04.I02 M00.F04.I03 M00.F04.I04
func TestRoleMenusSetListClear(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	role := createRole(t, mux, tok, fx, "menus", false)
	roleID := role["id"].(string)
	m1, m2 := testsupport.NewUUID(), testsupport.NewUUID()
	for _, mid := range []string{m1, m2} {
		if err := tx.Create(&entity.SysMenu{
			Id: mid, ClientId: fx.Client.ClientId, ParentId: testsupport.RootMenuID,
			Title: "菜单" + mid[:4], Type: 2, SortOrder: 1, Status: 1, CreatedAt: time.Now(),
		}).Error; err != nil {
			t.Fatalf("menu 种子失败: %v", err)
		}
	}
	base := rolesBase(fx) + "/" + roleID + "/menus"
	var updatedBefore string
	tx.Model(&entity.SysRole{}).Where("id = ?", roleID).First(&entity.SysRole{})
	var roleRow entity.SysRole
	tx.Where("id = ?", roleID).First(&roleRow)
	updatedBefore = roleRow.UpdatedAt.Format(time.RFC3339Nano)
	time.Sleep(10 * time.Millisecond) // updated_at 触碰需时钟前进

	// PUT 全量替换：请求乱序 + 重复 → 响应去重升序，不回读
	rec := doJSON(mux, http.MethodPut, base, tok, map[string]any{"menuIds": []string{m2, m1, m2}})
	if rec.Code != 200 {
		t.Fatalf("set 应 200: %d %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	ids, _ := body["menuIds"].([]any)
	if len(ids) != 2 || ids[0] != m1 || ids[1] != m2 {
		t.Fatalf("menuIds 应去重升序: %v", body["menuIds"])
	}
	if body["roleId"] != roleID || body["tenantId"] != fx.Tenant.Id {
		t.Fatalf("聚合视图形状不符: %v", body)
	}
	// touch role
	tx.Where("id = ?", roleID).First(&roleRow)
	if roleRow.UpdatedAt.Format(time.RFC3339Nano) == updatedBefore {
		t.Fatal("set 后 role.updated_at 应被 touch")
	}
	// GET 回显
	rec = getWithBearer(mux, base, tok)
	got := decodeBody(rec)["menuIds"].([]any)
	if len(got) != 2 || got[0] != m1 {
		t.Fatalf("GET 应回升序授权集: %v", got)
	}
	// PUT 空 → 全删
	rec = doJSON(mux, http.MethodPut, base, tok, map[string]any{"menuIds": []string{}})
	if rec.Code != 200 {
		t.Fatalf("空 set 应 200: %d", rec.Code)
	}
	var n int64
	tx.Model(&entity.SysRoleMenu{}).Where("role_id = ?", roleID).Count(&n)
	if n != 0 {
		t.Fatalf("空 set 应清空: %d", n)
	}
	// DELETE clear：幂等 204 不预查（未知 roleId 也 204）
	rec = doJSON(mux, http.MethodDelete, base, tok, nil)
	if rec.Code != 204 {
		t.Fatalf("clear 应 204: %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodDelete, rolesBase(fx)+"/"+testsupport.NewUUID()+"/menus", tok, nil)
	if rec.Code != 204 {
		t.Fatalf("未知 roleId clear 也应 204: %d", rec.Code)
	}
	// 跨租户 role 查授权集 → 404
	rec = getWithBearer(mux, "/api/v1/tenants/"+fx.Tenant.Id+"/roles/"+testsupport.NewUUID()+"/menus", tok)
	if rec.Code != 404 {
		t.Fatalf("未知 role 授权集应 404: %d", rec.Code)
	}
}
