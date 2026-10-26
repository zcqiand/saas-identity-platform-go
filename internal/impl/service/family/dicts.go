// Package family — 家族服务件（镜像 rails app/services/family/*）：
// 字典映射与成员视图装配。smallint 枚举 ↔ API 字符串的映射分支与
// springboot 各 assembler 逐分支同构。
package family

// MemberStatus tenant_member.status：1=active 2=invited 3=suspended 其余=disabled。
func MemberStatus(v int16) string {
	switch v {
	case 1:
		return "active"
	case 2:
		return "invited"
	case 3:
		return "suspended"
	default:
		return "disabled"
	}
}

// MemberStatusCode 逆映射：active→1 invited→2 suspended→3 其余→0。
func MemberStatusCode(s string) int16 {
	switch s {
	case "active":
		return 1
	case "invited":
		return 2
	case "suspended":
		return 3
	default:
		return 0
	}
}

// UserStatus sys_user.status：无 suspended 档（invitations / members 视图用）。
func UserStatus(v int16) string {
	switch v {
	case 1:
		return "active"
	case 2:
		return "invited"
	default:
		return "disabled"
	}
}

// MenuType sys_menu.type：1=directory 2=menu 3=button，未知→menu。
func MenuType(v int16) string {
	switch v {
	case 1:
		return "directory"
	case 3:
		return "button"
	default:
		return "menu"
	}
}

// MenuTypeCode 逆映射：未知值报错（写入路径 fail-fast，不猜）。
func MenuTypeCode(s string) (int16, error) {
	switch s {
	case "directory":
		return 1, nil
	case "menu":
		return 2, nil
	case "button":
		return 3, nil
	default:
		return 0, errBadMenuType()
	}
}
