// tenant_applications_impl.go — tag: tenant-applications 的 ServerInterface 落地
// （M00.F05.I01-I04）。protected + 租户缝；clientId 是 string code 非 UUID。
package api

import (
	"encoding/json"
	"net/http"

	genapi "saas-identity-platform-go/internal/gen/api"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// TenantApplicationsListTenantApplications GET .../applications —— 分页信封（M00.F05.I01）。
func (s *Server) TenantApplicationsListTenantApplications(w http.ResponseWriter, r *http.Request, tenantId string, params genapi.TenantApplicationsListTenantApplicationsParams) {
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
	payload, err := (&service.TenantApplicationsService{DB: s.DB}).List(tenantId, page, pageSize)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantApplicationsSubscribeTenantApplication POST .../applications —— 订阅（M00.F05.I02）。
func (s *Server) TenantApplicationsSubscribeTenantApplication(w http.ResponseWriter, r *http.Request, tenantId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	var body genapi.SubscribeTenantApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantApplicationsService{DB: s.DB}).
		Subscribe(tenantId, body.ClientId, body.ExpireTime)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantApplicationsUpdateTenantApplication PATCH .../applications/{clientId} —— 更新（M00.F05.I03）。
func (s *Server) TenantApplicationsUpdateTenantApplication(w http.ResponseWriter, r *http.Request, tenantId string, clientId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	var body genapi.UpdateTenantApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.TenantApplicationsService{DB: s.DB}).
		Update(tenantId, clientId, &body.Status, body.ExpireTime)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// TenantApplicationsRemoveTenantApplication DELETE .../applications/{clientId} —— 幂等 204（M00.F05.I04）。
func (s *Server) TenantApplicationsRemoveTenantApplication(w http.ResponseWriter, r *http.Request, tenantId string, clientId string) {
	if _, ok := s.guardTenant(w, r); !ok {
		return
	}
	if !checkUUID(w, tenantId) {
		return
	}
	if err := (&service.TenantApplicationsService{DB: s.DB}).Remove(tenantId, clientId); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
