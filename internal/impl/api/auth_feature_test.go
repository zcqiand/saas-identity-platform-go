package api_test

// auth_feature_test.go — M01.F04 登录/登出 Feature 测试（真库 saas_test，事务回滚隔离）。
// 语义逐 rails Auth::LoginService + test/integration/auth_flow_test.rb 平移：
// 未知用户与错密码同 401；连续 5 错锁 15min → 423 空体；成功清零计数；
// 无 active membership → 403（禁 200+null tenant 兜底）；logout 恒 204。

import (
	"strings"
	"testing"
	"time"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/testsupport"
)

// fn: M01.F04.I01
func TestLoginSuccess200FullShape(t *testing.T) {
	mux, fx, tx := featureMux(t)
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": fx.User.Username, "password": fx.Password, "clientId": fx.Client.ClientId,
	})
	if rec.Code != 200 {
		t.Fatalf("正确登录应 200，得到 %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(rec)
	if body["tokenType"] != "Bearer" {
		t.Fatalf("tokenType 应为 Bearer: %v", body["tokenType"])
	}
	if body["expiresIn"] != float64(3600) {
		t.Fatalf("expiresIn 应为 3600: %v", body["expiresIn"])
	}
	if body["clientId"] != fx.Client.ClientId {
		t.Fatalf("clientId 回显不符: %v", body["clientId"])
	}
	if body["userId"] != fx.User.Id {
		t.Fatalf("userId 不符: %v", body["userId"])
	}
	if body["currentTenantId"] != fx.Tenant.Id {
		t.Fatalf("currentTenantId 不符: %v", body["currentTenantId"])
	}
	access, _ := body["accessToken"].(string)
	if len(strings.Split(access, ".")) != 3 {
		t.Fatalf("accessToken 应为三段 JWT: %q", access)
	}
	refresh, _ := body["refreshToken"].(string)
	if !strings.HasPrefix(refresh, "rt_") {
		t.Fatalf("refreshToken 应为 rt_ 前缀: %q", refresh)
	}
	user, _ := body["user"].(map[string]any)
	if user["username"] != fx.User.Username {
		t.Fatalf("user.username 不符: %v", user["username"])
	}
	tenants, ok := body["availableTenants"].([]any)
	if !ok || len(tenants) != 1 {
		t.Fatalf("availableTenants 应为单元素数组: %v", body["availableTenants"])
	}
	m0, _ := tenants[0].(map[string]any)
	if m0["tenantId"] != fx.Tenant.Id || m0["status"] != "active" {
		t.Fatalf("membership 形状不符: %v", m0)
	}
	if _, ok := m0["joinedAt"]; !ok {
		t.Fatalf("membership 应含 joinedAt: %v", m0)
	}

	// refresh token 落库形状（镜像 rails persist_token_pair：30 天、带 tenantId、未撤销）
	var rt entity.OauthRefreshToken
	if err := tx.Where("refresh_token = ?", refresh).First(&rt).Error; err != nil {
		t.Fatalf("refresh 行未落库: %v", err)
	}
	if rt.Revoked || rt.TenantId == nil || *rt.TenantId != fx.Tenant.Id || rt.ClientId != fx.Client.ClientId {
		t.Fatalf("refresh 行形状不符: %+v", rt)
	}
	if !rt.ExpiresAt.After(time.Now().Add(29 * 24 * time.Hour)) {
		t.Fatalf("refresh 有效期应约 30 天: %v", rt.ExpiresAt)
	}
	var at entity.OauthAccessToken
	if err := tx.Where("id = ?", rt.AccessTokenId).First(&at).Error; err != nil {
		t.Fatalf("access 行未落库: %v", err)
	}
	if !strings.HasPrefix(at.TokenId, "at_") || at.AccessToken != "n/a" {
		t.Fatalf("access 行形状不符: token_id=%q access_token=%q", at.TokenId, at.AccessToken)
	}
}

// fn: M01.F04.I01
func TestLoginWrongPasswordAndUnknownUser401SameShape(t *testing.T) {
	mux, fx, _ := featureMux(t)
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": fx.User.Username, "password": "nope", "clientId": fx.Client.ClientId,
	})
	if rec.Code != 401 || decodeBody(rec)["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("错密码应 401 INVALID_CREDENTIALS: %d %s", rec.Code, rec.Body.String())
	}
	rec = postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": "ghost_" + testsupport.NewUUID()[:8], "password": "x", "clientId": fx.Client.ClientId,
	})
	if rec.Code != 401 || decodeBody(rec)["code"] != "INVALID_CREDENTIALS" {
		t.Fatalf("未知用户应同形状 401: %d %s", rec.Code, rec.Body.String())
	}
}

