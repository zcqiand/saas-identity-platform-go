package oauth

// authorize.go — OAuth authorize 码流语义（镜像 rails Oauth::AuthorizeService，
// M04.F03.I01）：Bearer 手动 decode（handler 侧），client 不存在 400、redirect
// 白名单精确相等或前缀+? 边界（子路径不算）、code="ac_<uuid>" 5 分钟过期。

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
)

// AuthorizeService authorize 面。
type AuthorizeService struct {
	DB *gorm.DB

	ClientID     string
	RedirectURI  string
	ResponseType string
	Scope        string
	State        string
	// UserID/TenantID 来自 Bearer claims（handler 解出后注入）。
	UserID   string
	TenantID string
}

// Call 校验链 → 落 oauth_code → {code, state}。
func (s *AuthorizeService) Call() (map[string]any, error) {
	if s.ResponseType != "code" {
		return nil, familyerrors.ErrBadRequest(`responseType: must be "code"`)
	}
	var client entity.OauthClient
	err := s.DB.Where("client_id = ?", s.ClientID).First(&client).Error
	if err != nil {
		return nil, familyerrors.ErrBadRequest("INVALID_CLIENT: unknown clientId " + s.ClientID)
	}
	if !redirectAllowed(client.RedirectUris, s.RedirectURI) {
		return nil, familyerrors.ErrBadRequest("INVALID_REDIRECT_URI: " + s.RedirectURI)
	}
	code := "ac_" + ids.NewUUID()
	var scope *string
	if s.Scope != "" {
		scope = &s.Scope
	}
	if err := s.DB.Create(&entity.OauthCode{
		Id:          ids.NewUUID(),
		Code:        code,
		ClientId:    s.ClientID,
		UserId:      s.UserID,
		TenantId:    s.TenantID,
		RedirectUri: &s.RedirectURI,
		Scope:       scope,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		CreatedAt:   time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	return map[string]any{"code": code, "state": s.State}, nil
}

// redirectAllowed 精确相等 或 前缀+? 边界（子路径不算）。
func redirectAllowed(redirectUris, uri string) bool {
	if uri == "" {
		return false
	}
	for _, allowed := range strings.Split(redirectUris, ",") {
		allowed = strings.TrimSpace(allowed)
		if uri == allowed {
			return true
		}
		if strings.HasPrefix(uri, allowed) && len(uri) > len(allowed) && uri[len(allowed)] == '?' {
			return true
		}
	}
	return false
}
