package errors

// errors_test.go — 错误形状映射（镜像 rails render_error / ktor StatusPages）：
// 错误体 {code,message}；423=空体；未知错误→500 INTERNAL 不泄露细节。

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusAndBodyPerKind(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string // "" 表示空体
	}{
		{"invalid credentials", ErrInvalidCredentials(), 401, `{"code":"INVALID_CREDENTIALS","message":"invalid credentials"}`},
		{"forbidden", ErrForbidden("forbidden"), 403, `{"code":"FORBIDDEN","message":"forbidden"}`},
		{"not found", ErrNotFound("record not found"), 404, `{"code":"NOT_FOUND","message":"record not found"}`},
		{"bad request", ErrBadRequest("bad request"), 400, `{"code":"BAD_REQUEST","message":"bad request"}`},
		{"locked empty body", ErrLocked(), 423, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Write(rec, tc.err)
			if rec.Code != tc.wantStatus {
				t.Fatalf("期望 %d，得到 %d", tc.wantStatus, rec.Code)
			}
			got := strings.TrimRight(rec.Body.String(), "\n")
			if got != tc.wantBody {
				t.Fatalf("体不符：got %q want %q", got, tc.wantBody)
			}
		})
	}
}

func TestUnknownErrorBecomes500Internal(t *testing.T) {
	rec := httptest.NewRecorder()
	Write(rec, errors.New("secret db dsn leaked"))
	if rec.Code != 500 {
		t.Fatalf("未知错误应 500，得到 %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret db dsn leaked") {
		t.Fatalf("500 不得泄露内部错误细节：%s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"code":"INTERNAL"`) {
		t.Fatalf("500 体应含 code=INTERNAL：%s", rec.Body.String())
	}
}

func TestAsRecoversTypedError(t *testing.T) {
	var target *E
	ok := As(ErrLocked(), &target)
	if !ok || target.Status != 423 {
		t.Fatal("As 应还原 *E")
	}
	if As(errors.New("plain"), &target) {
		t.Fatal("非 E 错误不应命中")
	}
}
