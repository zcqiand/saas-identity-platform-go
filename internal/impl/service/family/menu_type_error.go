package family

import "errors"

// 消息与 rails ArgumentError 字面一致（CT 全等断言依赖）。
func errBadMenuType() error {
	return errors.New("type: must be one of directory/menu/button")
}
