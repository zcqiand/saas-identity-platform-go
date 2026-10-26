// me_impl.go — tag: me 的 ServerInterface 落地。protected 面：每个 handler 首行
// 过 Bearer 守卫（rails include TenantGuard 镜像），userID/tenantID 只认 JWT claims。
package api

import (
	"net/http"

	genapi "saas-identity-platform-go/internal/gen/api"
	"saas-identity-platform-go/internal/impl/auth"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// s.auth 守卫入口薄包装（各 protected handler 首行调用）。
func (s *Server) auth(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	return auth.Authenticate(w, r)
}

// MeWhoami GET /api/v1/me —— 当前用户身份 + memberships + resolved tenant（M01.F01.I01）。
func (s *Server) MeWhoami(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.auth(w, r)
	if !ok {
		return
	}
	sub, _ := claims["sub"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	payload, err := (&service.MeService{DB: s.DB}).Whoami(sub, tenantID)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// MeListMyTenants GET /api/v1/me/tenants —— active memberships 裸数组（M01.F03.I01）。
func (s *Server) MeListMyTenants(w http.ResponseWriter, r *http.Request, params genapi.MeListMyTenantsParams) {
	claims, ok := s.auth(w, r)
	if !ok {
		return
	}
	sub, _ := claims["sub"].(string)
	payload, err := (&service.MeService{DB: s.DB}).ListMyTenants(sub)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// MeGetMyMenus GET /api/v1/me/menus —— 有效菜单树（M04.F04.I08）。
func (s *Server) MeGetMyMenus(w http.ResponseWriter, r *http.Request, params genapi.MeGetMyMenusParams) {
	claims, ok := s.auth(w, r)
	if !ok {
		return
	}
	sub, _ := claims["sub"].(string)
	payload, err := (&service.MeService{DB: s.DB}).GetMyMenus(sub)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// MeSwitchTenant POST /api/v1/me/tenants/{tenantId}/switch —— 换签 JWT（M01.F03.I02）。
func (s *Server) MeSwitchTenant(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.MeSwitchTenantParams) {
	claims, ok := s.auth(w, r)
	if !ok {
		return
	}
	sub, _ := claims["sub"].(string)
	payload, err := (&service.MeService{DB: s.DB}).SwitchTenant(sub, tenantId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}
