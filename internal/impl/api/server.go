// Package api — ServerInterface 实现壳（REQ-2026-003 Task 3 路由壳）。
//
// API 面只认生成物（suite-hard-rules §4）：路由注册由 internal/gen/api 的
// HandlerFromMux 完成，本结构只实现 47 个方法签名；每域实现落地时把对应方法
// 从 notImplemented 壳替换为 service 委派（ktor delegate 同构——handler 只做
// 装配与状态码映射，业务进 service，CLAUDE.md §2）。
package api

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	genapi "saas-identity-platform-go/internal/gen/api"
)

// Server 实现 genapi.ServerInterface。DB 为装配期注入的连接（*gorm.DB；
// 各域实现接管后经 service 使用，路由壳阶段可为 nil 的域不触库）。
type Server struct {
	DB *gorm.DB
}

// 编译期保证 Server 满足生成接口——签名漂移在 build 门即红。
var _ genapi.ServerInterface = (*Server)(nil)

// NewServer 构造路由壳（db 注入给已落地的域）。
func NewServer(db *gorm.DB) *Server { return &Server{DB: db} }

func notImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": "NOT_IMPLEMENTED"})
}

func (s *Server) AdminClientsListClients(w http.ResponseWriter, r *http.Request, params genapi.AdminClientsListClientsParams) {
	notImplemented(w)
}
func (s *Server) AdminClientsCreateClient(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
func (s *Server) AdminClientsDeleteClient(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) AdminClientsGetClient(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) AdminClientsUpdateClient(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) AdminClientsSetClientStatus(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) AdminTenantsListTenants(w http.ResponseWriter, r *http.Request, params genapi.AdminTenantsListTenantsParams) {
	notImplemented(w)
}
func (s *Server) AdminTenantsCreateTenant(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
func (s *Server) AdminTenantsDeleteTenant(w http.ResponseWriter, r *http.Request, id string) {
	notImplemented(w)
}
func (s *Server) AdminTenantsGetTenant(w http.ResponseWriter, r *http.Request, id string) {
	notImplemented(w)
}
func (s *Server) AdminTenantsUpdateTenant(w http.ResponseWriter, r *http.Request, id string) {
	notImplemented(w)
}

// SessionsLogin / SessionsLogout 落地见 auth_impl.go（tag: auth 已接管）。
func (s *Server) ClientsGetClient(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusListSysMenus(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusCreateSysMenu(w http.ResponseWriter, r *http.Request, clientId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusDeleteSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusGetSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusUpdateSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusMoveSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	notImplemented(w)
}
func (s *Server) ClientMenusReorderSysMenus(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	notImplemented(w)
}
func (s *Server) MeWhoami(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
func (s *Server) MeGetMyMenus(w http.ResponseWriter, r *http.Request, params genapi.MeGetMyMenusParams) {
	notImplemented(w)
}
func (s *Server) MeListMyTenants(w http.ResponseWriter, r *http.Request, params genapi.MeListMyTenantsParams) {
	notImplemented(w)
}
func (s *Server) MeSwitchTenant(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.MeSwitchTenantParams) {
	notImplemented(w)
}
func (s *Server) OAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
func (s *Server) OAuthToken(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
func (s *Server) TenantApplicationsListTenantApplications(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.TenantApplicationsListTenantApplicationsParams) {
	notImplemented(w)
}
func (s *Server) TenantApplicationsSubscribeTenantApplication(w http.ResponseWriter, r *http.Request, tenantId string) {
	notImplemented(w)
}
func (s *Server) TenantApplicationsRemoveTenantApplication(w http.ResponseWriter, r *http.Request, tenantId string, clientId string) {
	notImplemented(w)
}
func (s *Server) TenantApplicationsUpdateTenantApplication(w http.ResponseWriter, r *http.Request, tenantId string, clientId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersListTenantUsers(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.TenantMembersListTenantUsersParams) {
	notImplemented(w)
}
func (s *Server) TenantMembersCreateTenantUser(w http.ResponseWriter, r *http.Request, tenantId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersInviteTenantUser(w http.ResponseWriter, r *http.Request, tenantId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersDeleteTenantUser(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersGetTenantUser(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersUpdateTenantUser(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersAssignTenantMemberRoles(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	notImplemented(w)
}
func (s *Server) TenantMembersChangeTenantUserStatus(w http.ResponseWriter, r *http.Request, tenantId string, userId string) {
	notImplemented(w)
}
func (s *Server) TenantRolesListSysRoles(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.TenantRolesListSysRolesParams) {
	notImplemented(w)
}
func (s *Server) TenantRolesCreateSysRole(w http.ResponseWriter, r *http.Request, tenantId string) {
	notImplemented(w)
}
func (s *Server) TenantRolesDeleteSysRole(w http.ResponseWriter, r *http.Request, tenantId string, roleId string) {
	notImplemented(w)
}
func (s *Server) TenantRolesGetSysRole(w http.ResponseWriter, r *http.Request, tenantId string, roleId string) {
	notImplemented(w)
}
func (s *Server) TenantRolesUpdateSysRole(w http.ResponseWriter, r *http.Request, tenantId string, roleId string) {
	notImplemented(w)
}
func (s *Server) TenantRoleMenusClearSysRoleMenus(w http.ResponseWriter, r *http.Request, tenantId string, roleId string, params genapi.TenantRoleMenusClearSysRoleMenusParams) {
	notImplemented(w)
}
func (s *Server) TenantRoleMenusListSysRoleMenus(w http.ResponseWriter, r *http.Request, tenantId string, roleId string, params genapi.TenantRoleMenusListSysRoleMenusParams) {
	notImplemented(w)
}
func (s *Server) TenantRoleMenusSetSysRoleMenus(w http.ResponseWriter, r *http.Request, tenantId string, roleId string, params genapi.TenantRoleMenusSetSysRoleMenusParams) {
	notImplemented(w)
}
