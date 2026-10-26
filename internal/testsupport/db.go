// Package testsupport — 测试期真库 harness：连 saas_test（.env.test 提供
// DATABASE_URL），WithTx 以真事务包裹、Cleanup 强制回滚——任何写入不落库，
// 保护种子态不被污染（rails L4 交叉证据依赖同库干净）。
package testsupport

import (
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB 连 saas_test 并登记连接清理。DATABASE_URL 从进程 env 或仓根 .env.test
// 补载（go test 无 dotenv），仍缺则 Fatal——禁默认值兜底（suite-hard-rules §1）。
func DB(t *testing.T) *gorm.DB {
	t.Helper()
	loadEnvTest(t)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("缺少 DATABASE_URL（进程 env 与仓根 .env.test 均未提供），禁默认值兜底")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// WithTx 开真事务执行 fn，t.Cleanup 强制 Rollback（fn 正常返回也不提交）。
func WithTx(t *testing.T, fn func(tx *gorm.DB)) {
	t.Helper()
	tx := DB(t).Begin()
	if tx.Error != nil {
		t.Fatalf("开启事务失败: %v", tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	fn(tx)
}

// loadEnvTest 从 cwd 逐级上溯找 go.mod 定位仓根，把 .env.test 中尚未设置的
// 键补进进程 env（已显式设置的优先，不覆盖）。
func loadEnvTest(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("取 cwd 失败: %v", err)
	}
	for {
		if _, err := os.Stat(dir + string(os.PathSeparator) + "go.mod"); err == nil {
			break
		}
		parent := dir[:strings.LastIndexAny(dir, `/\`)]
		if parent == "" || parent == dir {
			t.Fatal("未找到 go.mod（仓根），无法定位 .env.test")
		}
		dir = parent
	}
	raw, err := os.ReadFile(dir + string(os.PathSeparator) + ".env.test")
	if err != nil {
		return // 无 .env.test 时交由调用方对必需键 fail-fast
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(strings.TrimSpace(key), strings.TrimSpace(val))
		}
	}
}
