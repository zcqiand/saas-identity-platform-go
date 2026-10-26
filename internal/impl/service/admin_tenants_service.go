package service

// admin_tenants_service.go — admin 租户管理语义（镜像 rails AdminTenantsController，
// M00.F01.I01-I05）。JWT 必需、无租户校验。DTO status 恒 'active' 是 springboot
// toDto 的忠实镜像（无 0 分支，suspended 不可见 —— 已知 quirk，勿修）。

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
)

// AdminTenantsService admin 租户面。
type AdminTenantsService struct {
	DB *gorm.DB
}

// TenantDTO 租户形状：status 恒 'active'（quirk 勿修；写侧 active→1 其余→0）。
func TenantDTO(t *entity.Tenant) map[string]any {
	return map[string]any{
		"id":         t.Id,
		"tenant_key": t.TenantKey,
		"name":       t.Name,
		"status":     "active",
		"created_at": t.CreatedAt,
		"updated_at": t.UpdatedAt,
	}
}

func (s *AdminTenantsService) findTenant(id string) (*entity.Tenant, error) {
	var tenant entity.Tenant
	err := s.DB.Where("id = ?", id).First(&tenant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// List 分页信封。
func (s *AdminTenantsService) List(page, pageSize int) (map[string]any, error) {
	q := s.DB.Model(&entity.Tenant{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []entity.Tenant
	if err := q.Limit(pageSize).Offset(page * pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for i := range rows {
		items = append(items, TenantDTO(&rows[i]))
	}
	return map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}, nil
}

// Create 创建租户：tenantKey/name 必填 400；status=1。
func (s *AdminTenantsService) Create(tenantKey, name string) (map[string]any, error) {
	if tenantKey == "" {
		return nil, familyerrors.ErrBadRequest("tenantKey: is required")
	}
	if name == "" {
		return nil, familyerrors.ErrBadRequest("name: is required")
	}
	tenant := entity.Tenant{
		Id:        ids.NewUUID(),
		TenantKey: tenantKey,
		Name:      name,
		Status:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.DB.Create(&tenant).Error; err != nil {
		return nil, err
	}
	return TenantDTO(&tenant), nil
}

// Get 详情。
func (s *AdminTenantsService) Get(id string) (map[string]any, error) {
	tenant, err := s.findTenant(id)
	if err != nil {
		return nil, err
	}
	return TenantDTO(tenant), nil
}

// Update name present 应用；status 'active'→1 其余→0；DTO 仍恒 active。
func (s *AdminTenantsService) Update(id string, name string, status string) (map[string]any, error) {
	tenant, err := s.findTenant(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if name != "" {
		updates["name"] = name
	}
	if status != "" {
		code := int16(0)
		if status == "active" {
			code = 1
		}
		updates["status"] = code
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.DB.Model(&entity.Tenant{}).Where("id = ?", tenant.Id).
			Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(tenant.Id)
}

// Delete 行不存在也 204（不预查，镜像）。
func (s *AdminTenantsService) Delete(id string) error {
	return s.DB.Where("id = ?", id).Delete(&entity.Tenant{}).Error
}
