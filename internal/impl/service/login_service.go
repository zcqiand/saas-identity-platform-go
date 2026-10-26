// Package service — 业务服务层（镜像 rails app/services/*）。handler 只装配，
// 语义全在 service（CLAUDE.md §2 禁止 handler 写业务）。
package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/impl/auth"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
	"saas-identity-platform-go/internal/impl/service/family"
)

// 锁定阈值（rails SysUser::LOCKOUT_THRESHOLD / LOCKOUT_MINUTES 同值：5 次 / 15 分钟）。
const (
	LockoutThreshold = 5
	LockoutMinutes   = 15
)

// LoginService 密码登录语义（镜像 rails Auth::LoginService，M01.F04.I01）：
// 未知用户与错密码同 401（不泄露存在性）；锁窗口内正确密码也 423；成功清零计数。
type LoginService struct {
	DB       *gorm.DB
	Username string
	Password string
	ClientID string
}

// Call 执行登录链，返回 render.Camelize 形状的 payload（snake 键）。
func (s *LoginService) Call() (map[string]any, error) {
	user, err := s.verifyCredentials()
	if err != nil {
		return nil, err
	}
	if err := s.DB.Model(&entity.SysUser{}).Where("id = ?", user.Id).
		Updates(map[string]any{"failed_attempts": 0, "locked_until": nil}).Error; err != nil {
		return nil, err
	}
	// 租户先解析（镜像 springboot AuthController.login 顺序：tenant check → persist 内验 clientId）
	tenantID, err := s.currentTenantID(user)
	if err != nil {
		return nil, err
	}
	var client entity.OauthClient
	if err := s.DB.Where("client_id = ?", s.ClientID).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, familyerrors.ErrBadRequest("unknown clientId")
		}
		return nil, err
	}
	refreshToken, err := s.persistTokenPair(user, tenantID)
	if err != nil {
		return nil, err
	}
	return s.loginPayload(user, tenantID, refreshToken)
}

func (s *LoginService) verifyCredentials() (*entity.SysUser, error) {
	var user entity.SysUser
	err := s.DB.Where("username = ?", s.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrInvalidCredentials()
	}
	if err != nil {
		return nil, err
	}
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, familyerrors.ErrLocked()
	}
	if !auth.PasswordMatches(user.Password, s.Password) {
		if err := s.registerFailure(&user); err != nil {
			return nil, err
		}
		return nil, familyerrors.ErrInvalidCredentials()
	}
	return &user, nil
}

// registerFailure 失败计数（M01.F04.I01 合并的失败锁定）：达阈值置 locked_until。
func (s *LoginService) registerFailure(user *entity.SysUser) error {
	attempts := user.FailedAttempts + 1
	updates := map[string]any{"failed_attempts": attempts, "locked_until": nil}
	if attempts >= LockoutThreshold {
		updates["locked_until"] = time.Now().Add(LockoutMinutes * time.Minute)
	}
	return s.DB.Model(&entity.SysUser{}).Where("id = ?", user.Id).Updates(updates).Error
}

// currentTenantID 首个 active membership 且 tenant 本身 active，否则 403
// （家族真源：禁 200 + null tenant 兜底，runtime-identity-fallback 禁令）。
func (s *LoginService) currentTenantID(user *entity.SysUser) (string, error) {
	var member entity.TenantMember
	err := s.DB.Where("user_id = ? AND status = ?", user.Id, 1).
		Order("created_at").First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", familyerrors.ErrForbidden("no active tenant membership")
	}
	if err != nil {
		return "", err
	}
	var tenant entity.Tenant
	if err := s.DB.Where("id = ?", member.TenantId).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", familyerrors.ErrForbidden("no active tenant")
		}
		return "", err
	}
	if tenant.Status != 1 {
		return "", familyerrors.ErrForbidden("no active tenant")
	}
	return member.TenantId, nil
}

// persistTokenPair 落库 token 对（镜像 springboot persistTokenPair）：access 行
// access_token="n/a"，refresh 行 "rt_<uuid>" 30 天，两行都带 tenantId（refresh
// rotate 继承 → 响应必有 tenantId；漏写 = normalize 剔 nil 后四方比对分叉）。
func (s *LoginService) persistTokenPair(user *entity.SysUser, tenantID string) (string, error) {
	now := time.Now()
	at := entity.OauthAccessToken{
		Id:          ids.NewUUID(),
		TokenId:     "at_" + ids.NewUUID(),
		AccessToken: "n/a",
		ClientId:    s.ClientID,
		UserId:      &user.Id,
		TenantId:    &tenantID,
		TokenType:   "Bearer",
		ExpiresAt:   now.Add(time.Hour),
		Revoked:     false,
		CreatedAt:   now,
	}
	if err := s.DB.Create(&at).Error; err != nil {
		return "", err
	}
	refreshToken := "rt_" + ids.NewUUID()
	rt := entity.OauthRefreshToken{
		Id:            ids.NewUUID(),
		RefreshToken:  refreshToken,
		AccessTokenId: at.Id,
		ClientId:      s.ClientID,
		UserId:        &user.Id,
		TenantId:      &tenantID,
		ExpiresAt:     now.Add(30 * 24 * time.Hour),
		Revoked:       false,
		CreatedAt:     now,
	}
	if err := s.DB.Create(&rt).Error; err != nil {
		return "", err
	}
	return refreshToken, nil
}

func (s *LoginService) loginPayload(user *entity.SysUser, tenantID, refreshToken string) (map[string]any, error) {
	token, err := auth.Encode(map[string]any{"sub": user.Id, "tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	ttl, err := auth.TTLSeconds()
	if err != nil {
		return nil, err
	}
	// availableTenants：active memberships；clientId 有订阅集则过滤到订阅内，订阅集空则不过滤
	var subscribed []string
	if err := s.DB.Model(&entity.TenantApplication{}).
		Where("client_id = ?", s.ClientID).Pluck("tenant_id", &subscribed).Error; err != nil {
		return nil, err
	}
	q := s.DB.Where("user_id = ? AND status = ?", user.Id, 1)
	if len(subscribed) > 0 {
		q = q.Where("tenant_id IN ?", subscribed)
	}
	var members []entity.TenantMember
	if err := q.Order("created_at").Find(&members).Error; err != nil {
		return nil, err
	}
	available := make([]map[string]any, 0, len(members))
	for i := range members {
		m, err := family.Membership(s.DB, &members[i])
		if err != nil {
			return nil, err
		}
		available = append(available, m)
	}
	return map[string]any{
		"user": map[string]any{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
		},
		"user_id":           user.Id,
		"current_tenant_id": tenantID,
		"access_token":      token,
		"refresh_token":     refreshToken,
		"token_type":        "Bearer",
		"expires_in":        ttl,
		"client_id":         s.ClientID,
		"available_tenants": available,
	}, nil
}
