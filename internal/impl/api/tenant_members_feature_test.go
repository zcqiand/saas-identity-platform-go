package api_test

// tenant_members_feature_test.go — tag: tenant-members 的 Feature 测试（M00.F02 + M01.F02.I01）。
// 语义逐 rails TenantMembersController 平移：寻址 {userId}=sys_user.id 经
// (tenant_id,user_id) 找 member；status 过滤 DB 级（total=过滤后）；删 member 不删
// user 行；status 双写 member+user；邀请建真 user（status=2）嵌套视图；角色挂载
// 只收本租户 role 外来静默忽略；跨租户路径 403。

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/testsupport"
)

func membersBase(fx testsupport.AuthFixtures) string {
	return "/api/v1/tenants/" + fx.Tenant.Id + "/members"
}

func createMember(t *testing.T, mux http.Handler, tok string, fx testsupport.AuthFixtures, username string) map[string]any {
	t.Helper()
	rec := doJSON(mux, http.MethodPost, membersBase(fx), tok, map[string]any{
		"username": username, "password": "password8", "email": username + "@test.local",
	})
	if rec.Code != 200 {
		t.Fatalf("createMember 前置失败: %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody(rec)
}

// fn: M00.F02.I01
func TestTenantMembersListEnvelopeAndStatusFilter(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)
	createMember(t, mux, tok, fx, "go_mem_a")
	third := createMember(t, mux, tok, fx, "go_mem_b")
	doJSON(mux, http.MethodPatch, membersBase(fx)+"/"+third["id"].(string)+"/status", tok,
		map[string]string{"status": "invited"})

	rec := getWithBearer(mux, membersBase(fx), tok)
	if rec.Code != 200 {
		t.Fatalf("list 应 200，得到 %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	for _, key := range []string{"items", "page", "pageSize", "total"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("分页信封缺 %s: %v", key, body)
		}
	}
	if body["page"] != float64(0) || body["pageSize"] != float64(20) {
		t.Fatalf("分页默认值应为 0/20: %v %v", body["page"], body["pageSize"])
	}
	if body["total"] != float64(3) {
		t.Fatalf("total 应为 3（alice+2）: %v", body["total"])
	}
	// status 过滤 DB 级：total=过滤后计数
	rec = getWithBearer(mux, membersBase(fx)+"?status=invited", tok)
	filtered := decodeBody(rec)
	if filtered["total"] != float64(1) {
		t.Fatalf("invited 过滤 total 应为 1: %v", filtered["total"])
	}
	// 跨租户路径 → 403 FORBIDDEN
	rec = getWithBearer(mux, "/api/v1/tenants/"+testsupport.NewUUID()+"/members", tok)
	if rec.Code != 403 || decodeBody(rec)["code"] != "FORBIDDEN" {
		t.Fatalf("跨租户路径应 403 FORBIDDEN: %d %s", rec.Code, rec.Body.String())
	}
}

// fn: M00.F02.I02 M00.F02.I03 M00.F02.I04
func TestTenantMembersCreateGetUpdate(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)
	// 缺 email → 400（顺序 username→password→email）
	rec := doJSON(mux, http.MethodPost, membersBase(fx), tok, map[string]any{
		"username": "go_x", "password": "password8",
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "email") {
		t.Fatalf("缺 email 应 400: %d %s", rec.Code, rec.Body.String())
	}
	// 短密码 → 400（契约 @Size(min=8,max=256)）
	rec = doJSON(mux, http.MethodPost, membersBase(fx), tok, map[string]any{
		"username": "go_x", "password": "short", "email": "go_x@test.local",
	})
	if rec.Code != 400 {
		t.Fatalf("短密码应 400: %d %s", rec.Code, rec.Body.String())
	}
	// 正常创建 → 扁平视图
	rec = doJSON(mux, http.MethodPost, membersBase(fx), tok, map[string]any{
		"username": "go_mem_c", "password": "password8", "email": "go_mem_c@test.local", "mobile": "138",
	})
	if rec.Code != 200 {
		t.Fatalf("创建应 200: %d %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	if body["username"] != "go_mem_c" || body["status"] != "active" || body["tenantId"] != fx.Tenant.Id {
		t.Fatalf("扁平视图形状不符: %v", body)
	}
	if ids, ok := body["roleIds"].([]any); !ok || len(ids) != 0 {
		t.Fatalf("roleIds 应为空数组: %v", body["roleIds"])
	}
	userID := body["id"].(string)
	// 详情
	rec = getWithBearer(mux, membersBase(fx)+"/"+userID, tok)
	if rec.Code != 200 || decodeBody(rec)["id"] != userID {
		t.Fatalf("详情应 200 同 id: %d %s", rec.Code, rec.Body.String())
	}
	// 更新只动 email/mobile
	rec = doJSON(mux, http.MethodPatch, membersBase(fx)+"/"+userID, tok,
		map[string]any{"email": "moved@test.local"})
	if rec.Code != 200 || decodeBody(rec)["email"] != "moved@test.local" {
		t.Fatalf("更新 email 应生效: %d %s", rec.Code, rec.Body.String())
	}
}

// fn: M00.F02.I05
func TestTenantMembersDeleteKeepsUserRow(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	created := createMember(t, mux, tok, fx, "go_mem_d")
	userID := created["id"].(string)
	rec := doJSON(mux, http.MethodDelete, membersBase(fx)+"/"+userID, tok, nil)
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("删除应 204 空体: %d %q", rec.Code, rec.Body.String())
	}
	rec = getWithBearer(mux, membersBase(fx)+"/"+userID, tok)
	if rec.Code != 404 {
		t.Fatalf("删后详情应 404: %d", rec.Code)
	}
	// user 行不删
	var n int64
	tx.Model(&entity.SysUser{}).Where("id = ?", userID).Count(&n)
	if n != 1 {
		t.Fatalf("user 行应保留: %d", n)
	}
}

// fn: M00.F02.I06
func TestTenantMembersInviteNestedView(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	// email 空 → 400
	rec := doJSON(mux, http.MethodPost, membersBase(fx)+"/invitations", tok, map[string]string{})
	if rec.Code != 400 {
		t.Fatalf("空 email 应 400: %d", rec.Code)
	}
	rec = doJSON(mux, http.MethodPost, membersBase(fx)+"/invitations", tok,
		map[string]string{"email": "newbie@test.local"})
	if rec.Code != 200 {
		t.Fatalf("邀请应 200: %d %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	member, _ := body["member"].(map[string]any)
	user, _ := body["user"].(map[string]any)
	if member == nil || user == nil {
		t.Fatalf("嵌套视图应有 member/user: %v", body)
	}
	if member["status"] != "active" || member["memberName"] != "newbie@test.local" {
		t.Fatalf("member 形状不符: %v", member)
	}
	if user["username"] != "newbie@test.local" || user["status"] != "invited" {
		t.Fatalf("user 形状不符（username=email, status=invited）: %v", user)
	}
	if roles, ok := body["roles"].([]any); !ok || len(roles) != 0 {
		t.Fatalf("roles 应为空数组: %v", body["roles"])
	}
	// 建的是真 user 行（status=2）
	var n int64
	tx.Model(&entity.SysUser{}).Where("username = ? AND status = 2", "newbie@test.local").Count(&n)
	if n != 1 {
		t.Fatalf("邀请应建真 user 行: %d", n)
	}
}

// fn: M00.F02.I08
func TestTenantMembersChangeStatusDualWrite(t *testing.T) {
	mux, fx, tx := featureMux(t)
	tok := loginForToken(t, mux, fx)
	created := createMember(t, mux, tok, fx, "go_mem_e")
	userID := created["id"].(string)
	rec := doJSON(mux, http.MethodPatch, membersBase(fx)+"/"+userID+"/status", tok,
		map[string]string{"status": "suspended"})
	if rec.Code != 200 || decodeBody(rec)["status"] != "suspended" {
		t.Fatalf("status 切换应 200 suspended: %d %s", rec.Code, rec.Body.String())
	}
	// 双写：member.status 与 user.status 同值
	var member entity.TenantMember
	if err := tx.Where("user_id = ? AND tenant_id = ?", userID, fx.Tenant.Id).First(&member).Error; err != nil {
		t.Fatalf("member 回查失败: %v", err)
	}
	var user entity.SysUser
	if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
		t.Fatalf("user 回查失败: %v", err)
	}
	if member.Status != 3 || user.Status != 3 {
		t.Fatalf("双写不符: member=%d user=%d", member.Status, user.Status)
	}
}

// fn: M01.F02.I01
func TestAssignTenantMemberRolesOwnTenantOnly(t *testing.T) {
	mux, fx, tx := featureMux(t)
	mf := testsupport.SeedMenus(t, tx, fx) // 自带本租户 role
	tok := loginForToken(t, mux, fx)
	created := createMember(t, mux, tok, fx, "go_mem_f")
	userID := created["id"].(string)
	// 外来 role（挂裸租户）+ 未知 id
	foreignTenant := entity.Tenant{
		Id: testsupport.NewUUID(), TenantKey: "go_fx_" + testsupport.NewUUID()[:8],
		Name: "FX", Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := tx.Create(&foreignTenant).Error; err != nil {
		t.Fatalf("外来租户种子失败: %v", err)
	}
	foreignRole := entity.SysRole{
		Id: testsupport.NewUUID(), TenantId: foreignTenant.Id, ClientId: fx.Client.ClientId,
		RoleCode: "fx_role", RoleName: "FX", Status: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := tx.Create(&foreignRole).Error; err != nil {
		t.Fatalf("外来 role 种子失败: %v", err)
	}
	rec := doJSON(mux, http.MethodPut, membersBase(fx)+"/"+userID+"/roles", tok,
		map[string]any{"roleIds": []string{mf.Role.Id, foreignRole.Id, testsupport.NewUUID()}})
	if rec.Code != 200 {
		t.Fatalf("角色挂载应 200: %d %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	ids, _ := body["roleIds"].([]any)
	if len(ids) != 1 || ids[0] != mf.Role.Id {
		t.Fatalf("只应保留本租户 role: %v", body["roleIds"])
	}
	var n int64
	tx.Model(&entity.TenantMemberRole{}).Where("member_id = ?", created["id"]).Count(&n)
	_ = n // member 行 id ≠ user id；下方经 member 查
	var member entity.TenantMember
	if err := tx.Where("user_id = ? AND tenant_id = ?", userID, fx.Tenant.Id).First(&member).Error; err != nil {
		t.Fatalf("member 回查失败: %v", err)
	}
	tx.Model(&entity.TenantMemberRole{}).Where("member_id = ?", member.Id).Count(&n)
	if n != 1 {
		t.Fatalf("member_role 绑定应为 1: %d", n)
	}
}
