package testsupport

// db_test.go — WithTx 事务回滚 harness 自测：tx 内写入（含 DDL，PG 事务性 DDL）
// 在 WithTx 返回后必须不可见——任何不回滚=污染 saas_test 种子态，rails L4 交叉证据吃红。

import (
	"testing"

	"gorm.io/gorm"
)

func TestWithTxRollsBackEverything(t *testing.T) {
	WithTx(t, func(tx *gorm.DB) {
		if err := tx.Exec(`create table testsupport_probe(id int)`).Error; err != nil {
			t.Fatalf("tx 内建表失败: %v", err)
		}
		if err := tx.Exec(`insert into testsupport_probe values (1)`).Error; err != nil {
			t.Fatalf("tx 内写入失败: %v", err)
		}
	})
	// 新连接视角：探针表应不存在（整事务已回滚）
	var n int
	if err := DB(t).Raw(
		`select count(*) from information_schema.tables where table_name = 'testsupport_probe'`,
	).Scan(&n).Error; err != nil {
		t.Fatalf("探针查询失败: %v", err)
	}
	if n != 0 {
		t.Fatal("WithTx 返回后写入未回滚——harness 失职，会污染 saas_test")
	}
}
