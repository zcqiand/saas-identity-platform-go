package oauth

// token.go — OAuth token 双 grant（镜像 rails Oauth::TokenService，M04.F03.I02/I03）：
// - authorization_code：code 一次性消费（过期/redirect 不匹配均删行，重放 400）
// - refresh_token：revoked 400；rotate（旧行标 revoked，新行 client_id 沿用旧行，
//   scope 继承 access token 行）
// 错误形状 {code: INVALID_*, message} 400（springboot ResponseEntity 镜像）。

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/impl/auth"
	"saas-identity-platform-go/internal/impl/ids"
)

// TokenService token 面。
type TokenService struct {
	DB *gorm.DB

	GrantType    string
	ClientID     string
	Code         string
	RefreshToken string
	RedirectURI  string
}

func invalid(code, message string) *familyE { return &familyE{Code: code, Message: message} }

// familyE token 端点错误：400 + {code,message}（与家族错误形状同形）。
type familyE struct {
	Code    string
	Message string
}

func (e *familyE) Error() string { return e.Message }

// AsFamilyE 把 error 还原为 token 端点错误（handler 侧判定用）。
func AsFamilyE(err error) (code, message string, ok bool) {
	var e *familyE
	if errors.As(err, &e) {
		return e.Code, e.Message, true
	}
	return "", "", false
}

// Call 按 grantType 分派。
func (s *TokenService) Call() (map[string]any, error) {
	var client entity.OauthClient
	err := s.DB.Where("client_id = ?", s.ClientID).First(&client).Error
	if err != nil {
		return nil, invalid("INVALID_CLIENT", "unknown clientId")
	}
	switch s.GrantType {
	case "authorization_code":
		return s.exchangeCode()
	case "refresh_token":
		return s.rotateRefresh()
	default:
		return nil, invalid("UNSUPPORTED_GRANT_TYPE", "unsupported grantType "+s.GrantType)
	}
}

func (s *TokenService) exchangeCode() (map[string]any, error) {
	if s.Code == "" {
		return nil, invalid("INVALID_REQUEST", "code is required")
	}
	if s.RedirectURI == "" {
		return nil, invalid("INVALID_REQUEST", "redirectUri is required")
	}
	var row entity.OauthCode
	err := s.DB.Where("code = ?", s.Code).First(&row).Error
	if err != nil || row.ClientId != s.ClientID {
		// code 不存在或绑错 client → 400（不删行，rails find_by nil 分支镜像）
		return nil, invalid("INVALID_GRANT", "code 不存在或已被使用")
	}
	if !row.ExpiresAt.After(time.Now()) {
		_ = s.DB.Where("id = ?", row.Id).Delete(&entity.OauthCode{}).Error
		return nil, invalid("INVALID_GRANT", "expired code")
	}
	if row.RedirectUri == nil || *row.RedirectUri != s.RedirectURI {
		_ = s.DB.Where("id = ?", row.Id).Delete(&entity.OauthCode{}).Error
		return nil, invalid("INVALID_GRANT", "redirectUri mismatch")
	}
	// 一次性消费
	if err := s.DB.Where("id = ?", row.Id).Delete(&entity.OauthCode{}).Error; err != nil {
		return nil, err
	}
	scope := ""
	if row.Scope != nil {
		scope = *row.Scope
	}
	return s.issue(row.ClientId, row.UserId, row.TenantId, scope)
}

func (s *TokenService) rotateRefresh() (map[string]any, error) {
	if s.RefreshToken == "" {
		return nil, invalid("INVALID_REQUEST", "refreshToken is required")
	}
	var row entity.OauthRefreshToken
	err := s.DB.Where("refresh_token = ?", s.RefreshToken).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, invalid("INVALID_GRANT", "refresh token 不存在")
	}
	if err != nil {
		return nil, err
	}
	if row.Revoked {
		return nil, invalid("INVALID_GRANT", "revoked refresh_token")
	}
	scope := ""
	var access entity.OauthAccessToken
	if err := s.DB.Where("id = ?", row.AccessTokenId).First(&access).Error; err == nil && access.Scope != nil {
		scope = *access.Scope
	}
	if err := s.DB.Model(&entity.OauthRefreshToken{}).Where("id = ?", row.Id).
		Update("revoked", true).Error; err != nil {
		return nil, err
	}
	userID, tenantID := "", ""
	if row.UserId != nil {
		userID = *row.UserId
	}
	if row.TenantId != nil {
		tenantID = *row.TenantId
	}
	return s.issue(row.ClientId, userID, tenantID, scope)
}

// issue 新 refresh 落库（"rt_<uuid>" 30 天）+ HS256 access JWT（镜像 persistTokenPair）。
func (s *TokenService) issue(clientID, userID, tenantID, scope string) (map[string]any, error) {
	accessToken, err := auth.Encode(map[string]any{"sub": userID, "tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	access := entity.OauthAccessToken{
		Id:          ids.NewUUID(),
		TokenId:     "at_" + ids.NewUUID(),
		AccessToken: "n/a",
		ClientId:    clientID,
		UserId:      strPtr(userID),
		TenantId:    strPtr(tenantID),
		Scope:       strPtrNonEmpty(scope),
		TokenType:   "Bearer",
		ExpiresAt:   now.Add(time.Hour),
		Revoked:     false,
		CreatedAt:   now,
	}
	if err := s.DB.Create(&access).Error; err != nil {
		return nil, err
	}
	refresh := "rt_" + ids.NewUUID()
	if err := s.DB.Create(&entity.OauthRefreshToken{
		Id:            ids.NewUUID(),
		RefreshToken:  refresh,
		AccessTokenId: access.Id,
		ClientId:      clientID,
		UserId:        strPtr(userID),
		TenantId:      strPtr(tenantID),
		ExpiresAt:     now.Add(30 * 24 * time.Hour),
		Revoked:       false,
		CreatedAt:     now,
	}).Error; err != nil {
		return nil, err
	}
	ttl, err := auth.TTLSeconds()
	if err != nil {
		return nil, err
	}
	// scope 逗号转空格（rails split(',').join(' ') 镜像）
	outScope := ""
	for i, part := range strings.Split(scope, ",") {
		if i > 0 {
			outScope += " "
		}
		outScope += part
	}
	return map[string]any{
		"access_token":  accessToken,
		"refresh_token": refresh,
		"token_type":    "Bearer",
		"expires_in":    ttl,
		"scope":         outScope,
		"user_id":       userID,
		"client_id":     clientID,
		"tenant_id":     tenantID,
	}, nil
}

func strPtr(s string) *string { return &s }

func strPtrNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
