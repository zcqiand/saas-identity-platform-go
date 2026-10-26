// clients_impl.go — tag: clients 公共元数据（M04.F01.I06）。匿名可读，不挂守卫
// （/api/v1/clients/单段 白名单镜像）。
package api

import (
	"net/http"

	familyerrors "saas-identity-platform-go/internal/impl/errors"
	"saas-identity-platform-go/internal/impl/render"
	"saas-identity-platform-go/internal/impl/service"
)

// ClientsGetClient GET /clients/{clientId} —— 匿名元数据。
func (s *Server) ClientsGetClient(w http.ResponseWriter, r *http.Request, clientId string) {
	payload, err := (&service.ClientsService{DB: s.DB}).Get(clientId)
	if err != nil {
		familyerrors.Write(w, err)
		return
	}
	render.WriteJSON(w, http.StatusOK, payload)
}