// fn: M01.F04.I01
func TestLoginUnknownClientId400(t *testing.T) {
	mux, fx, _ := featureMux(t)
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": fx.User.Username, "password": fx.Password, "clientId": "ghost-app",
	})
	if rec.Code != 400 {
		t.Fatalf("未知 clientId 应 400，得到 %d: %s", rec.Code, rec.Body.String())
	}
}

// fn: M01.F04.I01
func TestLoginFiveFailuresLock423EmptyBody(t *testing.T) {
	mux, fx, tx := featureMux(t)
	lockee := fx.User // 复用种子用户，走完 5 错→423→库内 locked_until 落库
	for i := 0; i < 5; i++ {
		rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
			"username": lockee.Username, "password": "wrong", "clientId": fx.Client.ClientId,
		})
		if rec.Code != 401 {
			t.Fatalf("第 %d 次错密码应 401，得到 %d", i+1, rec.Code)
		}
	}
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": lockee.Username, "password": fx.Password, "clientId": fx.Client.ClientId,
	})
	if rec.Code != 423 {
		t.Fatalf("锁窗口内正确密码也应 423，得到 %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("423 应空体: %q", rec.Body.String())
	}
	var stored entity.SysUser
	if err := tx.Where("id = ?", lockee.Id).First(&stored).Error; err != nil {
		t.Fatalf("用户回查失败: %v", err)
	}
	if stored.FailedAttempts != 5 || stored.LockedUntil == nil || !stored.LockedUntil.After(time.Now()) {
		t.Fatalf("锁定状态未落库: attempts=%d locked_until=%v", stored.FailedAttempts, stored.LockedUntil)
	}
}

// fn: M01.F04.I01
func TestLoginSuccessResetsFailureCount(t *testing.T) {
	mux, fx, tx := featureMux(t)
	for i := 0; i < 2; i++ {
		postJSON(mux, "/api/v1/auth/login", map[string]string{
			"username": fx.User.Username, "password": "wrong", "clientId": fx.Client.ClientId,
		})
	}
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": fx.User.Username, "password": fx.Password, "clientId": fx.Client.ClientId,
	})
	if rec.Code != 200 {
		t.Fatalf("2 错后正确密码仍应 200（未到阈值），得到 %d", rec.Code)
	}
	var stored entity.SysUser
	if err := tx.Where("id = ?", fx.User.Id).First(&stored).Error; err != nil {
		t.Fatalf("用户回查失败: %v", err)
	}
	if stored.FailedAttempts != 0 || stored.LockedUntil != nil {
		t.Fatalf("成功登录应清零计数: attempts=%d locked_until=%v", stored.FailedAttempts, stored.LockedUntil)
	}
}

// fn: M01.F04.I01
func TestLoginNoActiveMembership403(t *testing.T) {
	mux, fx, tx := featureMux(t)
	loner := fx.User
	loner.Id = testsupport.NewUUID()
	loner.Username = "go_loner_" + testsupport.NewUUID()[:8]
	loner.Email = testsupport.Ptr("go_loner_" + testsupport.NewUUID()[:8] + "@test.local")
	if err := tx.Create(&loner).Error; err != nil {
		t.Fatalf("无 membership 用户种子失败: %v", err)
	}
	rec := postJSON(mux, "/api/v1/auth/login", map[string]string{
		"username": loner.Username, "password": fx.Password, "clientId": fx.Client.ClientId,
	})
	if rec.Code != 403 || decodeBody(rec)["code"] != "FORBIDDEN" {
		t.Fatalf("无 active membership 应 403 FORBIDDEN（禁 200+null tenant），得到 %d %s", rec.Code, rec.Body.String())
	}
}

// fn: M01.F04.I06
func TestLogoutIs204Empty(t *testing.T) {
	mux, _, _ := featureMux(t)
	rec := postJSON(mux, "/api/v1/auth/logout", map[string]string{})
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("logout 应 204 空体，得到 %d %q", rec.Code, rec.Body.String())
	}
}
