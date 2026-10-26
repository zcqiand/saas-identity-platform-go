package testsupport

// fixtures.go — Feature 测试种子（真库链 saas_test，事务回滚保种子，无 fixtures 文件；
// 镜像 rails test/integration/support.rb 的 AC 程序化思路）。所有业务键带随机后缀，
// 与库内家族种子数据（alice / saas-console）互不冲突。

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
)

// AuthFixtures 一套登录链种子：user + tenant + active membership + oauth client。
type AuthFixtures struct {
	User     entity.SysUser
	Tenant   entity.Tenant
	Member   entity.TenantMember
	Client   entity.OauthClient
	Password string // 明文（User.Password 存的是 plain: 形式）
}

// SeedAuth 落一套登录链种子并返回实体（含生成键）。须在 Tx 内调用。
func SeedAuth(t *testing.T, db *gorm.DB) AuthFixtures {
	t.Helper()
	sfx := NewUUID()[:8]
	fx := AuthFixtures{
		Password: "dev123456",
		User: entity.SysUser{
			Id:             NewUUID(),
			Username:       "go_alice_" + sfx,
			Password:       "plain:dev123456",
			Email:          Ptr("go_alice_" + sfx + "@test.local"),
			Status:         1,
			FailedAttempts: 0,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		Tenant: entity.Tenant{
			Id:        NewUUID(),
			TenantKey: "go_t_" + sfx,
			Name:      "Go Test Tenant " + sfx,
			Status:    1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Client: entity.OauthClient{
			Id:                   NewUUID(),
			ClientId:             "go-console-" + sfx,
			ClientSecret:         "go-secret-" + sfx,
			ClientName:           "Go Test Client " + sfx,
			GrantTypes:           "password authorization_code refresh_token",
			RedirectUris:         "http://localhost/callback",
			AccessTokenValidity:  3600,
			RefreshTokenValidity: 2592000,
			AutoApprove:          false,
			Status:               1,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		},
	}
	fx.Member = entity.TenantMember{
		Id:        NewUUID(),
		TenantId:  fx.Tenant.Id,
		UserId:    fx.User.Id,
		IsOwner:   true,
		Status:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	for _, seed := range []func() *gorm.DB{
		func() *gorm.DB { return db.Create(&fx.User) },
		func() *gorm.DB { return db.Create(&fx.Tenant) },
		func() *gorm.DB { return db.Create(&fx.Client) },
		func() *gorm.DB { return db.Create(&fx.Member) },
	} {
		if err := seed().Error; err != nil {
			t.Fatalf("SeedAuth 种子失败: %v", err)
		}
	}
	return fx
}
