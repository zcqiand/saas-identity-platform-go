package family

// member_views.go — 会员视图装配（镜像 rails Family::MemberViews / springboot
// MemberViewAssembler）。roleIds 三处同源（members 视图 / login.availableTenants /
// me）：member_role join 原样，不按 sys_role.tenant_id 过滤——赋权端点负责只收本租户 role。

import (
	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
)

// RoleIds member 的角色 ID 列表（tenant_member_role join 原样，按 role_id 序）。
func RoleIds(db *gorm.DB, memberID string) ([]string, error) {
	var ids []string
	if err := db.Model(&entity.TenantMemberRole{}).
		Where("member_id = ?", memberID).
		Order("role_id").
		Pluck("role_id", &ids).Error; err != nil {
		return nil, err
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// Membership TenantMembership 通用项：login.availableTenants / me.tenants /
// me.memberships 共用（snake 键交给 render.Camelize）。
func Membership(db *gorm.DB, m *entity.TenantMember) (map[string]any, error) {
	roleIds, err := RoleIds(db, m.Id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":        m.Id,
		"user_id":   m.UserId,
		"tenant_id": m.TenantId,
		"role_ids":  roleIds,
		"status":    MemberStatus(m.Status),
		"joined_at": m.CreatedAt,
	}, nil
}

// MembershipsFor 全量 memberships（whoami 用，含非 active），created_at 序。
func MembershipsFor(db *gorm.DB, userID string) ([]map[string]any, error) {
	var members []entity.TenantMember
	if err := db.Where("user_id = ?", userID).Order("created_at").Find(&members).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(members))
	for i := range members {
		m, err := Membership(db, &members[i])
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
