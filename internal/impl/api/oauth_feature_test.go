package api_test

// oauth_feature_test.go — Task 10 oauth 面（M04.F03：authorize 码流 + token
// authorization_code/refresh_token 双 grant）。authorize 手动 Bearer decode；
// token 端点匿名（SecurityConfig permitAll 镜像）。code 一次性消费。

import (
	"net/http"
	"strings"
	"testing"

	"saas-identity-platform-go/internal/impl/auth"
)

// signAccessClaims 构造测试用 JWT（auth.Encode 注入 iat/exp/iss/aud）。
func signAccessClaims(payload map[string]any) (string, error) {
	return auth.Encode(payload)
}

// verifyAccessClaims 验签并返回 claims。
func verifyAccessClaims(token string) (map[string]any, error) {
	return auth.Verify(token)
}

const oauthBase = "/api/v1/oauth"

// fn: M04.F03.I01
// fn: M04.F03.I02
// fn: M04.F03.I03
func TestOauthAuthorizeAndTokenGrants(t *testing.T) {
	mux, fx, _ := featureMux(t)
	tok := loginForToken(t, mux, fx)

	// ---- authorize：错误路径 ----
	rec := doJSON(mux, http.MethodPost, oauthBase+"/authorize", tok, map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://localhost/callback",
		"responseType": "token", "state": "s1",
	})
	if rec.Code != 400 {
		t.Fatalf("responseType 非 code 应 400，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, oauthBase+"/authorize", tok, map[string]any{
		"clientId": "no-such", "redirectUri": "http://localhost/callback",
		"responseType": "code", "state": "s1",
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_CLIENT") {
		t.Fatalf("未知 clientId 应 400 INVALID_CLIENT，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, oauthBase+"/authorize", tok, map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://evil.example/cb",
		"responseType": "code", "state": "s1",
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_REDIRECT_URI") {
		t.Fatalf("白名单外 redirect 应 400，得 %d %s", rec.Code, rec.Body.String())
	}
	// 白名单边界：精确相等/前缀+? 算；子路径不算
	rec = doJSON(mux, http.MethodPost, oauthBase+"/authorize", tok, map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://localhost/callback/extra",
		"responseType": "code", "state": "s1",
	})
	if rec.Code != 400 {
		t.Fatalf("子路径 redirect 不应放行，得 %d %s", rec.Code, rec.Body.String())
	}

	// ---- authorize：成功（前缀+? 边界放行）----
	rec = doJSON(mux, http.MethodPost, oauthBase+"/authorize", tok, map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://localhost/callback?next=/a",
		"responseType": "code", "scope": "read,write", "state": "st-9",
	})
	if rec.Code != 200 {
		t.Fatalf("authorize 应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	authBody := decodeBody(rec)
	code, _ := authBody["code"].(string)
	if !strings.HasPrefix(code, "ac_") {
		t.Fatalf("code 应 ac_ 前缀，得 %v", authBody["code"])
	}
	if authBody["state"] != "st-9" {
		t.Fatalf("state 应回显，得 %v", authBody["state"])
	}

	// ---- token authorization_code：redirect 不匹配 → 消费 code → 400 ----
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "authorization_code", "clientId": fx.Client.ClientId,
		"code": code, "redirectUri": "http://localhost/callback?other=1",
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_GRANT") {
		t.Fatalf("redirect 不匹配应 400 INVALID_GRANT，得 %d %s", rec.Code, rec.Body.String())
	}
	// 重放已消费 code → 400
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "authorization_code", "clientId": fx.Client.ClientId,
		"code": code, "redirectUri": "http://localhost/callback?next=/a",
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_GRANT") {
		t.Fatalf("重放已消费 code 应 400，得 %d %s", rec.Code, rec.Body.String())
	}

	// ---- token authorization_code：完整成功流 ----
	rec = doJSON(mux, http.MethodPost, oauthBase+"/authorize", tok, map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://localhost/callback",
		"responseType": "code", "scope": "read,write", "state": "s2",
	})
	code2, _ := decodeBody(rec)["code"].(string)
	if code2 == "" {
		t.Fatal("第二次 authorize 未返回 code")
	}
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "authorization_code", "clientId": fx.Client.ClientId,
		"code": code2, "redirectUri": "http://localhost/callback",
	})
	if rec.Code != 200 {
		t.Fatalf("换码应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	pair := decodeBody(rec)
	access, _ := pair["accessToken"].(string)
	refresh, _ := pair["refreshToken"].(string)
	if access == "" || refresh == "" || pair["tokenType"] != "Bearer" {
		t.Fatalf("token 对形状不符: %v", pair)
	}
	if pair["scope"] != "read write" {
		t.Fatalf("scope 逗号应转空格，得 %v", pair["scope"])
	}
	if pair["userId"] != fx.User.Id || pair["tenantId"] != fx.Tenant.Id {
		t.Fatalf("token 归属不符: %v", pair)
	}
	claims, err := verifyAccessClaims(access)
	if err != nil {
		t.Fatalf("access 应为可验 HS256 JWT: %v", err)
	}
	if claims["sub"] != fx.User.Id || claims["tenant_id"] != fx.Tenant.Id {
		t.Fatalf("JWT claims 不符: %v", claims)
	}

	// ---- refresh_token grant：rotate + 旧 token 重放 400 ----
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "refresh_token", "clientId": fx.Client.ClientId,
		"refreshToken": refresh,
	})
	if rec.Code != 200 {
		t.Fatalf("refresh 应 200，得 %d %s", rec.Code, rec.Body.String())
	}
	pair2 := decodeBody(rec)
	refresh2, _ := pair2["refreshToken"].(string)
	if refresh2 == "" || refresh2 == refresh {
		t.Fatalf("rotate 应发新 refresh，得 %v", pair2["refreshToken"])
	}
	if pair2["scope"] != "read write" {
		t.Fatalf("rotate scope 应继承 access 行，得 %v", pair2["scope"])
	}
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "refresh_token", "clientId": fx.Client.ClientId,
		"refreshToken": refresh, // 旧行已 revoked
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "revoked") {
		t.Fatalf("重放 revoked refresh 应 400，得 %d %s", rec.Code, rec.Body.String())
	}

	// ---- token 杂项错误 ----
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "password", "clientId": fx.Client.ClientId,
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "UNSUPPORTED_GRANT_TYPE") {
		t.Fatalf("不支持 grant 应 400 UNSUPPORTED，得 %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(mux, http.MethodPost, oauthBase+"/token", "", map[string]any{
		"grantType": "authorization_code", "clientId": "no-such", "code": "ac_x",
		"redirectUri": "http://localhost/callback",
	})
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "INVALID_CLIENT") {
		t.Fatalf("未知 client 应 400 INVALID_CLIENT，得 %d %s", rec.Code, rec.Body.String())
	}
}

// fn: M04.F03.I01
func TestOauthAuthorizeRequiresBearerWithTenant(t *testing.T) {
	mux, fx, _ := featureMux(t)
	// 匿名 → 401
	rec := doJSON(mux, http.MethodPost, oauthBase+"/authorize", "", map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://localhost/callback",
		"responseType": "code", "state": "s",
	})
	if rec.Code != 401 {
		t.Fatalf("匿名 authorize 应 401，得 %d", rec.Code)
	}
	// 有效 JWT 但无 tenant_id claim → 401（decode_bearer! 镜像）
	bare, err := signAccessClaims(map[string]any{"sub": fx.User.Id})
	if err != nil {
		t.Fatalf("构造无租户 JWT 失败: %v", err)
	}
	rec = doJSON(mux, http.MethodPost, oauthBase+"/authorize", bare, map[string]any{
		"clientId": fx.Client.ClientId, "redirectUri": "http://localhost/callback",
		"responseType": "code", "state": "s",
	})
	if rec.Code != 401 {
		t.Fatalf("无 tenant_id claim 应 401，得 %d %s", rec.Code, rec.Body.String())
	}
}
