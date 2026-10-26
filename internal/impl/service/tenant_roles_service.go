package service

// tenant_roles_service.go — 租户角色语义（镜像 rails TenantRolesController，M00.F03）。
// role.tenant_id != path tenantId → 404（不泄露跨租户存在性）。

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
)

// TenantRolesService 租户角色面。
type TenantRolesService struct {
	DB *gorm.DB
}

// RoleDTO 角色形状（snake 键交给 render.Camelize）。
func RoleDTO(r *entity.SysRole) map[string]any {
	return map[string]any{
		"id":          r.Id,
		"tenant_id":   r.TenantId,
		"client_id":   r.ClientId,
		"role_code":   r.RoleCode,
		"role_name":   r.RoleName,
		"description": r.Description,
		"is_preset":   r.IsPreset,
		"status":      r.Status,
		"created_at":  r.CreatedAt,
		"updated_at":  r.UpdatedAt,
	}
}

// locateRole role 不在 path 租户下 → 404（find_by! + tenant 比对镜像）。
func (s *TenantRolesService) locateRole(tenantID, roleID string) (*entity.SysRole, error) {
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

// List 角色分页信封（clientId 参数被忽略，rails 镜像）。
func (s *TenantRolesService) List(tenantID string, page, pageSize int) (map[string]any, error) {
	q := s.DB.Where("tenant_id = ?", tenantID)
	var total int64
	if err := q.Model(&entity.SysRole{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []entity.SysRole
	if err := q.Limit(pageSize).Offset(page * pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for i := range rows {
		items = append(items, RoleDTO(&rows[i]))
	}
	return map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}, nil
}

// Create 创建角色：clientId/roleCode/roleName 必填 400；status=1。
func (s *TenantRolesService) Create(tenantID, clientID, roleCode, roleName string, description *string, isPreset bool) (map[string]any, error) {
	for key, val := range map[string]string{
		"clientId": clientID, "roleCode": roleCode, "roleName": roleName,
	} {
		if val == "" {
			return nil, familyerrors.ErrBadRequest(key + ": is required")
		}
	}
	role := entity.SysRole{
		Id:          ids.NewUUID(),
		TenantId:    tenantID,
		ClientId:    clientID,
		RoleCode:    roleCode,
		RoleName:    roleName,
		Description: description,
		IsPreset:    isPreset,
		Status:      1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.DB.Create(&role).Error; err != nil {
		return nil, err
	}
	return RoleDTO(&role), nil
}

// Get 角色详情。
func (s *TenantRolesService) Get(tenantID, roleID string) (map[string]any, error) {
	role, err := s.locateRole(tenantID, roleID)
	if err != nil {
		return nil, err
	}
	return RoleDTO(role), nil
}

// Update 只应用 roleName/description（status 字段可空不应用，rails 镜像）。
func (s *TenantRolesService) Update(tenantID, roleID string, roleName, description *string) (map[string]any, error) {
	role, err := s.locateRole(tenantID, roleID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if roleName != nil {
		updates["role_name"] = *roleName
	}
	if description != nil {
		updates["description"] = *description
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.DB.Model(&entity.SysRole{}).Where("id = ?", role.Id).Updates(updates).Error; err != nil {
			return nil, err
		}
		if roleName != nil {
			role.RoleName = *roleName
		}
		if description != nil {
			role.Description = description
		}
	}
	return RoleDTO(role), nil
}

// Delete 删 role + role_menu + member_role 绑定；204。
func (s *TenantRolesService) Delete(tenantID, roleID string) error {
	role, err := s.locateRole(tenantID, roleID)
	if err != nil {
		return err
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", role.Id).Delete(&entity.SysRoleMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", role.Id).Delete(&entity.TenantMemberRole{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", role.Id).Delete(&entity.SysRole{}).Error
	})
}
