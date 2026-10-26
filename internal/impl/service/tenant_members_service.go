package service

// tenant_members_service.go — 租户成员语义（镜像 rails TenantMembersController，
// M00.F02 + M01.F02.I01）。寻址：{userId} = sys_user.id，经 (tenant_id,user_id) 找
// member 行，寻不到 404。status 双写 member+user（家族口径）。

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"saas-identity-platform-go/internal/gen/entity"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
	"saas-identity-platform-go/internal/impl/service/family"
)

// TenantMembersService 租户成员面。
type TenantMembersService struct {
	DB *gorm.DB
}

// locateMember 寻 (tenant_id,user_id) 的 member 行 + user 行（无 user 行 404）。
func (s *TenantMembersService) locateMember(tenantID, userID string) (*entity.TenantMember, *entity.SysUser, error) {
	var member entity.TenantMember
	err := s.DB.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, familyerrors.ErrNotFound("record not found")
	}
	if err != nil {
		return nil, nil, err
	}
	var user entity.SysUser
	err = s.DB.Where("id = ?", member.UserId).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, familyerrors.ErrNotFound("member user missing")
	}
	if err != nil {
		return nil, nil, err
	}
	return &member, &user, nil
}

// List 成员列表：status 过滤 DB 级（total=过滤后计数）；createdAt DESC,id ASC；
// member 无对应 user 行的条目被滤掉。
func (s *TenantMembersService) List(tenantID string, status string, page, pageSize int) (map[string]any, error) {
	q := s.DB.Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", family.MemberStatusCode(status))
	}
	var total int64
	if err := q.Model(&entity.TenantMember{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var window []entity.TenantMember
	if err := q.Order("created_at DESC, id ASC").
		Limit(pageSize).Offset(page * pageSize).Find(&window).Error; err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(window))
	for i := range window {
		userIDs = append(userIDs, window[i].UserId)
	}
	var users []entity.SysUser
	if len(userIDs) > 0 {
		if err := s.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}
	}
	byID := make(map[string]*entity.SysUser, len(users))
	for i := range users {
		byID[users[i].Id] = &users[i]
	}
	items := []map[string]any{}
	for i := range window {
		user, ok := byID[window[i].UserId]
		if !ok {
			continue
		}
		view, err := family.FlatUserView(s.DB, &window[i], user)
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}
	return map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	}, nil
}

// Create 创建成员：password "plain:" 形式落库 + active member 行；返回扁平视图。
func (s *TenantMembersService) Create(tenantID, username, password string, email, mobile *string) (map[string]any, error) {
	now := time.Now()
	user := entity.SysUser{
		Id:        ids.NewUUID(),
		Username:  username,
		Password:  "plain:" + password,
		Email:     email,
		Mobile:    mobile,
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	member := entity.TenantMember{
		Id:         ids.NewUUID(),
		TenantId:   tenantID,
		UserId:     user.Id,
		MemberName: &username,
		IsOwner:    false,
		Status:     1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.DB.Create(&member).Error; err != nil {
		return nil, err
	}
	return family.FlatUserView(s.DB, &member, &user)
}

// Get 成员详情（扁平视图）。
func (s *TenantMembersService) Get(tenantID, userID string) (map[string]any, error) {
	member, user, err := s.locateMember(tenantID, userID)
	if err != nil {
		return nil, err
	}
	return family.FlatUserView(s.DB, member, user)
}

// Update 只更新 user 行 email/mobile（契约已删 status 字段——状态唯一通道是
// /status 端点）；非 nil 才应用。
func (s *TenantMembersService) Update(tenantID, userID string, email, mobile *string) (map[string]any, error) {
	member, user, err := s.locateMember(tenantID, userID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if email != nil {
		updates["email"] = *email
	}
	if mobile != nil {
		updates["mobile"] = *mobile
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := s.DB.Model(&entity.SysUser{}).Where("id = ?", user.Id).Updates(updates).Error; err != nil {
			return nil, err
		}
		if email != nil {
			user.Email = email
		}
		if mobile != nil {
			user.Mobile = mobile
		}
	}
	return family.FlatUserView(s.DB, member, user)
}

// Delete 删 member_role 绑定 + member 行；user 行不删；204。
func (s *TenantMembersService) Delete(tenantID, userID string) error {
	member, _, err := s.locateMember(tenantID, userID)
	if err != nil {
		return err
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("member_id = ?", member.Id).Delete(&entity.TenantMemberRole{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", member.Id).Delete(&entity.TenantMember{}).Error
	})
}

// ChangeStatus 双写 member.status 与 user.status 同值。
func (s *TenantMembersService) ChangeStatus(tenantID, userID, status string) (map[string]any, error) {
	member, user, err := s.locateMember(tenantID, userID)
	if err != nil {
		return nil, err
	}
	code := family.MemberStatusCode(status)
	now := time.Now()
	if err := s.DB.Model(&entity.TenantMember{}).Where("id = ?", member.Id).
		Updates(map[string]any{"status": code, "updated_at": now}).Error; err != nil {
		return nil, err
	}
	if err := s.DB.Model(&entity.SysUser{}).Where("id = ?", user.Id).
		Updates(map[string]any{"status": code, "updated_at": now}).Error; err != nil {
		return nil, err
	}
	member.Status = code
	user.Status = code
	return family.FlatUserView(s.DB, member, user)
}

// AssignRoles body {roleIds}：先全删再插；只接受本租户 sys_role 的 roleId，
// 外来/未知静默忽略；返回扁平视图。
func (s *TenantMembersService) AssignRoles(tenantID, userID string, roleIDs []string) (map[string]any, error) {
	member, user, err := s.locateMember(tenantID, userID)
	if err != nil {
		return nil, err
	}
	if roleIDs == nil {
		roleIDs = []string{}
	}
	var validIDs []string
	if len(roleIDs) > 0 {
		if err := s.DB.Model(&entity.SysRole{}).Where("tenant_id = ? AND id IN ?", tenantID, roleIDs).
			Pluck("id", &validIDs).Error; err != nil {
			return nil, err
		}
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("member_id = ?", member.Id).Delete(&entity.TenantMemberRole{}).Error; err != nil {
			return err
		}
		for _, rid := range validIDs {
			if err := tx.Create(&entity.TenantMemberRole{MemberId: member.Id, RoleId: rid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return family.FlatUserView(s.DB, member, user)
}

// Invite 邀请：email 空 → 400；建真 sys_user（username=email, 空 password,
// status=2）；member status=1；响应嵌套视图。
func (s *TenantMembersService) Invite(tenantID string, email string, mobile *string) (map[string]any, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, familyerrors.ErrBadRequest("email: is required")
	}
	now := time.Now()
	user := entity.SysUser{
		Id:        ids.NewUUID(),
		Username:  email,
		Password:  "",
		Email:     &email,
		Mobile:    mobile,
		Status:    2,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	member := entity.TenantMember{
		Id:         ids.NewUUID(),
		TenantId:   tenantID,
		UserId:     user.Id,
		MemberName: &email,
		IsOwner:    false,
		Status:     1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.DB.Create(&member).Error; err != nil {
		return nil, err
	}
	return family.NestedView(&member, &user), nil
}
