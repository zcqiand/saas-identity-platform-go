// oauth_impl.go — tag: oauth 的 ServerInterface 落地（M04.F03）。authorize 手动
// Bearer decode（须含 tenant_id claim），token 端点匿名（permitAll 镜像）。
package api

import (
	"encoding/json"
	"net/http"

	genapi "saas-identity-platform-go/internal/gen/api"
	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	oauthsvc "saas-identity-platform-go/internal/impl/service/oauth"
)

// OAuthAuthorize POST /oauth/authorize —— Bearer + tenant_id claim 必需（M04.F03.I01）。
func (s *Server) OAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.auth(w, r)
	if !ok {
		return
	}
	tenantID, _ := claims["tenant_id"].(string)
	if tenantID == "" {
		// rails decode_bearer! 后续 @payload['tenant_id'] 缺失 → 401 镜像
		familyerrors.Write(w, familyerrors.ErrInvalidCredentials())
		return
	}
	userID, _ := claims["sub"].(string)
	var body genapi.AuthorizeCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	scope := ""
	if body.Scope != nil {
		scope = *body.Scope
	}
	payload, err := (&oauthsvc.AuthorizeService{
		DB: s.DB, ClientID: body.ClientId, RedirectURI: body.RedirectUri,
		ResponseType: string(body.ResponseType), Scope: scope, State: body.State,
		UserID: userID, TenantID: tenantID,
	}).Call()
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}

// OAuthToken POST /oauth/token —— 匿名；INVALID_* 走 400 {code,message}（M04.F03.I02/I03）。
func (s *Server) OAuthToken(w http.ResponseWriter, r *http.Request) {
	var body genapi.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		familyerrors.Write(w, familyerrors.ErrBadRequest("malformed JSON body"))
		return
	}
	code, refreshToken, redirectURI := "", "", ""
	if body.Code != nil {
		code = *body.Code
	}
	if body.RefreshToken != nil {
		refreshToken = *body.RefreshToken
	}
	if body.RedirectUri != nil {
		redirectURI = *body.RedirectUri
	}
	payload, err := (&oauthsvc.TokenService{
		DB: s.DB, GrantType: string(body.GrantType), ClientID: body.ClientId,
		Code: code, RefreshToken: refreshToken, RedirectURI: redirectURI,
	}).Call()
	if err != nil {
		if errorCode, message, ok := oauthsvc.AsFamilyE(err); ok {
			familyerrors.Write(w, &familyerrors.E{
				Status:  http.StatusBadRequest,
				Code:    errorCode,
				Message: message,
			})
			return
		}
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}
