package auth

// password_test.go — 密码校验全口径（rails SysUser#password_matches? 同语义）：
// plain: 前缀直等 + bcrypt 兜底 + 非法 hash → false（kotlin 只实现 plain 不照抄）。

import "testing"

const bcryptHash = "$2a$04$2ZWzTq7OtNAUiSv.S2kzvuotpsHrTydjuXsXnDG9FM6HmLnheSo2u" // bcrypt.GenerateFromPassword("password", MinCost) 实算

func TestPlainPrefixDirectMatch(t *testing.T) {
	if !PasswordMatches("plain:dev123456", "dev123456") {
		t.Fatal("plain: 前缀直等应命中")
	}
	if PasswordMatches("plain:dev123456", "wrong") {
		t.Fatal("plain: 直等错密码不得命中")
	}
}

func TestBcryptFallback(t *testing.T) {
	if !PasswordMatches(bcryptHash, "password") {
		t.Fatal("bcrypt 正确密码应命中")
	}
	if PasswordMatches(bcryptHash, "wrong") {
		t.Fatal("bcrypt 错密码不得命中")
	}
}

func TestInvalidHashReturnsFalse(t *testing.T) {
	for _, hash := range []string{"", "not-a-hash", "$2a$10$truncated"} {
		if PasswordMatches(hash, "anything") {
			t.Fatalf("非法 hash %q 应返回 false 不 panic", hash)
		}
	}
}
