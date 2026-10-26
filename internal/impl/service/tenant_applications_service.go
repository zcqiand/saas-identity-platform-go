package service

// tenant_applications_service.go — 租户应用订阅语义（镜像 rails
// TenantApplicationsController，M00.F05）。订阅前校验链：未知/空 clientId 先撞
// find_by! → 404；dup 预检 → 400 clean message。

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
)

// TenantApplicationsService 租户应用订阅面。
type TenantApplicationsService struct {
	DB *gorm.DB
}

// AppDTO 订阅形状（snake 键交给 render.Camelize）。
func AppDTO(a *entity.TenantApplication) map[string]any {
	return map[string]any{
		"id":          a.Id,
		"tenant_id":   a.TenantId,
		"client_id":   a.ClientId,
		"status":      a.Status,
		"expire_time": a.ExpireTime,
		"created_at":  a.CreatedAt,
	}
}

// List 分页信封（rails render_paginated 镜像：无显式排序）。
func (s *TenantApplicationsService) List(tenantID string, page, pageSize int) (map[string]any, error) {
	q := s.DB.Where("tenant_id = ?", tenantID)
	var total int64
	if err := q.Model(&entity.TenantApplication{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []entity.TenantApplication
	if err := q.Limit(pageSize).Offset(page * pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for i := range rows {
		items = append(items, AppDTO(&rows[i]))
	}
	return map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}, nil
}

// Subscribe 订阅：clientId 先撞 oauth_client（含空 clientId → 404，rails 顺序镜像）→
// dup 预检 400 → status=1 + expireTime 落库。
func (s *TenantApplicationsService) Subscribe(tenantID, clientID string, expireTime *time.Time) (map[string]any, error) {
	var client entity.OauthClient
	err := s.DB.Where("client_id = ?", clientID).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, err
	}
	if clientID == "" {
		return nil, familyerrors.ErrBadRequest("clientId: is required")
	}
	var n int64
	if err := s.DB.Model(&entity.TenantApplication{}).
		Where("tenant_id = ? AND client_id = ?", tenantID, clientID).Count(&n).Error; err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, familyerrors.ErrBadRequest(
			"subscription already exists: tenant=" + tenantID + " client=" + clientID)
	}
	app := entity.TenantApplication{
		Id:         ids.NewUUID(),
		TenantId:   tenantID,
		ClientId:   clientID,
		Status:     1,
		ExpireTime: expireTime,
		CreatedAt:  time.Now(),
	}
	if err := s.DB.Create(&app).Error; err != nil {
		return nil, err
	}
	return AppDTO(&app), nil
}

// Update 只动 present 的字段（status 契约必填 int32，0=disabled 无条件应用——
// springboot oracle 语义 CT I76；rails blank 0 不写是已定性家族偏差，go 不随）。
func (s *TenantApplicationsService) Update(tenantID, clientID string, status *int32, expireTime *time.Time) (map[string]any, error) {
	var app entity.TenantApplication
	err := s.DB.Where("tenant_id = ? AND client_id = ?", tenantID, clientID).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if status != nil {
		updates["status"] = *status
	}
	if expireTime != nil {
		updates["expire_time"] = *expireTime
	}
	if len(updates) > 0 {
		if err := s.DB.Model(&entity.TenantApplication{}).Where("id = ?", app.Id).
			Updates(updates).Error; err != nil {
			return nil, err
		}
		if status != nil {
			app.Status = int16(*status)
		}
		if expireTime != nil {
			app.ExpireTime = expireTime
		}
	}
	return AppDTO(&app), nil
}

// Remove 删订阅行；不存在也 204（ifPresent 静默，镜像）。
func (s *TenantApplicationsService) Remove(tenantID, clientID string) error {
	return s.DB.Where("tenant_id = ? AND client_id = ?", tenantID, clientID).
		Delete(&entity.TenantApplication{}).Error
}
