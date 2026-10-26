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

// RootMenuID 菜单树根父 ID（rails SysMenu::ROOT_MENU_ID 同值：零 UUID）。
const RootMenuID = "00000000-0000-0000-0000-000000000000"

// MenuFixtures 一条 member→role→menus 授权链：root 目录（sort_order 2）+
// 其子菜单（sort_order 1，验证仅根层排序）。
type MenuFixtures struct {
	Role      entity.SysRole
	DirMenu   entity.SysMenu
	ChildMenu entity.SysMenu
}

// SeedMenus 在 fx.Member 名下落授权链并返回菜单实体。
func SeedMenus(t *testing.T, db *gorm.DB, fx AuthFixtures) MenuFixtures {
	t.Helper()
	now := time.Now()
	mf := MenuFixtures{
		Role: entity.SysRole{
			Id:        NewUUID(),
			TenantId:  fx.Tenant.Id,
			ClientId:  fx.Client.ClientId,
			RoleCode:  "go_role",
			RoleName:  "Go Role",
			IsPreset:  false,
			Status:    1,
			CreatedAt: now,
			UpdatedAt: now,
		},
		DirMenu: entity.SysMenu{
			Id:        NewUUID(),
			ClientId:  fx.Client.ClientId,
			ParentId:  RootMenuID,
			Title:     "目录甲",
			Type:      1,
			Path:      Ptr("/dir"),
			SortOrder: 2,
			Status:    1,
			CreatedAt: now,
		},
		ChildMenu: entity.SysMenu{
			Id:        NewUUID(),
			ClientId:  fx.Client.ClientId,
			ParentId:  "",
			Title:     "子菜单乙",
			Type:      2,
			Path:      Ptr("/dir/child"),
			SortOrder: 1,
			Status:    1,
			CreatedAt: now,
		},
	}
	// ChildMenu.ParentId 依赖 DirMenu.Id，单独赋值避开结构体字面量顺序
	mf.ChildMenu.ParentId = mf.DirMenu.Id
	seeds := []func() *gorm.DB{
		func() *gorm.DB { return db.Create(&mf.Role) },
		func() *gorm.DB { return db.Create(&mf.DirMenu) },
		func() *gorm.DB { return db.Create(&mf.ChildMenu) },
		func() *gorm.DB {
			return db.Create(&entity.TenantMemberRole{MemberId: fx.Member.Id, RoleId: mf.Role.Id})
		},
		func() *gorm.DB {
			return db.Create(&entity.SysRoleMenu{RoleId: mf.Role.Id, MenuId: mf.DirMenu.Id})
		},
		func() *gorm.DB {
			return db.Create(&entity.SysRoleMenu{RoleId: mf.Role.Id, MenuId: mf.ChildMenu.Id})
		},
	}
	for _, seed := range seeds {
		if err := seed().Error; err != nil {
			t.Fatalf("SeedMenus 种子失败: %v", err)
		}
	}
	return mf
}

// SeedSecondTenant 给 fx.User 追加一个第二租户 + 指定 status 的 membership
// （switch 链用：invited=2 可切、suspended=3 可切、disabled=0 不可切）。
func SeedSecondTenant(t *testing.T, db *gorm.DB, fx AuthFixtures, memberStatus int16) entity.Tenant {
	t.Helper()
	now := time.Now()
	tenant := entity.Tenant{
		Id:        NewUUID(),
		TenantKey: "go_t2_" + NewUUID()[:8],
		Name:      "Go Second Tenant",
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	member := entity.TenantMember{
		Id:        NewUUID(),
		TenantId:  tenant.Id,
		UserId:    fx.User.Id,
		Status:    memberStatus,
		CreatedAt: now.Add(time.Second), // 晚于首个 membership，Order(created_at) 稳定
		UpdatedAt: now,
	}
	for _, seed := range []func() *gorm.DB{
		func() *gorm.DB { return db.Create(&tenant) },
		func() *gorm.DB { return db.Create(&member) },
	} {
		if err := seed().Error; err != nil {
			t.Fatalf("SeedSecondTenant 种子失败: %v", err)
		}
	}
	return tenant
}
