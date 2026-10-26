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

// SessionsLogin / SessionsLogout 落地见 auth_impl.go（tag: auth 已接管）。
func (s *Server) OAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
func (s *Server) OAuthToken(w http.ResponseWriter, r *http.Request) {
	notImplemented(w)
}
