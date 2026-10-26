package service

// admin_clients_service.go — admin 应用管理语义（镜像 rails AdminClientsController，
// M04.F01.I01-I05 + M04.F02.I01）。JWT 必需、无租户校验；寻址一律
// oauth_client.client_id 业务字符串，非行 UUID。

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
)

// AdminClientsService admin 应用面。
type AdminClientsService struct {
	DB *gorm.DB
}

// ClientDTO 应用形状（snake 键交给 render.Camelize）。
func ClientDTO(c *entity.OauthClient) map[string]any {
	return map[string]any{
		"id":                     c.Id,
		"client_id":              c.ClientId,
		"client_name":            c.ClientName,
		"grant_types":            c.GrantTypes,
		"redirect_uris":          c.RedirectUris,
		"scopes":                 c.Scopes,
		"access_token_validity":  c.AccessTokenValidity,
		"refresh_token_validity": c.RefreshTokenValidity,
		"auto_approve":           c.AutoApprove,
		"status":                 c.Status,
		"created_at":             c.CreatedAt,
		"updated_at":             c.UpdatedAt,
	}
}

func (s *AdminClientsService) findClient(clientID string) (*entity.OauthClient, error) {
	var client entity.OauthClient
	err := s.DB.Where("client_id = ?", clientID).First(&client).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// List 分页信封。
func (s *AdminClientsService) List(page, pageSize int) (map[string]any, error) {
	q := s.DB.Model(&entity.OauthClient{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []entity.OauthClient
	if err := q.Limit(pageSize).Offset(page * pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(rows))
	for i := range rows {
		items = append(items, ClientDTO(&rows[i]))
	}
	return map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}, nil
}

// positiveOr validity 缺省/<=0 → default（应用层默认，非 DB DEFAULT 7200/2592000）。
func positiveOr(v *int32, def int32) int32 {
	if v == nil || *v <= 0 {
		return def
	}
	return *v
}

// Create 创建应用：200 非 201（家族口径）；status=1。
func (s *AdminClientsService) Create(in ClientCreateInput) (map[string]any, error) {
	client := entity.OauthClient{
		Id:                   ids.NewUUID(),
		ClientId:             in.ClientID,
		ClientName:           in.ClientName,
		ClientSecret:         in.ClientSecret,
		GrantTypes:           in.GrantTypes,
		RedirectUris:         in.RedirectUris,
		Scopes:               in.Scopes,
		AccessTokenValidity:  positiveOr(in.AccessTokenValidity, 3600),
		RefreshTokenValidity: positiveOr(in.RefreshTokenValidity, 86400),
		AutoApprove:          in.AutoApprove,
		Status:               1,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
	if err := s.DB.Create(&client).Error; err != nil {
		return nil, err
	}
	return ClientDTO(&client), nil
}

// Get 详情。
func (s *AdminClientsService) Get(clientID string) (map[string]any, error) {
	client, err := s.findClient(clientID)
	if err != nil {
		return nil, err
	}
	return ClientDTO(client), nil
}

// Update 只应用 clientName/redirectUris/scopes（其余字段忽略，springboot 镜像）。
func (s *AdminClientsService) Update(clientID string, clientName, redirectUris, scopes *string) (map[string]any, error) {
	client, err := s.findClient(clientID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if clientName != nil {
		updates["client_name"] = *clientName
	}
	if redirectUris != nil {
		updates["redirect_uris"] = *redirectUris
	}
	if scopes != nil {
		updates["scopes"] = *scopes
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.DB.Model(&entity.OauthClient{}).Where("id = ?", client.Id).
			Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(client.ClientId)
}

// SetStatus body {status}：0 是 blank 不应用（rails present? 语义），非 0 才写。
func (s *AdminClientsService) SetStatus(clientID string, status int32) (map[string]any, error) {
	client, err := s.findClient(clientID)
	if err != nil {
		return nil, err
	}
	if status != 0 {
		if err := s.DB.Model(&entity.OauthClient{}).Where("id = ?", client.Id).
			Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error; err != nil {
			return nil, err
		}
		client.Status = int16(status)
	}
	return ClientDTO(client), nil
}

// Delete destroy! 语义：预查，不存在 404；204 由 handler 映射。
func (s *AdminClientsService) Delete(clientID string) error {
	client, err := s.findClient(clientID)
	if err != nil {
		return err
	}
	return s.DB.Where("id = ?", client.Id).Delete(&entity.OauthClient{}).Error
}

// ClientCreateInput 创建入参。
type ClientCreateInput struct {
	ClientID             string
	ClientName           string
	ClientSecret         string
	GrantTypes           string
	RedirectUris         string
	Scopes               *string
	AccessTokenValidity  *int32
	RefreshTokenValidity *int32
	AutoApprove          bool
}
