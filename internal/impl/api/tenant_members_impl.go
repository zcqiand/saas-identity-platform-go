// tenant_members_impl.go — tag: tenant-members 的 ServerInterface 落地
// （M00.F02 + M01.F02.I01）。protected + 路径租户一致性校验，语义在 service。
package api

import (
	"encoding/json"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
	genapi "saas-identity-platform-go/internal/gen/api"
	"saas-identity-platform-go/internal/impl/auth"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/ids"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// checkUUID 路径参数形制校验（rails bad_uuid_param 镜像：不合格式 400 不触库）。
func checkUUID(w http.ResponseWriter, values ...string) bool {
	for _, v := range values {
		if !ids.IsValidUUID(v) {
			familyerrors.Write(w, familyerrors.ErrBadRequest("malformed uuid: "+v))
			return false
		}
	}
	return true
}

// guardTenant protected 面 + 租户缝：守卫 → claims 进 context → path tenantId 比对。
// 返回 false 表示响应已写（401/403）。
func (s *Server) guardTenant(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	claims, ok := s.auth(w, r)
	if !ok {
		return nil, false
	}
	r = r.WithContext(auth.WithClaims(r.Context(), claims))
	if err := auth.RequirePathTenant(r); err != nil {
		familyerrors.Write(w, err)
		return nil, false
	}
	return r, true
}

// pageParams 分页默认 0/20（rails page_params 镜像：page<0 / pageSize<1 → 400）。
func pageParams(w http.ResponseWriter, page, pageSize *int32) (int, int, bool) {
	p, ps := 0, 20
	if page != nil {
		p = int(*page)
	}
	if pageSize != nil {
		ps = int(*pageSize)
	}
	if p < 0 {
		familyerrors.Write(w, familyerrors.ErrBadRequest("page must be >= 0"))
		return 0, 0, false
	}
	if ps < 1 {
		familyerrors.Write(w, familyerrors.ErrBadRequest("pageSize must be >= 1"))
		return 0, 0, false
	}
	return p, ps, true
}

// emailStr openapi_types.Email → *string（nil 保持 nil）。
func emailStr(e *openapi_types.Email) *string {
	if e == nil {
		return nil
	}
	s := string(*e)
	return &s
}

// TenantMembersListTenantUsers GET .../members —— 分页 + status DB 级过滤（M00.F02.I01）。
func (s *Server) TenantMembersListTenantUsers(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.TenantMembersListTenantUsersParams) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	page, pageSize, ok := pageParams(w, params.Page, params.PageSize)
	if !ok {
		return
	}
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).List(tenantId, status, page, pageSize)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantMembersCreateTenantUser POST .../members —— 创建成员（M00.F02.I02）。
func (s *Server) TenantMembersCreateTenantUser(w http.ResponseWriter, r *http.Request, tenantId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	var body genapi.CreateSysUserRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	// 必填顺序镜像 rails：username → password → email；password @Size(8,256)
	for _, kv := range []struct{ key, val string }{
		{"username", body.Username}, {"password", body.Password},
	} {
		if kv.val == "" {
			familyerrors.Write(w, familyerrors.ErrBadRequest(kv.key+": is required"))
			return
		}
	}
	if body.Email == nil || *body.Email == "" {
		familyerrors.Write(w, familyerrors.ErrBadRequest("email: is required"))
		return
	}
	if len(body.Password) < 8 || len(body.Password) > 256 {
		familyerrors.Write(w, familyerrors.ErrBadRequest("password: size must be between 8 and 256"))
		return
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).
		Create(tenantId, body.Username, body.Password, emailStr(body.Email), body.Mobile)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantMembersGetTenantUser GET .../members/{userId} —— 详情（M00.F02.I03）。
func (s *Server) TenantMembersGetTenantUser(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, userId) {
		return
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).Get(tenantId, userId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantMembersUpdateTenantUser PATCH .../members/{userId} —— 只动 email/mobile（M00.F02.I04）。
func (s *Server) TenantMembersUpdateTenantUser(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, userId) {
		return
	}
	var body genapi.UpdateSysUserRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).
		Update(tenantId, userId, emailStr(body.Email), body.Mobile)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantMembersDeleteTenantUser DELETE .../members/{userId} —— 删 member 不删 user（M00.F02.I05）。
func (s *Server) TenantMembersDeleteTenantUser(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, userId) {
		return
	}
	if err := (&service.TenantMembersService{DB: s.DB}).Delete(tenantId, userId); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TenantMembersChangeTenantUserStatus PATCH .../members/{userId}/status —— 双写（M00.F02.I08）。
func (s *Server) TenantMembersChangeTenantUserStatus(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, userId) {
		return
	}
	var body genapi.TenantMembersChangeTenantUserStatusJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).
		ChangeStatus(tenantId, userId, string(body.Status))
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantMembersAssignTenantMemberRoles PUT .../members/{userId}/roles —— 全量覆盖
// （M01.F02.I01）：只收本租户 roleId，外来/未知静默忽略。
func (s *Server) TenantMembersAssignTenantMemberRoles(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, userId) {
		return
	}
	var body genapi.SetTenantMemberRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).
		AssignRoles(tenantId, userId, body.RoleIds)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantMembersInviteTenantUser POST .../members/invitations —— 嵌套视图（M00.F02.I06）。
func (s *Server) TenantMembersInviteTenantUser(w http.ResponseWriter, r *http.Request, tenantId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	var body genapi.TenantMembersInviteTenantUserJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	email := ""
	if body.Email != nil {
		email = *body.Email
	}
	payload, err := (&service.TenantMembersService{DB: s.DB}).Invite(tenantId, email, body.Mobile)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}
