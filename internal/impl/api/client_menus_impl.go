// client_menus_impl.go — tag: client-menus 的 ServerInterface 落地（M04.F04.I01-I07）。
// protected 但无租户缝（无 TenantGuard，rails 镜像）；clientId 是 string code 非 UUID。
package api

import (
	"encoding/json"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
	genapi "saas-identity-platform-go/internal/gen/api"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// authOrStop 仅守卫（无租户缝）。
func (s *Server) authOrStop(w http.ResponseWriter, r *http.Request) bool {
	_, ok := s.auth(w, r)
	return ok
}

// uuidStr *openapi_types.UUID → string（nil → ""）。
func uuidStr(u *openapi_types.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

// ClientMenusListSysMenus GET .../menus —— 扁平列表（M04.F04.I01）。
func (s *Server) ClientMenusListSysMenus(w http.ResponseWriter, r *http.Request, clientId string) {
	if !s.authOrStop(w, r) {
		return
	}
	payload, err := (&service.ClientMenusService{DB: s.DB}).List(clientId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// ClientMenusCreateSysMenu POST .../menus —— title 必填；type 缺省 directory（M04.F04.I02）。
func (s *Server) ClientMenusCreateSysMenu(w http.ResponseWriter, r *http.Request, clientId string) {
	if !s.authOrStop(w, r) {
		return
	}
	var body genapi.CreateSysMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.ClientMenusService{DB: s.DB}).Create(
		clientId, body.Title, string(body.Type), uuidStr(body.ParentId),
		body.Path, body.Component, body.Perms, body.Icon, body.SortOrder)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// ClientMenusGetSysMenu GET .../menus/{menuId} —— 详情（M04.F04.I03）。
func (s *Server) ClientMenusGetSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, menuId) {
		return
	}
	payload, err := (&service.ClientMenusService{DB: s.DB}).Get(menuId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// ClientMenusUpdateSysMenu PATCH .../menus/{menuId} —— 只应用白名单字段（M04.F04.I04）。
func (s *Server) ClientMenusUpdateSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, menuId) {
		return
	}
	var body genapi.UpdateSysMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.ClientMenusService{DB: s.DB}).Update(menuId, service.MenuUpdateInput{
		Title: body.Title, Path: body.Path, Component: body.Component,
		Perms: body.Perms, Icon: body.Icon, SortOrder: body.SortOrder,
	})
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// ClientMenusMoveSysMenu PATCH .../menus/{menuId}/parent —— 改挂父级（M04.F04.I07）。
func (s *Server) ClientMenusMoveSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, menuId) {
		return
	}
	var body genapi.ClientMenusMoveSysMenuJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	parentID := ""
	if body.ParentId != nil {
		parentID = *body.ParentId
	}
	payload, err := (&service.ClientMenusService{DB: s.DB}).Move(menuId, parentID)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// ClientMenusReorderSysMenus PUT .../menus/{menuId}/reorder —— 同级排序（M04.F04.I06）。
func (s *Server) ClientMenusReorderSysMenus(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, menuId) {
		return
	}
	var body genapi.ReorderSysMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	payload, err := (&service.ClientMenusService{DB: s.DB}).Reorder(menuId, clientId, body.OrderedMenuIds)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// ClientMenusDeleteSysMenu DELETE .../menus/{menuId} —— 不预查恒 204（M04.F04.I05）。
func (s *Server) ClientMenusDeleteSysMenu(w http.ResponseWriter, r *http.Request, clientId string, menuId string) {
	if !s.authOrStop(w, r) {
		return
	}
	if !checkUUID(w, menuId) {
		return
	}
	if err := (&service.ClientMenusService{DB: s.DB}).Delete(menuId); err != nil {
		familyerrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
