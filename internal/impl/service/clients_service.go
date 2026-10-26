package service

// clients_service.go — 公共 client 元数据（镜像 rails ClientsController，M04.F01.I06）。
// 匿名可读（/api/v1/clients/单段 在 SecurityConfig 白名单的镜像）：handler 不挂守卫。

import (
	"errors"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
)

// ClientsService client 公共元数据面。
type ClientsService struct {
	DB *gorm.DB
}

// Get 匿名元数据 {client_id, client_name, status}；未知 → 404。
func (s *ClientsService) Get(clientID string) (map[string]any, error) {
	var client entity.OauthClient
	err := s.DB.Where("client_id = ?", clientID).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"client_id":   client.ClientId,
		"client_name": client.ClientName,
		"status":      client.Status,
	}, nil
}
