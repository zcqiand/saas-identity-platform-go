// tenant_roles_impl.go — tag: tenant-roles + tenant-role-menus 的 ServerInterface
// 落地（M00.F03 + M00.F04）。protected + 路径租户一致性校验，语义在 service。
package api

import (
	"encoding/json"
	"net/http"

	genapi "saas-identity-platform-go/internal/gen/api"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// TenantRolesListSysRoles GET .../roles —— 角色分页信封（M00.F03.I01）。
func (s *Server) TenantRolesListSysRoles(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.TenantRolesListSysRolesParams) {
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
	payload, err := (&service.TenantRolesService{DB: s.DB}).List(tenantId, page, pageSize)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantRolesCreateSysRole POST .../roles —— 必填校验 + status=1（M00.F03.I02）。
func (s *Server) TenantRolesCreateSysRole(w http.ResponseWriter, r *http.Request, tenantId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	var body genapi.CreateSysRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	isPreset := false
	if body.IsPreset != nil {
		isPreset = *body.IsPreset
	}
	payload, err := (&service.TenantRolesService{DB: s.DB}).
		Create(tenantId, body.ClientId, body.RoleCode, body.RoleName, body.Description, isPreset)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantRolesGetSysRole GET .../roles/{roleId} —— 详情；跨租户 404（M00.F03.I03）。
func (s *Server) TenantRolesGetSysRole(w http.ResponseWriter, r *http.Request, tenantId string, roleId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, roleId) {
		return
	}
	payload, err := (&service.TenantRolesService{DB: s.DB}).Get(tenantId, roleId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantRolesUpdateSysRole PATCH .../roles/{roleId} —— 只动 roleName/description（M00.F03.I04）。
func (s *Server) TenantRolesUpdateSysRole(w http.ResponseWriter, r *http.Request, tenantId string, roleId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, roleId) {
		return
	}
	var body genapi.UpdateSysRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantRolesService{DB: s.DB}).
		Update(tenantId, roleId, body.RoleName, body.Description)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantRolesDeleteSysRole DELETE .../roles/{roleId} —— 连带清 role_menu/member_role（M00.F03.I05）。
func (s *Server) TenantRolesDeleteSysRole(w http.ResponseWriter, r *http.Request, tenantId string, roleId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, roleId) {
		return
	}
	if err := (&service.TenantRolesService{DB: s.DB}).Delete(tenantId, roleId); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TenantRoleMenusListSysRoleMenus GET .../roles/{roleId}/menus —— 当前授权集（M00.F04.I03）。
func (s *Server) TenantRoleMenusListSysRoleMenus(w http.ResponseWriter, r *http.Request, tenantId string, roleId string, params genapi.TenantRoleMenusListSysRoleMenusParams) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, roleId) {
		return
	}
	payload, err := (&service.TenantRoleMenusService{DB: s.DB}).List(tenantId, roleId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantRoleMenusSetSysRoleMenus PUT .../roles/{roleId}/menus —— 全量替换 + touch（M00.F04.I02）。
func (s *Server) TenantRoleMenusSetSysRoleMenus(w http.ResponseWriter, r *http.Request, tenantId string, roleId string, params genapi.TenantRoleMenusSetSysRoleMenusParams) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, roleId) {
		return
	}
	var body genapi.SetSysRoleMenusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantRoleMenusService{DB: s.DB}).Set(tenantId, roleId, body.MenuIds)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantRoleMenusClearSysRoleMenus DELETE .../roles/{roleId}/menus —— 幂等 204（M00.F04.I04）。
func (s *Server) TenantRoleMenusClearSysRoleMenus(w http.ResponseWriter, r *http.Request, tenantId string, roleId string, params genapi.TenantRoleMenusClearSysRoleMenusParams) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId, roleId) {
		return
	}
	if err := (&service.TenantRoleMenusService{DB: s.DB}).Clear(roleId); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
