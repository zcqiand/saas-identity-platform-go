package service

// me_service.go — 当前用户面语义（镜像 rails MeController + Me::MenuTree）。
// 全部方法入参 userID/tenantID 取自 JWT claims（守卫已在 handler 入口完成）。

import (
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	"saas-identity-platform-go/internal/impl/auth"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
	"saas-identity-platform-go/internal/impl/service/family"
)

// rootMenuID 菜单树根父 ID（rails SysMenu::ROOT_MENU_ID 同值：零 UUID）。
const rootMenuID = "00000000-0000-0000-0000-000000000000"

// MeService 当前用户面（whoami / tenants / menus / switch）。
type MeService struct {
	DB *gorm.DB
}

// Whoami 当前用户身份 + 全量 memberships（含非 active）+ resolved tenant
// （claim tenant_id 缺失时回落首个 membership——rails resolved_tenant_id 镜像；
// 用户已不存在返回空 memberships 形状，防御分支无身份不报错）。
func (s *MeService) Whoami(userID, claimTenantID string) (map[string]any, error) {
	var user entity.SysUser
	err := s.DB.Where("id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]any{"memberships": []map[string]any{}}, nil
	}
	if err != nil {
		return nil, err
	}
	resolved := claimTenantID
	if resolved == "" {
		var m entity.TenantMember
		err := s.DB.Where("user_id = ?", userID).Order("created_at").First(&m).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
		case err != nil:
			return nil, err
		default:
			resolved = m.TenantId
		}
	}
	ms, err := family.MembershipsFor(s.DB, userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":                user.Id,
		"email":             user.Email,
		"memberships":       ms,
		"current_tenant_id": resolved,
	}, nil
}

// ListMyTenants active memberships 裸数组（clientId 参数被忽略，rails 镜像）。
func (s *MeService) ListMyTenants(userID string) ([]map[string]any, error) {
	var members []entity.TenantMember
	if err := s.DB.Where("user_id = ? AND status = ?", userID, 1).
		Order("created_at").Find(&members).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(members))
	for i := range members {
		m, err := family.Membership(s.DB, &members[i])
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// GetMyMenus 当前用户有效菜单（M04.F04.I08）：member(全量)→member_role→role_menu
// →sys_menu→按 clientId 分组建树。根 parentId 零 UUID → 键剔除；孤儿（parent 不在
// 授权集）提升为 root；仅根层按 sort_order 升序，子层不再排序。
func (s *MeService) GetMyMenus(userID string) (map[string]any, error) {
	var memberIDs []string
	if err := s.DB.Model(&entity.TenantMember{}).Where("user_id = ?", userID).
		Pluck("id", &memberIDs).Error; err != nil {
		return nil, err
	}
	if len(memberIDs) == 0 {
		return map[string]any{}, nil
	}
	var roleIDs []string
	if err := s.DB.Model(&entity.TenantMemberRole{}).Where("member_id IN ?", memberIDs).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return map[string]any{}, nil
	}
	var menuIDs []string
	if err := s.DB.Model(&entity.SysRoleMenu{}).Where("role_id IN ?", roleIDs).
		Distinct().Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, err
	}
	if len(menuIDs) == 0 {
		return map[string]any{}, nil
	}
	var menus []entity.SysMenu
	if err := s.DB.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
		return nil, err
	}
	grouped := map[string][]entity.SysMenu{}
	for _, m := range menus {
		grouped[m.ClientId] = append(grouped[m.ClientId], m)
	}
	out := map[string]any{}
	for clientID, rows := range grouped {
		out[clientID] = buildMenuTree(rows)
	}
	return out, nil
}

// menuNode 单节点形状（snake 键交给 render.Camelize；nil 值键自动剔除）。
func menuNode(m *entity.SysMenu) map[string]any {
	var parentID any
	if m.ParentId != rootMenuID {
		parentID = m.ParentId
	}
	return map[string]any{
		"id":         m.Id,
		"client_id":  m.ClientId,
		"parent_id":  parentID,
		"title":      m.Title,
		"type":       family.MenuType(m.Type),
		"path":       m.Path,
		"component":  m.Component,
		"perms":      m.Perms,
		"icon":       m.Icon,
		"sort_order": m.SortOrder,
		"children":   []map[string]any{},
	}
}

func buildMenuTree(rows []entity.SysMenu) []map[string]any {
	nodes := make(map[string]map[string]any, len(rows))
	for i := range rows {
		nodes[rows[i].Id] = menuNode(&rows[i])
	}
	roots := []map[string]any{}
	for i := range rows {
		m := &rows[i]
		if m.ParentId == rootMenuID {
			roots = append(roots, nodes[m.Id])
			continue
		}
		parent, ok := nodes[m.ParentId]
		if !ok { // 孤儿（parent 不在授权集）提升为 root
			roots = append(roots, nodes[m.Id])
			continue
		}
		parent["children"] = append(parent["children"].([]map[string]any), nodes[m.Id])
	}
	sort.SliceStable(roots, func(a, b int) bool {
		sa, _ := roots[a]["sort_order"].(int32)
		sb, _ := roots[b]["sort_order"].(int32)
		return sa < sb
	})
	return roots
}

// SwitchTenant 切换租户：tenant 存在否则 404；member 存在且 status≠0 否则 404
// （invited/suspended 可切）。新 JWT 换 tenant_id；refresh 临时形状不落库。
func (s *MeService) SwitchTenant(userID, tenantID string) (map[string]any, error) {
	if !ids.IsValidUUID(tenantID) {
		return nil, familyerrors.ErrBadRequest("malformed uuid: " + tenantID)
	}
	var tenant entity.Tenant
	err := s.DB.Where("id = ?", tenantID).First(&tenant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("tenant not found")
	}
	if err != nil {
		return nil, err
	}
	var member entity.TenantMember
	err = s.DB.Where("user_id = ? AND tenant_id = ?", userID, tenant.Id).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && member.Status == 0) {
		return nil, familyerrors.ErrNotFound("membership not found")
	}
	if err != nil {
		return nil, err
	}
	token, err := auth.Encode(map[string]any{"sub": userID, "tenant_id": tenant.Id})
	if err != nil {
		return nil, err
	}
	ttl, err := auth.TTLSeconds()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"access_token":  token,
		"refresh_token": family.EphemeralRefreshToken(userID),
		"expires_at":    time.Now().Add(time.Duration(ttl) * time.Second),
		"tenant_id":     tenant.Id,
	}, nil
}
