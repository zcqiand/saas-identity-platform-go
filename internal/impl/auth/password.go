package auth

import "golang.org/x/crypto/bcrypt"

// PasswordMatches 密码校验全口径（rails SysUser#password_matches? 同语义）：
// plain: 前缀直等（家族 dev 种子约定）+ bcrypt 兜底 + 非法 hash → false（不 panic）。
func PasswordMatches(hash, raw string) bool {
	if hash == "plain:"+raw {
		return true
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}
