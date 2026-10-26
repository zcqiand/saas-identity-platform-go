package service

// client_menus_service.go — 应用菜单管理语义（镜像 rails ClientMenusController，
// M04.F04）。protected 但无租户缝；get/update/delete 不校验 menu.client_id ==
// path clientId（仅 findById，镜像）。

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
	"saas-identity-platform-go/internal/impl/service/family"
)

// ClientMenusService 应用菜单面。
type ClientMenusService struct {
	DB *gorm.DB
}

// MenuDTO 菜单形状：parentId 零值 UUID 不转 null、无 updatedAt（rails 镜像）。
func MenuDTO(m *entity.SysMenu) map[string]any {
	return map[string]any{
		"id":         m.Id,
		"client_id":  m.ClientId,
		"parent_id":  m.ParentId,
		"title":      m.Title,
		"type":       family.MenuType(m.Type),
		"path":       m.Path,
		"component":  m.Component,
		"perms":      m.Perms,
		"icon":       m.Icon,
		"sort_order": m.SortOrder,
		"status":     m.Status,
		"created_at": m.CreatedAt,
	}
}

func (s *ClientMenusService) listRows(clientID string) ([]map[string]any, error) {
	var menus []entity.SysMenu
	if err := s.DB.Where("client_id = ?", clientID).Order("sort_order").Find(&menus).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(menus))
	for i := range menus {
		out = append(out, MenuDTO(&menus[i]))
	}
	return out, nil
}

// List 扁平列表（非树），按 sort_order。
func (s *ClientMenusService) List(clientID string) ([]map[string]any, error) {
	return s.listRows(clientID)
}

// Create 创建菜单：title 必填 400；type 缺省 directory 的 rails 语义是 to_i 缺省 1？
// rails：body[:type].present? ? menu_type_code : 1 —— 缺省 1（directory）。
// 注意 menu_dto 读出时经 menu_type：1 → "directory"。parentId 缺省=根。
func (s *ClientMenusService) Create(clientID, title, typeStr string, parentID string,
	path, component, perms, icon *string, sortOrder *int32) (map[string]any, error) {
	if title == "" {
		return nil, familyerrors.ErrBadRequest("title: is required")
	}
	typeCode := int16(1)
	if typeStr != "" {
		code, err := family.MenuTypeCode(typeStr)
		if err != nil {
			return nil, familyerrors.ErrBadRequest(err.Error())
		}
		typeCode = code
	}
	if parentID == "" {
		parentID = rootMenuID
	}
	sort := int32(0)
	if sortOrder != nil {
		sort = *sortOrder
	}
	menu := entity.SysMenu{
		Id:        ids.NewUUID(),
		ClientId:  clientID,
		ParentId:  parentID,
		Title:     title,
		Type:      typeCode,
		Path:      path,
		Component: component,
		Perms:     perms,
		Icon:      icon,
		SortOrder: sort,
		Status:    1,
		CreatedAt: time.Now(),
	}
	if err := s.DB.Create(&menu).Error; err != nil {
		return nil, err
	}
	return MenuDTO(&menu), nil
}

// Get 详情（仅 findById，不校验 clientId 归属）。
func (s *ClientMenusService) Get(menuID string) (map[string]any, error) {
	var menu entity.SysMenu
	if err := s.DB.Where("id = ?", menuID).First(&menu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, familyerrors.ErrNotFound("record not found")
		}
		return nil, err
	}
	return MenuDTO(&menu), nil
}

// Update 只应用 title/path/component/perms/icon/sortOrder（parentId/type/status 忽略，
// 镜像 springboot 注释）。
func (s *ClientMenusService) Update(menuID string, in MenuUpdateInput) (map[string]any, error) {
	var menu entity.SysMenu
	if err := s.DB.Where("id = ?", menuID).First(&menu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, familyerrors.ErrNotFound("record not found")
		}
		return nil, err
	}
	updates := map[string]any{}
	if in.Title != nil {
		updates["title"] = *in.Title
	}
	if in.Path != nil {
		updates["path"] = *in.Path
	}
	if in.Component != nil {
		updates["component"] = *in.Component
	}
	if in.Perms != nil {
		updates["perms"] = *in.Perms
	}
	if in.Icon != nil {
		updates["icon"] = *in.Icon
	}
	if in.SortOrder != nil {
		updates["sort_order"] = *in.SortOrder
	}
	if len(updates) > 0 {
		if err := s.DB.Model(&entity.SysMenu{}).Where("id = ?", menu.Id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(menu.Id)
}

// Move body {parentId} 非空则改挂；响应=菜单 DTO；不存在 404。
func (s *ClientMenusService) Move(menuID, parentID string) (map[string]any, error) {
	var menu entity.SysMenu
	if err := s.DB.Where("id = ?", menuID).First(&menu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, familyerrors.ErrNotFound("record not found")
		}
		return nil, err
	}
	if parentID != "" {
		if err := s.DB.Model(&entity.SysMenu{}).Where("id = ?", menu.Id).
			Update("parent_id", parentID).Error; err != nil {
			return nil, err
		}
		menu.ParentId = parentID
	}
	return MenuDTO(&menu), nil
}

// Reorder body {orderedMenuIds}：indexOf(menuId) >= 0 写 sortOrder=idx（只改当前
// 菜单一条）；响应=该 clientId 全量扁平列表；菜单不存在 404。
func (s *ClientMenusService) Reorder(menuID, clientID string, orderedMenuIDs []string) ([]map[string]any, error) {
	var menu entity.SysMenu
	if err := s.DB.Where("id = ?", menuID).First(&menu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, familyerrors.ErrNotFound("record not found")
		}
		return nil, err
	}
	if orderedMenuIDs == nil {
		return nil, familyerrors.ErrBadRequest("orderedMenuIds: is required")
	}
	for idx, id := range orderedMenuIDs {
		if id == menu.Id {
			if err := s.DB.Model(&entity.SysMenu{}).Where("id = ?", menu.Id).
				Update("sort_order", idx).Error; err != nil {
				return nil, err
			}
			break
		}
	}
	return s.listRows(clientID)
}

// Delete deleteById 语义：不预查，恒成功（204 由 handler 映射）。
func (s *ClientMenusService) Delete(menuID string) error {
	return s.DB.Where("id = ?", menuID).Delete(&entity.SysMenu{}).Error
}

// MenuUpdateInput 更新入参（只带可应用字段）。
type MenuUpdateInput struct {
	Title     *string
	Path      *string
	Component *string
	Perms     *string
	Icon      *string
	SortOrder *int32
}
