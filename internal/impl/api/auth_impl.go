// auth_impl.go — tag: auth 的 ServerInterface 落地（M01.F04）。
// /auth/** 是 public 面（rails SessionsController 无 TenantGuard 镜像），handler 只
// 做装配与状态码映射，登录语义全在 service.LoginService。
package api

import (
	"encoding/json"
	"net/http"

	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"

	genapi "saas-identity-platform-go/internal/gen/api"
)

// SessionsLogin POST /api/v1/auth/login —— 密码登录（M01.F04.I01）。
func (s *Server) SessionsLogin(w http.ResponseWriter, r *http.Request) {
	var body genapi.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	// 必填校验顺序镜像 rails login_params：username → password → clientId
	for _, kv := range []struct {
		key string
		val string
	}{
		{"username", body.Username},
		{"password", body.Password},
		{"clientId", body.ClientId},
	} {
		if kv.val == "" {
			familyerrors.Write(w, familyerrors.ErrBadRequest(kv.key+": is required"))
			return
		}
	}
	svc := service.LoginService{
		DB:       s.DB,
		Username: body.Username,
		Password: body.Password,
		ClientID: body.ClientId,
	}
	payload, err := svc.Call()
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// SessionsLogout POST /api/v1/auth/logout —— 204 恒定，无任何 token 撤销副作用
// （springboot logout 镜像，M01.F04.I06）。
func (s *Server) SessionsLogout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
