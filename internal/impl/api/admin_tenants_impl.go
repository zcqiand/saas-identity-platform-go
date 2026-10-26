// admin_tenants_impl.go — tag: admin-tenants 的 ServerInterface 落地（M00.F01.I01-I05）。
// JWT 必需、无租户校验；{id} 是行 UUID。
package api

import (
	"encoding/json"
	"net/http"

	genapi "saas-identity-platform-go/internal/gen/api"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// AdminTenantsListTenants GET /admin/tenants —— 分页信封（M00.F01.I01）。
func (s *Server) AdminTenantsListTenants(w http.ResponseWriter, r *http.Request, params genapi.AdminTenantsListTenantsParams) {
	if !s.authOrStop(w, r) {
		return
	}
	page, pageSize, ok := pageParams(w, params.Page, params.PageSize)
	if !ok {
		return
	}
	payload, err := (&service.AdminTenantsService{DB: s.DB}).List(page, pageSize)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminTenantsCreateTenant POST /admin/tenants —— tenantKey/name 必填（M00.F01.I02）。
func (s *Server) AdminTenantsCreateTenant(w http.ResponseWriter, r *http.Request) {
	if !s.authOrStop(w, r) {
		return
	}
	var body genapi.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.AdminTenantsService{DB: s.DB}).Create(body.TenantKey, body.Name)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminTenantsGetTenant GET /admin/tenants/{id} —— 详情（M00.F01.I03）。
func (s *Server) AdminTenantsGetTenant(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, id) {
		return
	}
	payload, err := (&service.AdminTenantsService{DB: s.DB}).Get(id)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminTenantsUpdateTenant PATCH /admin/tenants/{id} —— name/status（M00.F01.I04）。
func (s *Server) AdminTenantsUpdateTenant(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, id) {
		return
	}
	var body genapi.UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	name, status := "", ""
	if body.Name != nil {
		name = *body.Name
	}
	if body.Status != nil {
		status = string(*body.Status)
	}
	payload, err := (&service.AdminTenantsService{DB: s.DB}).Update(id, name, status)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminTenantsDeleteTenant DELETE /admin/tenants/{id} —— 不预查幂等 204（M00.F01.I05）。
func (s *Server) AdminTenantsDeleteTenant(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, id) {
		return
	}
	if err := (&service.AdminTenantsService{DB: s.DB}).Delete(id); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
