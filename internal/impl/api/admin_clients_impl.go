// admin_clients_impl.go — tag: admin-clients 的 ServerInterface 落地
// （M04.F01.I01-I05 + M04.F02.I01）。JWT 必需、无租户校验；clientId 是业务字符串。
package api

import (
	"encoding/json"
	"net/http"

	genapi "saas-identity-platform-go/internal/gen/api"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// AdminClientsListClients GET /admin/clients —— 分页信封（M04.F01.I01）。
func (s *Server) AdminClientsListClients(w http.ResponseWriter, r *http.Request, params genapi.AdminClientsListClientsParams) {
	if !s.authOrStop(w, r) {
		return
	}
	page, pageSize, ok := pageParams(w, params.Page, params.PageSize)
	if !ok {
		return
	}
	payload, err := (&service.AdminClientsService{DB: s.DB}).List(page, pageSize)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminClientsCreateClient POST /admin/clients —— validity 缺省 3600/86400（M04.F01.I02）。
func (s *Server) AdminClientsCreateClient(w http.ResponseWriter, r *http.Request) {
	if !s.authOrStop(w, r) {
		return
	}
	var body genapi.CreateOAuthClientRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.AdminClientsService{DB: s.DB}).Create(service.ClientCreateInput{
		ClientID: body.ClientId, ClientName: body.ClientName, ClientSecret: body.ClientSecret,
		GrantTypes: body.GrantTypes, RedirectUris: body.RedirectUris, Scopes: body.Scopes,
		AccessTokenValidity: body.AccessTokenValidity, RefreshTokenValidity: body.RefreshTokenValidity,
		AutoApprove: boolPtr(body.AutoApprove),
	})
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminClientsGetClient GET /admin/clients/{clientId} —— 详情（M04.F01.I03）。
func (s *Server) AdminClientsGetClient(w http.ResponseWriter, r *http.Request, clientId string) {
	if !s.authOrStop(w, r) {
		return
	}
	payload, err := (&service.AdminClientsService{DB: s.DB}).Get(clientId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminClientsUpdateClient PATCH /admin/clients/{clientId} —— 白名单字段（M04.F01.I04）。
func (s *Server) AdminClientsUpdateClient(w http.ResponseWriter, r *http.Request, clientId string) {
	if !s.authOrStop(w, r) {
		return
	}
	var body genapi.UpdateOAuthClientRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.AdminClientsService{DB: s.DB}).
		Update(clientId, body.ClientName, body.RedirectUris, body.Scopes)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminClientsSetClientStatus PATCH /admin/clients/{clientId}/status —— 0 不应用（M04.F02.I01）。
func (s *Server) AdminClientsSetClientStatus(w http.ResponseWriter, r *http.Request, clientId string) {
	if !s.authOrStop(w, r) {
		return
	}
	var body genapi.AdminClientsSetClientStatusJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.AdminClientsService{DB: s.DB}).SetStatus(clientId, body.Status)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// AdminClientsDeleteClient DELETE /admin/clients/{clientId} —— destroy! 预查 404（M04.F01.I05）。
func (s *Server) AdminClientsDeleteClient(w http.ResponseWriter, r *http.Request, clientId string) {
	if !s.authOrStop(w, r) {
		return
	}
	if err := (&service.AdminClientsService{DB: s.DB}).Delete(clientId); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// boolPtr *bool 保持 nil 语义；false → false。
func boolPtr(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
