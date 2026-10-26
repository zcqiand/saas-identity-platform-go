package service

// tenant_role_menus_service.go — 角色菜单授权语义（镜像 rails
// TenantRoleMenusController，M00.F04.I02/I03/I04）。聚合视图 RoleMenuGrant：
// { roleId, tenantId, menuIds(升序), updatedAt(=sys_role.updated_at) }。

import (
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
)

// TenantRoleMenusService 角色菜单授权面。
type TenantRoleMenusService struct {
	DB *gorm.DB
}

func (s *TenantRoleMenusService) locateRole(tenantID, roleID string) (*entity.SysRole, error) {
	var role entity.SysRole
	err := s.DB.Where("id = ?", roleID).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, err
	}
	if role.TenantId != tenantID {
		return nil, familyerrors.ErrNotFound("role not in tenant")
	}
	return &role, nil
}

// List 当前授权集（menu_id 升序）。
func (s *TenantRoleMenusService) List(tenantID, roleID string) (map[string]any, error) {
	role, err := s.locateRole(tenantID, roleID)
	if err != nil {
		return nil, err
	}
	var menuIDs []string
	if err := s.DB.Model(&entity.SysRoleMenu{}).Where("role_id = ?", role.Id).
		Order("menu_id").Pluck("menu_id", &menuIDs).Error; err != nil {
		return nil, err
	}
	if menuIDs == nil {
		menuIDs = []string{}
	}
	return map[string]any{
		"role_id":    role.Id,
		"tenant_id":  tenantID,
		"menu_ids":   menuIDs,
		"updated_at": role.UpdatedAt,
	}, nil
}

// Set 全量替换（差量实现镜像）：锁 role 行 → 删不在集合的 → 插缺失（幂等）→
// touch role。响应直接从请求集构造（去重排序），不回读。
func (s *TenantRoleMenusService) Set(tenantID, roleID string, menuIDs []string) (map[string]any, error) {
	role, err := s.locateRole(tenantID, roleID)
	if err != nil {
		return nil, err
	}
	if menuIDs == nil {
		menuIDs = []string{}
	}
	requested := dedupeSorted(menuIDs)
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var locked entity.SysRole
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", role.Id).First(&locked).Error; err != nil {
			return err
		}
		if len(requested) == 0 {
			if err := tx.Where("role_id = ?", locked.Id).Delete(&entity.SysRoleMenu{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Where("role_id = ? AND menu_id NOT IN ?", locked.Id, requested).
				Delete(&entity.SysRoleMenu{}).Error; err != nil {
				return err
			}
			for _, mid := range requested {
				var n int64
				if err := tx.Model(&entity.SysRoleMenu{}).
					Where("role_id = ? AND menu_id = ?", locked.Id, mid).Count(&n).Error; err != nil {
					return err
				}
				if n == 0 {
					if err := tx.Create(&entity.SysRoleMenu{RoleId: locked.Id, MenuId: mid}).Error; err != nil {
						return err
					}
				}
			}
		}
		return tx.Model(&entity.SysRole{}).Where("id = ?", locked.Id).
			Update("updated_at", time.Now()).Error
	})
	if err != nil {
		return nil, err
	}
	var reloaded entity.SysRole
	if err := s.DB.Where("id = ?", role.Id).First(&reloaded).Error; err != nil {
		return nil, err
	}
	return map[string]any{
		"role_id":    role.Id,
		"tenant_id":  tenantID,
		"menu_ids":   requested,
		"updated_at": reloaded.UpdatedAt,
	}, nil
}

// Clear bulk 全删；roleId 不存在也 204（不预查，rails 镜像）。
func (s *TenantRoleMenusService) Clear(roleID string) error {
	return s.DB.Where("role_id = ?", roleID).Delete(&entity.SysRoleMenu{}).Error
}

func dedupeSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
